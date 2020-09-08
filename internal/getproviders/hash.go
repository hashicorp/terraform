package getproviders

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

const h1Prefix = "h1:"
const zipHashPrefix = "zh:"

// PackageHash computes a hash of the contents of the package at the given
// location, using whichever hash algorithm is the current default.
//
// Currently, this method returns version 1 hashes as produced by the
// function PackageHashV1, but this function may switch to other versions in
// later releases. Call PackageHashV1 directly if you specifically need a V1
// hash.
//
// PackageHash can be used only with the two local package location types
// PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func PackageHash(loc PackageLocation) (string, error) {
	return PackageHashV1(loc)
}

// PackageMatchesHash returns true if the package at the given location matches
// the given hash, or false otherwise.
//
// If it cannot read from the given location, or if the given hash is in an
// unsupported format, PackageMatchesHash returns an error.
//
// There is currently only one hash format, as implemented by HashV1. However,
// if others are introduced in future PackageMatchesHash may accept multiple
// formats, and may generate errors for any formats that become obsolete.
//
// PackageMatchesHash can be used only with the two local package location types
// PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func PackageMatchesHash(loc PackageLocation, want string) (bool, error) {
	switch {
	case strings.HasPrefix(want, h1Prefix):
		got, err := PackageHashV1(loc)
		if err != nil {
			return false, err
		}
		return got == want, nil
	default:
		return false, fmt.Errorf("unsupported hash format (this may require a newer version of Terraform)")
	}
}

// PreferredHash examines all of the given hash strings and returns the one
// that the current version of Terraform considers to provide the strongest
// verification.
//
// Returns an empty string if none of the given hashes are of a supported
// format. If PreferredHash returns a non-empty string then it will be one
// of the hash strings in "given", and that hash is the one that must pass
// verification in order for a package to be considered valid.
func PreferredHash(given []string) string {
	for _, s := range given {
		if strings.HasPrefix(s, h1Prefix) {
			return s
		}
	}
	return ""
}

// PackageHashLegacyZipSHA implements the old provider package hashing scheme
// of taking a SHA256 hash of the containing .zip archive itself, rather than
// of the contents of the archive.
//
// The result is a hash string with the "zh:" prefix, which is intended to
// represent "zip hash". After the prefix is a lowercase-hex encoded SHA256
// checksum, intended to exactly match the formatting used in the registry
// API (apart from the prefix) so that checksums can be more conveniently
// compared by humans.
//
// Because this hashing scheme uses the official provider .zip file as its
// input, it accepts only PackageLocalArchive locations.
func PackageHashLegacyZipSHA(loc PackageLocalArchive) (string, error) {
	archivePath, err := filepath.EvalSymlinks(string(loc))
	if err != nil {
		return "", err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	gotHash := h.Sum(nil)
	return fmt.Sprintf("%s%x", zipHashPrefix, gotHash), nil
}

// HashLegacyZipSHAFromSHA is a convenience method to produce the schemed-string
// hash format from an already-calculated hash of a provider .zip archive.
//
// This just adds the "zh:" prefix and encodes the string in hex, so that the
// result is in the same format as PackageHashLegacyZipSHA.
func HashLegacyZipSHAFromSHA(sum [sha256.Size]byte) string {
	return fmt.Sprintf("%s%x", zipHashPrefix, sum[:])
}

// PackageHashV1 computes a hash of the contents of the package at the given
// location using hash algorithm 1.
//
// The hash covers the paths to files in the directory and the contents of
// those files. It does not cover other metadata about the files, such as
// permissions.
//
// This function is named "PackageHashV1" in anticipation of other hashing
// algorithms being added in a backward-compatible way in future. The result
// from PackageHashV1 always begins with the prefix "h1:" so that callers can
// distinguish the results of potentially multiple different hash algorithms in
// future.
//
// PackageHashV1 can be used only with the two local package location types
// PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func PackageHashV1(loc PackageLocation) (string, error) {
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
	//
	// Internally, dirhash.Hash1 produces a string containing a sequence of
	// newline-separated path+filehash pairs for all of the files in the
	// directory, and then finally produces a hash of that string to return.
	// In both cases, the hash algorithm is SHA256.

	switch loc := loc.(type) {

	case PackageLocalDir:
		// We'll first dereference a possible symlink at our PackageDir location,
		// as would be created if this package were linked in from another cache.
		packageDir, err := filepath.EvalSymlinks(string(loc))
		if err != nil {
			return "", err
		}

		return dirhash.HashDir(packageDir, "", dirhash.Hash1)

	case PackageLocalArchive:
		archivePath, err := filepath.EvalSymlinks(string(loc))
		if err != nil {
			return "", err
		}

		return dirhash.HashZip(archivePath, dirhash.Hash1)

	default:
		return "", fmt.Errorf("cannot hash package at %s", loc.String())
	}
}

// Hash computes a hash of the contents of the package at the location
// associated with the reciever, using whichever hash algorithm is the current
// default.
//
// This method will change to use new hash versions as they are introduced
// in future. If you need a specific hash version, call the method for that
// version directly instead, such as HashV1.
//
// Hash can be used only with the two local package location types
// PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func (m PackageMeta) Hash() (string, error) {
	return PackageHash(m.Location)
}

// MatchesHash returns true if the package at the location associated with
// the receiver matches the given hash, or false otherwise.
//
// If it cannot read from the given location, or if the given hash is in an
// unsupported format, MatchesHash returns an error.
//
// MatchesHash can be used only with the two local package location types
// PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func (m PackageMeta) MatchesHash(want string) (bool, error) {
	return PackageMatchesHash(m.Location, want)
}

// HashV1 computes a hash of the contents of the package at the location
// associated with the receiver using hash algorithm 1.
//
// The hash covers the paths to files in the directory and the contents of
// those files. It does not cover other metadata about the files, such as
// permissions.
//
// HashV1 can be used only with the two local package location types
// PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func (m PackageMeta) HashV1() (string, error) {
	return PackageHashV1(m.Location)
}
