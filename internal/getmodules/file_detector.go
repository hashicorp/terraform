// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package getmodules

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// fileDetector is a replacement for go-getter's own file detector which
// better meets Terraform's needs: specifically, it rejects relative filesystem
// paths with a somewhat-decent error message.
//
// This is a replacement for some historical hackery we did where we tried to
// avoid calling into go-getter altogether in this situation. This is,
// therefore, a copy of getter.FileDetector but with the "not absolute path"
// case replaced with a similar result as Terraform's old heuristic would've
// returned: a custom error type that the caller can react to in order to
// produce a hint error message if desired.
type fileDetector struct{}

func (d *fileDetector) Detect(src, pwd string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if !filepath.IsAbs(src) {
		return "", true, &MaybeRelativePathErr{src}
	}

	return fmtFileURL(src), true, nil
}

func fmtFileURL(path string) string {
	if runtime.GOOS == "windows" {
		// Make sure we're using "/" on Windows. URLs are "/"-based.
		path = filepath.ToSlash(path)
		return fmt.Sprintf("file://%s", path)
	}

	// Make sure that we don't start with "/" since we add that below.
	if path[0] == '/' {
		path = path[1:]
	}
	return fmt.Sprintf("file:///%s", path)
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
