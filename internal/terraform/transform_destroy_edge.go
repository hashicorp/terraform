package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
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
//	B_d -> A_d -> A -> B
//
// Notice that A destroy depends on B destroy, while B create depends on
// A create. They're inverted. This must be done for example because often
// dependent resources will block parent resources from deleting. Concrete
// example: VPC with subnets, the VPC can't be deleted while there are
// still subnets.
type DestroyEdgeTransformer struct{}

// tryInterProviderDestroyEdge checks if we're inserting a destroy edge
// across a provider boundary, and only adds the edge if it results in no cycles.
//
// FIXME: The cycles can arise in valid configurations when a provider depends
// on resources from another provider. In the future we may want to inspect
// the dependencies of the providers themselves, to avoid needing to use the
// blunt hammer of checking for cycles.
//
// A reduced example of this dependency problem looks something like:
/*

createA <-               createB
  |        \            /    |
  |         providerB <-     |
  v                     \    v
destroyA ------------->  destroyB

*/
//
// The edge from destroyA to destroyB would be skipped in this case, but there
// are still other combinations of changes which could connect the A and B
// groups around providerB in various ways.
//
// The most difficult problem here happens during a full destroy operation.
// That creates a special case where resources on which a provider depends must
// exist for evaluation before they are destroyed. This means that any provider
// dependencies must wait until all that provider's resources have first been
// destroyed. This is where these cross-provider edges are still required to
// ensure the correct order.
func (t *DestroyEdgeTransformer) tryInterProviderDestroyEdge(g *Graph, from, to dag.Vertex) {
	e := dag.BasicEdge(from, to)
	g.Connect(e)

	pc, ok := from.(GraphNodeProviderConsumer)
	if !ok {
		return
	}
	fromProvider := pc.Provider()

	pc, ok = to.(GraphNodeProviderConsumer)
	if !ok {
		return
	}
	toProvider := pc.Provider()

	sameProvider := fromProvider.Equals(toProvider)

	// Check for cycles, and back out the edge if there are any.
	// The cycles we are looking for only appears between providers, so don't
	// waste time checking for cycles if both nodes use the same provider.
	if !sameProvider && len(g.Cycles()) > 0 {
		log.Printf("[DEBUG] DestroyEdgeTransformer: skipping inter-provider edge %s->%s which creates a cycle",
			dag.VertexName(from), dag.VertexName(to))
		g.RemoveEdge(e)
	}
}

func (t *DestroyEdgeTransformer) Transform(g *Graph) error {
	// Build a map of what is being destroyed (by address string) to
	// the list of destroyers.
	destroyers := make(map[string][]GraphNodeDestroyer)

	// Record the creators, which will need to depend on the destroyers if they
	// are only being updated.
	creators := make(map[string][]GraphNodeCreator)

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
			addr := n.CreateAddr().ContainingResource().Config().String()
			creators[addr] = append(creators[addr], n)
		}
	}

	// If we aren't destroying anything, there will be no edges to make
	// so just exit early and avoid future work.
	if len(destroyers) == 0 {
		return nil
	}

	// Connect destroy dependencies as stored in the state
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
						t.tryInterProviderDestroyEdge(g, desDep, des)
					} else {
						log.Printf("[TRACE] DestroyEdgeTransformer: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(desDep), dag.VertexName(des))
					}
				}

				// We can have some create or update nodes which were
				// dependents of the destroy node. If they have no destroyer
				// themselves, make the connection directly from the creator.
				for _, createDep := range creators[resAddr.String()] {
					if !graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(createDep, des) {
						log.Printf("[DEBUG] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(createDep), dag.VertexName(des))
						t.tryInterProviderDestroyEdge(g, createDep, des)
					} else {
						log.Printf("[TRACE] DestroyEdgeTransformer: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(createDep), dag.VertexName(des))
					}
				}
			}
		}
	}

	// connect creators to any destroyers on which they may depend
	for _, cs := range creators {
		for _, c := range cs {
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
							// Root module output values (which the following
							// condition matches) are exempt because we know
							// there is only ever exactly one instance of the
							// root module, and so it's not actually important
							// to expand it and so this lets us do a bit more
							// pruning than we'd be able to do otherwise.
							if tmp, ok := v.(graphNodeTemporaryValue); ok && !tmp.temporaryValue() {
								continue
							}

							// expanders can always depend on module expansion
							// themselves
							return
						case GraphNodeResourceInstance:
							// resource instances always depend on their
							// resource node, which is an expander
							return
						}
					}

				case GraphNodeProvider:
					// Providers that may have been required by expansion nodes
					// that we no longer need can also be removed.
					if g.UpEdges(n).Len() > 0 {
						return
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
