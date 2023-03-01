package deferring

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Deferred is the main type tracking which objects have already been deferred
// and what other objects might have their planning deferred as a result.
type Deferred struct {
	// See the description of the resourceDeps argument of [NewDeferred].
	resourceDeps addrs.DirectedGraph[addrs.ConfigResource]

	// deferred is the set of addresses of all resources that either had their
	// expansion deferred or whose provider requested deferral during of a
	// specific instance during planning. The detail about exactly which
	// instances were deferred is tracked in the plan, instead of in this
	// data structure.
	deferredResources addrs.Set[addrs.ConfigResource]

	// forceAllDeferred is a special case for when Terraform Core is planning
	// a configuration that is downstream of some other configuration that
	// included deferrals, which means that nothing at all in this configuration
	// can be definitively planned yet.
	forceAllDeferred bool

	mu sync.Mutex
}

// NewDeferred allocates and returns a new [Deferred] which can track the
// deferral statuses for objects during Terraform Core evaluation and answer
// questions about whether downstream objects ought to be deferred as a result
// of existing deferrals.
//
// resourceDeps describes the dependency graph of unexpanded resources,
// which we need to decide whether a particular resource ought to be
// deferred as a result of one of its upstreams being deferred.
// This graph must always describe the same effective dependences as the
// main execution graph in Terraform Core, so that questions about
// downstream deferrals will arrive in the correct order for their
// information to be available.
//
// The caller must not modify anything reachable through the given arguments
// after passing them to this function.
func NewDeferred(resourceDeps addrs.DirectedGraph[addrs.ConfigResource]) *Deferred {
	return &Deferred{
		resourceDeps:      resourceDeps,
		deferredResources: addrs.MakeSet[addrs.ConfigResource](),
	}
}

// ForceAllDeferred makes all future queries about whether an object should be
// deferred indicate that it _should_ be deferred.
//
// This should typically be called before beginning the Terraform Core graph
// walk to represent the special situation where we're planning a configuration
// that depends on some other configuration which itself had deferred actions,
// and therefore all of the actions we plan must be deferred.
func (d *Deferred) ForceAllDeferred() {
	d.mu.Lock()
	d.forceAllDeferred = true
	d.mu.Unlock()
}

// ReportResourceDeferred records that a particular resource has at least one
// deferred instance, or that its entire expansion was deferred.
//
// Terraform Core must call this for any resource that it generates deferred
// actions for, even if that deferral was forced by one of the "Should"
// functions on this same object, because our analyses typically consider only
// direct dependencies on the assumption that Terraform Core is going to visit
// everything in order anyway.
func (d *Deferred) ReportResourceDeferred(addr addrs.ConfigResource) {
	// NOTE: We're currently only tracking deferral for entire ConfigResource
	// addresses, which means that we don't track which module instances each
	// deferral belongs to and so this analysis will be very conservative when
	// considering deferrals inside multi-instance modules.
	//
	// We might improve the accuracy of this later, but we'll wait to see if
	// that's justified based on experience with real-world usage. Even this
	// imprecise analysis is presumably better than failing outright whenever
	// unknown values appear in an unfortunate place.

	d.mu.Lock()
	d.deferredResources.Add(addr)
	d.mu.Unlock()
}

func (d *Deferred) ShouldDeferResourceInstanceAction(addr addrs.AbsResourceInstance) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.forceAllDeferred {
		return true
	}

	// We use DirectDependenciesOf because we expect Terraform Core to visit
	// all of the resources in dependency order and to have already called
	// ReportResourceDeferred for any direct upstreams that were deferred for
	// any reason.
	deps := d.resourceDeps.DirectDependenciesOf(addr.ConfigResource())
	for _, depAddr := range deps {
		if d.deferredResources.Has(depAddr) {
			return true
		}
	}
	return false
}
