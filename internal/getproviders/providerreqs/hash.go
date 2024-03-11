// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providerreqs

import (
	"fmt"
	"strings"
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
