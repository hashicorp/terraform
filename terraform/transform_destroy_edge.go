package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDestroyer must be implemented by nodes that destroy resources.
type GraphNodeDestroyer interface {
	dag.Vertex

	// DestroyAddr is the address of the resource that is being
	// destroyed by this node. If this returns nil, then this node
	// is not destroying anything.
	DestroyAddr() *addrs.AbsResourceInstance
}

// GraphNodeCreator must be implemented by nodes that create OR update resources.
type GraphNodeCreator interface {
	// CreateAddr is the address of the resource being created or updated
	CreateAddr() *addrs.AbsResourceInstance
}

// DestroyEdgeTransformer is a GraphTransformer that creates the proper
// references for destroy resources. Destroy resources are more complex
// in that they must be depend on the destruction of resources that
// in turn depend on the CREATION of the node being destroy.
//
// That is complicated. Visually:
//
//   B_d -> A_d -> A -> B
//
// Notice that A destroy depends on B destroy, while B create depends on
// A create. They're inverted. This must be done for example because often
// dependent resources will block parent resources from deleting. Concrete
// example: VPC with subnets, the VPC can't be deleted while there are
// still subnets.
type DestroyEdgeTransformer struct {
	// These are needed to properly build the graph of dependencies
	// to determine what a destroy node depends on. Any of these can be nil.
	Config *configs.Config
	State  *states.State

	// If configuration is present then Schemas is required in order to
	// obtain schema information from providers and provisioners in order
	// to properly resolve implicit dependencies.
	Schemas *Schemas
}

func (t *DestroyEdgeTransformer) Transform(g *Graph) error {
	// Build a map of what is being destroyed (by address string) to
	// the list of destroyers.
	destroyers := make(map[string][]GraphNodeDestroyer)

	// Record the creators, which will need to depend on the destroyers if they
	// are only being updated.
	creators := make(map[string]GraphNodeCreator)

	// destroyersByResource records each destroyer by the ConfigResource
	// address.  We use this because dependencies are only referenced as
	// resources and have no index or module instance information, but we will
	// want to connect all the individual instances for correct ordering.
	destroyersByResource := make(map[string][]GraphNodeDestroyer)
	for _, v := range g.Vertices() {
		switch n := v.(type) {
		case GraphNodeDestroyer:
			addrP := n.DestroyAddr()
			if addrP == nil {
				log.Printf("[WARN] DestroyEdgeTransformer: %q (%T) has no destroy address", dag.VertexName(n), v)
				continue
			}
			addr := *addrP

			key := addr.String()
			log.Printf("[TRACE] DestroyEdgeTransformer: %q (%T) destroys %s", dag.VertexName(n), v, key)
			destroyers[key] = append(destroyers[key], n)

			resAddr := addr.ContainingResource().Config().String()
			destroyersByResource[resAddr] = append(destroyersByResource[resAddr], n)
		case GraphNodeCreator:
			addr := n.CreateAddr()
			creators[addr.String()] = n
		}
	}

	// If we aren't destroying anything, there will be no edges to make
	// so just exit early and avoid future work.
	if len(destroyers) == 0 {
		return nil
	}

	// Connect destroy despendencies as stored in the state
	for _, ds := range destroyers {
		for _, des := range ds {
			ri, ok := des.(GraphNodeResourceInstance)
			if !ok {
				continue
			}

			for _, resAddr := range ri.StateDependencies() {
				for _, desDep := range destroyersByResource[resAddr.String()] {
					if !graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(desDep, des) {
						log.Printf("[TRACE] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(desDep), dag.VertexName(des))
						g.Connect(dag.BasicEdge(desDep, des))
					} else {
						log.Printf("[TRACE] DestroyEdgeTransformer: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(desDep), dag.VertexName(des))
					}
				}
			}
		}
	}

	// connect creators to any destroyers on which they may depend
	for _, c := range creators {
		ri, ok := c.(GraphNodeResourceInstance)
		if !ok {
			continue
		}

		for _, resAddr := range ri.StateDependencies() {
			for _, desDep := range destroyersByResource[resAddr.String()] {
				if !graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(c, desDep) {
					log.Printf("[TRACE] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(c), dag.VertexName(desDep))
					g.Connect(dag.BasicEdge(c, desDep))
				} else {
					log.Printf("[TRACE] DestroyEdgeTransformer: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(c), dag.VertexName(desDep))
				}
			}
		}
	}

	// Go through and connect creators to destroyers. Going along with
	// our example, this makes: A_d => A
	for _, v := range g.Vertices() {
		cn, ok := v.(GraphNodeCreator)
		if !ok {
			continue
		}

		addr := cn.CreateAddr()
		if addr == nil {
			continue
		}

		for _, d := range destroyers[addr.String()] {
			// For illustrating our example
			a_d := d.(dag.Vertex)
			a := v

			log.Printf(
				"[TRACE] DestroyEdgeTransformer: connecting creator %q with destroyer %q",
				dag.VertexName(a), dag.VertexName(a_d))

			g.Connect(dag.BasicEdge(a, a_d))

			// Attach the destroy node to the creator
			// There really shouldn't be more than one destroyer, but even if
			// there are, any of them will represent the correct
			// CreateBeforeDestroy status.
			if n, ok := cn.(GraphNodeAttachDestroyer); ok {
				if d, ok := d.(GraphNodeDestroyerCBD); ok {
					n.AttachDestroyNode(d)
				}
			}
		}
	}

	return nil
}

// Remove any nodes that aren't needed when destroying modules.
// Variables, outputs, locals, and expanders may not be able to evaluate
// correctly, so we can remove these if nothing depends on them. The module
// closers also need to disable their use of expansion if the module itself is
// no longer present.
type pruneUnusedNodesTransformer struct {
}

func (t *pruneUnusedNodesTransformer) Transform(g *Graph) error {
	// We need a reverse depth first walk of modules, processing them in order
	// from the leaf modules to the root. This allows us to remove unneeded
	// dependencies from child modules, freeing up nodes in the parent module
	// to also be removed.

	nodes := g.Vertices()

	for removed := true; removed; {
		removed = false

		for i := 0; i < len(nodes); i++ {
			// run this in a closure, so we can return early rather than
			// dealing with complex looping and labels
			func() {
				n := nodes[i]
				switch n := n.(type) {
				case graphNodeTemporaryValue:
					// root module outputs indicate they are not temporary by
					// returning false here.
					if !n.temporaryValue() {
						return
					}

					// temporary values, which consist of variables, locals,
					// and outputs, must be kept if anything refers to them.
					for _, v := range g.UpEdges(n) {
						// keep any value which is connected through a
						// reference
						if _, ok := v.(GraphNodeReferencer); ok {
							return
						}
					}

				case graphNodeExpandsInstances:
					// Any nodes that expand instances are kept when their
					// instances may need to be evaluated.
					for _, v := range g.UpEdges(n) {
						switch v.(type) {
						case graphNodeExpandsInstances:
							// expanders can always depend on module expansion
							// themselves
							return
						case GraphNodeResourceInstance:
							// resource instances always depend on their
							// resource node, which is an expander
							return
						}
					}

				default:
					return
				}

				log.Printf("[DEBUG] pruneUnusedNodes: %s is no longer needed, removing", dag.VertexName(n))
				g.Remove(n)
				removed = true

				// remove the node from our iteration as well
				last := len(nodes) - 1
				nodes[i], nodes[last] = nodes[last], nodes[i]
				nodes = nodes[:last]
			}()
		}
	}

	return nil
}
