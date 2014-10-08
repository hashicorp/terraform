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
			if err != nil {
				return 42, "error", nil
			}

			return 42, "success", nil
		},
	}

	c.WaitForState()
	return err
}
