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

// Nodes that register instances in the instances.Expander are not needed
// during apply if there are no instances that will lookup the expansion. This
// is the case when a module tree is removed or during a full destroy, and we
// may not be able to evaluate the expansion expression.
type pruneUnusedExpanderTransformer struct {
}

func (t *pruneUnusedExpanderTransformer) Transform(g *Graph) error {
	// We need a reverse depth first walk of modules, but it needs to be
	// recursive so that we can process the lead modules first.

	// collect all nodes into their containing module
	type mod struct {
		addr  addrs.Module
		nodes []dag.Vertex
	}

	// first collect the nodes into their respective modules
	moduleMap := make(map[string]*mod)
	for _, v := range g.Vertices() {
		var path addrs.Module
		switch v := v.(type) {
		case instanceExpander:
			path = v.expandsInstances()

		case graphNodeModuleCloser:
			// module closers are connected like module calls, and report
			// their parent module address
			path = v.CloseModule()

		case GraphNodeModulePath:
			path = v.ModulePath()
		}
		m, ok := moduleMap[path.String()]
		if !ok {
			m = &mod{}
			moduleMap[path.String()] = m
		}

		m.addr = path
		m.nodes = append(m.nodes, v)
	}

	// now we need to restructure the modules so we can sort them
	var modules []*mod

	for _, mod := range moduleMap {
		modules = append(modules, mod)
	}

	// Sort them by path length, longest first, so that we process the deepest
	// modules first.  The order of modules at the same tree level doesn't
	// matter, we just need to ensure that child modules are processed before
	// parent modules.
	sort.Slice(modules, func(i, j int) bool {
		return len(modules[i].addr) > len(modules[j].addr)
	})

	for _, module := range modules {
		t.removeUnused(module.nodes, g)
	}

	return nil
}

func (t *pruneUnusedExpanderTransformer) removeUnused(nodes []dag.Vertex, g *Graph) {
	// since we have no defined structure within the module, just cycle through
	// the nodes until there are no more removals
	removed := true
	for {
		if !removed {
			return
		}
		removed = false

		last := len(nodes) - 1

	NEXT:
		for i := 0; i < len(nodes); i++ {
			n := nodes[i]
			switch n.(type) {
			case graphNodeTemporaryValue:
				if n, ok := n.(GraphNodeModulePath); ok {
					// root outputs always have a dependency on remote state
					if n.ModulePath().IsRoot() {
						continue NEXT
					}
				}
				for _, vv := range g.UpEdges(n) {
					if _, ok := vv.(GraphNodeReferencer); ok {
						continue NEXT
					}
				}

			case instanceExpander:
				for _, vv := range g.UpEdges(n) {
					if _, ok := vv.(requiresInstanceExpansion); ok {
						continue NEXT
					}
				}

			default:
				continue NEXT
			}

			removed = true

			//// connect through edges
			//for _, d := range g.DownEdges(n) {
			//    for _, u := range g.UpEdges(n) {
			//        g.Connect(dag.BasicEdge(u, d))
			//    }
			//}

			g.Remove(n)

			// remove the node from our iteration as well
			nodes[i], nodes[last] = nodes[last], nodes[i]
			nodes = nodes[:last]
			last--
		}
	}
}
