// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package getproviders

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

// Version represents a particular single version of a provider.
type Version = providerreqs.Version

// UnspecifiedVersion is the zero value of Version, representing the absense
// of a version number.
var UnspecifiedVersion Version = providerreqs.UnspecifiedVersion

// VersionList represents a list of versions. It is a []Version with some
// extra methods for convenient filtering.
type VersionList = providerreqs.VersionList

// VersionSet represents a set of versions, usually describing the acceptable
// versions that can be selected under a particular version constraint provided
// by the end-user.
type VersionSet = providerreqs.VersionSet

// VersionConstraints represents a set of version constraints, which can
// define the membership of a VersionSet by exclusion.
type VersionConstraints = providerreqs.VersionConstraints

// Warnings represents a list of warnings returned by a Registry source.
type Warnings = providerreqs.Warnings

// Requirements gathers together requirements for many different providers
// into a single data structure, as a convenient way to represent the full
// set of requirements for a particular configuration or state or both.
//
// If an entry in a Requirements has a zero-length VersionConstraints then
// that indicates that the provider is required but that any version is
// acceptable. That's different than a provider being absent from the map
// altogether, which means that it is not required at all.
type Requirements = providerreqs.Requirements

// Selections gathers together version selections for many different providers.
//
// This is the result of provider installation: a specific version selected
// for each provider given in the requested Requirements, selected based on
// the given version constraints.
type Selections = providerreqs.Selections

// ParseVersion parses a "semver"-style version string into a Version value,
// which is the version syntax we use for provider versions.
func ParseVersion(str string) (Version, error) {
	return providerreqs.ParseVersion(str)
}

// MustParseVersion is a variant of ParseVersion that panics if it encounters
// an error while parsing.
func MustParseVersion(str string) Version {
	return providerreqs.MustParseVersion(str)
}

// ParseVersionConstraints parses a "Ruby-like" version constraint string
// into a VersionConstraints value.
func ParseVersionConstraints(str string) (VersionConstraints, error) {
	return providerreqs.ParseVersionConstraints(str)
}

// MustParseVersionConstraints is a variant of ParseVersionConstraints that
// panics if it encounters an error while parsing.
func MustParseVersionConstraints(str string) VersionConstraints {
	return providerreqs.MustParseVersionConstraints(str)
}

// MeetingConstraints returns a version set that contains all of the versions
// that meet the given constraints, specified using the Spec type from the
// constraints package.
func MeetingConstraints(vc VersionConstraints) VersionSet {
	return providerreqs.MeetingConstraints(vc)
}

// Platform represents a target platform that a provider is or might be
// available for.
type Platform struct {
	OS, Arch string
}

func (p Platform) String() string {
	return p.OS + "_" + p.Arch
}

// LessThan returns true if the receiver should sort before the other given
// Platform in an ordered list of platforms.
//
// The ordering is lexical first by OS and then by Architecture.
// This ordering is primarily just to ensure that results of
// functions in this package will be deterministic. The ordering is not
// intended to have any semantic meaning and is subject to change in future.
func (p Platform) LessThan(other Platform) bool {
	switch {
	case p.OS != other.OS:
		return p.OS < other.OS
	default:
		return p.Arch < other.Arch
	}
}

// ParsePlatform parses a string representation of a platform, like
// "linux_amd64", or returns an error if the string is not valid.
func ParsePlatform(str string) (Platform, error) {
	parts := strings.Split(str, "_")
	if len(parts) != 2 {
		return Platform{}, fmt.Errorf("must be two words separated by an underscore")
	}

	os, arch := parts[0], parts[1]
	if strings.ContainsAny(os, " \t\n\r") {
		return Platform{}, fmt.Errorf("OS portion must not contain whitespace")
	}
	if strings.ContainsAny(arch, " \t\n\r") {
		return Platform{}, fmt.Errorf("architecture portion must not contain whitespace")
	}

	return Platform{
		OS:   os,
		Arch: arch,
	}, nil
}

// CurrentPlatform is the platform where the current program is running.
//
// If attempting to install providers for use on the same system where the
// installation process is running, this is the right platform to use.
var CurrentPlatform = Platform{
	OS:   runtime.GOOS,
	Arch: runtime.GOARCH,
}

// PackageMeta represents the metadata related to a particular downloadable
// provider package targeting a single platform.
//
// Package findproviders does no signature verification or protocol version
// compatibility checking of its own. A caller receving a PackageMeta must
// verify that it has a correct signature and supports a protocol version
// accepted by the current version of Terraform before trying to use the
// described package.
type PackageMeta struct {
	Provider addrs.Provider
	Version  Version

	ProtocolVersions VersionList
	TargetPlatform   Platform

	Filename string
	Location PackageLocation

	// Authentication, if non-nil, is a request from the source that produced
	// this meta for verification of the target package after it has been
	// retrieved from the indicated Location.
	//
	// Different sources will support different authentication strategies --
	// or possibly no strategies at all -- depending on what metadata they
	// have available to them, such as checksums provided out-of-band by the
	// original package author, expected signing keys, etc.
	//
	// If Authentication is non-nil then no authentication is requested.
	// This is likely appropriate only for packages that are already available
	// on the local system.
	Authentication PackageAuthentication
}

// LessThan returns true if the receiver should sort before the given other
// PackageMeta in a sorted list of PackageMeta.
//
// Sorting preference is given first to the provider address, then to the
// taget platform, and the to the version number (using semver precedence).
// Packages that differ only in semver build metadata have no defined
// precedence and so will always return false.
//
// This ordering is primarily just to maximize the chance that results of
// functions in this package will be deterministic. The ordering is not
// intended to have any semantic meaning and is subject to change in future.
func (m PackageMeta) LessThan(other PackageMeta) bool {
	switch {
	case m.Provider != other.Provider:
		return m.Provider.LessThan(other.Provider)
	case m.TargetPlatform != other.TargetPlatform:
		return m.TargetPlatform.LessThan(other.TargetPlatform)
	case m.Version != other.Version:
		return m.Version.LessThan(other.Version)
	default:
		return false
	}
}

// UnpackedDirectoryPath determines the path under the given base
// directory where SearchLocalDirectory or the FilesystemMirrorSource would
// expect to find an unpacked copy of the receiving PackageMeta.
//
// The result always uses forward slashes as path separator, even on Windows,
// to produce a consistent result on all platforms. Windows accepts both
// direction of slash as long as each individual path string is self-consistent.
func (m PackageMeta) UnpackedDirectoryPath(baseDir string) string {
	return UnpackedDirectoryPathForPackage(baseDir, m.Provider, m.Version, m.TargetPlatform)
}

// PackedFilePath determines the path under the given base
// directory where SearchLocalDirectory or the FilesystemMirrorSource would
// expect to find packed copy (a .zip archive) of the receiving PackageMeta.
//
// The result always uses forward slashes as path separator, even on Windows,
// to produce a consistent result on all platforms. Windows accepts both
// direction of slash as long as each individual path string is self-consistent.
func (m PackageMeta) PackedFilePath(baseDir string) string {
	return PackedFilePathForPackage(baseDir, m.Provider, m.Version, m.TargetPlatform)
}

// AcceptableHashes returns a set of hashes that could be recorded for
// comparison to future results for the same provider version, to implement a
// "trust on first use" scheme.
//
// The AcceptableHashes result is a platform-agnostic set of hashes, with the
// intent that in most cases it will be used as an additional cross-check in
// addition to a platform-specific hash check made during installation. However,
// there are some situations (such as verifying an already-installed package
// that's on local disk) where Terraform would check only against the results
// of this function, meaning that it would in principle accept another
// platform's package as a substitute for the correct platform. That's a
// pragmatic compromise to allow lock files derived from the result of this
// method to be portable across platforms.
//
// Callers of this method should typically also verify the package using the
// object in the Authentication field, and consider how much trust to give
// the result of this method depending on the authentication result: an
// unauthenticated result or one that only verified a checksum could be
// considered less trustworthy than one that checked the package against
// a signature provided by the origin registry.
//
// The AcceptableHashes result is actually provided by the object in the
// Authentication field. AcceptableHashes therefore returns an empty result
// for a PackageMeta that has no authentication object, or has one that does
// not make use of hashes.
func (m PackageMeta) AcceptableHashes() []Hash {
	auth, ok := m.Authentication.(PackageAuthenticationHashes)
	if !ok {
		return nil
	}
	return auth.AcceptableHashes()
}

// PackageLocation represents a location where a provider distribution package
// can be obtained. A value of this type contains one of the following
// concrete types: PackageLocalArchive, PackageLocalDir, or PackageHTTPURL.
type PackageLocation interface {
	packageLocation()
	String() string
}

// PackageLocalArchive is the location of a provider distribution archive file
// in the local filesystem. Its value is a local filesystem path using the
// syntax understood by Go's standard path/filepath package on the operating
// system where Terraform is running.
type PackageLocalArchive string

func (p PackageLocalArchive) packageLocation() {}
func (p PackageLocalArchive) String() string   { return string(p) }

// PackageLocalDir is the location of a directory containing an unpacked
// provider distribution archive in the local filesystem. Its value is a local
// filesystem path using the syntax understood by Go's standard path/filepath
// package on the operating system where Terraform is running.
type PackageLocalDir string

func (p PackageLocalDir) packageLocation() {}
func (p PackageLocalDir) String() string   { return string(p) }

// PackageHTTPURL is a provider package location accessible via HTTP.
// Its value is a URL string using either the http: scheme or the https: scheme.
type PackageHTTPURL string

func (p PackageHTTPURL) packageLocation() {}
func (p PackageHTTPURL) String() string   { return string(p) }

// PackageMetaList is a list of PackageMeta. It's just []PackageMeta with
// some methods for convenient sorting and filtering.
type PackageMetaList []PackageMeta

func (l PackageMetaList) Len() int {
	return len(l)
}

func (l PackageMetaList) Less(i, j int) bool {
	return l[i].LessThan(l[j])
}

func (l PackageMetaList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Sort performs an in-place, stable sort on the contents of the list, using
// the ordering given by method Less. This ordering is primarily to help
// encourage deterministic results from functions and does not have any
// semantic meaning.
func (l PackageMetaList) Sort() {
	sort.Stable(l)
}

// FilterPlatform constructs a new PackageMetaList that contains only the
// elements of the receiver that are for the given target platform.
//
// Pass CurrentPlatform to filter only for packages targeting the platform
// where this code is running.
func (l PackageMetaList) FilterPlatform(target Platform) PackageMetaList {
	var ret PackageMetaList
	for _, m := range l {
		if m.TargetPlatform == target {
			ret = append(ret, m)
		}
	}
	return ret
}

// FilterProviderExactVersion constructs a new PackageMetaList that contains
// only the elements of the receiver that relate to the given provider address
// and exact version.
//
// The version matching for this function is exact, including matching on
// semver build metadata, because it's intended for handling a single exact
// version selected by the caller from a set of available versions.
func (l PackageMetaList) FilterProviderExactVersion(provider addrs.Provider, version Version) PackageMetaList {
	var ret PackageMetaList
	for _, m := range l {
		if m.Provider == provider && m.Version == version {
			ret = append(ret, m)
		}
	}
	return ret
}

// FilterProviderPlatformExactVersion is a combination of both
// FilterPlatform and FilterProviderExactVersion that filters by all three
// criteria at once.
func (l PackageMetaList) FilterProviderPlatformExactVersion(provider addrs.Provider, platform Platform, version Version) PackageMetaList {
	var ret PackageMetaList
	for _, m := range l {
		if m.Provider == provider && m.Version == version && m.TargetPlatform == platform {
			ret = append(ret, m)
		}
	}
	return ret
}

// VersionConstraintsString returns a canonical string representation of
// a VersionConstraints value.
func VersionConstraintsString(spec VersionConstraints) string {
	return providerreqs.VersionConstraintsString(spec)
}
