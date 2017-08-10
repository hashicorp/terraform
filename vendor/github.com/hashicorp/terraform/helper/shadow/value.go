package shadow

import (
	"errors"
	"sync"
)

// ErrClosed is returned by any closed values.
//
// A "closed value" is when the shadow has been notified that the real
// side is complete and any blocking values will _never_ be satisfied
// in the future. In this case, this error is returned. If a value is already
// available, that is still returned.
var ErrClosed = errors.New("shadow closed")

// Value is a struct that coordinates a value between two
// parallel routines. It is similar to atomic.Value except that when
// Value is called if it isn't set it will wait for it.
//
// The Value can be closed with Close, which will cause any future
// blocking operations to return immediately with ErrClosed.
type Value struct {
	lock     sync.Mutex
	cond     *sync.Cond
	value    interface{}
	valueSet bool
}

func (v *Value) Lock() {
	v.lock.Lock()
}

func (v *Value) Unlock() {
	v.lock.Unlock()
}

// Close closes the value. This can never fail. For a definition of
// "close" see the struct docs.
func (w *Value) Close() error {
	w.lock.Lock()
	set := w.valueSet
	w.lock.Unlock()

	// If we haven't set the value, set it
	if !set {
		w.SetValue(ErrClosed)
	}

	// Done
	return nil
}

// Value returns the value that was set.
func (w *Value) Value() interface{} {
	w.lock.Lock()
	defer w.lock.Unlock()

	// If we already have a value just return
	for !w.valueSet {
		// No value, setup the condition variable if we have to
		if w.cond == nil {
			w.cond = sync.NewCond(&w.lock)
		}

		// Wait on it
		w.cond.Wait()
	}

	// Return the value
	return w.value
}

// SetValue sets the value.
func (w *Value) SetValue(v interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// Set the value
	w.valueSet = true
	w.value = v

	// If we have a condition, clear it
	if w.cond != nil {
		w.cond.Broadcast()
		w.cond = nil
	}
}
