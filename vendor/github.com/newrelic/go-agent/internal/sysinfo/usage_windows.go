package sysinfo

import (
	"syscall"
	"time"
)

func filetimeToDuration(ft *syscall.Filetime) time.Duration {
	ns := ft.Nanoseconds()
	return time.Duration(ns)
}

// GetUsage gathers process times.
func GetUsage() (Usage, error) {
	var creationTime syscall.Filetime
	var exitTime syscall.Filetime
	var kernelTime syscall.Filetime
	var userTime syscall.Filetime

	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return Usage{}, err
	}

	err = syscall.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime)
	if err != nil {
		return Usage{}, err
	}

	return Usage{
		System: filetimeToDuration(&kernelTime),
		User:   filetimeToDuration(&userTime),
	}, nil
}
