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
	closed  bool
}

// Close closes the value. This can never fail. For a definition of
// "close" see the ErrClosed docs.
func (w *KeyedValue) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	// Set closed to true always
	w.closed = true

	// For all waiters, complete with ErrClosed
	for k, val := range w.waiters {
		val.SetValue(ErrClosed)
		delete(w.waiters, k)
	}

	return nil
}

// Value returns the value that was set for the given key, or blocks
// until one is available.
func (w *KeyedValue) Value(k string) interface{} {
	w.lock.Lock()
	v, val := w.valueWaiter(k)
	w.lock.Unlock()

	// If we have no waiter, then return the value
	if val == nil {
		return v
	}

	// We have a waiter, so wait
	return val.Value()
}

// WaitForChange waits for the value with the given key to be set again.
// If the key isn't set, it'll wait for an initial value. Note that while
// it is called "WaitForChange", the value isn't guaranteed to _change_;
// this will return when a SetValue is called for the given k.
func (w *KeyedValue) WaitForChange(k string) interface{} {
	w.lock.Lock()
	w.once.Do(w.init)

	// If we're closed, we're closed
	if w.closed {
		w.lock.Unlock()
		return ErrClosed
	}

	// Check for an active waiter. If there isn't one, make it
	val := w.waiters[k]
	if val == nil {
		val = new(Value)
		w.waiters[k] = val
	}
	w.lock.Unlock()

	// And wait
	return val.Value()
}

// ValueOk gets the value for the given key, returning immediately if the
// value doesn't exist. The second return argument is true if the value exists.
func (w *KeyedValue) ValueOk(k string) (interface{}, bool) {
	w.lock.Lock()
	defer w.lock.Unlock()

	v, val := w.valueWaiter(k)
	return v, val == nil
}

func (w *KeyedValue) SetValue(k string, v interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.setValue(k, v)
}

// Init will initialize the key to a given value only if the key has
// not been set before. This is safe to call multiple times and in parallel.
func (w *KeyedValue) Init(k string, v interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// If we have a waiter, set the value.
	_, val := w.valueWaiter(k)
	if val != nil {
		w.setValue(k, v)
	}
}

// Must be called with w.lock held.
func (w *KeyedValue) init() {
	w.values = make(map[string]interface{})
	w.waiters = make(map[string]*Value)
}

// setValue is like SetValue but assumes the lock is held.
func (w *KeyedValue) setValue(k string, v interface{}) {
	w.once.Do(w.init)

	// Set the value, always
	w.values[k] = v

	// If we have a waiter, set it
	if val, ok := w.waiters[k]; ok {
		val.SetValue(v)
		delete(w.waiters, k)
	}
}

// valueWaiter gets the value or the Value waiter for a given key.
//
// This must be called with lock held.
func (w *KeyedValue) valueWaiter(k string) (interface{}, *Value) {
	w.once.Do(w.init)

	// If we have this value already, return it
	if v, ok := w.values[k]; ok {
		return v, nil
	}

	// If we're closed, return that
	if w.closed {
		return ErrClosed, nil
	}

	// No pending value, check for a waiter
	val := w.waiters[k]
	if val == nil {
		val = new(Value)
		w.waiters[k] = val
	}

	// Return the waiter
	return nil, val
}
