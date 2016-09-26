package shadow

import (
	"sync"
)

// Value is a struct that coordinates a value between two
// parallel routines. It is similar to atomic.Value except that when
// Value is called if it isn't set it will wait for it.
type Value struct {
	lock     sync.Mutex
	cond     *sync.Cond
	value    interface{}
	valueSet bool
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
