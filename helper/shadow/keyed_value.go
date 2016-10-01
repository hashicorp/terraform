package shadow

import (
	"sync"
)

// KeyedValue is a struct that coordinates a value by key. If a value is
// not available for a give key, it'll block until it is available.
type KeyedValue struct {
	lock    sync.Mutex
	once    sync.Once
	values  map[string]interface{}
	waiters map[string]*Value
}

// Value returns the value that was set for the given key, or blocks
// until one is available.
func (w *KeyedValue) Value(k string) interface{} {
	w.lock.Lock()
	w.once.Do(w.init)

	// If we have this value already, return it
	if v, ok := w.values[k]; ok {
		w.lock.Unlock()
		return v
	}

	// No pending value, check for a waiter
	val := w.waiters[k]
	if val == nil {
		val = new(Value)
		w.waiters[k] = val
	}
	w.lock.Unlock()

	// Return the value once we have it
	return val.Value()
}

func (w *KeyedValue) SetValue(k string, v interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.once.Do(w.init)

	// Set the value, always
	w.values[k] = v

	// If we have a waiter, set it
	if val, ok := w.waiters[k]; ok {
		val.SetValue(v)
		w.waiters[k] = nil
	}
}

// Must be called with w.lock held.
func (w *KeyedValue) init() {
	w.values = make(map[string]interface{})
	w.waiters = make(map[string]*Value)
}
