package terraform

// EvalLiteral is an EvalNode implementation that returns a literal
// value. This is very useful for testing as well as in practice.
type EvalLiteral struct {
	Value     interface{}
	ValueType EvalType
}

func (n *EvalLiteral) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalLiteral) Eval(EvalContext, []interface{}) (interface{}, error) {
	return n.Value, nil
}

func (n *EvalLiteral) Type() EvalType {
	return n.ValueType
}
