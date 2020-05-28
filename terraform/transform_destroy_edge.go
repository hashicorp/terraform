package terraform

import (
	"log"
	"sort"

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

	// destroyersByResource records each destroyer by the AbsResourceAddress.
	// We use this because dependencies are only referenced as resources, but we
	// will want to connect all the individual instances for correct ordering.
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

			resAddr := addr.Resource.Resource.Absolute(addr.Module).String()
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
					log.Printf("[TRACE] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(desDep), dag.VertexName(des))
					g.Connect(dag.BasicEdge(desDep, des))

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
				log.Printf("[TRACE] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(c), dag.VertexName(desDep))
				g.Connect(dag.BasicEdge(c, desDep))

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

	// First collect the nodes into their respective modules based on
	// configuration path.
	moduleMap := make(map[string]pruneUnusedNodesMod)
	for _, v := range g.Vertices() {
		var path addrs.Module
		switch v := v.(type) {
		case GraphNodeModulePath:
			path = v.ModulePath()
		default:
			continue
		}
		m := moduleMap[path.String()]
		m.addr = path
		m.nodes = append(m.nodes, v)

		// We need to keep track of the closers, to make sure they don't look
		// for an expansion if there's nothing being expanded.
		if c, ok := v.(*nodeCloseModule); ok {
			m.closer = c
		}
		moduleMap[path.String()] = m
	}

	// now we need to restructure the modules so we can sort them
	var modules []pruneUnusedNodesMod

	for _, mod := range moduleMap {
		modules = append(modules, mod)
	}

	// Sort them by path length, longest first, so that start with the deepest
	// modules.  The order of modules at the same tree level doesn't matter, we
	// just need to ensure that child modules are processed before parent
	// modules.
	sort.Slice(modules, func(i, j int) bool {
		return len(modules[i].addr) > len(modules[j].addr)
	})

	for _, mod := range modules {
		mod.removeUnused(g)
	}

	return nil
}

// pruneUnusedNodesMod is a container to hold the nodes that belong to a
// particular configuration module for the pruneUnusedNodesTransformer
type pruneUnusedNodesMod struct {
	addr   addrs.Module
	nodes  []dag.Vertex
	closer *nodeCloseModule
}

// Remove any unused locals, variables, outputs and expanders.  Since module
// closers can also lookup expansion info to detect orphaned instances, disable
// them if their associated expander is removed.
func (m *pruneUnusedNodesMod) removeUnused(g *Graph) {
	// We modify the nodes slice during processing here.
	// Make a copy so no one is surprised by this changing in the future.
	nodes := make([]dag.Vertex, len(m.nodes))
	copy(nodes, m.nodes)

	// since we have no defined structure within the module, just cycle through
	// the nodes in each module until there are no more removals
	removed := true
	for {
		if !removed {
			return
		}
		removed = false

		for i := 0; i < len(nodes); i++ {
			// run this in a closure, so we can return early rather than
			// dealing with complex looping and labels
			func() {
				n := nodes[i]
				switch n.(type) {
				case graphNodeTemporaryValue:
					// temporary value, which consist of variables, locals, and
					// outputs, must be kept if anything refers to them.
					if n, ok := n.(GraphNodeModulePath); ok {
						// root outputs always have an implicit dependency on
						// remote state.
						if n.ModulePath().IsRoot() {
							return
						}
					}
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
}
