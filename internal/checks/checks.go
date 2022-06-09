package checks

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Checks represents a full set of checks from across an entire configuration,
// along with their current statuses.
//
// Both read and write access to a Checks is concurrency-safe.
type Checks struct {
	mu sync.Mutex

	// objectChecks is a map from the unique key of an addrs.Checkable to
	// a slice of the checks related to that checkable object, in the order
	// they were declared in the configuration.
	//
	// We establish the map keys and slice sizes of this data structure during
	// initial construction based on what we find in the configuration.
	// Subsequent updates then just mutate the Status and ErrorMessage fields
	// of the leaf Checks in-place, without any further allocations.
	objectChecks map[addrs.UniqueKey][]Check
}

// Check represents a check declared in the configuration and its current
// status.
type Check struct {
	// Addr is the address of the check.
	//
	// The Container field of Addr is the object that the check belongs to.
	// For example, for a resource precondition or post condition the container
	// is the address of an instance of the resource instance that the
	// condition was configured against.
	Object addrs.Checkable

	// Status is the most recently determined status for the check. During
	// planning this can temporarily become CheckPending if the condition
	// expression depends on an unknown value, but in any stable state snapshot
	// it will always be either CheckPassed or CheckFailed.
	Status CheckStatus

	// ErrorMessage is the error message string configured by the author of
	// the check.
	//
	// This is populated only when CheckStatus is either CheckPassed or
	// CheckFailed, because we evaluate the error message expression only when
	// evaluating the condition for a check.
	//
	// The error message is essentially a natural-language description of the
	// _opposite_ of the assertion the check condition makes, and so it may
	// be confusing to show it when reporting that a check has passed.
	ErrorMessage string
}
