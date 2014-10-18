package resource

import (
	"time"
)

// RetryFunc is the function retried until it succeeds.
type RetryFunc func() error

// Retry is a basic wrapper around StateChangeConf that will just retry
// a function until it no longer returns an error.
func Retry(timeout time.Duration, f RetryFunc) error {
	var err error
	c := &StateChangeConf{
		Pending:    []string{"error"},
		Target:     "success",
		Timeout:    timeout,
		MinTimeout: 500 * time.Millisecond,
		Refresh: func() (interface{}, string, error) {
			err = f()
			if err == nil {
				return 42, "success", nil
			}

			if rerr, ok := err.(RetryError); ok {
				err = rerr.Err
				return nil, "quit", err
			}

			return 42, "error", nil
		},
	}

	c.WaitForState()
	return err
}

// RetryError, if returned, will quit the retry immediately with the
// Err.
type RetryError struct {
	Err error
}

func (e RetryError) Error() string {
	return e.Err.Error()
}
