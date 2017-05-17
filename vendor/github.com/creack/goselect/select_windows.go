// +build windows

package goselect

import "syscall"

//sys _select(nfds int, readfds *FDSet, writefds *FDSet, exceptfds *FDSet, timeout *syscall.Timeval) (total int, err error) = ws2_32.select
//sys __WSAFDIsSet(handle syscall.Handle, fdset *FDSet) (isset int, err error) = ws2_32.__WSAFDIsSet

func sysSelect(n int, r, w, e *FDSet, timeout *syscall.Timeval) error {
	_, err := _select(n, r, w, e, timeout)
	return err
}
