package providercache

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"

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

// Hash computes a hash of the contents of the package directory associated
// with the receiving cached provider, using whichever hash algorithm is
// the current default.
//
// Currently, this method returns version 1 hashes as produced by the
// method HashV1, but this function may switch to other versions in later
// releases. Call HashV1 directly if you specifically need a V1 hash.
func (cp *CachedProvider) Hash() (string, error) {
	return cp.HashV1()
}

// MatchesHash returns true if the package on disk matches the given hash,
// or false otherwise. If it cannot traverse the package directory and read
// all of the files in it, or if the hash is in an unsupported format,
// CheckHash returns an error.
//
// There is currently only one hash format, as implemented by HashV1. However,
// if others are introduced in future MatchesHash may accept multiple formats,
// and may generate errors for any formats that become obsolete.
func (cp *CachedProvider) MatchesHash(want string) (bool, error) {
	switch {
	case strings.HasPrefix(want, "h1"):
		got, err := cp.HashV1()
		if err != nil {
			return false, err
		}
		return got == want, nil
	default:
		return false, fmt.Errorf("unsupported hash format (this may require a newer version of Terraform)")
	}
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
	// Our HashV1 is really just the Go Modules hash version 1, which is
	// sufficient for our needs and already well-used for identity of
	// Go Modules distribution packages. It is also blocked from incompatible
	// changes by being used in a wide array of go.sum files already.
	//
	// In particular, it also supports computing an equivalent hash from
	// an unpacked zip file, which is not important for Terraform workflow
	// today but is likely to become so in future if we adopt a top-level
	// lockfile mechanism that is intended to be checked in to version control,
	// rather than just a transient lock for a particular local cache directory.
	// (In that case we'd need to check hashes of _packed_ packages, too.)

	// We'll first dereference a possible symlink at our PackageDir location,
	// as would be created if this package were linked in from another cache.
	packageDir, err := filepath.EvalSymlinks(cp.PackageDir)
	if err != nil {
		return "", err
	}

	// Internally, dirhash.Hash1 produces a string containing a sequence of
	// newline-separated path+filehash pairs for all of the files in the
	// directory, and then finally produces a hash of that string to return.
	// In both cases, the hash algorithm is SHA256.
	return dirhash.HashDir(packageDir, "", dirhash.Hash1)
}
