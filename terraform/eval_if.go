package terraform

// EvalIf is an EvalNode that is a conditional.
type EvalIf struct {
	If   func(EvalContext) (bool, error)
	Node EvalNode
}

// TODO: test
func (n *EvalIf) Eval(ctx EvalContext) (interface{}, error) {
	yes, err := n.If(ctx)
	if err != nil {
		return nil, err
	}

	if yes {
		return EvalRaw(n.Node, ctx)
	}

	return nil, nil
}
