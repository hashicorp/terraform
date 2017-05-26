// +build plan9

package goselect

const FD_SETSIZE = 0

// FDSet wraps syscall.FdSet with convenience methods
type FDSet struct{}

// Set adds the fd to the set
func (fds *FDSet) Set(fd uintptr) {}

// Clear remove the fd from the set
func (fds *FDSet) Clear(fd uintptr) {}

// IsSet check if the given fd is set
func (fds *FDSet) IsSet(fd uintptr) bool { return false }

// Zero empties the Set
func (fds *FDSet) Zero() {}
