// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// graphWalkOpts captures some transient values we use (and possibly mutate)
// during a graph walk.
//
// The way these options get used unfortunately varies between the different
// walkOperation types. This is a historical design wart that dates back to
// us using the same graph structure for all operations; hopefully we'll
// make the necessary differences between the walk types more explicit someday.
type graphWalkOpts struct {
	InputState *states.State
	Changes    *plans.Changes
	Config     *configs.Config

	// ExternalProviderConfigs is used for walks that make use of configured
	// providers (e.g. plan and apply) to satisfy situations where the root
	// module itself declares that it expects to have providers passed in
	// from outside.
	//
	// This should not be populated for walks that use only unconfigured
	// providers, such as validate. Populating it for those walks might cause
	// strange things to happen, because our graph walking machinery doesn't
	// always take into account what walk type it's dealing with.
	ExternalProviderConfigs map[addrs.RootProviderConfig]providers.Interface

	// ExternalDependencyDeferred indicates that something that this entire
	// configuration depends on (outside the view of this modules runtime)
	// has deferred changes, and therefore we must treat _all_ actions
	// as deferred to produce the correct overall dependency ordering.
	ExternalDependencyDeferred bool

	// PlanTimeCheckResults should be populated during the apply phase with
	// the snapshot of check results that was generated during the plan step.
	//
	// This then propagates the decisions about which checkable objects exist
	// from the plan phase into the apply phase without having to re-compute
	// the module and resource expansion.
	PlanTimeCheckResults *states.CheckResults

	// PlanTimeTimestamp should be populated during the plan phase by retrieving
	// the current UTC timestamp, and should be read from the plan file during
	// the apply phase.
	PlanTimeTimestamp time.Time

	// Overrides contains the set of overrides we should apply during this
	// operation.
	Overrides *mocking.Overrides

	MoveResults refactoring.MoveResults

	ProviderFuncResults *providers.FunctionResults
}

func (c *Context) walk(graph *Graph, operation walkOperation, opts *graphWalkOpts) (*ContextGraphWalker, tfdiags.Diagnostics) {
	log.Printf("[DEBUG] Starting graph walk: %s", operation.String())

	walker := c.graphWalker(graph, operation, opts)

	// Watch for a stop so we can call the provider Stop() API.
	watchStop, watchWait := c.watchStop(walker)

	// Walk the real graph, this will block until it completes
	diags := graph.Walk(walker)

	// Close the channel so the watcher stops, and wait for it to return.
	close(watchStop)
	<-watchWait

	return walker, diags
}

func (c *Context) graphWalker(graph *Graph, operation walkOperation, opts *graphWalkOpts) *ContextGraphWalker {
	var state *states.SyncState
	var refreshState *states.SyncState
	var prevRunState *states.SyncState

	// NOTE: None of the SyncState objects must directly wrap opts.InputState,
	// because we use those to mutate the state object and opts.InputState
	// belongs to our caller and thus we must treat it as immutable.
	//
	// To account for that, most of our SyncState values created below end up
	// wrapping a _deep copy_ of opts.InputState instead.
	inputState := opts.InputState
	if inputState == nil {
		// Lots of callers use nil to represent the "empty" case where we've
		// not run Apply yet, so we tolerate that.
		inputState = states.NewState()
	}

	switch operation {
	case walkValidate:
		// validate should not use any state
		state = states.NewState().SyncWrapper()

		// validate currently uses the plan graph, so we have to populate the
		// refreshState and the prevRunState.
		refreshState = states.NewState().SyncWrapper()
		prevRunState = states.NewState().SyncWrapper()

	case walkPlan, walkPlanDestroy, walkImport:
		state = inputState.DeepCopy().SyncWrapper()
		refreshState = inputState.DeepCopy().SyncWrapper()
		prevRunState = inputState.DeepCopy().SyncWrapper()

		// For both of our new states we'll discard the previous run's
		// check results, since we can still refer to them from the
		// prevRunState object if we need to.
		state.DiscardCheckResults()
		refreshState.DiscardCheckResults()

	default:
		state = inputState.DeepCopy().SyncWrapper()
		// Only plan-like walks use refreshState and prevRunState

		// Discard the input state's check results, because we should create
		// a new set as a result of the graph walk.
		state.DiscardCheckResults()
	}

	changes := opts.Changes
	if changes == nil {
		// Several of our non-plan walks end up sharing codepaths with the
		// plan walk and thus expect to generate planned changes even though
		// we don't care about them. To avoid those crashing, we'll just
		// insert a placeholder changes object which'll get discarded
		// afterwards.
		changes = plans.NewChanges()
	}

	if opts.Config == nil {
		panic("Context.graphWalker call without Config")
	}

	checkState := checks.NewState(opts.Config)
	if opts.PlanTimeCheckResults != nil {
		// We'll re-report all of the same objects we determined during the
		// plan phase so that we can repeat the checks during the apply
		// phase to finalize them.
		for _, configElem := range opts.PlanTimeCheckResults.ConfigResults.Elems {
			if configElem.Value.ObjectAddrsKnown() {
				configAddr := configElem.Key
				checkState.ReportCheckableObjects(configAddr, configElem.Value.ObjectResults.Keys())
			}
		}
	}

	// We'll produce a derived graph that only includes the static resource
	// blocks, since we need that for deferral tracking.
	resourceGraph := graph.ResourceGraph()
	deferred := deferring.NewDeferred(resourceGraph)
	if opts.ExternalDependencyDeferred {
		deferred.SetExternalDependencyDeferred()
	}

	return &ContextGraphWalker{
		Context:                 c,
		State:                   state,
		Config:                  opts.Config,
		RefreshState:            refreshState,
		Overrides:               opts.Overrides,
		PrevRunState:            prevRunState,
		Changes:                 changes.SyncWrapper(),
		NamedValues:             namedvals.NewState(),
		Deferrals:               deferred,
		Checks:                  checkState,
		InstanceExpander:        instances.NewExpander(),
		ExternalProviderConfigs: opts.ExternalProviderConfigs,
		MoveResults:             opts.MoveResults,
		Operation:               operation,
		StopContext:             c.runContext,
		PlanTimestamp:           opts.PlanTimeTimestamp,
		providerFuncResults:     opts.ProviderFuncResults,
	}
}
