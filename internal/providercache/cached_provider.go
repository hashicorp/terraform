package providercache

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

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
func (cp *CachedProvider) Hash() (getproviders.Hash, error) {
	return getproviders.PackageHash(cp.PackageLocation())
}

// MatchesHash returns true if the package on disk matches the given hash,
// or false otherwise. If it cannot traverse the package directory and read
// all of the files in it, or if the hash is in an unsupported format,
// MatchesHash returns an error.
//
// MatchesHash may accept hashes in a number of different formats. Over time
// the set of supported formats may grow and shrink.
func (cp *CachedProvider) MatchesHash(want getproviders.Hash) (bool, error) {
	return getproviders.PackageMatchesHash(cp.PackageLocation(), want)
}

// MatchesAnyHash returns true if the package on disk matches the given hash,
// or false otherwise. If it cannot traverse the package directory and read
// all of the files in it, MatchesAnyHash returns an error.
//
// Unlike the singular MatchesHash, MatchesAnyHash considers unsupported hash
// formats as successfully non-matching, rather than returning an error.
func (cp *CachedProvider) MatchesAnyHash(allowed []getproviders.Hash) (bool, error) {
	return getproviders.PackageMatchesAnyHash(cp.PackageLocation(), allowed)
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
func (cp *CachedProvider) HashV1() (getproviders.Hash, error) {
	return getproviders.PackageHashV1(cp.PackageLocation())
}

// ExecutableFile inspects the cached provider's unpacked package directory for
// something that looks like it's intended to be the executable file for the
// plugin.
//
// This is a bit messy and heuristic-y because historically Terraform used the
// filename itself for local filesystem discovery, allowing some variance in
// the filenames to capture extra metadata, whereas now we're using the
// directory structure leading to the executable instead but need to remain
// compatible with the executable names bundled into existing provider packages.
//
// It will return an error if it can't find a file following the expected
// convention in the given directory.
//
// If found, the path always uses slashes as path separators, even on Windows,
// so that the results are consistent between platforms. Windows accepts both
// slashes and backslashes as long as the separators are consistent within a
// particular path string.
func (cp *CachedProvider) ExecutableFile() (string, error) {
	infos, err := ioutil.ReadDir(cp.PackageDir)
	if err != nil {
		// If the directory itself doesn't exist or isn't readable then we
		// can't access an executable in it.
		return "", fmt.Errorf("could not read package directory: %s", err)
	}

	// For a provider named e.g. tf.example.com/awesomecorp/happycloud, we
	// expect an executable file whose name starts with
	// "terraform-provider-happycloud", followed by zero or more additional
	// characters. If there _are_ additional characters then the first one
	// must be an underscore or a period, like in thse examples:
	// - terraform-provider-happycloud_v1.0.0
	// - terraform-provider-happycloud.exe
	//
	// We don't require the version in the filename to match because the
	// executable's name is no longer authoritative, but packages of "official"
	// providers may continue to use versioned executable names for backward
	// compatibility with Terraform 0.12.
	//
	// We also presume that providers packaged for Windows will include the
	// necessary .exe extension on their filenames but do not explicitly check
	// for that. If there's a provider package for Windows that has a file
	// without that suffix then it will be detected as an executable but then
	// we'll presumably fail later trying to run it.
	wantPrefix := "terraform-provider-" + cp.Provider.Type

	// We'll visit all of the directory entries and take the first (in
	// name-lexical order) that looks like a plausible provider executable
	// name. A package with multiple files meeting these criteria is degenerate
	// but we will tolerate it by ignoring the subsequent entries.
	for _, info := range infos {
		if info.IsDir() {
			continue // A directory can never be an executable
		}
		name := info.Name()
		if !strings.HasPrefix(name, wantPrefix) {
			continue
		}
		remainder := name[len(wantPrefix):]
		if len(remainder) > 0 && (remainder[0] != '_' && remainder[0] != '.') {
			continue // subsequent characters must be delimited by _ or .
		}
		return filepath.ToSlash(filepath.Join(cp.PackageDir, name)), nil
	}

	return "", fmt.Errorf("could not find executable file starting with %s", wantPrefix)
}
