package terraform

// EvalIf is an EvalNode that is a conditional.
type EvalIf struct {
	If   func(EvalContext) (bool, error)
	Then EvalNode
	Else EvalNode
}

// TODO: test
func (n *EvalIf) Eval(ctx EvalContext) (interface{}, error) {
	yes, err := n.If(ctx)
	if err != nil {
		return nil, err
	}

	if yes {
		return EvalRaw(n.Then, ctx)
	} else {
		if n.Else != nil {
			return EvalRaw(n.Else, ctx)
		}
	}

	return nil, nil
}
