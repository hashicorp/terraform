// +build linux

package goselect

import "syscall"

func sysSelect(n int, r, w, e *FDSet, timeout *syscall.Timeval) error {
	_, err := syscall.Select(n, (*syscall.FdSet)(r), (*syscall.FdSet)(w), (*syscall.FdSet)(e), timeout)
	return err
}
