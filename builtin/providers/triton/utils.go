package triton

import (
	"errors"
	"time"
)

var (
	// ErrTimeout is returned when waiting for state change
	ErrTimeout = errors.New("timed out while waiting for resource change")
)

func waitFor(f func() (bool, error), every, timeout time.Duration) error {
	start := time.Now()

	for time.Since(start) <= timeout {
		stop, err := f()
		if err != nil {
			return err
		}

		if stop {
			return nil
		}

		time.Sleep(every)
	}

	return ErrTimeout
}
