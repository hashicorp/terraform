// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package clistate

import (
	"io"
	"syscall"
)

// use fcntl POSIX locks for the most consistent behavior across platforms, and
// hopefully some campatibility over NFS and CIFS.
func (s *LocalState) lock() error {
	flock := &syscall.Flock_t{
		Type:   syscall.F_RDLCK | syscall.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFileOut.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}

func (s *LocalState) unlock() error {
	flock := &syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFileOut.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}
