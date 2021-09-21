package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
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

	RootVariableValues InputValues
	MoveResults        refactoring.MoveResults
}

func (c *Context) walk(graph *Graph, operation walkOperation, opts *graphWalkOpts) (*ContextGraphWalker, tfdiags.Diagnostics) {
	log.Printf("[DEBUG] Starting graph walk: %s", operation.String())

	walker := c.graphWalker(operation, opts)

	// Watch for a stop so we can call the provider Stop() API.
	watchStop, watchWait := c.watchStop(walker)

	// Walk the real graph, this will block until it completes
	diags := graph.Walk(walker)

	// Close the channel so the watcher stops, and wait for it to return.
	close(watchStop)
	<-watchWait

	return walker, diags
}

func (c *Context) graphWalker(operation walkOperation, opts *graphWalkOpts) *ContextGraphWalker {
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

	case walkPlan, walkPlanDestroy:
		state = inputState.DeepCopy().SyncWrapper()
		refreshState = inputState.DeepCopy().SyncWrapper()
		prevRunState = inputState.DeepCopy().SyncWrapper()

	default:
		state = inputState.DeepCopy().SyncWrapper()
		// Only plan-like walks use refreshState and prevRunState
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

	return &ContextGraphWalker{
		Context:            c,
		State:              state,
		Config:             opts.Config,
		RefreshState:       refreshState,
		PrevRunState:       prevRunState,
		Changes:            changes.SyncWrapper(),
		InstanceExpander:   instances.NewExpander(),
		MoveResults:        opts.MoveResults,
		Operation:          operation,
		StopContext:        c.runContext,
		RootVariableValues: opts.RootVariableValues,
	}
}
