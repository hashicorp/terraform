package terraform

// NodeEvalableProvider represents a provider during an "eval" walk.
// This special provider node type just initializes a provider and
// fetches its schema, without configuring it or otherwise interacting
// with it.
type NodeEvalableProvider struct {
	*NodeAbstractProvider
}

// GraphNodeExecutable
func (n *NodeEvalableProvider) Execute(ctx EvalContext, op walkOperation) error {
	_, err := ctx.InitProvider(n.Addr)
	return err
}
