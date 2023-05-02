// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows
// +build windows

package clistate

import (
	"math"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procCreateEventW = modkernel32.NewProc("CreateEventW")
)

const (
	// dwFlags defined for LockFileEx
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx
	_LOCKFILE_FAIL_IMMEDIATELY = 1
	_LOCKFILE_EXCLUSIVE_LOCK   = 2
)

func (s *LocalState) lock() error {
	// even though we're failing immediately, an overlapped event structure is
	// required
	ol, err := newOverlapped()
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(ol.HEvent)

	return lockFileEx(
		syscall.Handle(s.stateFileOut.Fd()),
		_LOCKFILE_EXCLUSIVE_LOCK|_LOCKFILE_FAIL_IMMEDIATELY,
		0,              // reserved
		0,              // bytes low
		math.MaxUint32, // bytes high
		ol,
	)
}

func (s *LocalState) unlock() error {
	// the file is closed in Unlock
	return nil
}

func lockFileEx(h syscall.Handle, flags, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procLockFileEx.Addr(),
		6,
		uintptr(h),
		uintptr(flags),
		uintptr(reserved),
		uintptr(locklow),
		uintptr(lockhigh),
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// newOverlapped creates a structure used to track asynchronous
// I/O requests that have been issued.
func newOverlapped() (*syscall.Overlapped, error) {
	event, err := createEvent(nil, true, false, nil)
	if err != nil {
		return nil, err
	}
	return &syscall.Overlapped{HEvent: event}, nil
}

func createEvent(sa *syscall.SecurityAttributes, manualReset bool, initialState bool, name *uint16) (handle syscall.Handle, err error) {
	var _p0 uint32
	if manualReset {
		_p0 = 1
	}
	var _p1 uint32
	if initialState {
		_p1 = 1
	}

	r0, _, e1 := syscall.Syscall6(
		procCreateEventW.Addr(),
		4,
		uintptr(unsafe.Pointer(sa)),
		uintptr(_p0),
		uintptr(_p1),
		uintptr(unsafe.Pointer(name)),
		0,
		0,
	)
	handle = syscall.Handle(r0)
	if handle == syscall.InvalidHandle {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
