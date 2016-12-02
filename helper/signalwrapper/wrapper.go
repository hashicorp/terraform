// Package signalwrapper is used to run functions that are sensitive to
// signals that may be received from outside the process. It can also be
// used as just an async function runner that is cancellable and we may
// abstract this further into another package in the future.
package signalwrapper

import (
	"log"
	"os"
	"os/signal"
	"sync"
)

// CancellableFunc is a function that cancels if it receives a message
// on the given channel. It must return an error if any occurred. It should
// return no error if it was cancelled successfully since it is assumed
// that this function will probably be called again at some future point
// since it was interrupted.
type CancellableFunc func(<-chan struct{}) error

// Run wraps and runs the given cancellable function and returns the Wrapped
// struct that can be used to listen for events, cancel on other events
// (such as timeouts), etc.
func Run(f CancellableFunc) *Wrapped {
	// Determine the signals we're listening to. Prematurely making
	// this a slice since I predict a future where we'll add others and
	// the complexity in doing so is low.
	signals := []os.Signal{os.Interrupt}

	// Register a listener for the signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)

	// Create the channel we'll use to "cancel"
	cancelCh := make(chan struct{})

	// This channel keeps track of whether the function we're running
	// completed successfully and the errors it may have had. It is
	// VERY IMPORTANT that the errCh is buffered to at least 1 so that
	// it doesn't block when finishing.
	doneCh := make(chan struct{})
	errCh := make(chan error, 1)

	// Build our wrapped result
	wrapped := &Wrapped{
		ErrCh:    errCh,
		errCh:    errCh,
		cancelCh: cancelCh,
	}

	// Start the function
	go func() {
		log.Printf("[DEBUG] signalwrapper: executing wrapped function")
		err := f(cancelCh)

		// Close the done channel _before_ sending the error in case
		// the error channel read blocks (it shouldn't) to avoid interrupts
		// doing anything.
		close(doneCh)

		// Mark completion
		log.Printf("[DEBUG] signalwrapper: wrapped function execution ended")
		wrapped.done(err)
	}()

	// Goroutine to track interrupts and make sure we do at-most-once
	// delivery of an interrupt since we're using a channel.
	go func() {
		// Clean up after this since this is the only function that
		// reads signals.
		defer signal.Stop(sigCh)

		select {
		case <-doneCh:
			// Everything happened naturally
		case <-sigCh:
			log.Printf("[DEBUG] signalwrapper: signal received, cancelling wrapped function")

			// Stop the function. Goroutine since we don't care about
			// the result and we'd like to end this goroutine as soon
			// as possible to avoid any more signals coming in.
			go wrapped.Cancel()
		}
	}()

	return wrapped
}

// Wrapped is the return value of wrapping a function. This has channels
// that can be used to wait for a result as well as functions to help with
// different behaviors.
type Wrapped struct {
	// Set and consumed by user

	// ErrCh is the channel to listen for real-time events on the wrapped
	// function. A nil error sent means the execution completed without error.
	// This is an exactly once delivery channel.
	ErrCh <-chan error

	// Set by creator
	errCh    chan<- error
	cancelCh chan<- struct{}

	// Set automatically
	once       sync.Once
	cancelCond *sync.Cond
	cancelLock *sync.Mutex
	resultErr  error
	resultSet  bool
}

// Cancel stops the running function and blocks until it returns. The
// resulting value is returned.
//
// It is safe to call this multiple times. This will return the resulting
// error value each time.
func (w *Wrapped) Cancel() error {
	w.once.Do(w.init)
	w.cancelLock.Lock()

	// If we have a result set, return that
	if w.resultSet {
		w.cancelLock.Unlock()
		return w.resultErr
	}

	// If we have a cancel channel, close it to signal and set it to
	// nil so we never do that again.
	if w.cancelCh != nil {
		close(w.cancelCh)
		w.cancelCh = nil
	}

	// Wait for the result to be set
	defer w.cancelLock.Unlock()
	w.cancelCond.Wait()
	return w.resultErr
}

// Wait waits for the completion of the wrapped function and returns the result.
//
// This can be called multiple times safely.
func (w *Wrapped) Wait() error {
	w.once.Do(w.init)
	w.cancelLock.Lock()
	defer w.cancelLock.Unlock()

	// If we don't have a result yet, wait for that
	if !w.resultSet {
		w.cancelCond.Wait()
	}

	// Return the result
	return w.resultErr
}

// done marks this wrapped function as done with the resulting value.
// This must only be called once.
func (w *Wrapped) done(err error) {
	w.once.Do(w.init)
	w.cancelLock.Lock()

	// Set the result
	w.resultErr = err
	w.resultSet = true

	// Notify any waiters
	w.cancelCond.Broadcast()

	// Unlock since the next call can be blocking
	w.cancelLock.Unlock()

	// Notify any channel listeners
	w.errCh <- err
}

func (w *Wrapped) init() {
	// Create the condition variable
	var m sync.Mutex
	w.cancelCond = sync.NewCond(&m)
	w.cancelLock = &m
}
