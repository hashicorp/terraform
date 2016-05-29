package random

import (
	"hash/crc64"
	"math/rand"
	"time"
)

// NewRand returns a seeded random number generator, using a seed derived
// from the provided string.
//
// If the seed string is empty, the current time is used as a seed.
func NewRand(seed string) *rand.Rand {
	var seedInt int64
	if seed != "" {
		crcTable := crc64.MakeTable(crc64.ISO)
		seedInt = int64(crc64.Checksum([]byte(seed), crcTable))
	} else {
		seedInt = time.Now().Unix()
	}

	randSource := rand.NewSource(seedInt)
	return rand.New(randSource)
}
