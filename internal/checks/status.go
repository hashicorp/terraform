package checks

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// Status represents the status of an individual check associated with a
// checkable object.
type Status rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=Status

const (
	// StatusUnknown represents that there is not yet a conclusive result
	// for the check, either because we haven't yet visited its associated
	// object or because the check condition itself depends on a value not
	// yet known during planning.
	StatusUnknown Status = 0
	// NOTE: Our implementation relies on StatusUnknown being the zero value
	// of Status.

	// StatusPass represents that Terraform Core has evaluated the check's
	// condition and it returned true, indicating success.
	StatusPass Status = 'P'

	// StatusFail represents that Terraform Core has evaluated the check's
	// condition and it returned false, indicating failure.
	StatusFail Status = 'F'

	// StatusError represents that Terraform Core tried to evaluate the check's
	// condition but encountered an error while evaluating the check expression.
	//
	// This is different than StatusFail because StatusFail indiciates that
	// the condition was valid and returned false, whereas StatusError
	// indicates that the condition was not valid at all.
	StatusError Status = 'E'
)

// ForExpectedFailure reinterprets the reciever in a context where failure is
// expected, such as in a test step that is intentionally causing a condition
// to fail in order to test the condition itself.
//
// This method swaps [StatusPass] for [StatusFail] and vice-versa. It also
// treates [StatusUnknown] as [StatusError] because any test which is asserting
// failure for a particular checkable object must produce a definitive result
// for that object. [StatusError] also stays as [StatusError] because that
// indicates that the check was totally invalid, and that's not the same thing
// as the failure of a valid check.
func (s Status) ForExpectedFailure() Status {
	switch s {
	case StatusPass:
		return StatusFail
	case StatusFail:
		return StatusPass
	case StatusUnknown:
		return StatusError
	default:
		return s
	}
}

// StatusForCtyValue returns the Status value corresponding to the given
// cty Value, which must be one of either cty.True, cty.False, or
// cty.UnknownVal(cty.Bool) or else this function will panic.
//
// The current behavior of this function is:
//
//	cty.True                  StatusPass
//	cty.False                 StatusFail
//	cty.UnknownVal(cty.Bool)  StatusUnknown
//
// Any other input will panic. Note that there's no value that can produce
// StatusError, because in case of a condition error there will not typically
// be a result value at all.
func StatusForCtyValue(v cty.Value) Status {
	if !v.Type().Equals(cty.Bool) {
		panic(fmt.Sprintf("cannot use %s as check status", v.Type().FriendlyName()))
	}
	if v.IsNull() {
		panic("cannot use null as check status")
	}

	switch {
	case v == cty.True:
		return StatusPass
	case v == cty.False:
		return StatusFail
	case !v.IsKnown():
		return StatusUnknown
	default:
		// Should be impossible to get here unless something particularly
		// weird is going on, like a marked condition result.
		panic(fmt.Sprintf("cannot use %#v as check status", v))
	}
}

// AggregateCheckStatus is a helper for finding an approximate status that
// describes the "strongest" status from a set of statuses which are presumably
// from some child objects.
//
// "Strongest" here means a prioritization order where errors trump failures,
// failures trump passes, and passes trump unknowns. This prioritization order
// reflects that if there's at least one failure then the overall status
// cannot possibly be "pass" no matter if there are other sibling checks
// passing.
//
// If the given set of objects is zero-length then the result is always
// StatusPass, assuming that the absense of checks means an automatic pass.
// Callers should check for this case separately if they need different
// treatment of an empty set.
func AggregateCheckStatus(statuses ...Status) Status {
	if len(statuses) == 0 { // Easy path
		return StatusPass
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0

	for _, status := range statuses {
		switch status {
		case StatusPass:
			// ok
		case StatusFail:
			failCount++
		case StatusError:
			errorCount++
		default:
			unknownCount++
		}
	}

	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

// AggregateCheckStatusSlice is a helper for finding an approximate status that
// describes the "strongest" status from a set of statuses which are presumably
// from some child objects, represented as a slice.
//
// "Strongest" here means a prioritization order where errors trump failures,
// failures trump passes, and passes trump unknowns. This prioritization order
// reflects that if there's at least one failure then the overall status
// cannot possibly be "pass" no matter if there are other sibling checks
// passing.
//
// If the given set of objects is zero-length then the result is always
// StatusPass, assuming that the absense of checks means an automatic pass.
// Callers should check for this case separately if they need different
// treatment of an empty set.
//
// The separate getStatus callback allows extracting status information
// from each element of "objects" in turn without first allocating a separate
// slice to copy them all into. If T is already [Status] then you can use
// [AggregateCheckStatusIdentity] for getStatus.
func AggregateCheckStatusSlice[T any](objects []T, getStatus func(T) Status) Status {
	if len(objects) == 0 { // Easy path
		return StatusPass
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0

	for _, obj := range objects {
		status := getStatus(obj)
		switch status {
		case StatusPass:
			// ok
		case StatusFail:
			failCount++
		case StatusError:
			errorCount++
		default:
			unknownCount++
		}
	}

	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

// AggregateCheckStatusMap is a helper for finding an approximate status that
// describes the "strongest" status from a set of statuses which are presumably
// from some child objects, represented as a map.
//
// "Strongest" here means a prioritization order where errors trump failures,
// failures trump passes, and passes trump unknowns. This prioritization order
// reflects that if there's at least one failure then the overall status
// cannot possibly be "pass" no matter if there are other sibling checks
// passing.
//
// If the given set of objects is zero-length then the result is always
// StatusPass, assuming that the absense of checks means an automatic pass.
// Callers should check for this case separately if they need different
// treatment of an empty set.
//
// The separate getStatus callback allows extracting status information
// from each element of "objects" in turn without first allocating a separate
// slice to copy them all into.
func AggregateCheckStatusMap[K comparable, V any](objects map[K]V, getStatus func(K, V) Status) Status {
	if len(objects) == 0 { // Easy path
		return StatusPass
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0

	for k, v := range objects {
		status := getStatus(k, v)
		switch status {
		case StatusPass:
			// ok
		case StatusFail:
			failCount++
		case StatusError:
			errorCount++
		default:
			unknownCount++
		}
	}

	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

// AggregateCheckStatusAddrsMap is a helper for finding an approximate status that
// describes the "strongest" status from a set of statuses which are presumably
// from some child objects, represented as an addrs.Map.
//
// "Strongest" here means a prioritization order where errors trump failures,
// failures trump passes, and passes trump unknowns. This prioritization order
// reflects that if there's at least one failure then the overall status
// cannot possibly be "pass" no matter if there are other sibling checks
// passing.
//
// If the given set of objects is zero-length then the result is always
// StatusPass, assuming that the absense of checks means an automatic pass.
// Callers should check for this case separately if they need different
// treatment of an empty set.
//
// The separate getStatus callback allows extracting status information
// from each element of "objects" in turn without first allocating a separate
// slice to copy them all into.
func AggregateCheckStatusAddrsMap[K addrs.UniqueKeyer, V any](objects addrs.Map[K, V], getStatus func(K, V) Status) Status {
	if objects.Len() == 0 { // Easy path
		return StatusPass
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0

	for _, elem := range objects.Elems {
		status := getStatus(elem.Key, elem.Value)
		switch status {
		case StatusPass:
			// ok
		case StatusFail:
			failCount++
		case StatusError:
			errorCount++
		default:
			unknownCount++
		}
	}

	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

// AggregateCheckStatusAddrsSet is a helper for finding an approximate status that
// describes the "strongest" status from a set of statuses which are presumably
// from some child objects, represented as an addrs.Set.
//
// "Strongest" here means a prioritization order where errors trump failures,
// failures trump passes, and passes trump unknowns. This prioritization order
// reflects that if there's at least one failure then the overall status
// cannot possibly be "pass" no matter if there are other sibling checks
// passing.
//
// If the given set of objects is zero-length then the result is always
// StatusPass, assuming that the absense of checks means an automatic pass.
// Callers should check for this case separately if they need different
// treatment of an empty set.
//
// The separate getStatus callback allows extracting status information
// from each element of "objects" in turn without first allocating a separate
// slice to copy them all into. If T is already [Status] then you can use
// [AggregateCheckStatusIdentity] for getStatus.
func AggregateCheckStatusAddrsSet[T addrs.UniqueKeyer](objects addrs.Set[T], getStatus func(T) Status) Status {
	if len(objects) == 0 { // Easy path
		return StatusPass
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0

	for _, obj := range objects {
		status := getStatus(obj)
		switch status {
		case StatusPass:
			// ok
		case StatusFail:
			failCount++
		case StatusError:
			errorCount++
		default:
			unknownCount++
		}
	}

	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

func AggregateCheckStatusIdentity(status Status) Status {
	return status
}
