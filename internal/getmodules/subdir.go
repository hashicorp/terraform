// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package getmodules

import (
	"path"

	getter "github.com/hashicorp/go-getter"
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
	// We delegate this mostly to go-getter, because older Terraform
	// versions just used go-getter directly and so we need to preserve
	// its various quirks for compatibility reasons.
	//
	// However, note that in Terraform we _always_ split off the subdirectory
	// portion and handle it within Terraform-level code, _never_ passing
	// a subdirectory portion down into go-getter's own Get function, because
	// Terraform's ability to refer between local paths inside the same
	// package depends on Terraform itself always being aware of where the
	// package's root directory ended up on disk, and always needs the
	// package installed wholesale.
	packageAddr, subDir = getter.SourceDirSubdir(given)
	if subDir != "" {
		subDir = path.Clean(subDir)
	}
	return packageAddr, subDir
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
	// We just delegate this entirely to go-getter, because older Terraform
	// versions just used go-getter directly and so we need to preserve
	// its various quirks for compatibility reasons.
	return getter.SubdirGlob(instDir, subDir)
}
