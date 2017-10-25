package testharness

// A Waiter is a readable channel that will never produce any items but
// will eventually be closed to signal when a consumer may proceed.
//
// A Waiter may be used just like any other readable channel, but it also
// has a convenience method Wait that provides a concise way to block until
// the waiter is closed.
type Waiter <-chan struct{}

// Wait blocks until the receiver's channel is closed.
func (w Waiter) Wait() {
	if w == nil {
		// Should never happen, but if it does we'd rather end immediately
		// than spin forever.
		return
	}
	for {
		_, ok := <-w
		if !ok {
			return
		}
	}
}
