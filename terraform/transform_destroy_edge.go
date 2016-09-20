package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDestroyer must be implemented by nodes that destroy resources.
type GraphNodeDestroyer interface {
	dag.Vertex

	// ResourceAddr is the address of the resource that is being
	// destroyed by this node. If this returns nil, then this node
	// is not destroying anything.
	DestroyAddr() *ResourceAddress
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
	// Module and State are only needed to look up dependencies in
	// any way possible. Either can be nil if not availabile.
	Module *module.Tree
	State  *State
}

func (t *DestroyEdgeTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] DestroyEdgeTransformer: Beginning destroy edge transformation...")

	// Build a map of what is being destroyed (by address string) to
	// the list of destroyers. In general there will only be one destroyer
	// but to make it more robust we support multiple.
	destroyers := make(map[string][]GraphNodeDestroyer)
	for _, v := range g.Vertices() {
		dn, ok := v.(GraphNodeDestroyer)
		if !ok {
			continue
		}

		addr := dn.DestroyAddr()
		if addr == nil {
			continue
		}

		key := addr.String()
		log.Printf(
			"[TRACE] DestroyEdgeTransformer: %s destroying %q",
			dag.VertexName(dn), key)
		destroyers[key] = append(destroyers[key], dn)
	}

	// If we aren't destroying anything, there will be no edges to make
	// so just exit early and avoid future work.
	if len(destroyers) == 0 {
		return nil
	}

	// This is strange but is the easiest way to get the dependencies
	// of a node that is being destroyed. We use another graph to make sure
	// the resource is in the graph and ask for references. We have to do this
	// because the node that is being destroyed may NOT be in the graph.
	//
	// Example: resource A is force new, then destroy A AND create A are
	// in the graph. BUT if resource A is just pure destroy, then only
	// destroy A is in the graph, and create A is not.
	steps := []GraphTransformer{
		&AttachResourceConfigTransformer{Module: t.Module},
		&AttachStateTransformer{State: t.State},
	}

	// Go through the all destroyers and find what they're destroying.
	// Use this to find the dependencies, look up if any of them are being
	// destroyed, and to make the proper edge.
	for d, dns := range destroyers {
		// d is what is being destroyed. We parse the resource address
		// which it came from it is a panic if this fails.
		addr, err := ParseResourceAddress(d)
		if err != nil {
			panic(err)
		}

		// This part is a little bit weird but is the best way to
		// find the dependencies we need to: build a graph and use the
		// attach config and state transformers then ask for references.
		node := &NodeApplyableResource{Addr: addr}
		{
			var g Graph
			g.Add(node)
			for _, s := range steps {
				if err := s.Transform(&g); err != nil {
					return err
				}
			}
		}

		// Get the references of the creation node. If it has none,
		// then there are no edges to make here.
		prefix := modulePrefixStr(normalizeModulePath(addr.Path))
		deps := modulePrefixList(node.References(), prefix)
		log.Printf(
			"[TRACE] DestroyEdgeTransformer: creation of %q depends on %#v",
			d, deps)
		if len(deps) == 0 {
			continue
		}

		// We have dependencies, check if any are being destroyed
		// to build the list of things that we must depend on!
		//
		// In the example of the struct, if we have:
		//
		//   B_d => A_d => A => B
		//
		// Then at this point in the algorithm we started with A_d,
		// we built A (to get dependencies), and we found B. We're now looking
		// to see if B_d exists.
		var depDestroyers []dag.Vertex
		for _, d := range deps {
			if ds, ok := destroyers[d]; ok {
				for _, d := range ds {
					depDestroyers = append(depDestroyers, d.(dag.Vertex))
					log.Printf(
						"[TRACE] DestroyEdgeTransformer: destruction of %q depends on %s",
						addr.String(), dag.VertexName(d))
				}
			}
		}

		// Go through and make the connections. Use the variable
		// names "a_d" and "b_d" to reference our example.
		for _, a_d := range dns {
			for _, b_d := range depDestroyers {
				g.Connect(dag.BasicEdge(b_d, a_d))
			}
		}
	}

	return nil
}
