package terraform

// EvalReturnError is an EvalNode implementation that returns an
// error if it is present.
//
// This is useful for scenarios where an error has been captured by
// another EvalNode (like EvalApply) for special EvalTree-based error
// handling, and that handling has completed, so the error should be
// returned normally.
type EvalReturnError struct {
	Error *error
}

func (n *EvalReturnError) Eval(ctx EvalContext) (interface{}, error) {
	if n.Error == nil {
		return nil, nil
	}

	return nil, *n.Error
}
