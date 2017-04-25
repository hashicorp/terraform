package shadow

import (
	"sync"
)

// ComparedValue is a struct that finds a value by comparing some key
// to the list of stored values. This is useful when there is no easy
// uniquely identifying key that works in a map (for that, use KeyedValue).
//
// ComparedValue is very expensive, relative to other Value types. Try to
// limit the number of values stored in a ComparedValue by potentially
// nesting it within a KeyedValue (a keyed value points to a compared value,
// for example).
type ComparedValue struct {
	// Func is a function that is given the lookup key and a single
	// stored value. If it matches, it returns true.
	Func func(k, v interface{}) bool

	lock    sync.Mutex
	once    sync.Once
	closed  bool
	values  []interface{}
	waiters map[interface{}]*Value
}

// Close closes the value. This can never fail. For a definition of
// "close" see the ErrClosed docs.
func (w *ComparedValue) Close() error {
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
func (w *ComparedValue) Value(k interface{}) interface{} {
	v, val := w.valueWaiter(k)
	if val == nil {
		return v
	}

	return val.Value()
}

// ValueOk gets the value for the given key, returning immediately if the
// value doesn't exist. The second return argument is true if the value exists.
func (w *ComparedValue) ValueOk(k interface{}) (interface{}, bool) {
	v, val := w.valueWaiter(k)
	return v, val == nil
}

func (w *ComparedValue) SetValue(v interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.once.Do(w.init)

	// Check if we already have this exact value (by simply comparing
	// with == directly). If we do, then we don't insert it again.
	found := false
	for _, v2 := range w.values {
		if v == v2 {
			found = true
			break
		}
	}

	if !found {
		// Set the value, always
		w.values = append(w.values, v)
	}

	// Go through the waiters
	for k, val := range w.waiters {
		if w.Func(k, v) {
			val.SetValue(v)
			delete(w.waiters, k)
		}
	}
}

func (w *ComparedValue) valueWaiter(k interface{}) (interface{}, *Value) {
	w.lock.Lock()
	w.once.Do(w.init)

	// Look for a pre-existing value
	for _, v := range w.values {
		if w.Func(k, v) {
			w.lock.Unlock()
			return v, nil
		}
	}

	// If we're closed, return that
	if w.closed {
		w.lock.Unlock()
		return ErrClosed, nil
	}

	// Pre-existing value doesn't exist, create a waiter
	val := w.waiters[k]
	if val == nil {
		val = new(Value)
		w.waiters[k] = val
	}
	w.lock.Unlock()

	// Return the waiter
	return nil, val
}

// Must be called with w.lock held.
func (w *ComparedValue) init() {
	w.waiters = make(map[interface{}]*Value)
	if w.Func == nil {
		w.Func = func(k, v interface{}) bool { return k == v }
	}
}
