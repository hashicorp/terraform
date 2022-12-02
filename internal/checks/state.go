package checks

import (
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

// State is a container for state tracking of all of the the checks declared in
// a particular Terraform configuration and their current statuses.
//
// A State object is mutable during plan and apply operations but should
// otherwise be treated as a read-only snapshot of the status of checks
// at a particular moment.
//
// The checks State tracks a few different concepts:
//   - configuration objects: items in the configuration which statically
//     declare some checks associated with zero or more checkable objects.
//   - checkable objects: dynamically-determined objects that are each
//     associated with one configuration object.
//   - checks: a single check that is declared as part of a configuration
//     object and then resolved once for each of its associated checkable
//     objects.
//   - check statuses: the current state of a particular check associated
//     with a particular checkable object.
//
// This container type is concurrency-safe for both reads and writes through
// its various methods.
type State struct {
	mu sync.Mutex

	statuses    addrs.Map[addrs.ConfigCheckable, *configCheckableState]
	failureMsgs addrs.Map[addrs.Check, string]
}

// configCheckableState is an internal part of type State that represents
// the evaluation status for a particular addrs.ConfigCheckable address.
//
// Its initial state, at the beginning of a run, is that it doesn't even know
// how many checkable objects will be dynamically-declared yet. Terraform Core
// will notify the State object of the associated Checkables once
// it has decided the appropriate expansion of that configuration object,
// and then will gradually report the results of each check once the graph
// walk reaches it.
//
// This must be accessed only while holding the mutex inside the associated
// State object.
type configCheckableState struct {
	// checkTypes captures the expected number of checks of each type
	// associated with object declared by this configuration construct. Since
	// checks are statically declared (even though the checkable objects
	// aren't) we can compute this only from the configuration.
	checkTypes map[addrs.CheckType]int

	// objects represents the set of dynamic checkable objects associated
	// with this configuration construct. This is initially nil to represent
	// that we don't know the objects yet, and is replaced by a non-nil map
	// once Terraform Core reports the expansion of this configuration
	// construct.
	//
	// The leaf Status values will initially be StatusUnknown
	// and then gradually updated by Terraform Core as it visits the
	// individual checkable objects and reports their status.
	objects addrs.Map[addrs.Checkable, map[addrs.CheckType][]Status]
}

// NOTE: For the "Report"-prefixed methods that we use to gradually update
// the structure with results during a plan or apply operation, see the
// state_report.go file also in this package.

// NewState returns a new State object representing the check statuses of
// objects declared in the given configuration.
//
// The configuration determines which configuration objects and associated
// checks we'll be expecting to see, so that we can seed their statuses as
// all unknown until we see affirmative reports sent by the Report-prefixed
// methods on Checks.
func NewState(config *configs.Config) *State {
	return &State{
		statuses: initialStatuses(config),
	}
}

// ConfigHasChecks returns true if and only if the given address refers to
// a configuration object that this State object is expecting to recieve
// statuses for.
//
// Other methods of Checks will typically panic if given a config address
// that would not have returned true from ConfigHasChecked.
func (c *State) ConfigHasChecks(addr addrs.ConfigCheckable) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statuses.Has(addr)
}

// AllConfigAddrs returns all of the addresses of all configuration objects
// that could potentially produce checkable objects at runtime.
//
// This is a good starting point for reporting on the outcome of all of the
// configured checks at the configuration level of granularity, e.g. for
// automated testing reports where we want to report the status of all
// configured checks even if the graph walk aborted before we reached any
// of their objects.
func (c *State) AllConfigAddrs() addrs.Set[addrs.ConfigCheckable] {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statuses.Keys()
}

// ObjectAddrs returns the addresses of individual checkable objects belonging
// to the configuration object with the given address.
//
// This will panic if the given address isn't a known configuration object
// that has checks.
func (c *State) ObjectAddrs(configAddr addrs.ConfigCheckable) addrs.Set[addrs.Checkable] {
	c.mu.Lock()
	defer c.mu.Unlock()

	st, ok := c.statuses.GetOk(configAddr)
	if !ok {
		panic(fmt.Sprintf("unknown configuration object %s", configAddr))
	}

	ret := addrs.MakeSet[addrs.Checkable]()
	for _, elem := range st.objects.Elems {
		ret.Add(elem.Key)
	}
	return ret

}

// AggregateCheckStatus returns a summarization of all of the check results
// for a particular configuration object into a single status.
//
// The given address must refer to an object within the configuration that
// this Checks was instantiated from, or this method will panic.
func (c *State) AggregateCheckStatus(addr addrs.ConfigCheckable) Status {
	c.mu.Lock()
	defer c.mu.Unlock()

	st, ok := c.statuses.GetOk(addr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown configuration object %s", addr))
	}

	if st.objects.Elems == nil {
		// If we don't even know how many objects we have for this
		// configuration construct then that summarizes as unknown.
		// (Note: this is different than Elems being a non-nil empty map,
		// which means that we know there are zero objects and therefore
		// the aggregate result will be pass to pass below.)
		return StatusUnknown
	}

	// Otherwise, our result depends on how many of our known objects are
	// in each status.
	errorCount := 0
	failCount := 0
	unknownCount := 0

	for _, objects := range st.objects.Elems {
		for _, checks := range objects.Value {
			for _, status := range checks {
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
		}
	}

	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

// ObjectCheckStatus returns a summarization of all of the check results
// for a particular checkable object into a single status.
//
// The given address must refer to a checkable object that Terraform Core
// previously reported while doing a graph walk, or this method will panic.
func (c *State) ObjectCheckStatus(addr addrs.Checkable) Status {
	c.mu.Lock()
	defer c.mu.Unlock()

	configAddr := addr.ConfigCheckable()

	st, ok := c.statuses.GetOk(configAddr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown object %s", addr))
	}
	if st.objects.Elems == nil {
		panic(fmt.Sprintf("request for status of %s before establishing the checkable objects for %s", addr, configAddr))
	}
	checks, ok := st.objects.GetOk(addr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown object %s", addr))
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0
	for _, statuses := range checks {
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
	}
	return summarizeCheckStatuses(errorCount, failCount, unknownCount)
}

// ObjectCheckStatusByConditionType is like ObjectCheckStatus but it
// restricts its analysis only to conditions of the given type.
//
// This can be useful when some condition types are prerequisites for others,
// so that the downstream type can check whether its upstreams are failing
// and skip its own work if so.
func (c *State) ObjectCheckStatusByConditionType(addr addrs.Checkable, condType addrs.CheckType) Status {
	c.mu.Lock()
	defer c.mu.Unlock()

	configAddr := addr.ConfigCheckable()

	st, ok := c.statuses.GetOk(configAddr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown object %s", addr))
	}
	if st.objects.Elems == nil {
		panic(fmt.Sprintf("request for status of %s before establishing the checkable objects for %s", addr, configAddr))
	}
	checks, ok := st.objects.GetOk(addr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown object %s", addr))
	}

	errorCount := 0
	failCount := 0
	unknownCount := 0
	for _, status := range checks[condType] {
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

// ObjectFailureMessages returns the zero or more failure messages reported
// for the object with the given address.
//
// Failure messages are recorded only for checks whose status is StatusFail,
// but since this aggregates together the results of all of the checks
// on the given object it's possible for there to be a mixture of failures
// and errors at the same time, which would aggregate as StatusError in
// ObjectCheckStatus's result because errors are defined as "stronger"
// than failures.
func (c *State) ObjectFailureMessages(addr addrs.Checkable) []string {
	var ret []string

	configAddr := addr.ConfigCheckable()

	st, ok := c.statuses.GetOk(configAddr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown object %s", addr))
	}
	if st.objects.Elems == nil {
		panic(fmt.Sprintf("request for status of %s before establishing the checkable objects for %s", addr, configAddr))
	}
	checksByType, ok := st.objects.GetOk(addr)
	if !ok {
		panic(fmt.Sprintf("request for status of unknown object %s", addr))
	}

	for checkType, checks := range checksByType {
		for i, status := range checks {
			if status == StatusFail {
				checkAddr := addrs.NewCheck(addr, checkType, i)
				msg := c.failureMsgs.Get(checkAddr)
				if msg != "" {
					ret = append(ret, msg)
				}
			}
		}
	}

	// We always return the messages in a lexical sort order just so that
	// it'll be consistent between runs if we still have the same problems.
	sort.Strings(ret)

	return ret
}

func summarizeCheckStatuses(errorCount, failCount, unknownCount int) Status {
	switch {
	case errorCount > 0:
		// If we saw any errors then we'll treat the whole thing as errored.
		return StatusError
	case failCount > 0:
		// If anything failed then this whole configuration construct failed.
		return StatusFail
	case unknownCount > 0:
		// If nothing failed but we still have unknowns then our outcome isn't
		// known yet.
		return StatusUnknown
	default:
		// If we have no failures and no unknowns then either we have all
		// passes or no checkable objects at all, both of which summarize as
		// a pass.
		return StatusPass
	}
}
