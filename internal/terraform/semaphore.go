// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

// Semaphore is a wrapper around a channel to provide
// utility methods to clarify that we are treating the
// channel as a semaphore
type Semaphore chan struct{}

// NewSemaphore creates a semaphore that allows up
// to a given limit of simultaneous acquisitions
func NewSemaphore(n int) Semaphore {
	if n <= 0 {
		panic("semaphore with limit <=0")
	}
	ch := make(chan struct{}, n)
	return Semaphore(ch)
}

// Acquire is used to acquire an available slot.
// Blocks until available.
func (s Semaphore) Acquire() {
	s <- struct{}{}
}

// TryAcquire is used to do a non-blocking acquire.
// Returns a bool indicating success
func (s Semaphore) TryAcquire() bool {
	select {
	case s <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release is used to return a slot. Acquire must
// be called as a pre-condition.
func (s Semaphore) Release() {
	select {
	case <-s:
	default:
		panic("release without an acquire")
	}
}
