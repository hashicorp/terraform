package terraform

// NodeApplyableProvider represents a provider during an apply.
type NodeApplyableProvider struct {
	*NodeAbstractProvider
}

// GraphNodeEvalable
func (n *NodeApplyableProvider) EvalTree() EvalNode {
	return ProviderEvalTree(n.NameValue, n.ProviderConfig())
}
