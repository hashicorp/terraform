package terraform

// GraphNodeDestroyer must be implemented by nodes that destroy resources.
type GraphNodeDestroyer interface {
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
type DestroyEdgeTransformer struct{}

func (t *DestroyEdgeTransformer) Transform(g *Graph) error {
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
		destroyers[key] = append(destroyers[key], dn)
	}

	// If we aren't destroying anything, there will be no edges to make
	// so just exit early and avoid future work.
	if len(destroyers) == 0 {
		return nil
	}

	// Go through the all destroyers and find what they're destroying.
	// Use this to find the dependencies, look up if any of them are being
	// destroyed, and to make the proper edge.
	for _, ds := range destroyers {
		for _, d := range ds {
			// TODO
			println(d)
		}
	}

	return nil
}
