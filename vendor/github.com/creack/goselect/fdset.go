// +build !freebsd,!windows,!plan9

package goselect

import "syscall"

const FD_SETSIZE = syscall.FD_SETSIZE

// FDSet wraps syscall.FdSet with convenience methods
type FDSet syscall.FdSet

// Set adds the fd to the set
func (fds *FDSet) Set(fd uintptr) {
	fds.Bits[fd/NFDBITS] |= (1 << (fd % NFDBITS))
}

// Clear remove the fd from the set
func (fds *FDSet) Clear(fd uintptr) {
	fds.Bits[fd/NFDBITS] &^= (1 << (fd % NFDBITS))
}

// IsSet check if the given fd is set
func (fds *FDSet) IsSet(fd uintptr) bool {
	return fds.Bits[fd/NFDBITS]&(1<<(fd%NFDBITS)) != 0
}

// Keep a null set to avoid reinstatiation
var nullFdSet = &FDSet{}

// Zero empties the Set
func (fds *FDSet) Zero() {
	copy(fds.Bits[:], (nullFdSet).Bits[:])
}
