package terraform

// EvalNoop is an EvalNode that does nothing.
type EvalNoop struct{}

func (EvalNoop) Eval(EvalContext) (interface{}, error) {
	return nil, nil
}
