package terraform

import (
	"strings"
)

// Semaphore is a wrapper around a channel to provide
// utility methods to clarify that we are treating the
// channel as a semaphore
type Semaphore chan struct{}

// NewSemaphore creates a semaphore that allows up
// to a given limit of simultaneous acquisitions
func NewSemaphore(n int) Semaphore {
	if n == 0 {
		panic("semaphore with limit 0")
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

// resourceProvider returns the provider name for the given type.
func resourceProvider(t, alias string) string {
	if alias != "" {
		return alias
	}

	idx := strings.IndexRune(t, '_')
	if idx == -1 {
		// If no underscores, the resource name is assumed to be
		// also the provider name, e.g. if the provider exposes
		// only a single resource of each type.
		return t
	}

	return t[:idx]
}

// strSliceContains checks if a given string is contained in a slice
// When anybody asks why Go needs generics, here you go.
func strSliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
