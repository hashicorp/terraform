// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package replacefile

import (
	"os"
)

// AtomicRename renames from the source path to the destination path,
// atomically replacing any file that might already exist at the destination.
//
// Typically this operation can succeed only if the source and destination
// are within the same physical filesystem, so this function is best reserved
// for cases where the source and destination exist in the same directory and
// only the local filename differs between them.
//
// The Unix implementation of AtomicRename relies on the atomicity of renaming
// that is required by the ISO C standard, which in turn assumes that Go's
// implementation of rename is calling into a system call that preserves that
// guarantee.
func AtomicRename(source, destination string) error {
	// On Unix systems, a rename is sufficiently atomic.
	return os.Rename(source, destination)
}
