// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deferring

import (
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Deferred keeps track of deferrals that have already happened, to help
// guide decisions about whether downstream operations might also need to be
// deferred.
type Deferred struct {

	// deferralAllowed marks whether deferred actions are supported by the
	// current runtime. At time of writing, the modules runtime does not support
	// deferral, but the stacks runtime does.
	deferralAllowed bool

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

	// dataSourceInstancesDeferred tracks the data source instances that have
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
	dataSourceInstancesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]]

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
	resourceInstancesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]]

	// ephemeralResourceInstancesDeferred tracks the ephemeral resource instances
	// that have been deferred despite their full addresses being known. This can happen
	// either because an upstream change was already deferred, or because
	// during planning the owning provider indicated that it doesn't yet have
	// enough information to produce a plan.
	//
	// These are grouped by the static resource configuration address because
	// there can potentially be various different deferrals for the same
	// configuration block at different amounts of instance expansion under
	// different prefixes, and so some queries require us to search across
	// all of those options to decide if each instance is relevant.
	ephemeralResourceInstancesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]]

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
	partialExpandedResourcesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]]

	// partialExpandedDataSourcesDeferred tracks placeholders that cover an
	// unbounded set of potential data sources in situations where we don't yet
	// even have enough information to predict which instances of a data source
	// will exist.
	//
	// Data sources are never written into the plan, even when deferred, so we
	// are tracking these for purely internal reasons. If a resource depends on
	// a deferred data source, then that resource should be deferred as well.
	partialExpandedDataSourcesDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]]

	// partialExpandedEphemeralResourceDeferred tracks placeholders that cover an
	// unbounded set of potential data sources in situations where we don't yet
	// even have enough information to predict which instances of a data source
	// will exist.
	//
	// Data sources are never written into the plan, even when deferred, so we
	// are tracking these for purely internal reasons. If a resource depends on
	// a deferred data source, then that resource should be deferred as well.
	partialExpandedEphemeralResourceDeferred addrs.Map[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]]

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

// NewDeferred constructs a new empty [Deferred] object. The enabled argument
// controls whether the receiver will actually track any deferrals. If false,
// all methods will return false and no deferrals will be recorded.
func NewDeferred(enabled bool) *Deferred {
	return &Deferred{
		deferralAllowed:                          enabled,
		resourceInstancesDeferred:                addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]](),
		ephemeralResourceInstancesDeferred:       addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]](),
		dataSourceInstancesDeferred:              addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]](),
		partialExpandedResourcesDeferred:         addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]](),
		partialExpandedDataSourcesDeferred:       addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]](),
		partialExpandedEphemeralResourceDeferred: addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]](),
		partialExpandedModulesDeferred:           addrs.MakeSet[addrs.PartialExpandedModule](),
	}
}

// GetDeferredChanges returns a slice of all the deferred changes that have
// been reported to the receiver.
func (d *Deferred) GetDeferredChanges() []*plans.DeferredResourceInstanceChange {
	var changes []*plans.DeferredResourceInstanceChange

	if !d.deferralAllowed {
		return changes
	}

	for _, configMapElem := range d.resourceInstancesDeferred.Elems {
		for _, changeElem := range configMapElem.Value.Elems {
			changes = append(changes, changeElem.Value)
		}
	}
	for _, configMapElem := range d.dataSourceInstancesDeferred.Elems {
		for _, changeElem := range configMapElem.Value.Elems {
			changes = append(changes, changeElem.Value)
		}
	}
	for _, configMapElem := range d.partialExpandedResourcesDeferred.Elems {
		for _, changeElem := range configMapElem.Value.Elems {
			changes = append(changes, changeElem.Value)
		}
	}
	for _, configMapElem := range d.partialExpandedDataSourcesDeferred.Elems {
		for _, changeElem := range configMapElem.Value.Elems {
			changes = append(changes, changeElem.Value)
		}
	}
	return changes
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

// DeferralAllowed checks whether deferred actions are supported by the current
// runtime.
func (d *Deferred) DeferralAllowed() bool {
	// Gracefully recover from being called on nil, for tests that use
	// MockEvalContext without a real Deferred pointer set up.
	if d == nil {
		return false
	}
	return d.deferralAllowed
}

// HaveAnyDeferrals returns true if at least one deferral has been registered
// with the receiver.
//
// This method is intended as a summary result to propagate to the modules
// runtime caller so it can know if it should treat any downstream objects
// as having their own changes deferred without having to duplicate the
// modules runtime's rules for what counts as a deferral.
func (d *Deferred) HaveAnyDeferrals() bool {
	return d.deferralAllowed &&
		(d.externalDependencyDeferred ||
			d.resourceInstancesDeferred.Len() != 0 ||
			d.dataSourceInstancesDeferred.Len() != 0 ||
			d.ephemeralResourceInstancesDeferred.Len() != 0 ||
			d.partialExpandedResourcesDeferred.Len() != 0 ||
			d.partialExpandedDataSourcesDeferred.Len() != 0 ||
			d.partialExpandedEphemeralResourceDeferred.Len() != 0 ||
			len(d.partialExpandedModulesDeferred) != 0)
}

// GetDeferredResourceInstanceValue returns the deferred value for the given
// resource instance, if any.
func (d *Deferred) GetDeferredResourceInstanceValue(addr addrs.AbsResourceInstance) (cty.Value, bool) {
	if !d.deferralAllowed {
		return cty.NilVal, false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.ConfigResource()
	var instancesMap addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]]

	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		instancesMap = d.resourceInstancesDeferred
	case addrs.DataResourceMode:
		instancesMap = d.dataSourceInstancesDeferred
	case addrs.EphemeralResourceMode:
		instancesMap = d.ephemeralResourceInstancesDeferred
	default:
		panic(fmt.Sprintf("unexpected resource mode %q for %s", addr.Resource.Resource.Mode, addr))
	}

	change, ok := instancesMap.Get(configAddr).GetOk(addr)
	if !ok {
		return cty.NilVal, false
	}

	return change.Change.After, true
}

// GetDeferredResourceInstances returns a map of all the deferred instances of
// the given resource.
func (d *Deferred) GetDeferredResourceInstances(addr addrs.AbsResource) map[addrs.InstanceKey]cty.Value {
	if !d.deferralAllowed {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.Config()
	var instancesMap addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]]

	switch addr.Resource.Mode {
	case addrs.ManagedResourceMode:
		instancesMap = d.resourceInstancesDeferred
	case addrs.DataResourceMode:
		instancesMap = d.dataSourceInstancesDeferred
	case addrs.EphemeralResourceMode:
		instancesMap = d.ephemeralResourceInstancesDeferred

	default:
		panic(fmt.Sprintf("unexpected resource mode %q for %s", addr.Resource.Mode, addr))
	}

	instances, ok := instancesMap.GetOk(configAddr)
	if !ok {
		return nil
	}

	result := make(map[addrs.InstanceKey]cty.Value)
	for _, elem := range instances.Elems {
		instanceAddr := elem.Key
		change := elem.Value

		if addr.Resource.Mode == addrs.EphemeralResourceMode {
			// Deferred ephemeral resources always have an unknown value.
			result[instanceAddr.Resource.Key] = cty.UnknownVal(cty.DynamicPseudoType).Mark(marks.Ephemeral)
			continue
		}
		// instances contains all the resources identified by the config address
		// regardless of the instances of the module they might be in. We need
		// to filter out the instances that are not part of the module we are
		// interested in.
		if addr.Equal(instanceAddr.ContainingResource()) {
			result[instanceAddr.Resource.Key] = change.Change.After
		}
	}
	return result
}

// ShouldDeferResourceInstanceChanges returns true if the receiver knows some
// reason why the resource instance with the given address should have its
// planned action deferred for a future plan/apply round.
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
// method will panic in that case.
func (d *Deferred) ShouldDeferResourceInstanceChanges(addr addrs.AbsResourceInstance, deps []addrs.ConfigResource) bool {
	if !d.deferralAllowed {
		return false
	}
	configAddr := addr.ConfigResource()

	// Since d.DependenciesDeferred will also acquire the lock we don't use
	// the normal defer d.mu.Unlock() but handle it manually.
	d.mu.Lock()
	if d.resourceInstancesDeferred.Get(configAddr).Has(addr) || d.dataSourceInstancesDeferred.Get(configAddr).Has(addr) || d.ephemeralResourceInstancesDeferred.Get(configAddr).Has(addr) {
		d.mu.Unlock()
		// Asking for whether a resource instance should be deferred when
		// it was already reported as deferred suggests a programming error
		// in the caller, because the test for whether a change should be
		// deferred should always come before reporting that it has been.
		panic(fmt.Sprintf("checking whether %s should be deferred when it was already deferred", addr))
	}
	d.mu.Unlock()

	return d.DependenciesDeferred(deps)
}

// DependenciesDeferred returns true if any of the given configuration
// resources have had their planned actions deferred, either because they
// themselves were deferred or because they depend on something that was
// deferred.
//
// As
func (d *Deferred) DependenciesDeferred(deps []addrs.ConfigResource) bool {
	if !d.deferralAllowed {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.externalDependencyDeferred {
		return true
	}

	// If neither of our resource-deferral-tracking collections have anything
	// in them then we definitely don't need to defer. This special case is
	// here primarily to minimize the amount of code from here that will run
	// when the deferred-actions-related experiments are inactive, so we can
	// minimize the risk of impacting non-participants.
	// (Maybe we'll remove this check once this stuff is non-experimental.)
	if d.resourceInstancesDeferred.Len() == 0 &&
		d.dataSourceInstancesDeferred.Len() == 0 &&
		d.ephemeralResourceInstancesDeferred.Len() == 0 &&
		d.partialExpandedResourcesDeferred.Len() == 0 &&
		d.partialExpandedDataSourcesDeferred.Len() == 0 &&
		d.partialExpandedEphemeralResourceDeferred.Len() == 0 {
		return false
	}

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
	for _, configDep := range deps {
		if d.resourceInstancesDeferred.Has(configDep) || d.dataSourceInstancesDeferred.Has(configDep) || d.ephemeralResourceInstancesDeferred.Has(configDep) {
			// For now we don't consider exactly which instances of that
			// configuration block were deferred; there being at least
			// one is enough.
			return true
		}
		if d.partialExpandedResourcesDeferred.Has(configDep) {
			return true
		}
		if d.partialExpandedDataSourcesDeferred.Has(configDep) {
			return true
		}
		if d.partialExpandedEphemeralResourceDeferred.Has(configDep) {
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
func (d *Deferred) ReportResourceExpansionDeferred(addr addrs.PartialExpandedResource, change *plans.ResourceInstanceChange) {
	if change == nil {
		// This indicates a bug in Terraform, we shouldn't ever be setting a
		// null change. Note, if we don't make this check here, then we'll
		// just crash later anyway. This way the stack trace points to the
		// source of the problem.
		panic("change must not be nil")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if addr.Resource().Mode != addrs.ManagedResourceMode {
		// Use ReportDataSourceExpansionDeferred for data sources and ReportEphemeralResourceExpansionDeferred for ephemeral resources.
		panic(fmt.Sprintf("unexpected resource mode %q for %s", addr.Resource().Mode, addr))
	}

	configAddr := addr.ConfigResource()
	if !d.partialExpandedResourcesDeferred.Has(configAddr) {
		d.partialExpandedResourcesDeferred.Put(configAddr, addrs.MakeMap[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]())
	}

	configMap := d.partialExpandedResourcesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each distinct partial-expanded
		// prefix only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, &plans.DeferredResourceInstanceChange{
		DeferredReason: providers.DeferredReasonInstanceCountUnknown,
		Change:         change,
	})
}

// ReportDataSourceExpansionDeferred reports that we cannot even predict which
// instances of a data source will be declared and thus we must defer all
// planning for that data source.
func (d *Deferred) ReportDataSourceExpansionDeferred(addr addrs.PartialExpandedResource, change *plans.ResourceInstanceChange) {
	if change == nil {
		// This indicates a bug in Terraform, we shouldn't ever be setting a
		// null change. Note, if we don't make this check here, then we'll
		// just crash later anyway. This way the stack trace points to the
		// source of the problem.
		panic("change must not be nil")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if addr.Resource().Mode != addrs.DataResourceMode {
		// Use ReportResourceExpansionDeferred for resources and ReportEphemeralResourceExpansionDeferred for ephemeral resources.
		panic(fmt.Sprintf("unexpected resource mode %q for %s", addr.Resource().Mode, addr))
	}

	configAddr := addr.ConfigResource()
	if !d.partialExpandedDataSourcesDeferred.Has(configAddr) {
		d.partialExpandedDataSourcesDeferred.Put(configAddr, addrs.MakeMap[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]())
	}

	configMap := d.partialExpandedDataSourcesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each distinct partial-expanded
		// prefix only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, &plans.DeferredResourceInstanceChange{
		DeferredReason: providers.DeferredReasonInstanceCountUnknown,
		Change:         change,
	})
}

func (d *Deferred) ReportEphemeralResourceExpansionDeferred(addr addrs.PartialExpandedResource) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if addr.Resource().Mode != addrs.EphemeralResourceMode {
		// Use ReportResourceExpansionDeferred for resources and ReportDataSourceExpansionDeferred for data sources.
		panic(fmt.Sprintf("unexpected resource mode %q for %s", addr.Resource().Mode, addr))
	}

	configAddr := addr.ConfigResource()
	if !d.partialExpandedEphemeralResourceDeferred.Has(configAddr) {
		d.partialExpandedEphemeralResourceDeferred.Put(configAddr, addrs.MakeMap[addrs.PartialExpandedResource, *plans.DeferredResourceInstanceChange]())
	}

	configMap := d.partialExpandedEphemeralResourceDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each distinct partial-expanded
		// prefix only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, &plans.DeferredResourceInstanceChange{
		DeferredReason: providers.DeferredReasonInstanceCountUnknown,
		Change:         nil, // since we don't serialize this we can get away with no change, we store the addr, that should be enough
	})
}

// ReportResourceInstanceDeferred records that a fully-expanded resource
// instance has had its planned action deferred to a future round for a reason
// other than its address being only partially-decided.
func (d *Deferred) ReportResourceInstanceDeferred(addr addrs.AbsResourceInstance, reason providers.DeferredReason, change *plans.ResourceInstanceChange) {
	if change == nil {
		// This indicates a bug in Terraform, we shouldn't ever be setting a
		// null change. Note, if we don't make this check here, then we'll
		// just crash later anyway. This way the stack trace points to the
		// source of the problem.
		panic("change must not be nil")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.ConfigResource()
	if !d.resourceInstancesDeferred.Has(configAddr) {
		d.resourceInstancesDeferred.Put(configAddr, addrs.MakeMap[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]())
	}

	configMap := d.resourceInstancesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each resource instance only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, &plans.DeferredResourceInstanceChange{
		DeferredReason: reason,
		Change:         change,
	})
}

func (d *Deferred) ReportDataSourceInstanceDeferred(addr addrs.AbsResourceInstance, reason providers.DeferredReason, change *plans.ResourceInstanceChange) {
	if change == nil {
		// This indicates a bug in Terraform, we shouldn't ever be setting a
		// null change. Note, if we don't make this check here, then we'll
		// just crash later anyway. This way the stack trace points to the
		// source of the problem.
		panic("change must not be nil")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.ConfigResource()
	if !d.dataSourceInstancesDeferred.Has(configAddr) {
		d.dataSourceInstancesDeferred.Put(configAddr, addrs.MakeMap[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]())
	}

	configMap := d.dataSourceInstancesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each resource instance only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, &plans.DeferredResourceInstanceChange{
		DeferredReason: reason,
		Change:         change,
	})
}

func (d *Deferred) ReportEphemeralResourceInstanceDeferred(addr addrs.AbsResourceInstance, reason providers.DeferredReason) {
	d.mu.Lock()
	defer d.mu.Unlock()

	configAddr := addr.ConfigResource()
	if !d.ephemeralResourceInstancesDeferred.Has(configAddr) {
		d.ephemeralResourceInstancesDeferred.Put(configAddr, addrs.MakeMap[addrs.AbsResourceInstance, *plans.DeferredResourceInstanceChange]())
	}

	configMap := d.ephemeralResourceInstancesDeferred.Get(configAddr)
	if configMap.Has(addr) {
		// This indicates a bug in the caller, since our graph walk should
		// ensure that we visit and evaluate each resource instance only once.
		panic(fmt.Sprintf("duplicate deferral report for %s", addr))
	}
	configMap.Put(addr, &plans.DeferredResourceInstanceChange{
		DeferredReason: reason,
		Change:         nil, // Since we don't serialize this we can get away with not storing a change
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

// UnexpectedProviderDeferralDiagnostic is a diagnostic that indicates that a
// provider was deferred although deferrals were not allowed.
func UnexpectedProviderDeferralDiagnostic(addrs addrs.AbsResourceInstance) tfdiags.Diagnostic {
	return tfdiags.Sourceless(tfdiags.Error, "Provider deferred changes when Terraform did not allow deferrals", fmt.Sprintf("The provider signaled a deferred action for %q, but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.", addrs.String()))
}
