package slowmessage

import (
	"time"
)

// SlowFunc is the function that could be slow. Usually, you'll have to
// wrap an existing function in a lambda to make it match this type signature.
type SlowFunc func() error

// CallbackFunc is the function that is triggered when the threshold is reached.
type CallbackFunc func()

// Do calls sf. If threshold time has passed, cb is called. Note that this
// call will be made concurrently to sf still running.
func Do(threshold time.Duration, sf SlowFunc, cb CallbackFunc) error {
	// Call the slow function
	errCh := make(chan error, 1)
	go func() {
		errCh <- sf()
	}()

	// Wait for it to complete or the threshold to pass
	select {
	case err := <-errCh:
		return err
	case <-time.After(threshold):
		// Threshold reached, call the callback
		cb()
	}

	// Wait an indefinite amount of time for it to finally complete
	return <-errCh
}
