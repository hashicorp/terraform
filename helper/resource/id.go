package resource

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

const UniqueIdPrefix = `terraform-`

// Helper for a resource to generate a unique identifier w/ default prefix
func UniqueId() string {
	return PrefixedUniqueId(UniqueIdPrefix)
}

// Helper for a resource to generate a unique identifier w/ given prefix
//
// This uses a simple RFC 4122 v4 UUID with some basic cosmetic filters
// applied (base32, remove padding, downcase) to make visually distinguishing
// identifiers easier.
func PrefixedUniqueId(prefix string) string {
	return fmt.Sprintf("%s%s", prefix,
		strings.ToLower(
			strings.Replace(
				base32.StdEncoding.EncodeToString(uuidV4()),
				"=", "", -1)))
}

func uuidV4() []byte {
	var uuid [16]byte

	// Set all the other bits to randomly (or pseudo-randomly) chosen
	// values.
	rand.Read(uuid[:])

	// Set the two most significant bits (bits 6 and 7) of the
	// clock_seq_hi_and_reserved to zero and one, respectively.
	uuid[8] = (uuid[8] | 0x80) & 0x8f

	// Set the four most significant bits (bits 12 through 15) of the
	// time_hi_and_version field to the 4-bit version number from Section 4.1.3.
	uuid[6] = (uuid[6] | 0x40) & 0x4f

	return uuid[:]
}
