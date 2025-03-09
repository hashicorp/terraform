// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"path"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
)

// ModuleSource is the general type for all three of the possible module source
// address types. The concrete implementations of this are [ModuleSourceLocal],
// [ModuleSourceRegistry], and [ModuleSourceRemote].
//
// The parser for this address type lives in package moduleaddrs, because remote
// module source address parsing depends on go-getter and that's too heavy a
// dependency to impose on everything that imports this package addrs.
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
//
// The parser for this address type lives in package moduleaddrs. It doesn't
// really need to because it doesn't have any special dependencies, but the
// remote source address parser needs to live over there and so it's clearer
// to just have all of the parsers live together in that other package.
type ModuleSourceLocal string

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
//
// The parser for this address type lives in package moduleaddrs. It doesn't
// really need to because it doesn't have any special dependencies, but the
// remote source address parser needs to live over there and so it's clearer
// to just have all of the parsers live together in that other package.
type ModuleSourceRegistry tfaddr.Module

// DefaultModuleRegistryHost is the hostname used for registry-based module
// source addresses that do not have an explicit hostname.
const DefaultModuleRegistryHost = tfaddr.DefaultModuleRegistryHost

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
//
// The parser for this address type lives in package moduleaddrs, because remote
// module source address parsing depends on go-getter and that's too heavy a
// dependency to impose on everything that imports this package addrs.
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
