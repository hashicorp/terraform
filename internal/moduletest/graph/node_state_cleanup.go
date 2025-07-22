// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeExecutable = (*NodeStateCleanup)(nil)
)

// NodeStateCleanup is responsible for cleaning up the state of resources
// defined in the state file. It uses the provided stateKey to identify the
// specific state to clean up and opts for additional configuration options.
type NodeStateCleanup struct {
	stateKey string
	opts     *graphOptions
	parallel bool

	// If customCleanupRun is provided, it will be applied to the state file instead
	// of running the default destroy operation during cleanup.
	customCleanupRun *moduletest.Run
}

func (n *NodeStateCleanup) Name() string {
	return fmt.Sprintf("cleanup.%s", n.stateKey)
}

// Execute destroys the resources created in the state file.
// This function should never return non-fatal error diagnostics, as that would
// prevent further cleanup from happening. Instead, the diagnostics
// will be rendered directly.
func (n *NodeStateCleanup) Execute(evalCtx *EvalContext) {
	var diags tfdiags.Diagnostics
	file := n.opts.File
	state := evalCtx.GetFileState(n.stateKey)
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)
	if n.customCleanupRun != nil {
		state.Run = n.customCleanupRun
	}

	if n.shouldSkipCleanup(evalCtx, state, file) {
		return
	}

	// If the state is empty, we still write it so we can store the
	// output values, but we don't need to run a destroy operation.
	if n.emptyState(state.State) {
		// TODO(liamcervante): No diagnostics here!
		diags = diags.Append(evalCtx.WriteFileState(n.stateKey, state))
		evalCtx.Renderer().DestroySummary(diags, state.Run, file, state.State)
		return
	}

	if state.Run == nil {
		log.Printf("[ERROR] TestFileRunner: found inconsistent run block and state file in %s for module %s", file.Name, n.stateKey)

		// The state can have a nil run block if it only executed a plan
		// command. In which case, we shouldn't have reached here as the
		// state should also have been empty and this will have been skipped
		// above. If we do reach here, then something has gone badly wrong
		// and we can't really recover from it.

		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Inconsistent state", fmt.Sprintf("Found inconsistent state while cleaning up %s. This is a bug in Terraform - please report it", file.Name)))
		file.UpdateStatus(moduletest.Error)
		state.Reason = ReasonError
		diags = diags.Append(evalCtx.WriteFileState(n.stateKey, state))
		evalCtx.Renderer().DestroySummary(diags, nil, file, state.State)
		return
	}

	diags = diags.Append(n.performCleanup(evalCtx, state))
	evalCtx.Renderer().DestroySummary(diags, state.Run, file, state.State)
}

func (n *NodeStateCleanup) performCleanup(evalCtx *EvalContext, state *TestFileState) tfdiags.Diagnostics {
	runNode := &NodeTestRun{run: state.Run, opts: n.opts}
	updated := state.State
	startTime := time.Now().UTC()
	waiter := NewOperationWaiter(nil, evalCtx, runNode, moduletest.Running, startTime.UnixMilli())
	var diags tfdiags.Diagnostics
	cancelled := waiter.Run(func() {
		updated, diags = n.cleanup(evalCtx, runNode, waiter)
	})
	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	// Update the statefile's resources and return the test file status
	n.updateStateResources(runNode, state, updated)

	if state.Run.Config.Backend != nil && !diags.HasErrors() {
		// We don't create a state artefact when the node state's corresponding run block has a backend,
		// UNLESS an error occurs when returning the state to match that run block's config during cleanup.
		return diags
	}

	diags = diags.Append(evalCtx.WriteFileState(n.stateKey, state))
	return diags
}

func (n *NodeStateCleanup) updateStateResources(runNode *NodeTestRun, fileState *TestFileState, updated *states.State) {
	// After destruction, we still want to preserve the output values
	// from the original state file. This is important because other runs
	// may reference outputs from this state file.
	outputs := fileState.State.RootOutputValues

	// Update the state file with the new state.
	fileState.State = updated
	fileState.State.RootOutputValues = outputs

	// Update the test file status
	status := moduletest.Pass
	switch {
	case n.customCleanupRun != nil: // skip_cleanup=true
		status = runNode.run.Status
		fileState.Reason = ReasonSkip
	case !n.emptyState(updated):
		// Then we ran a destroy operation, but failed to adequately clean up the state, so mark as errored.
		status = moduletest.Error
		fileState.Reason = ReasonError
	default:
		// Keep the default status of Pass
	}
	n.opts.File.UpdateStatus(status)
}

func (n *NodeStateCleanup) cleanup(ctx *EvalContext, runNode *NodeTestRun, waiter *operationWaiter) (*states.State, tfdiags.Diagnostics) {
	file := n.opts.File
	stateKey := runNode.run.Config.StateKey
	fileState := ctx.GetFileState(stateKey)
	state := fileState.State
	run := runNode.run
	log.Printf("[TRACE] TestFileRunner: called destroy for %s/%s", file.Name, run.Name)

	ctx.Renderer().Run(run, file, moduletest.TearDown, 0)

	variables, diags := runNode.GetVariables(ctx, false)
	if diags.HasErrors() {
		return state, diags
	}

	// we ignore the diagnostics from here, because we will have reported them
	// during the initial execution of the run block and we would not have
	// executed the run block if there were any errors.
	providers, mocks, _ := runNode.getProviders(ctx)

	// If an apply override is provided, we can skip the destroy operation
	// and directly apply the override to the state file.
	if n.customCleanupRun != nil {
		runNode.testApply(ctx, variables, providers, mocks, waiter)
		return ctx.GetFileState(stateKey).State, runNode.run.Diagnostics
	}

	// During the destroy operation, we don't add warnings from this operation.
	// Anything that would have been reported here was already reported during
	// the original plan, and a successful destroy operation is the only thing
	// we care about.
	setVariables, _, _ := runNode.FilterVariablesToModule(variables)

	planOpts := &terraform.PlanOpts{
		Mode:              plans.DestroyMode,
		SetVariables:      setVariables,
		Overrides:         mocking.PackageOverrides(run.Config, file.Config, mocks),
		ExternalProviders: providers,
	}

	tfCtx, _ := terraform.NewContext(n.opts.ContextOpts)

	waiter.update(tfCtx, moduletest.TearDown, nil)
	plan, planDiags := tfCtx.Plan(run.ModuleConfig, state, planOpts)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return state, diags
	}

	_, updated, applyDiags := runNode.apply(tfCtx, plan, moduletest.TearDown, variables, providers, waiter)
	diags = diags.Append(applyDiags)
	return updated, diags
}

func (n *NodeStateCleanup) shouldSkipCleanup(evalCtx *EvalContext, state *TestFileState, file *moduletest.File) bool {
	// If the state was loaded as a result of an intentional skip, we
	// don't need to clean it up when in repair mode.
	if evalCtx.repair && state.Reason == ReasonSkip {
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for state %q due to repair mode", n.stateKey)
		return true
	}

	if state.Reason == ReasonDep {
		// If the state was loaded as a result of a dependency, we don't
		// need to clean it up.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for state %q due to dependency", n.stateKey)
		return true
	}

	if evalCtx.Cancelled() {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for state %q due to cancellation", n.stateKey)
		return true
	}

	return false
}

// emptyState checks if the state is empty, meaning it doesn't contain any
// managed resources.
func (n *NodeStateCleanup) emptyState(state *states.State) bool {
	empty := true
	if !state.Empty() {
		for _, module := range state.Modules {
			for _, resource := range module.Resources {
				if resource.Addr.Resource.Mode == addrs.ManagedResourceMode {
					empty = false
					break
				}
			}
		}
	}
	return empty
}
