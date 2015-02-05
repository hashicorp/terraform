package terraform

// EvalSequence is an EvalNode that evaluates in sequence.
type EvalSequence struct {
	Nodes []EvalNode
}

func (n *EvalSequence) Args() ([]EvalNode, []EvalType) {
	types := make([]EvalType, len(n.Nodes))
	for i, n := range n.Nodes {
		types[i] = n.Type()
	}

	return n.Nodes, types
}

func (n *EvalSequence) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	// TODO: test
	if len(args) == 0 {
		return nil, nil
	}

	return args[len(args)-1], nil
}

func (n *EvalSequence) Type() EvalType {
	if len(n.Nodes) == 0 {
		return EvalTypeNull
	}

	return n.Nodes[len(n.Nodes)-1].Type()
}
