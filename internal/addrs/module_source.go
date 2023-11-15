// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"path"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/getmodules"
)

// ModuleSource is the general type for all three of the possible module source
// address types. The concrete implementations of this are ModuleSourceLocal,
// ModuleSourceRegistry, and ModuleSourceRemote.
type ModuleSource interface {
	// String returns a full representation of the address, including any
	// additional components that are typically implied by omission in
	// user-written addresses.
	//
	// We typically use this longer representation in error message, in case
	// the inclusion of normally-omitted components is helpful in debugging
	// unexpected behavior.
	String() string

	// ForDisplay is similar to String but instead returns a representation of
	// the idiomatic way to write the address in configuration, omitting
	// components that are commonly just implied in addresses written by
	// users.
	//
	// We typically use this shorter representation in informational messages,
	// such as the note that we're about to start downloading a package.
	ForDisplay() string

	moduleSource()
}

var _ ModuleSource = ModuleSourceLocal("")
var _ ModuleSource = ModuleSourceRegistry{}
var _ ModuleSource = ModuleSourceRemote{}

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
// registry address in particular, use ParseModuleSourceRegistry instead, which
// can therefore expose more detailed error messages about registry address
// parsing in particular.
func ParseModuleSource(raw string) (ModuleSource, error) {
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

// ModuleSourceLocal is a ModuleSource representing a local path reference
// from the caller's directory to the callee's directory within the same
// module package.
//
// A "module package" here means a set of modules distributed together in
// the same archive, repository, or similar. That's a significant distinction
// because we always download and cache entire module packages at once,
// and then create relative references within the same directory in order
// to ensure all modules in the package are looking at a consistent filesystem
// layout. We also assume that modules within a package are maintained together,
// which means that cross-cutting maintenence across all of them would be
// possible.
//
// The actual value of a ModuleSourceLocal is a normalized relative path using
// forward slashes, even on operating systems that have other conventions,
// because we're representing traversal within the logical filesystem
// represented by the containing package, not actually within the physical
// filesystem we unpacked the package into. We should typically not construct
// ModuleSourceLocal values directly, except in tests where we can ensure
// the value meets our assumptions. Use ParseModuleSource instead if the
// input string is not hard-coded in the program.
type ModuleSourceLocal string

func parseModuleSourceLocal(raw string) (ModuleSourceLocal, error) {
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

	return ModuleSourceLocal(clean), nil
}

func isModuleSourceLocal(raw string) bool {
	for _, prefix := range moduleSourceLocalPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}

func (s ModuleSourceLocal) moduleSource() {}

func (s ModuleSourceLocal) String() string {
	// We assume that our underlying string was already normalized at
	// construction, so we just return it verbatim.
	return string(s)
}

func (s ModuleSourceLocal) ForDisplay() string {
	return string(s)
}

// ModuleSourceRegistry is a ModuleSource representing a module listed in a
// Terraform module registry.
//
// A registry source isn't a direct source location but rather an indirection
// over a ModuleSourceRemote. The job of a registry is to translate the
// combination of a ModuleSourceRegistry and a module version number into
// a concrete ModuleSourceRemote that Terraform will then download and
// install.
type ModuleSourceRegistry tfaddr.Module

// DefaultModuleRegistryHost is the hostname used for registry-based module
// source addresses that do not have an explicit hostname.
const DefaultModuleRegistryHost = tfaddr.DefaultModuleRegistryHost

// ParseModuleSourceRegistry is a variant of ParseModuleSource which only
// accepts module registry addresses, and will reject any other address type.
//
// Use this instead of ParseModuleSource if you know from some other surrounding
// context that an address is intended to be a registry address rather than
// some other address type, which will then allow for better error reporting
// due to the additional information about user intent.
func ParseModuleSourceRegistry(raw string) (ModuleSource, error) {
	// Before we delegate to the "real" function we'll just make sure this
	// doesn't look like a local source address, so we can return a better
	// error message for that situation.
	if isModuleSourceLocal(raw) {
		return ModuleSourceRegistry{}, fmt.Errorf("can't use local directory %q as a module registry address", raw)
	}

	src, err := tfaddr.ParseModuleSource(raw)
	if err != nil {
		return nil, err
	}
	return ModuleSourceRegistry{
		Package: src.Package,
		Subdir:  src.Subdir,
	}, nil
}

func (s ModuleSourceRegistry) moduleSource() {}

func (s ModuleSourceRegistry) String() string {
	if s.Subdir != "" {
		return s.Package.String() + "//" + s.Subdir
	}
	return s.Package.String()
}

func (s ModuleSourceRegistry) ForDisplay() string {
	if s.Subdir != "" {
		return s.Package.ForDisplay() + "//" + s.Subdir
	}
	return s.Package.ForDisplay()
}

// ModuleSourceRemote is a ModuleSource representing a remote location from
// which we can retrieve a module package.
//
// A ModuleSourceRemote can optionally include a "subdirectory" path, which
// means that it's selecting a sub-directory of the given package to use as
// the entry point into the package.
type ModuleSourceRemote struct {
	// Package is the address of the remote package that the requested
	// module belongs to.
	Package ModulePackage

	// If Subdir is non-empty then it represents a sub-directory within the
	// remote package which will serve as the entry-point for the package.
	//
	// Subdir uses a normalized forward-slash-based path syntax within the
	// virtual filesystem represented by the final package. It will never
	// include `../` or `./` sequences.
	Subdir string
}

func parseModuleSourceRemote(raw string) (ModuleSourceRemote, error) {
	var subDir string
	raw, subDir = getmodules.SplitPackageSubdir(raw)
	if strings.HasPrefix(subDir, "../") {
		return ModuleSourceRemote{}, fmt.Errorf("subdirectory path %q leads outside of the module package", subDir)
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
	norm, moreSubDir, err := getmodules.NormalizePackageAddress(raw)
	if err != nil {
		// We must pass through the returned error directly here because
		// the getmodules package has some special error types it uses
		// for certain cases where the UI layer might want to include a
		// more helpful error message.
		return ModuleSourceRemote{}, err
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
			return ModuleSourceRemote{}, fmt.Errorf("detected subdirectory path %q of %q leads outside of the module package", subDir, norm)
		}
	}

	return ModuleSourceRemote{
		Package: ModulePackage(norm),
		Subdir:  subDir,
	}, nil
}

func (s ModuleSourceRemote) moduleSource() {}

func (s ModuleSourceRemote) String() string {
	base := s.Package.String()

	if s.Subdir != "" {
		// Address contains query string
		if strings.Contains(base, "?") {
			parts := strings.SplitN(base, "?", 2)
			return parts[0] + "//" + s.Subdir + "?" + parts[1]
		}
		return base + "//" + s.Subdir
	}
	return base
}

func (s ModuleSourceRemote) ForDisplay() string {
	// The two string representations are identical for this address type.
	// This isn't really entirely true to the idea of "ForDisplay" since
	// it'll often include some additional components added in by the
	// go-getter detectors, but we don't have any function to turn a
	// "detected" string back into an idiomatic shorthand the user might've
	// entered.
	return s.String()
}

// FromRegistry can be called on a remote source address that was returned
// from a module registry, passing in the original registry source address
// that the registry was asked about, in order to get the effective final
// remote source address.
//
// Specifically, this method handles the situations where one or both of
// the two addresses contain subdirectory paths, combining both when necessary
// in order to ensure that both the registry's given path and the user's
// given path are both respected.
//
// This will return nonsense if given a registry address other than the one
// that generated the reciever via a registry lookup.
func (s ModuleSourceRemote) FromRegistry(given ModuleSourceRegistry) ModuleSourceRemote {
	ret := s // not a pointer, so this is a shallow copy

	switch {
	case s.Subdir != "" && given.Subdir != "":
		ret.Subdir = path.Join(s.Subdir, given.Subdir)
	case given.Subdir != "":
		ret.Subdir = given.Subdir
	}

	return ret
}
