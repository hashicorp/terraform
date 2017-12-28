package resource

import (
	"fmt"
	"strconv"
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
// After the prefix, the ID consists of an incrementing base 36 value.
// The value is made of the current timestamp in base 36 followed by an
// incrementing base 36 counter.
// This means that the identifier can grow in size depending on the number of
// items handled by terraform.
// Because the first part is using the timestamp, it is always possible to sort
// the identifiers of multiple resources alphabetically (with numbers first)
// to get the list of resources created from the oldest to the newest.
// The timestamp means that multiple IDs created with the same prefix will sort
// in the order of their creation, even across multiple terraform executions, as
// long as the clock is not turned back between calls.
func PrefixedUniqueId(prefix string) string {
	timestamp := strconv.FormatUint(time.Now().Unix(), 36)

	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	id := strconv.FormatUint(idCounter, 36)
	return fmt.Sprintf("%s%s%s", prefix, timestamp, id)
}
