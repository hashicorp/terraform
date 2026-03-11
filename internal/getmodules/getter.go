// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package getmodules

import (
	"context"
	"fmt"
	"log"
	"os"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	getter "github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/internal/copy"
)

// We configure our own go-getter getter set here, because the set of sources
// we support is part of Terraform's documentation and so we don't want any
// new sources introduced in go-getter to sneak in here and work even though
// they aren't documented. This also insulates us from any meddling that might
// be done by other go-getter callers linked into our executable.
//
// We don't use go-getter's own detectors because Terraform needs to be in
// control of its own source address syntax due to it being covered by our
// Terraform v1.x compatibility promises. However, we do still follow at least
// what go-getter would've done at some point in the past that now represent's
// Terraform's compatibility contract, and arrange for the result to be
// something that should be consumable just by the set of getters defined
// below.
//
// Note that over time we've found go-getter's design to be not wholly fit
// for Terraform's purposes in various ways, and so we're continuing to use
// it here because our backward compatibility with earlier versions depends
// on it, but we use go-getter very carefully and always only indirectly via
// the public API of this package so that we can get the subset of the
// go-getter functionality we need while working around some of the less
// helpful parts of its design. See the comments in various other functions
// in this package which call into go-getter for more information on what
// tradeoffs we're making here.

var goGetterNoDetectors = []getter.Detector{}

var goGetterDecompressors = map[string]getter.Decompressor{
	// Bzip2
	"bz2":      new(getter.Bzip2Decompressor),
	"tbz2":     new(getter.TarBzip2Decompressor),
	"tar.bz2":  new(getter.TarBzip2Decompressor),
	"tar.tbz2": new(getter.TarBzip2Decompressor),

	// Gzip
	"gz":     new(getter.GzipDecompressor),
	"tar.gz": new(getter.TarGzipDecompressor),
	"tgz":    new(getter.TarGzipDecompressor),

	// Xz
	"xz":     new(getter.XzDecompressor),
	"tar.xz": new(getter.TarXzDecompressor),
	"txz":    new(getter.TarXzDecompressor),

	// Zip
	"zip": new(getter.ZipDecompressor),
}

var goGetterGetters = map[string]getter.Getter{
	// Protocol-based getters
	"http":  getterHTTPGetter,
	"https": getterHTTPGetter,

	// Cloud storage getters
	"gcs": new(getter.GCSGetter),
	"s3":  new(getter.S3Getter),

	// Version control getters
	"git": new(getter.GitGetter),
	"hg":  new(getter.HgGetter),

	// Local and file-based getters
	"file": new(getter.FileGetter),
}

var getterHTTPClient = cleanhttp.DefaultClient()

var getterHTTPGetter = &getter.HttpGetter{
	Client:             getterHTTPClient,
	Netrc:              true,
	XTerraformGetLimit: 10,
}

// A reusingGetter is a helper for the module installer that remembers
// the final resolved addresses of all of the sources it has already been
// asked to install, and will copy from a prior installation directory if
// it has the same resolved source address.
//
// The keys in a reusingGetter are the normalized (post-detection) package
// addresses, and the values are the paths where each source was previously
// installed. (Users of this map should treat the keys as addrs.ModulePackage
// values, but we can't type them that way because the addrs package
// imports getmodules in order to indirectly access our go-getter
// configuration.)
type reusingGetter map[string]string

// getWithGoGetter fetches the package at the given address into the given
// target directory. The given address must already be in normalized form
// (using NormalizePackageAddress) or else the behavior is undefined.
//
// This function deals only in entire packages, so it's always the caller's
// responsibility to handle any subdirectory specification and select a
// suitable subdirectory of the given installation directory after installation
// has succeeded.
//
// This function would ideally accept packageAddr as a value of type
// addrs.ModulePackage, but we can't do that because the addrs package
// depends on this package for package address parsing. Therefore we just
// use a string here but assume that the caller got that value by calling
// the String method on a valid addrs.ModulePackage value.
//
// The errors returned by this function are those surfaced by the underlying
// go-getter library, which have very inconsistent quality as
// end-user-actionable error messages. At this time we do not have any
// reasonable way to improve these error messages at this layer because
// the underlying errors are not separately recognizable.
func (g reusingGetter) getWithGoGetter(ctx context.Context, instPath, packageAddr string) error {
	var err error

	if prevDir, exists := g[packageAddr]; exists {
		log.Printf("[TRACE] getmodules: copying previous install of %q from %s to %s", packageAddr, prevDir, instPath)
		err := os.Mkdir(instPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %s", instPath, err)
		}
		err = copy.CopyDir(instPath, prevDir)
		if err != nil {
			return fmt.Errorf("failed to copy from %s to %s: %s", prevDir, instPath, err)
		}
	} else {
		log.Printf("[TRACE] getmodules: fetching %q to %q", packageAddr, instPath)
		client := getter.Client{
			Src: packageAddr,
			Dst: instPath,
			Pwd: instPath,

			Mode: getter.ClientModeDir,

			Detectors:     goGetterNoDetectors, // our caller should've already done detection
			Decompressors: goGetterDecompressors,
			Getters:       goGetterGetters,
			Ctx:           ctx,
		}
		err = client.Get()
		if err != nil {
			return err
		}
		// Remember where we installed this so we might reuse this directory
		// on subsequent calls to avoid re-downloading.
		g[packageAddr] = instPath
	}

	// If we get down here then we've either downloaded the package or
	// copied a previous tree we downloaded, and so either way we should
	// have got the full module package structure written into instPath.
	return nil
}
