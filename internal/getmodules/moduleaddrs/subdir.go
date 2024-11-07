// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// SplitPackageSubdir detects whether the given address string has a
// subdirectory portion, and if so returns a non-empty subDir string
// along with the trimmed package address.
//
// If the given string doesn't have a subdirectory portion then it'll
// just be returned verbatim in packageAddr, with an empty subDir value.
//
// Although the rest of this package is focused only on direct remote
// module packages, this particular function and its companion
// ExpandSubdirGlobs are both also relevant for registry-based module
// addresses, because a registry translates such an address into a
// remote module package address and thus can contribute its own
// additions to the final subdirectory selection.
func SplitPackageSubdir(given string) (packageAddr, subDir string) {
	packageAddr, subDir = splitPackageSubdirRaw(given)
	if subDir != "" {
		subDir = path.Clean(subDir)
	}
	return packageAddr, subDir
}

func splitPackageSubdirRaw(src string) (packageAddr, subDir string) {
	// URL might contains another url in query parameters
	stop := len(src)
	if idx := strings.Index(src, "?"); idx > -1 {
		stop = idx
	}

	// Calculate an offset to avoid accidentally marking the scheme
	// as the dir.
	var offset int
	if idx := strings.Index(src[:stop], "://"); idx > -1 {
		offset = idx + 3
	}

	// First see if we even have an explicit subdir
	idx := strings.Index(src[offset:stop], "//")
	if idx == -1 {
		return src, ""
	}

	idx += offset
	subdir := src[idx+2:]
	src = src[:idx]

	// Next, check if we have query parameters and push them onto the
	// URL.
	if idx = strings.Index(subdir, "?"); idx > -1 {
		query := subdir[idx:]
		subdir = subdir[:idx]
		src += query
	}

	return src, subdir
}

// ExpandSubdirGlobs handles a subdir string that might contain glob syntax,
// turning it into a concrete subdirectory path by referring to the actual
// files on disk in the given directory which we assume contains the content
// of whichever package this is a subdirectory glob for.
//
// Subdir globs are used, for example, when a module registry wants to specify
// to select the contents of the single directory at the root of a conventional
// tar archive but it doesn't actually know the exact name of that directory.
// In that case it might specify a subdir of just "*", which this function
// will then expand into the single subdirectory found inside instDir, or
// return an error if the result would be ambiguous.
func ExpandSubdirGlobs(instDir string, subDir string) (string, error) {
	pattern := filepath.Join(instDir, subDir)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("subdir %q not found", subDir)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("subdir %q matches multiple paths", subDir)
	}

	return matches[0], nil
}
