package terraform

// EvalEarlyExitError is a special error return value that can be returned
// by eval nodes that does an early exit.
type EvalEarlyExitError struct{}

func (EvalEarlyExitError) Error() string { return "early exit" }
