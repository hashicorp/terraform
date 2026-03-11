// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"path"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"

	"github.com/hashicorp/terraform/internal/addrs"
)

// We have some of the module address parsers in here, rather than in
// package addrs, because right now our remote source address normalization
// is inextricably tied to the external go-getter library, which means any
// package that calls these functions must indirectly depend on go-getter.
//
// Package addrs is imported from almost everywhere, so any dependency it
// has becomes an indirect dependency of everything else. Only a few callers
// actually need to parse module source addresses, so it's pragmatic to have
// just those callers import this package, whereas packages that only need
// to work with addresses that were already parsed -- or don't need to interact
// with module source addresses _at all_ -- can avoid indirectly depending
// on go-getter and all of its various third-party dependencies.

var moduleSourceLocalPrefixes = []string{
	"./",
	"../",
	".\\",
	"..\\",
}

// ParseModuleSource parses a module source address as given in the "source"
// argument inside a "module" block in the configuration.
//
// For historical reasons this syntax is a bit overloaded, supporting three
// different address types:
//   - Local paths starting with either ./ or ../, which are special because
//     Terraform considers them to belong to the same "package" as the caller.
//   - Module registry addresses, given as either NAMESPACE/NAME/SYSTEM or
//     HOST/NAMESPACE/NAME/SYSTEM, in which case the remote registry serves
//     as an indirection over the third address type that follows.
//   - Various URL-like and other heuristically-recognized strings which
//     we currently delegate to the external library go-getter.
//
// There is some ambiguity between the module registry addresses and go-getter's
// very liberal heuristics and so this particular function will typically treat
// an invalid registry address as some other sort of remote source address
// rather than returning an error. If you know that you're expecting a
// registry address in particular, use [ParseModuleSourceRegistry] instead, which
// can therefore expose more detailed error messages about registry address
// parsing in particular.
func ParseModuleSource(raw string) (addrs.ModuleSource, error) {
	if isModuleSourceLocal(raw) {
		localAddr, err := parseModuleSourceLocal(raw)
		if err != nil {
			// This is to make sure we really return a nil ModuleSource in
			// this case, rather than an interface containing the zero
			// value of ModuleSourceLocal.
			return nil, err
		}
		return localAddr, nil
	}

	// For historical reasons, whether an address is a registry
	// address is defined only by whether it can be successfully
	// parsed as one, and anything else must fall through to be
	// parsed as a direct remote source, where go-getter might
	// then recognize it as a filesystem path. This is odd
	// but matches behavior we've had since Terraform v0.10 which
	// existing modules may be relying on.
	// (Notice that this means that there's never any path where
	// the registry source parse error gets returned to the caller,
	// which is annoying but has been true for many releases
	// without it posing a serious problem in practice.)
	if ret, err := ParseModuleSourceRegistry(raw); err == nil {
		return ret, nil
	}

	// If we get down here then we treat everything else as a
	// remote address. In practice there's very little that
	// go-getter doesn't consider invalid input, so even invalid
	// nonsense will probably interpreted as _something_ here
	// and then fail during installation instead. We can't
	// really improve this situation for historical reasons.
	remoteAddr, err := parseModuleSourceRemote(raw)
	if err != nil {
		// This is to make sure we really return a nil ModuleSource in
		// this case, rather than an interface containing the zero
		// value of ModuleSourceRemote.
		return nil, err
	}
	return remoteAddr, nil
}

func parseModuleSourceLocal(raw string) (addrs.ModuleSourceLocal, error) {
	// As long as we have a suitable prefix (detected by ParseModuleSource)
	// there is no failure case for local paths: we just use the "path"
	// package's cleaning logic to remove any redundant "./" and "../"
	// sequences and any duplicate slashes and accept whatever that
	// produces.

	// Although using backslashes (Windows-style) is non-idiomatic, we do
	// allow it and just normalize it away, so the rest of Terraform will
	// only see the forward-slash form.
	if strings.Contains(raw, `\`) {
		// Note: We use string replacement rather than filepath.ToSlash
		// here because the filepath package behavior varies by current
		// platform, but we want to interpret configured paths the same
		// across all platforms: these are virtual paths within a module
		// package, not physical filesystem paths.
		raw = strings.ReplaceAll(raw, `\`, "/")
	}

	// Note that we could've historically blocked using "//" in a path here
	// in order to avoid confusion with the subdir syntax in remote addresses,
	// but we historically just treated that as the same as a single slash
	// and so we continue to do that now for compatibility. Clean strips those
	// out and reduces them to just a single slash.
	clean := path.Clean(raw)

	// However, we do need to keep a single "./" on the front if it isn't
	// a "../" path, or else it would be ambigous with the registry address
	// syntax.
	if !strings.HasPrefix(clean, "../") {
		clean = "./" + clean
	}

	return addrs.ModuleSourceLocal(clean), nil
}

func isModuleSourceLocal(raw string) bool {
	for _, prefix := range moduleSourceLocalPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}

// ParseModuleSourceRegistry is a variant of ParseModuleSource which only
// accepts module registry addresses, and will reject any other address type.
//
// Use this instead of ParseModuleSource if you know from some other surrounding
// context that an address is intended to be a registry address rather than
// some other address type, which will then allow for better error reporting
// due to the additional information about user intent.
func ParseModuleSourceRegistry(raw string) (addrs.ModuleSource, error) {
	// Before we delegate to the "real" function we'll just make sure this
	// doesn't look like a local source address, so we can return a better
	// error message for that situation.
	if isModuleSourceLocal(raw) {
		return addrs.ModuleSourceRegistry{}, fmt.Errorf("can't use local directory %q as a module registry address", raw)
	}

	src, err := tfaddr.ParseModuleSource(raw)
	if err != nil {
		return nil, err
	}
	return addrs.ModuleSourceRegistry{
		Package: src.Package,
		Subdir:  src.Subdir,
	}, nil
}

func parseModuleSourceRemote(raw string) (addrs.ModuleSourceRemote, error) {
	var subDir string
	raw, subDir = SplitPackageSubdir(raw)
	if strings.HasPrefix(subDir, "../") {
		return addrs.ModuleSourceRemote{}, fmt.Errorf("subdirectory path %q leads outside of the module package", subDir)
	}

	// A remote source address is really just a go-getter address resulting
	// from go-getter's "detect" phase, which adds on the prefix specifying
	// which protocol it should use and possibly also adjusts the
	// protocol-specific part into different syntax.
	//
	// Note that for historical reasons this can potentially do network
	// requests in order to disambiguate certain address types, although
	// that's a legacy thing that is only for some specific, less-commonly-used
	// address types. Most just do local string manipulation. We should
	// aim to remove the network requests over time, if possible.
	norm, moreSubDir, err := NormalizePackageAddress(raw)
	if err != nil {
		// We must pass through the returned error directly here because
		// the getmodules package has some special error types it uses
		// for certain cases where the UI layer might want to include a
		// more helpful error message.
		return addrs.ModuleSourceRemote{}, err
	}

	if moreSubDir != "" {
		switch {
		case subDir != "":
			// The detector's own subdir goes first, because the
			// subdir we were given is conceptually relative to
			// the subdirectory that we just detected.
			subDir = path.Join(moreSubDir, subDir)
		default:
			subDir = path.Clean(moreSubDir)
		}
		if strings.HasPrefix(subDir, "../") {
			// This would suggest a bug in a go-getter detector, but
			// we'll catch it anyway to avoid doing something confusing
			// downstream.
			return addrs.ModuleSourceRemote{}, fmt.Errorf("detected subdirectory path %q of %q leads outside of the module package", subDir, norm)
		}
	}

	return addrs.ModuleSourceRemote{
		Package: addrs.ModulePackage(norm),
		Subdir:  subDir,
	}, nil
}
