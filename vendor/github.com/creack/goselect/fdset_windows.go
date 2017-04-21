// +build windows

package goselect

import "syscall"

const FD_SETSIZE = 64

// FDSet extracted from mingw libs source code
type FDSet struct {
	fd_count uint
	fd_array [FD_SETSIZE]uintptr
}

// Set adds the fd to the set
func (fds *FDSet) Set(fd uintptr) {
	var i uint
	for i = 0; i < fds.fd_count; i++ {
		if fds.fd_array[i] == fd {
			break
		}
	}
	if i == fds.fd_count {
		if fds.fd_count < FD_SETSIZE {
			fds.fd_array[i] = fd
			fds.fd_count++
		}
	}
}

// Clear remove the fd from the set
func (fds *FDSet) Clear(fd uintptr) {
	var i uint
	for i = 0; i < fds.fd_count; i++ {
		if fds.fd_array[i] == fd {
			for i < fds.fd_count-1 {
				fds.fd_array[i] = fds.fd_array[i+1]
				i++
			}
			fds.fd_count--
			break
		}
	}
}

// IsSet check if the given fd is set
func (fds *FDSet) IsSet(fd uintptr) bool {
	if isset, err := __WSAFDIsSet(syscall.Handle(fd), fds); err == nil && isset != 0 {
		return true
	}
	return false
}

// Zero empties the Set
func (fds *FDSet) Zero() {
	fds.fd_count = 0
}
