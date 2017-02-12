package shadow

import (
	"container/list"
	"sync"
)

// OrderedValue is a struct that keeps track of a value in the order
// it is set. Each time Value() is called, it will return the most recent
// calls value then discard it.
//
// This is unlike Value that returns the same value once it is set.
type OrderedValue struct {
	lock    sync.Mutex
	values  *list.List
	waiters *list.List
}

// Value returns the last value that was set, or blocks until one
// is received.
func (w *OrderedValue) Value() interface{} {
	w.lock.Lock()

	// If we have a pending value already, use it
	if w.values != nil && w.values.Len() > 0 {
		front := w.values.Front()
		w.values.Remove(front)
		w.lock.Unlock()
		return front.Value
	}

	// No pending value, create a waiter
	if w.waiters == nil {
		w.waiters = list.New()
	}

	var val Value
	w.waiters.PushBack(&val)
	w.lock.Unlock()

	// Return the value once we have it
	return val.Value()
}

// SetValue sets the latest value.
func (w *OrderedValue) SetValue(v interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// If we have a waiter, notify it
	if w.waiters != nil && w.waiters.Len() > 0 {
		front := w.waiters.Front()
		w.waiters.Remove(front)

		val := front.Value.(*Value)
		val.SetValue(v)
		return
	}

	// Add it to the list of values
	if w.values == nil {
		w.values = list.New()
	}

	w.values.PushBack(v)
}
