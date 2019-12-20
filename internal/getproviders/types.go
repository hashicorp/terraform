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
	ProtocolVersions VersionList
	TargetPlatform   Platform

	Filename    string
	DownloadURL string
	SHA256Sum   [sha256.Size]byte

	// TODO: Extra metadata for signature verification
}
