package terraform

// EvalIf is an EvalNode that is a conditional.
type EvalIf struct {
	If   func(EvalContext) (bool, error)
	Node EvalNode
}

func (n *EvalIf) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalIf) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	yes, err := n.If(ctx)
	if err != nil {
		return nil, err
	}

	if yes {
		return EvalRaw(n.Node, ctx)
	}

	return nil, nil
}

func (n *EvalIf) Type() EvalType {
	return EvalTypeNull
}
