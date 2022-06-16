package states

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
)

// CheckResults represents a snapshot of the status of a set of checks declared
// in configuration, updated after each Terraform Core run that changes the state
// or remote system in a way that might impact the check results.
//
// Unlike a checks.State, this type only tracks the leaf check results and
// doesn't retain any information about how the checks were declared in
// configuration. That's because this subset of the data is intended to survive
// from one run to the next, and the next run will probably have a changed
// configuration anyway and so it's only meaningful to consider changes
// to the presence of checks and their statuses between runs.
type CheckResults struct {
	Results []*CheckResult
}

// Check is the state of a single check, inside the Checks struct.
type CheckResult struct {
	CheckAddr addrs.Check
	Status    checks.Status

	// If Status is checks.StatusError then there might also be an error
	// message describing what problem the check detected.
	ErrorMessage string
}

// NewChecks constructs a new states.Checks object that is a snapshot of the
// check statuses recorded in the given checks.State object.
//
// This should be called only after a Terraform Core run has complete and
// recorded any results from running the checks in the given object.
func NewCheckResults(source *checks.State) *CheckResults {
	statuses := source.AllCheckStatuses()
	if statuses.Len() == 0 {
		return &CheckResults{}
	}

	results := make([]*CheckResult, 0, statuses.Len())
	for _, elem := range statuses.Elems {
		errMsg := source.CheckFailureMessage(elem.Key)
		results = append(results, &CheckResult{
			CheckAddr:    elem.Key,
			Status:       elem.Value,
			ErrorMessage: errMsg,
		})
	}

	return &CheckResults{results}
}

// AllCheckedObjects returns a set of all of the objects that have at least
// one check in the set of results.
func (r *CheckResults) AllCheckedObjects() addrs.Set[addrs.Checkable] {
	if r == nil || len(r.Results) == 0 {
		return nil
	}
	ret := addrs.MakeSet[addrs.Checkable]()
	for _, result := range r.Results {
		ret.Add(result.CheckAddr.Container)
	}
	return ret
}

// GetCheckResults scans over the checks and returns the first one that
// has the given address, or nil if there is no such check.
//
// In main code we shouldn't typically need to look up individual checks
// like this, since we'll usually be reporting check results in an aggregate
// form, but determining the result of a particular check is useful in our
// internal unit tests, and so this is here primarily for that purpose.
func (r *CheckResults) GetCheckResult(addr addrs.Check) *CheckResult {
	for _, result := range r.Results {
		if addrs.Equivalent(result.CheckAddr, addr) {
			return result
		}
	}
	return nil
}

func (r *CheckResults) DeepCopy() *CheckResults {
	if r == nil {
		return nil
	}
	if len(r.Results) == 0 {
		return &CheckResults{}
	}

	// Everything inside CheckResult is either a value type or is
	// treated as immutable by convention, so we don't need to
	// copy any deeper.
	results := make([]*CheckResult, len(r.Results))
	copy(results, r.Results)
	return &CheckResults{results}
}
