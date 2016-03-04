package acctest

import (
	"math/rand"
	"time"
)

// Helpers for generating random tidbits for use in identifiers to prevent
// collisions in acceptance tests.

// RandInt generates a random integer
func RandInt() int {
	reseed()
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int()
}

// RandString generates a random alphanumeric string of the length specified
func RandString(strlen int) string {
	return RandStringFromCharSet(strlen, CharSetAlphaNum)
}

// RandStringFromCharSet generates a random string by selecting characters from
// the charset provided
func RandStringFromCharSet(strlen int, charSet string) string {
	reseed()
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(result)
}

// Seeds random with current timestamp
func reseed() {
	rand.Seed(time.Now().UTC().UnixNano())
}

const (
	// CharSetAlphaNum is the alphanumeric character set for use with
	// RandStringFromCharSet
	CharSetAlphaNum = "abcdefghijklmnopqrstuvwxyz012346789"

	// CharSetAlpha is the alphabetical character set for use with
	// RandStringFromCharSet
	CharSetAlpha = "abcdefghijklmnopqrstuvwxyz"
)
