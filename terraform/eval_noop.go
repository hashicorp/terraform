package terraform

// EvalNoop is an EvalNode that does nothing.
type EvalNoop struct{}

func (EvalNoop) Args() ([]EvalNode, []EvalType) { return nil, nil }
func (EvalNoop) Eval(EvalContext, []interface{}) (interface{}, error) {
	return nil, nil
}
func (EvalNoop) Type() EvalType { return EvalTypeNull }
