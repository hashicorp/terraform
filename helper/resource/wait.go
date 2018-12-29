package resource

import (
	"sync"
	"time"
)

// Retry is a basic wrapper around StateChangeConf that will just retry
// a function until it no longer returns an error.
func Retry(timeout time.Duration, f RetryFunc) error {
	// These are used to pull the error out of the function; need a mutex to
	// avoid a data race.
	var resultErr error
	var resultErrMu sync.Mutex

	c := &StateChangeConf{
		Pending:    []string{"retryableerror"},
		Target:     []string{"success"},
		Timeout:    timeout,
		MinTimeout: 500 * time.Millisecond,
		Refresh: func() (interface{}, string, error) {
			rerr := f()

			resultErrMu.Lock()
			defer resultErrMu.Unlock()

			if rerr == nil {
				resultErr = nil
				return 42, "success", nil
			}

			resultErr = rerr.Err

			if rerr.Retryable {
				return 42, "retryableerror", nil
			}
			return nil, "quit", rerr.Err
		},
	}

	_, waitErr := c.WaitForState()

	// Need to acquire the lock here to be able to avoid race using resultErr as
	// the return value
	resultErrMu.Lock()
	defer resultErrMu.Unlock()

	// resultErr may be nil because the wait timed out and resultErr was never
	// set; this is still an error
	if resultErr == nil {
		return waitErr
	}
	// resultErr takes precedence over waitErr if both are set because it is
	// more likely to be useful
	return resultErr
}

// RetryFunc is the function retried until it succeeds.
type RetryFunc func() *RetryError

// RetryError is the required return type of RetryFunc. It forces client code
// to choose whether or not a given error is retryable.
type RetryError struct {
	Err       error
	Retryable bool
}

// RetryableError is a helper to create a RetryError that's retryable from a
// given error.
func RetryableError(err error) *RetryError {
	if err == nil {
		return nil
	}
	return &RetryError{Err: err, Retryable: true}
}

// NonRetryableError is a helper to create a RetryError that's _not_ retryable
// from a given error.
func NonRetryableError(err error) *RetryError {
	if err == nil {
		return nil
	}
	return &RetryError{Err: err, Retryable: false}
}
