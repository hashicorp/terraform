package structs

import (
	"fmt"
	"time"
)

const (
	// The default user that the executor uses to run tasks
	DefaultUnpriviledgedUser = "nobody"

	// CheckBufSize is the size of the check output result
	CheckBufSize = 4 * 1024
)

// WaitResult stores the result of a Wait operation.
type WaitResult struct {
	ExitCode int
	Signal   int
	Err      error
}

func NewWaitResult(code, signal int, err error) *WaitResult {
	return &WaitResult{
		ExitCode: code,
		Signal:   signal,
		Err:      err,
	}
}

func (r *WaitResult) Successful() bool {
	return r.ExitCode == 0 && r.Signal == 0 && r.Err == nil
}

func (r *WaitResult) String() string {
	return fmt.Sprintf("Wait returned exit code %v, signal %v, and error %v",
		r.ExitCode, r.Signal, r.Err)
}

// RecoverableError wraps an error and marks whether it is recoverable and could
// be retried or it is fatal.
type RecoverableError struct {
	Err         error
	Recoverable bool
}

// NewRecoverableError is used to wrap an error and mark it as recoverable or
// not.
func NewRecoverableError(e error, recoverable bool) *RecoverableError {
	return &RecoverableError{
		Err:         e,
		Recoverable: recoverable,
	}
}

func (r *RecoverableError) Error() string {
	return r.Err.Error()
}

// CheckResult encapsulates the result of a check
type CheckResult struct {

	// ExitCode is the exit code of the check
	ExitCode int

	// Output is the output of the check script
	Output string

	// Timestamp is the time at which the check was executed
	Timestamp time.Time

	// Duration is the time it took the check to run
	Duration time.Duration

	// Err is the error that a check returned
	Err error
}
