// +build !windows

package statemgr

import (
	"os"
	"syscall"
)

// use fcntl POSIX locks for the most consistent behavior across platforms, and
// hopefully some campatibility over NFS and CIFS.
func (s *Filesystem) lock() error {
	flock := &syscall.Flock_t{
		Type:   syscall.F_RDLCK | syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFileOut.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}

func (s *Filesystem) unlock() error {
	flock := &syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFileOut.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}
