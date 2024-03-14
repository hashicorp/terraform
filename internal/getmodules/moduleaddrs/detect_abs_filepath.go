// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// detectAbsFilePath detects strings that seem like they are trying to be
// file paths.
//
// If the path is absolute then it's transformed into a file:// URL. If it's
// relative then we return an error of type *MaybeRelativePathErr, so that
// the caller can return a special error message to diagnose that the author
// should have written a local source address if they wanted to use a relative
// path.
//
// This should always be the last detector, because unless the input is an
// empty string this will always claim everything it's given.
func detectAbsFilePath(src string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if !filepath.IsAbs(src) {
		return "", true, &MaybeRelativePathErr{src}
	}

	if runtime.GOOS == "windows" {
		// Make sure we're using "/" on Windows. URLs are "/"-based.
		src = filepath.ToSlash(src)
		return fmt.Sprintf("file://%s", src), true, nil
	}

	// Make sure that we don't start with "/" since we add that below.
	if src[0] == '/' {
		src = src[1:]
	}
	return fmt.Sprintf("file:///%s", src), true, nil
}

// MaybeRelativePathErr is the error type returned by NormalizePackageAddress
// if the source address looks like it might be intended to be a relative
// filesystem path but without the required "./" or "../" prefix.
//
// Specifically, NormalizePackageAddress will return a pointer to this type,
// so the error type will be *MaybeRelativePathErr.
//
// It has a name starting with "Maybe" because in practice we can get here
// with any string that isn't recognized as one of the supported schemes:
// treating the address as a local filesystem path is our fallback for when
// everything else fails, but it could just as easily be a typo in an attempt
// to use one of the other schemes and thus not a filesystem path at all.
type MaybeRelativePathErr struct {
	Addr string
}

func (e *MaybeRelativePathErr) Error() string {
	return fmt.Sprintf("Terraform cannot detect a supported external module source type for %s", e.Addr)
}
