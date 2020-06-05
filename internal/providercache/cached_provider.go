package providercache

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// CachedProvider represents a provider package in a cache directory.
type CachedProvider struct {
	// Provider and Version together identify the specific provider version
	// this cache entry represents.
	Provider addrs.Provider
	Version  getproviders.Version

	// PackageDir is the local filesystem path to the root directory where
	// the provider's distribution archive was unpacked.
	//
	// The path always uses slashes as path separators, even on Windows, so
	// that the results are consistent between platforms. Windows accepts
	// both slashes and backslashes as long as the separators are consistent
	// within a particular path string.
	PackageDir string

	// ExecutableFile is the local filesystem path to the main plugin executable
	// for the provider, which is always a file within the directory given
	// in PackageDir.
	//
	// The path always uses slashes as path separators, even on Windows, so
	// that the results are consistent between platforms. Windows accepts
	// both slashes and backslashes as long as the separators are consistent
	// within a particular path string.
	ExecutableFile string
}

// PackageLocation returns the package directory given in the PackageDir field
// as a getproviders.PackageLocation implementation.
//
// Because cached providers are always in the unpacked structure, the result is
// always of the concrete type getproviders.PackageLocalDir.
func (cp *CachedProvider) PackageLocation() getproviders.PackageLocalDir {
	return getproviders.PackageLocalDir(cp.PackageDir)
}

// Hash computes a hash of the contents of the package directory associated
// with the receiving cached provider, using whichever hash algorithm is
// the current default.
//
// If you need a specific version of hash rather than just whichever one is
// current default, call that version's corresponding method (e.g. HashV1)
// directly instead.
func (cp *CachedProvider) Hash() (string, error) {
	return getproviders.PackageHash(cp.PackageLocation())
}

// MatchesHash returns true if the package on disk matches the given hash,
// or false otherwise. If it cannot traverse the package directory and read
// all of the files in it, or if the hash is in an unsupported format,
// CheckHash returns an error.
//
// MatchesHash may accept hashes in a number of different formats. Over time
// the set of supported formats may grow and shrink.
func (cp *CachedProvider) MatchesHash(want string) (bool, error) {
	return getproviders.PackageMatchesHash(cp.PackageLocation(), want)
}

// HashV1 computes a hash of the contents of the package directory associated
// with the receiving cached provider using hash algorithm 1.
//
// The hash covers the paths to files in the directory and the contents of
// those files. It does not cover other metadata about the files, such as
// permissions.
//
// This function is named "HashV1" in anticipation of other hashing algorithms
// being added (in a backward-compatible way) in future. The result from
// HashV1 always begins with the prefix "h1:" so that callers can distinguish
// the results of potentially multiple different hash algorithms in future.
func (cp *CachedProvider) HashV1() (string, error) {
	return getproviders.PackageHashV1(cp.PackageLocation())
}
