// +build !linux,!windows,!plan9,!solaris

package goselect

import "syscall"

func sysSelect(n int, r, w, e *FDSet, timeout *syscall.Timeval) error {
	return syscall.Select(n, (*syscall.FdSet)(r), (*syscall.FdSet)(w), (*syscall.FdSet)(e), timeout)
}
