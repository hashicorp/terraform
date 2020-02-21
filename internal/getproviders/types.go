package getproviders

import (
	"crypto/sha256"
	"runtime"

	"github.com/apparentlymart/go-versions/versions"
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

	Filename  string
	Location  PackageLocation
	SHA256Sum [sha256.Size]byte

	// TODO: Extra metadata for signature verification
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
