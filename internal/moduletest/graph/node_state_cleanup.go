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
	// each state cleanup node is associated with a specific run,
	// but only the first execution of the run is used to
	// destroy the state.
	run  *moduletest.Run
	opts *graphOptions

	// If applyOverride is provided, it will be applied to the state file to reach
	// the final state, instead of running the destroy operation.
	applyOverride *moduletest.Run

	// deps is a map of dependent states that need to be stored if this
	// state cleanup fails.
	deps map[string]*NodeStateCleanup
}

func (n *NodeStateCleanup) Name() string {
	return fmt.Sprintf("cleanup.%s", n.run.Name)
}

// Execute performs the cleanup of resources defined in the state file.
// This function should never return non-fatal error diagnostics, as doing so
// would prevent further cleanup of other states. Instead, any diagnostics
// will be rendered directly to ensure the cleanup process continues.
func (n *NodeStateCleanup) Execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	n.execute(evalCtx)
	return nil
}

func (n *NodeStateCleanup) execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	file := n.opts.File
	stateKey := n.run.Config.StateKey
	state := evalCtx.GetFileState(stateKey)
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)
	if n.applyOverride != nil {
		state.Run = n.applyOverride
	}

	if n.shouldSkipCleanup(evalCtx, state, file) {
		return nil
	}

	if state.Run == nil {
		log.Printf("[ERROR] TestFileRunner: found inconsistent run block and state file in %s for module %s", file.Name, stateKey)

		// The state can have a nil run block if it only executed a plan
		// command. In which case, we shouldn't have reached here as the
		// state should also have been empty and this will have been skipped
		// above. If we do reach here, then something has gone badly wrong
		// and we can't really recover from it.

		diags := tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "Inconsistent state", fmt.Sprintf("Found inconsistent state while cleaning up %s. This is a bug in Terraform - please report it", file.Name))}
		file.UpdateStatus(moduletest.Error)
		state.Reason = ReasonError
		diags = diags.Append(evalCtx.WriteFileState(stateKey, state))
		evalCtx.Renderer().DestroySummary(diags, nil, file, state.State)

		return diags
	}
	TransformConfigForRun(evalCtx, state.Run, file)

	// store the state in the file before running the cleanup
	evalCtx.snapStates[stateKey] = &TestFileState{
		File:   file,
		Run:    state.Run,
		State:  state.State.DeepCopy(),
		Reason: state.Reason,
	}

	n.performCleanup(evalCtx, state)
	return nil
}

func (n *NodeStateCleanup) performCleanup(evalCtx *EvalContext, state *TestFileState) tfdiags.Diagnostics {
	runNode := &NodeTestRun{run: state.Run, opts: n.opts}
	updated := state.State
	stateKey := n.run.Config.StateKey
	startTime := time.Now().UTC()
	waiter := NewOperationWaiter(nil, evalCtx, runNode, moduletest.Running, startTime.UnixMilli())
	var destroyDiags tfdiags.Diagnostics
	cancelled := waiter.Run(func() {
		updated, destroyDiags = n.cleanup(evalCtx, runNode, waiter)
	})
	if cancelled {
		destroyDiags = destroyDiags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	evalCtx.Renderer().DestroySummary(destroyDiags, state.Run, n.opts.File, updated)
	n.updateStateResourcesOnly(evalCtx, runNode, updated)

	// Update the file status and write the updated state back to the file.
	status := moduletest.Pass
	if n.applyOverride != nil { // skip_cleanup=true
		status = runNode.run.Status
		state.Reason = ReasonSkip
	} else if !n.emptyState(updated) {
		// Then we ran a destroy operation, but failed to adequately clean up the state, so mark as errored.
		status = moduletest.Error
		state.Reason = ReasonError
	}
	n.opts.File.UpdateStatus(status)
	if state.Run.Config.Backend != nil && !destroyDiags.HasErrors() {
		// We don't create a state artefact when the node state's corresponding run block has a backend,
		// UNLESS an error occurs when returning the state to match that run block's config during cleanup.
		return nil
	}

	evalCtx.WriteFileState(stateKey, state)

	// failed state cleanup. We need to store the dependent states too
	if !n.emptyState(updated) {
		for _, dep := range n.deps {
			depState, ok := evalCtx.snapStates[dep.run.Config.StateKey]
			if !ok {
				continue
			}
			depState.Reason = ReasonDep
			evalCtx.WriteFileState(dep.run.Config.StateKey, depState)
		}
	}

	// We don't return destroyDiags here because the calling code sets the return code for the test operation
	// based on whether the tests passed or not; cleanup is not a factor.
	// Users will be aware of issues with cleanup due to destroyDiags being rendered to the View.
	return nil
}

func (n *NodeStateCleanup) updateStateResourcesOnly(ctx *EvalContext, runNode *NodeTestRun, state *states.State) {
	fileState := ctx.GetFileState(runNode.run.Config.StateKey)
	if fileState == nil {
		return
	}

	outputs := fileState.State.RootOutputValues
	fileState.State = state
	fileState.State.RootOutputValues = outputs
}

func (n *NodeStateCleanup) cleanup(ctx *EvalContext, runNode *NodeTestRun, waiter *operationWaiter) (*states.State, tfdiags.Diagnostics) {
	file := n.opts.File
	stateKey := runNode.run.Config.StateKey
	fileState := ctx.GetFileState(stateKey)
	state := fileState.State
	run := runNode.run
	log.Printf("[TRACE] TestFileRunner: called destroy for %s/%s", file.Name, run.Name)

	ctx.Renderer().Run(run, file, moduletest.TearDown, 0)

	var diags tfdiags.Diagnostics
	variables, variableDiags := runNode.GetVariables(ctx, false)
	diags = diags.Append(variableDiags)
	if diags.HasErrors() {
		return state, diags
	}

	// If an apply override is provided and the test is not in cleanup mode,
	// we can directly apply the override to the state file instead of performing
	// a destroy operation.
	if n.applyOverride != nil && n.opts.CommandMode != moduletest.CleanupMode {
		runNode.testApply(ctx, variables, waiter)
		return ctx.GetFileState(stateKey).State, runNode.run.Diagnostics
	}

	// During the destroy operation, we don't add warnings from this operation.
	// Anything that would have been reported here was already reported during
	// the original plan, and a successful destroy operation is the only thing
	// we care about.
	setVariables, _, _ := runNode.FilterVariablesToModule(variables)

	planOpts := &terraform.PlanOpts{
		Mode:         plans.DestroyMode,
		SetVariables: setVariables,
		Overrides:    mocking.PackageOverrides(run.Config, file.Config, run.ModuleConfig),
	}

	tfCtx, ctxDiags := terraform.NewContext(n.opts.ContextOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return state, diags
	}

	waiter.update(tfCtx, moduletest.TearDown, nil)
	plan, planDiags := tfCtx.Plan(run.ModuleConfig, state, planOpts)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return state, diags
	}

	_, updated, applyDiags := runNode.apply(tfCtx, plan, moduletest.TearDown, variables, waiter)
	diags = diags.Append(applyDiags)
	return updated, diags
}

func (n *NodeStateCleanup) shouldSkipCleanup(evalCtx *EvalContext, state *TestFileState, file *moduletest.File) bool {
	stateKey := n.run.Config.StateKey
	if _, ok := evalCtx.snapStates[stateKey]; ok {
		// This state's cleanup has already been processed, so we can skip it.
		log.Printf("[TRACE] TestStateManager: skipping state cleanup for %s", stateKey)
		return true
	}

	// If the state was loaded as a result of an intentional skip, we
	// don't need to clean it up when in repair mode.
	if evalCtx.repair && state.Reason == ReasonSkip {
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to repair mode", file.Name)
		return true
	}

	if state.Reason == ReasonDep {
		// If the state was loaded as a result of a dependency, we don't
		// need to clean it up.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to dependency", file.Name)
		return true
	}

	if evalCtx.Cancelled() {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return true
	}

	return n.emptyState(state.State)
}

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
