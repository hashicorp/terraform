// +build !windows

package sysinfo

import (
	"syscall"
	"time"
)

func timevalToDuration(tv syscall.Timeval) time.Duration {
	return time.Duration(tv.Nano()) * time.Nanosecond
}

// GetUsage gathers process times.
func GetUsage() (Usage, error) {
	ru := syscall.Rusage{}
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	if err != nil {
		return Usage{}, err
	}

	return Usage{
		System: timevalToDuration(ru.Stime),
		User:   timevalToDuration(ru.Utime),
	}, nil
}
