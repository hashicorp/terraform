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

// Hash is a specially-formatted string representing a checksum of a package
// or the contents of the package.
//
// A Hash string is always starts with a scheme, which is a short series of
// alphanumeric characters followed by a colon, and then the remainder of the
// string has a different meaning depending on the scheme prefix.
//
// The currently-valid schemes are defined as the constants of type HashScheme
// in this package.
//
// Callers outside of this package must not create Hash values via direct
// conversion. Instead, use either the HashScheme.New method on one of the
// HashScheme contents (for a hash of a particular scheme) or the ParseHash
// function (if hashes of any scheme are acceptable).
type Hash string

// NilHash is the zero value of Hash. It isn't a valid hash, so all of its
// methods will panic.
const NilHash = Hash("")

// ParseHash parses the string representation of a Hash into a Hash value.
//
// A particular version of Terraform only supports a fixed set of hash schemes,
// but this function intentionally allows unrecognized schemes so that we can
// silently ignore other schemes that may be introduced in the future. For
// that reason, the Scheme method of the returned Hash may return a value that
// isn't in one of the HashScheme constants in this package.
//
// This function doesn't verify that the value portion of the given hash makes
// sense for the given scheme. Invalid values are just considered to not match
// any packages.
//
// If this function returns an error then the returned Hash is invalid and
// must not be used.
func ParseHash(s string) (Hash, error) {
	colon := strings.Index(s, ":")
	if colon < 1 { // 1 because a zero-length scheme is not allowed
		return NilHash, fmt.Errorf("hash string must start with a scheme keyword followed by a colon")
	}
	return Hash(s), nil
}

// MustParseHash is a wrapper around ParseHash that panics if it returns an
// error.
func MustParseHash(s string) Hash {
	hash, err := ParseHash(s)
	if err != nil {
		panic(err.Error())
	}
	return hash
}

// Scheme returns the scheme of the recieving hash. If the receiver is not
// using valid syntax then this method will panic.
func (h Hash) Scheme() HashScheme {
	colon := strings.Index(string(h), ":")
	if colon < 0 {
		panic(fmt.Sprintf("invalid hash string %q", h))
	}
	return HashScheme(h[:colon+1])
}

// HasScheme returns true if the given scheme matches the receiver's scheme,
// or false otherwise.
//
// If the receiver is not using valid syntax then this method will panic.
func (h Hash) HasScheme(want HashScheme) bool {
	return h.Scheme() == want
}

// Value returns the scheme-specific value from the recieving hash. The
// meaning of this value depends on the scheme.
//
// If the receiver is not using valid syntax then this method will panic.
func (h Hash) Value() string {
	colon := strings.Index(string(h), ":")
	if colon < 0 {
		panic(fmt.Sprintf("invalid hash string %q", h))
	}
	return string(h[colon+1:])
}

// String returns a string representation of the receiving hash.
func (h Hash) String() string {
	return string(h)
}

// GoString returns a Go syntax representation of the receiving hash.
//
// This is here primarily to help with producing descriptive test failure
// output; these results are not particularly useful at runtime.
func (h Hash) GoString() string {
	if h == NilHash {
		return "getproviders.NilHash"
	}
	switch scheme := h.Scheme(); scheme {
	case HashScheme1:
		return fmt.Sprintf("getproviders.HashScheme1.New(%q)", h.Value())
	case HashSchemeZip:
		return fmt.Sprintf("getproviders.HashSchemeZip.New(%q)", h.Value())
	default:
		// This fallback is for when we encounter lock files or API responses
		// with hash schemes that the current version of Terraform isn't
		// familiar with. They were presumably introduced in a later version.
		return fmt.Sprintf("getproviders.HashScheme(%q).New(%q)", scheme, h.Value())
	}
}

// HashScheme is an enumeration of schemes that are allowed for values of type
// Hash.
type HashScheme string

const (
	// HashScheme1 is the scheme identifier for the first hash scheme.
	//
	// Use HashV1 (or one of its wrapper functions) to calculate hashes with
	// this scheme.
	HashScheme1 HashScheme = HashScheme("h1:")

	// HashSchemeZip is the scheme identifier for the legacy hash scheme that
	// applies to distribution archives (.zip files) rather than package
	// contents, and can therefore only be verified against the original
	// distribution .zip file, not an extracted directory.
	//
	// Use PackageHashLegacyZipSHA to calculate hashes with this scheme.
	HashSchemeZip HashScheme = HashScheme("zh:")
)

// New creates a new Hash value with the receiver as its scheme and the given
// raw string as its value.
//
// It's the caller's responsibility to make sure that the given value makes
// sense for the selected scheme.
func (hs HashScheme) New(value string) Hash {
	return Hash(string(hs) + value)
}

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
func PackageHash(loc PackageLocation) (Hash, error) {
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
func PackageMatchesHash(loc PackageLocation, want Hash) (bool, error) {
	switch want.Scheme() {
	case HashScheme1:
		got, err := PackageHashV1(loc)
		if err != nil {
			return false, err
		}
		return got == want, nil
	case HashSchemeZip:
		archiveLoc, ok := loc.(PackageLocalArchive)
		if !ok {
			return false, fmt.Errorf(`ziphash scheme ("zh:" prefix) is not supported for unpacked provider packages`)
		}
		got, err := PackageHashLegacyZipSHA(archiveLoc)
		if err != nil {
			return false, err
		}
		return got == want, nil
	default:
		return false, fmt.Errorf("unsupported hash format (this may require a newer version of Terraform)")
	}
}

// PackageMatchesAnyHash returns true if the package at the given location
// matches at least one of the given hashes, or false otherwise.
//
// If it cannot read from the given location, PackageMatchesAnyHash returns an
// error. Unlike the singular PackageMatchesHash, PackageMatchesAnyHash
// considers unsupported hash formats as successfully non-matching, rather
// than returning an error.
//
// PackageMatchesAnyHash can be used only with the two local package location
// types PackageLocalDir and PackageLocalArchive, because it needs to access the
// contents of the indicated package in order to compute the hash. If given
// a non-local location this function will always return an error.
func PackageMatchesAnyHash(loc PackageLocation, allowed []Hash) (bool, error) {
	// It's likely that we'll have multiple hashes of the same scheme in
	// the "allowed" set, in which case we'll avoid repeatedly re-reading the
	// given package by caching its result for each of the two
	// currently-supported hash formats. These will be NilHash until we
	// encounter the first hash of the corresponding scheme.
	var v1Hash, zipHash Hash
	for _, want := range allowed {
		switch want.Scheme() {
		case HashScheme1:
			if v1Hash == NilHash {
				got, err := PackageHashV1(loc)
				if err != nil {
					return false, err
				}
				v1Hash = got
			}
			if v1Hash == want {
				return true, nil
			}
		case HashSchemeZip:
			archiveLoc, ok := loc.(PackageLocalArchive)
			if !ok {
				// A zip hash can never match an unpacked directory
				continue
			}
			if zipHash == NilHash {
				got, err := PackageHashLegacyZipSHA(archiveLoc)
				if err != nil {
					return false, err
				}
				zipHash = got
			}
			if zipHash == want {
				return true, nil
			}
		default:
			// If it's not a supported format then it can't match.
			continue
		}
	}
	return false, nil
}

// PreferredHashes examines all of the given hash strings and returns the one
// that the current version of Terraform considers to provide the strongest
// verification.
//
// Returns an empty string if none of the given hashes are of a supported
// format. If PreferredHash returns a non-empty string then it will be one
// of the hash strings in "given", and that hash is the one that must pass
// verification in order for a package to be considered valid.
func PreferredHashes(given []Hash) []Hash {
	// For now this is just filtering for the two hash formats we support,
	// both of which are considered equally "preferred". If we introduce
	// a new scheme like "h2:" in future then, depending on the characteristics
	// of that new version, it might make sense to rework this function so
	// that it only returns "h1:" hashes if the input has no "h2:" hashes,
	// so that h2: is preferred when possible and h1: is only a fallback for
	// interacting with older systems that haven't been updated with the new
	// scheme yet.

	var ret []Hash
	for _, hash := range given {
		switch hash.Scheme() {
		case HashScheme1, HashSchemeZip:
			ret = append(ret, hash)
		}
	}
	return ret
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
func PackageHashLegacyZipSHA(loc PackageLocalArchive) (Hash, error) {
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
	return HashSchemeZip.New(fmt.Sprintf("%x", gotHash)), nil
}

// HashLegacyZipSHAFromSHA is a convenience method to produce the schemed-string
// hash format from an already-calculated hash of a provider .zip archive.
//
// This just adds the "zh:" prefix and encodes the string in hex, so that the
// result is in the same format as PackageHashLegacyZipSHA.
func HashLegacyZipSHAFromSHA(sum [sha256.Size]byte) Hash {
	return HashSchemeZip.New(fmt.Sprintf("%x", sum[:]))
}

// PackageHashV1 computes a hash of the contents of the package at the given
// location using hash algorithm 1. The resulting Hash is guaranteed to have
// the scheme HashScheme1.
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
func PackageHashV1(loc PackageLocation) (Hash, error) {
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

		// The dirhash.HashDir result is already in our expected h1:...
		// format, so we can just convert directly to Hash.
		s, err := dirhash.HashDir(packageDir, "", dirhash.Hash1)
		return Hash(s), err

	case PackageLocalArchive:
		archivePath, err := filepath.EvalSymlinks(string(loc))
		if err != nil {
			return "", err
		}

		// The dirhash.HashDir result is already in our expected h1:...
		// format, so we can just convert directly to Hash.
		s, err := dirhash.HashZip(archivePath, dirhash.Hash1)
		return Hash(s), err

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
func (m PackageMeta) Hash() (Hash, error) {
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
func (m PackageMeta) MatchesHash(want Hash) (bool, error) {
	return PackageMatchesHash(m.Location, want)
}

// MatchesAnyHash returns true if the package at the location associated with
// the receiver matches at least one of the given hashes, or false otherwise.
//
// If it cannot read from the given location, MatchesHash returns an error.
// Unlike the signular MatchesHash, MatchesAnyHash considers an unsupported
// hash format to be a successful non-match.
func (m PackageMeta) MatchesAnyHash(acceptable []Hash) (bool, error) {
	return PackageMatchesAnyHash(m.Location, acceptable)
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
func (m PackageMeta) HashV1() (Hash, error) {
	return PackageHashV1(m.Location)
}
