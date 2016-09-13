package terraform

// NodeResource is a graph node for referencing a resource.
type NodeResource struct {
	Addr *ResourceAddress // Addr is the address for this resource
}

func (n *NodeResource) Name() string {
	return n.Addr.String()
}
