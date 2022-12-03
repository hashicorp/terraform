package states

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
)

// CheckResults represents a summary snapshot of the status of a set of checks
// declared in configuration, updated after each Terraform Core run that
// changes the state or remote system in a way that might impact the check
// results.
//
// Unlike a checks.State, this type only tracks the overall results for
// each checkable object and doesn't aim to preserve the identity of individual
// checks in the configuration. For our UI reporting purposes, it is entire
// objects that pass or fail based on their declared checks; the individual
// checks have no durable identity between runs, and so are only a language
// design convenience to help authors describe various independent conditions
// with different failure messages each.
//
// CheckResults should typically be considered immutable once constructed:
// instead of updating it in-place,instead construct an entirely new
// CheckResults object based on a fresh checks.State.
type CheckResults struct {
	// ConfigResults has all of the individual check results grouped by the
	// configuration object they relate to.
	//
	// The top-level map here will always have a key for every configuration
	// object that includes checks at the time of evaluating the results,
	// even if there turned out to be no instances of that object and
	// therefore no individual check results.
	ConfigResults addrs.Map[addrs.ConfigCheckable, *CheckResultAggregate]
}

// CheckResultAggregate represents both the overall result for a particular
// configured object that has checks and the individual checkable objects
// it declared, if any.
type CheckResultAggregate struct {
	// Status is the aggregate status across all objects.
	//
	// Sometimes an error or check failure during planning will prevent
	// Terraform Core from even determining the individual checkable objects
	// associated with a downstream configuration object, and that situation is
	// described here by this Status being checks.StatusUnknown and there being
	// no elements in the ObjectResults field.
	//
	// That's different than Terraform Core explicitly reporting that there are
	// no instances of the config object (e.g. a resource with count = 0),
	// which leads to the aggregate status being checks.StatusPass while
	// ObjectResults is still empty.
	Status checks.Status

	ObjectResults addrs.Map[addrs.Checkable, *CheckResultObject]
}

// CheckResultObject is the check status for a single checkable object.
//
// This aggregates together all of the checks associated with a particular
// object into a single pass/fail/error/unknown result, because checkable
// objects have durable addresses that can survive between runs, but their
// individual checks do not. (Module authors are free to reorder their checks
// for a particular object in the configuration with no change in meaning.)
type CheckResultObject struct {
	// Status is the check status of the checkable object, derived from the
	// results of all of its individual checks.
	Status checks.Status

	// FailureMessages is an optional set of module-author-defined messages
	// describing the problems that the checks detected, for objects whose
	// status is checks.StatusFail.
	//
	// (checks.StatusError problems get reported as normal diagnostics during
	// evaluation instead, and so will not appear here.)
	FailureMessages []string
}

// NewCheckResults constructs a new states.CheckResults object that is a
// snapshot of the check statuses recorded in the given checks.State object.
//
// This should be called only after a Terraform Core run has completed and
// recorded any results from running the checks in the given object.
func NewCheckResults(source *checks.State) *CheckResults {
	ret := &CheckResults{
		ConfigResults: addrs.MakeMap[addrs.ConfigCheckable, *CheckResultAggregate](),
	}

	for _, configAddr := range source.AllConfigAddrs() {
		aggr := &CheckResultAggregate{
			Status:        source.AggregateCheckStatus(configAddr),
			ObjectResults: addrs.MakeMap[addrs.Checkable, *CheckResultObject](),
		}

		for _, objectAddr := range source.ObjectAddrs(configAddr) {
			obj := &CheckResultObject{
				Status:          source.ObjectCheckStatus(objectAddr),
				FailureMessages: source.ObjectFailureMessages(objectAddr),
			}
			aggr.ObjectResults.Put(objectAddr, obj)
		}

		ret.ConfigResults.Put(configAddr, aggr)
	}

	// If there aren't actually any configuration objects then we'll just
	// leave the map as a whole nil, because having it be zero-value makes
	// life easier for deep comparisons in unit tests elsewhere.
	if ret.ConfigResults.Len() == 0 {
		ret.ConfigResults.Elems = nil
	}

	return ret
}

// GetObjectResult looks up the result for a single object, or nil if there
// is no such object.
//
// In main code we shouldn't typically need to look up individual objects
// like this, since we'll usually be reporting check results in an aggregate
// form, but determining the result of a particular object is useful in our
// internal unit tests, and so this is here primarily for that purpose.
func (r *CheckResults) GetObjectResult(objectAddr addrs.Checkable) *CheckResultObject {
	if r == nil {
		return nil
	}
	configAddr := objectAddr.ConfigCheckable()

	aggr := r.ConfigResults.Get(configAddr)
	if aggr == nil {
		return nil
	}

	return aggr.ObjectResults.Get(objectAddr)
}

func (r *CheckResults) DeepCopy() *CheckResults {
	if r == nil {
		return nil
	}
	ret := &CheckResults{}
	if r.ConfigResults.Elems == nil {
		return ret
	}

	ret.ConfigResults = addrs.MakeMap[addrs.ConfigCheckable, *CheckResultAggregate]()

	for _, configElem := range r.ConfigResults.Elems {
		aggr := &CheckResultAggregate{
			Status: configElem.Value.Status,
		}

		if configElem.Value.ObjectResults.Elems != nil {
			aggr.ObjectResults = addrs.MakeMap[addrs.Checkable, *CheckResultObject]()

			for _, objectElem := range configElem.Value.ObjectResults.Elems {
				result := &CheckResultObject{
					Status: objectElem.Value.Status,

					// NOTE: We don't deep-copy this slice because it's
					// immutable once constructed by convention.
					FailureMessages: objectElem.Value.FailureMessages,
				}
				aggr.ObjectResults.Put(objectElem.Key, result)
			}
		}

		ret.ConfigResults.Put(configElem.Key, aggr)
	}

	return ret
}

// ObjectAddrsKnown determines whether the set of objects recorded in this
// aggregate is accurate (true) or if it's incomplete as a result of the
// run being interrupted before instance expansion.
func (r *CheckResultAggregate) ObjectAddrsKnown() bool {
	if r.ObjectResults.Len() != 0 {
		// If there are any object results at all then we definitely know.
		return true
	}

	// If we don't have any object addresses then we distinguish a known
	// empty set of objects from an unknown set of objects by the aggregate
	// status being unknown.
	return r.Status != checks.StatusUnknown
}
