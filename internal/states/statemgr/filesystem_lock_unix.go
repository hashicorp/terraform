// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package statemgr

import (
	"io"
	"log"
	"syscall"
)

// use fcntl POSIX locks for the most consistent behavior across platforms, and
// hopefully some campatibility over NFS and CIFS.
func (s *Filesystem) lock() error {
	log.Printf("[TRACE] statemgr.Filesystem: locking %s using fcntl flock", s.path)
	flock := &syscall.Flock_t{
		Type:   syscall.F_RDLCK | syscall.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFileOut.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}

func (s *Filesystem) unlock() error {
	log.Printf("[TRACE] statemgr.Filesystem: unlocking %s using fcntl flock", s.path)
	flock := &syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFileOut.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}
