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
	relAddr := addr.ProviderConfig

	return &EvalInitProvider{
		TypeName: relAddr.Type.LegacyString(),
		Addr:     addr.ProviderConfig,
	}
}
