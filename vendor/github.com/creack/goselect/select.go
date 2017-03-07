package goselect

import (
	"syscall"
	"time"
)

// Select wraps syscall.Select with Go types
func Select(n int, r, w, e *FDSet, timeout time.Duration) error {
	var timeval *syscall.Timeval
	if timeout >= 0 {
		t := syscall.NsecToTimeval(timeout.Nanoseconds())
		timeval = &t
	}

	return sysSelect(n, r, w, e, timeval)
}

// RetrySelect wraps syscall.Select with Go types, and retries a number of times, with a given retryDelay.
func RetrySelect(n int, r, w, e *FDSet, timeout time.Duration, retries int, retryDelay time.Duration) (err error) {
	for i := 0; i < retries; i++ {
		if err = Select(n, r, w, e, timeout); err != syscall.EINTR {
			return err
		}
		time.Sleep(retryDelay)
	}
	return err
}
