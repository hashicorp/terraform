package terraform

// NodeEvalableProvider represents a provider during an "eval" walk.
// This special provider node type just initializes a provider and
// fetches its schema, without configuring it or otherwise interacting
// with it.
type NodeEvalableProvider struct {
	*NodeAbstractProvider
}

// GraphNodeEvalable
func (n *NodeEvalableProvider) EvalTree() EvalNode {
	addr := n.Addr

	return &EvalInitProvider{
		TypeName: addr.ProviderConfig.LocalName, // FIXME: Should be an addrs.Provider
		Addr:     addr,
	}
}
