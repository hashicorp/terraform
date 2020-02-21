package getproviders

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/addrs"
)

// Version represents a particular single version of a provider.
type Version = versions.Version

// VersionList represents a list of versions. It is a []Version with some
// extra methods for convenient filtering.
type VersionList = versions.List

// ParseVersion parses a "semver"-style version string into a Version value,
// which is the version syntax we use for provider versions.
func ParseVersion(str string) (Version, error) {
	return versions.ParseVersion(str)
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
	underPos := strings.Index(str, "_")
	if underPos < 1 || underPos >= len(str)-2 {
		return Platform{}, fmt.Errorf("must be two words separated by an underscore")
	}

	os, arch := str[:underPos], str[underPos+1:]
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

	// FIXME: Our current hashing scheme only works for sources that have
	// access to the original distribution archives, so this isn't always
	// populated. Need to figure out a different approach where we can
	// consistently hash both from an archive file and from an extracted
	// archive to detect inconsistencies.
	SHA256Sum [sha256.Size]byte

	// TODO: Extra metadata for signature verification
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

// PackageLocation represents a location where a provider distribution package
// can be obtained. A value of this type contains one of the following
// concrete types: PackageLocalArchive, PackageLocalDir, or PackageHTTPURL.
type PackageLocation interface {
	packageLocation()
}

// PackageLocalArchive is the location of a provider distribution archive file
// in the local filesystem. Its value is a local filesystem path using the
// syntax understood by Go's standard path/filepath package on the operating
// system where Terraform is running.
type PackageLocalArchive string

func (p PackageLocalArchive) packageLocation() {}

// PackageLocalDir is the location of a directory containing an unpacked
// provider distribution archive in the local filesystem. Its value is a local
// filesystem path using the syntax understood by Go's standard path/filepath
// package on the operating system where Terraform is running.
type PackageLocalDir string

func (p PackageLocalDir) packageLocation() {}

// PackageHTTPURL is a provider package location accessible via HTTP.
// Its value is a URL string using either the http: scheme or the https: scheme.
type PackageHTTPURL string

func (p PackageHTTPURL) packageLocation() {}

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
