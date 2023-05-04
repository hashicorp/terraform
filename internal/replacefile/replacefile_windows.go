// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows
// +build windows

package replacefile

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// AtomicRename renames from the source path to the destination path,
// atomically replacing any file that might already exist at the destination.
//
// Typically this operation can succeed only if the source and destination
// are within the same physical filesystem, so this function is best reserved
// for cases where the source and destination exist in the same directory and
// only the local filename differs between them.
func AtomicRename(source, destination string) error {
	// On Windows, renaming one file over another is not atomic and certain
	// error conditions can result in having only the source file and nothing
	// at the destination file. Instead, we need to call into the MoveFileEx
	// Windows API function, setting two flags to opt in to replacing an
	// existing file.
	srcPtr, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return &os.LinkError{"replace", source, destination, err}
	}
	destPtr, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return &os.LinkError{"replace", source, destination, err}
	}

	flags := uint32(windows.MOVEFILE_REPLACE_EXISTING | windows.MOVEFILE_WRITE_THROUGH)
	err = windows.MoveFileEx(srcPtr, destPtr, flags)
	if err != nil {
		return &os.LinkError{"replace", source, destination, err}
	}
	return nil
}
