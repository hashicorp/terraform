package resource

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
)

const UniqueIdPrefix = `terraform-`

// idCounter is a randomly seeded monotonic counter for generating ordered
// unique ids.  It uses a big.Int so we can easily increment a long numeric
// string.  The max possible hex value here with 12 random bytes is
// "01000000000000000000000000", so there's no chance of rollover during
// operation.
var idMutex sync.Mutex
var idCounter = big.NewInt(0).SetBytes(randomBytes(12))

// Helper for a resource to generate a unique identifier w/ default prefix
func UniqueId() string {
	return PrefixedUniqueId(UniqueIdPrefix)
}

// Helper for a resource to generate a unique identifier w/ given prefix
//
// After the prefix, the ID consists of an incrementing 26 digit value (to match
// previous timestamp output).
func PrefixedUniqueId(prefix string) string {
	idMutex.Lock()
	defer idMutex.Unlock()
	return fmt.Sprintf("%s%026x", prefix, idCounter.Add(idCounter, big.NewInt(1)))
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}
