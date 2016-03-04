package resource

import (
	"sync"
	"time"
)

// RetryFunc is the function retried until it succeeds.
type RetryFunc func() error

// Retry is a basic wrapper around StateChangeConf that will just retry
// a function until it no longer returns an error.
func Retry(timeout time.Duration, f RetryFunc) error {
	// These are used to pull the error out of the function; need a mutex to
	// avoid a data race.
	var resultErr error
	var resultErrMu sync.Mutex

	c := &StateChangeConf{
		Pending:    []string{"error"},
		Target:     []string{"success"},
		Timeout:    timeout,
		MinTimeout: 500 * time.Millisecond,
		Refresh: func() (interface{}, string, error) {
			err := f()
			if err == nil {
				return 42, "success", nil
			}

			resultErrMu.Lock()
			defer resultErrMu.Unlock()
			resultErr = err
			if rerr, ok := err.(RetryError); ok {
				resultErr = rerr.Err
				return nil, "quit", rerr.Err
			}

			return 42, "error", nil
		},
	}

	c.WaitForState()

	// Need to acquire the lock here to be able to avoid race using resultErr as
	// the return value
	resultErrMu.Lock()
	defer resultErrMu.Unlock()
	return resultErr
}

// RetryError, if returned, will quit the retry immediately with the
// Err.
type RetryError struct {
	Err error
}

func (e RetryError) Error() string {
	return e.Err.Error()
}
