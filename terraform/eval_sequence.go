package terraform

// EvalSequence is an EvalNode that evaluates in sequence.
type EvalSequence struct {
	Nodes []EvalNode
}

func (n *EvalSequence) Eval(ctx EvalContext) (interface{}, error) {
	for _, n := range n.Nodes {
		if _, err := EvalRaw(n, ctx); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// EvalNodeFilterable impl.
func (n *EvalSequence) Filter(fn EvalNodeFilterFunc) {
	for i, node := range n.Nodes {
		n.Nodes[i] = fn(node)
	}
}
