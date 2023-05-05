// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package getmodules

import (
	getter "github.com/hashicorp/go-getter"
)

// NormalizePackageAddress uses the go-getter "detector" functionality in
// order to turn a user-supplied source address into a normalized address
// which always includes a prefix naming a protocol to fetch with and may
// also include a transformed/normalized version of the protocol-specific
// source address included afterward.
//
// This is part of the implementation of addrs.ParseModulePackage and of
// addrs.ParseModuleSource, so for most callers it'd be better to call
// one of those other functions instead. The addrs package can potentially
// perform other processing in addition to just the go-getter detection.
//
// Note that this function expects to recieve only a package address, not
// a full source address that might also include a subdirectory portion.
// The caller must trim off any subdirectory portion using
// getmodules.SplitPackageSubdir before calling this function, passing in
// just the packageAddr return value, or the result will be incorrect.
//
// The detectors in go-getter can potentially introduce their own
// package subdirectory portions. If that happens then this function will
// return the subdirectory portion as a non-empty subDir return value,
// which the caller must then use as a prefix for any subDir it already
// extracted from the user's given package address.
//
// Some of go-getter's detectors make outgoing HTTP requests, and so
// the behavior of this function may depend on the network connectivity
// of the system where Terraform is running. However, most of the getters
// we use are local-only, and so HTTP requests are only for some ambiguous
// edge-cases, such as the BitBucket detector which has a mechanism to
// detect whether to use Git or Mercurial, because earlier versions of
// BitBucket used to support both.
func NormalizePackageAddress(given string) (packageAddr, subDir string, err error) {
	// Because we're passing go-getter no base directory here, the file
	// detector will return an error if the user entered a relative filesystem
	// path without a "../" or "./" prefix and thus ended up in here.
	//
	// go-getter's error message for that case is very poor, and so we'll
	// try to heuristically detect that situation and return a better error
	// message.

	// NOTE: We're passing an empty string to the "current working directory"
	// here because that's only relevant for relative filesystem paths,
	// but Terraform handles relative filesystem paths itself outside of
	// go-getter and so it'd always be an error to pass one into here.
	// go-getter's "file" detector returns an error if it encounters a
	// relative path when the pwd argument is empty.
	//
	// (Absolute filesystem paths _are_ valid though, for annoying historical
	// reasons, and we treat them as remote packages even though "downloading"
	// them just means a recursive copy of the source directory tree.)

	result, err := getter.Detect(given, "", goGetterDetectors)
	if err != nil {
		// NOTE: go-getter's error messages are of very inconsistent quality
		// and many are not suitable for an end-user audience, but they are all
		// just strings and so we can't really do any sort of post-processing
		// to improve them and thus we just accept some bad error messages for
		// now.
		return "", "", err
	}

	packageAddr, subDir = SplitPackageSubdir(result)
	return packageAddr, subDir, nil
}
