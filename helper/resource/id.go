package resource

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
)

const UniqueIdPrefix = `terraform-`

// Helper for a resource to generate a unique identifier w/ default prefix
func UniqueId() string {
	return PrefixedUniqueId(UniqueIdPrefix)
}

// Helper for a resource to generate a unique identifier w/ given prefix
//
// After the prefix, the ID consists of a timestamp and 12 random base32
// characters.  The timestamp means that multiple IDs created with the same
// prefix will sort in the order of their creation.
func PrefixedUniqueId(prefix string) string {
	// Be precise to the level nanoseconds, but remove the dot before the
	// nanosecond. We assume that the randomCharacters call takes at least a
	// nanosecond, so that multiple calls to this function from the same goroutine
	// will have distinct ordered timestamps.
	timestamp := strings.Replace(
		time.Now().UTC().Format("20060102150405.000000000"),
		".",
		"", 1)
	// This uses 3 characters, so that the length of the unique ID is the same as
	// it was before we added the timestamp prefix, which happened to be 23
	// characters.
	return fmt.Sprintf("%s%s%s", prefix, timestamp, randomCharacters(3))
}

func randomCharacters(n int) string {
	// Base32 has 5 bits of information per character.
	b := randomBytes(n * 8 / 5)
	chars := strings.ToLower(
		strings.Replace(
			base32.StdEncoding.EncodeToString(b),
			"=", "", -1))
	// Trim extra characters.
	return chars[:n]
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}
