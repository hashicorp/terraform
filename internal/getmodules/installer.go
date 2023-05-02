// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package getmodules

import (
	"context"
)

// PackageFetcher is a low-level utility for fetching remote module packages
// into local filesystem directories in preparation for use by higher-level
// module installer functionality implemented elsewhere.
//
// A PackageFetcher works only with entire module packages and never with
// the individual modules within a package.
//
// A particular PackageFetcher instance remembers the target directory of
// any successfully-installed package so that it can optimize future calls
// that have the same package address by copying the local directory tree,
// rather than fetching the package from its origin repeatedly. There is
// no way to reset this cache, so a particular PackageFetcher instance should
// live only for the duration of a single initialization process.
type PackageFetcher struct {
	getter reusingGetter
}

func NewPackageFetcher() *PackageFetcher {
	return &PackageFetcher{
		getter: reusingGetter{},
	}
}

// FetchPackage downloads or otherwise retrieves the filesystem inside the
// package at the given address into the given local installation directory.
//
// packageAddr must be formatted as if it were the result of an
// addrs.ModulePackage.String() call. It's only defined as a raw string here
// because the getmodules package can't import the addrs package due to
// that creating a package dependency cycle.
//
// PackageFetcher only works with entire packages. If the caller is processing
// a module source address which includes a subdirectory portion then the
// caller must resolve that itself, possibly with the help of the
// getmodules.SplitPackageSubdir and getmodules.ExpandSubdirGlobs functions.
func (f *PackageFetcher) FetchPackage(ctx context.Context, instDir string, packageAddr string) error {
	return f.getter.getWithGoGetter(ctx, instDir, packageAddr)
}
