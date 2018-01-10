package resource

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const UniqueIdPrefix = `terraform-`

// idCounter is a monotonic counter for generating ordered unique ids.
var idMutex sync.Mutex
var idCounter uint32

// Helper for a resource to generate a unique identifier w/ default prefix
func UniqueId() string {
	return PrefixedUniqueId(UniqueIdPrefix)
}

// Helper for a resource to generate a unique identifier w/ given prefix
//
// After the prefix, the ID consists of an incrementing 26 digit value (to match
// previous timestamp output).  After the prefix, the ID consists of a timestamp
// and an incrementing 8 hex digit value The timestamp means that multiple IDs
// created with the same prefix will sort in the order of their creation, even
// across multiple terraform executions, as long as the clock is not turned back
// between calls, and as long as any given terraform execution generates fewer
// than 4 billion IDs.
func PrefixedUniqueId(prefix string) string {
	// Be precise to 4 digits of fractional seconds, but remove the dot before the
	// fractional seconds.
	timestamp := strings.Replace(
		time.Now().UTC().Format("20060102150405.0000"), ".", "", 1)

	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	return fmt.Sprintf("%s%s%08x", prefix, timestamp, idCounter)
}
