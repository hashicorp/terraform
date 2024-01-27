// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deferring

import (
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
)

// Deferred keeps track of deferrals that have already happened, to help
// guide decisions about whether downstream operations might also need to be
// deferred, and to provide some placeholder data for performing downstream
// checks against the subset of data we know despite the deferrals.
//
// This type only tracks information about object types that can _cause_
// deferred changes. Everything in the language can be _affected_ by deferred
// changes, such as by referring to an object whose changes were deferred or
// being declared in a module that was only partially-expanded, but we track
// the information about the other object types in other locations that are
// thematically closer to the type of object in question.
type Deferred struct {
	// resourceGraph is provided by the caller when instantiating a [Deferred],
	// and describes the dependency relationships between the static resource
	// declarations in the configuration.
	//
	// We use this as part of the rules for deciding whether a downstream
	// resource instance that could potentially be planned should be deferred
	// anyway due to its dependencies not yet being fully planned.
	resourceGraph addrs.DirectedGraph[addrs.ConfigResource]

	// externalDependencyDeferred marks the special situation where the
	// subsystem that's calling the modules runtime knows that some external
	// dependency of the configuration has deferred changes itself, and thus
	// all planned actions in this configuration must be deferred even if
	// the modules runtime can't find its own reason to do that.
	//
	// This is used by the stacks runtime when component B depends on
	// component A and component A's plan had deferred changes, so therefore
	// everything that component B might plan must also be deferred even
	// though the planning process for B cannot see into the plan for A.
	externalDependencyDeferred bool

	// Must hold this lock when accessing all fields after this one.
	mu sync.Mutex

	// resourceInstancesDeferred tracks the resource instances that have
	// been deferred despite their full addresses being known. This can happen
	// either because an upstream change was already deferred, or because
	// during planning the owning provider indicated that it doesn't yet have
	// enough information to produce a plan.
	//
	// These are grouped by the static resource configuration address because
	// there can potentially be various different deferrals for the same
	// configuration block at different amounts of instance expansion under
	// different prefixes, and so some queries require us to search across
	// all of those options to decide if each instance is relevant.
	resourceInstancesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, deferredResourceInstance]]

	// partialExpandedResourcesDeferred tracks placeholders that cover an
	// unbounded set of potential resource instances in situations where we
	// don't yet even have enough information to predict which instances of
	// a resource will exist.
	//
	// These are grouped by the static resource configuration address because
	// there can potentially be various different deferrals for the same
	// configuration block at different amounts of instance expansion under
	// different prefixes, and so some queries require us to search across
	// all of those options to find the one that matches most closely.
	partialExpandedResourcesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, deferredPartialExpandedResource]]

	// partialExpandedModulesDeferred tracks all of the partial-expanded module
	// prefixes we were notified about.
	//
	// We don't need to track anything for these other than that we saw them
	// reported, because the relevant data is tracked in [instances.Expander]
	// and [namedvals.State], but we do need to remember the addresses just
	// so that we can inform the caller that there was something deferred
	// even if there weren't any resources beneath the partial-expanded prefix.
	//
	// (If we didn't catch that then we'd mislead the caller into thinking
	// we fully-evaluated everything, which would be incorrect if any of the
	// root module output values are derived from the results of the
	// partial-expanded calls.)
	partialExpandedModulesDeferred addrs.Set[addrs.PartialExpandedModule]
}

// NewDeferred constructs a new [Deferred] that assumes that the given resource
// graph accurately describes all of the dependencies between static resource
// blocks in the configuration.
//
// Callers must not modify anything reachable through resourceGraph after
// calling this function.
func NewDeferred(resourceGraph addrs.DirectedGraph[addrs.ConfigResource]) *Deferred {
	return &Deferred{
		resourceGraph:                    resourceGraph,
		resourceInstancesDeferred:        addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, deferredResourceInstance]](),
		partialExpandedResourcesDeferred: addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, deferredPartialExpandedResource]](),
		partialExpandedModulesDeferred:   addrs.MakeSet[addrs.PartialExpandedModule](),
	}
}

// SetExternalDependencyDeferred modifies a freshly-constructed [Deferred]
// so that it will consider all resource instances as needing their actions
// deferred, even if there's no other reason to do that.
//
// This must be called zero or one times before any other use of the receiver.
// Changing this setting after a [Deferred] has already been used, or
// concurrently with any other method call, will cause inconsistent and
// undefined behavior.
func (d *Deferred) SetExternalDependencyDeferred() {
	d.externalDependencyDeferred = true
}

// HaveAnyDeferrals returns true if at least one deferral has been registered
// with the receiver.
//
// This method is intended as a summary result to propagate to the modules
// runtime caller so it can know if it should treat any downstream objects
// as having their own changes deferred without having to duplicate the
// modules runtime's rules for what counts as a deferral.
func (d *Deferred) HaveAnyDeferrals() bool {
	return d.externalDependencyDeferred ||
		d.resourceInstancesDeferred.Len() != 0 ||
		d.partialExpandedResourcesDeferred.Len() != 0 ||
		len(d.partialExpandedModulesDeferred) != 0
}

// ShouldDeferResourceChanges returns true if the receiver knows some reason
// why the resource instance with the given address should have its planned
// action deferred for a future plan/apply round.
//
// This method is specifically for resource instances whose full address is
// known and thus it would be possible in principle to plan changes, but we
// still need to respect dependency ordering and so any planned changes must
// be deferred if any upstream planned action was already deferred for
// some reason.
//
// Callers who get the answer true should announce an approximation of the
// action they would have planned to [Deferred.ReportResourceInstanceDeferred],
// but should skip writing that change into the live plan so that downstream
// evaluation will be based on the prior state (similar to in a refresh-only
// plan) rather than the result of the deferred action.
//
// It's invalid to call this method for an address that was already reported
// as deferred using [Deferred.ReportResourceInstanceDeferred], and so this
// method will panic in that case. Callers should always test whether a resource
// instance action should be deferred _before_ reporting that it has been.
func (d *Deferred) ShouldDeferResourceInstanceChanges(addr addrs.AbsResourceInstance) bool {
	if d.externalDependencyDeferred {
		// This is an easy case: _all_ actions must be deferred.
		return true
	}

	// If neither of our resource-deferral-tracking collections have anything
	// in them then we definitely don't need to defer. This special case is
	// here primarily to minimize the amount of code from here that will run
	// when the deferred-actions-related experiments are inactive, so we can
	// minimize the risk of impacting non-participants.
	// (Maybe we'll remove this check once this stuff is non-experimental.)
	if d.resourceInstancesDeferred.Len() == 0 && d.partialExpandedResourcesDeferred.Len() == 0 {
		return false
	}

	// Our resource graph describes relationships between the static resource
	// configuration blocks, not their dynamic instances, so we need to start
	// with the config address that the given instance belongs to.
	configAddr := addr.ConfigResource()

	if d.resourceInstancesDeferred.Get(configAddr).Has(addr) {
		// Asking for whether a resource instance should be deferred when
		// it was already reported as deferred suggests a programming error
		// in the caller, because the test for whether a change should be
		// deferred should always come before reporting that it has been.
		panic(fmt.Sprintf("checking whether %s should be deferred when it was already deferred", addr))
	}

	// We use DirectDependenciesOf rather than TransitiveDependenciesOf because
	// the reports to this object are driven by the modules runtime's existing
	// graph walk and so all of our direct dependencies should already have
	// had the opportunity to report themselves as deferred by the time this
	// question is being asked.
	configDeps := d.resourceGraph.DirectDependenciesOf(configAddr)

	// For this initial implementation we're taking the shortcut of assuming
	// that all of the configDeps are required. It would be better to do a
	// more precise analysis that takes into account how data could possibly
	// flow between instances of resources across different module paths,
	// but that may have some subtlety due to dynamic data flow, so we'll
	// need to do some more theory work to figure out what kind of analysis
	// we'd need to do to get this to be more precise.
	//
	// This conservative approach is a starting point so we can focus on
	// developing the workflow around deferred changes before making its
	// analyses more precise. This will defer more changes than strictly
	// necessary, but that's better than not deferring changes that should
	// have been deferred.
	//
	// (FWIW, it does seem like we _should_ be able to eliminate some
	// dynamic instances from consideration by relying on constraints such as
	// how a multi-instance module call can't have an object in one instance
	// depending on an object for another instance, but we'll need to make sure
	// any additional logic here is well-reasoned to avoid violating dependency
	// invariants.)
	for _, configDep := range configDeps {
		if d.resourceInstancesDeferred.Has(configDep) {
			// For now we don't consider exactly which instances of that
			// configuration block were deferred; there being at least
			// one is enough.
			return true
		}
		if d.partialExpandedResourcesDeferred.Has(configDep) {
			// For now we don't consider exactly which partial-expanded
			// prefixes of that configuration block were deferred; there being
			// at least one is enough.
			return true
		}

		// We don't check d.partialExpandedModulesDeferred here because
		// we expect that the graph nodes representing any resource under
		// a partial-expanded module prefix to call
		// d.ReportResourceExpansionDeferred once they find out that they
		// are under a partial-expanded prefix, and so
		// partialExpandedModulesDeferred is effectively just a less-detailed
		// summary of the information in partialExpandedResourcesDeferred.
		// (instances.Expander is the one responsible for letting the resource
		// node discover that it needs to do that; package deferred does
		// not participate directly in that concern.)
	}
	return false
}

// ReportResourceExpansionDeferred reports that we cannot even predict which
// instances of a resource will be declared and thus we must defer all planning
// for that resource.
//
// Use the most precise partial-expanded resource address possible, and provide
// a valuePlaceholder that has known values only for attributes/elements that
// we can guarantee will be equal across all potential resource instances
// under the partial-expanded prefix.
func (d *Deferred) ReportResourceExpansionDeferred(addr addrs.PartialExpandedResource, valuePlaceholder cty.Value) {
	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.ConfigResource()
	if !d.partialExpandedResourcesDeferred.Has(configAddr) {
		d.partialExpandedResourcesDeferred.Put(configAddr, addrs.MakeMap[addrs.PartialExpandedResource, deferredPartialExpandedResource]())
	}

	configMap := d.partialExpandedResourcesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each distinct partial-expanded
		// prefix only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, deferredPartialExpandedResource{
		valuePlaceholder: valuePlaceholder,
	})
}

// ReportResourceInstanceDeferred records that a fully-expanded resource
// instance has had its planned action deferred to a future round for a reason
// other than its address being only partially-decided.
//
// For example, this is the method to use if the reason for deferral is
// that [Deferred.ShouldDeferResourceInstanceChanges] returns true for the
// same address, or if the responsible provider indicated in its planning
// response that it does not have enough information to produce a final
// plan.
//
// expectedAction and expectedValue together provide an approximation of
// what Terraform is expecting to plan in a future round. expectedAction may
// be [plans.Undecided] if there isn't even enough information to decide on
// an action. expectedValue should use unknown values to stand in for values
// that cannot be predicted while being as precise as is practical; in the
// worst case it's okay to provide a totally-unknown value, but better to
// provide a known object with unknown values inside it when possible.
//
// TODO: Allow the caller to pass something representing the reason for the
// deferral, so we can distinguish between the different variations in the
// plan reported to the operator.
func (d *Deferred) ReportResourceInstanceDeferred(addr addrs.AbsResourceInstance, expectedAction plans.Action, expectedValue cty.Value) {
	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.ConfigResource()
	if !d.resourceInstancesDeferred.Has(configAddr) {
		d.resourceInstancesDeferred.Put(configAddr, addrs.MakeMap[addrs.AbsResourceInstance, deferredResourceInstance]())
	}

	configMap := d.resourceInstancesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each resource instance only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, deferredResourceInstance{
		plannedAction: expectedAction,
		plannedValue:  expectedValue,
	})
}

// ReportModuleExpansionDeferred reports that we cannot even predict which
// instances of a module call will be declared and thus we must defer all
// planning for everything inside that module.
//
// Use the most precise partial-expanded module address possible.
func (d *Deferred) ReportModuleExpansionDeferred(addr addrs.PartialExpandedModule) {
	if d.partialExpandedModulesDeferred.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each distinct partial-expanded
		// prefix only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	d.partialExpandedModulesDeferred.Add(addr)
}
