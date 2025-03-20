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

	// If applyOverride is provided, it will be applied to the state file to reach
	// the final state, instead of running the destroy operation.
	applyOverride *moduletest.Run
}

func (n *NodeStateCleanup) Name() string {
	return fmt.Sprintf("cleanup.%s", n.stateKey)
}

// Execute performs the cleanup of resources defined in the state file.
// This function should never return non-fatal error diagnostics, as doing so
// would prevent further cleanup of other states. Instead, any diagnostics
// will be rendered directly to ensure the cleanup process continues.
func (n *NodeStateCleanup) Execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	file := n.opts.File
	state := evalCtx.GetFileState(n.stateKey)
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)
	if n.applyOverride != nil {
		state.Run = n.applyOverride
	}

	if evalCtx.Cancelled() {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return nil
	}

	empty := true
	if !state.State.Empty() {
		for _, module := range state.State.Modules {
			for _, resource := range module.Resources {
				if resource.Addr.Resource.Mode == addrs.ManagedResourceMode {
					empty = false
					break
				}
			}
		}
	}

	if empty {
		// The state can be empty for a run block that just executed a plan
		// command, or a run block that only read data sources. We'll just
		// skip empty run blocks.
		return nil
	}

	if state.Run == nil {
		log.Printf("[ERROR] TestFileRunner: found inconsistent run block and state file in %s for module %s", file.Name, n.stateKey)

		// The state can have a nil run block if it only executed a plan
		// command. In which case, we shouldn't have reached here as the
		// state should also have been empty and this will have been skipped
		// above. If we do reach here, then something has gone badly wrong
		// and we can't really recover from it.

		diags := tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "Inconsistent state", fmt.Sprintf("Found inconsistent state while cleaning up %s. This is a bug in Terraform - please report it", file.Name))}
		file.UpdateStatus(moduletest.Error)
		diags = diags.Append(evalCtx.WriteFileState(n.stateKey, state))
		evalCtx.Renderer().DestroySummary(diags, nil, file, state.State)

		// intentionally return nil to allow further cleanup
		return nil
	}
	TransformConfigForRun(evalCtx, state.Run, file)

	runNode := &NodeTestRun{run: state.Run, opts: n.opts}
	updated := state.State
	startTime := time.Now().UTC()
	waiter := NewOperationWaiter(nil, evalCtx, runNode, moduletest.Running, startTime.UnixMilli())
	var destroyDiags tfdiags.Diagnostics
	cancelled := waiter.Run(func() {
		updated, destroyDiags = n.cleanup(evalCtx, runNode, waiter)
	})
	if cancelled {
		destroyDiags = destroyDiags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	evalCtx.Renderer().DestroySummary(destroyDiags, state.Run, file, updated)
	state.State = updated

	evalCtx.WriteFileState(n.stateKey, state)
	return nil
}

func (n *NodeStateCleanup) cleanup(ctx *EvalContext, runNode *NodeTestRun, waiter *operationWaiter) (*states.State, tfdiags.Diagnostics) {
	file := n.opts.File
	fileState := ctx.GetFileState(n.stateKey)
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

	// If the run block has an override, we don't need to run the destroy
	// operation. We can just apply the override to the state file and return.
	if n.applyOverride != nil {
		runNode.testApply(ctx, variables, waiter)
		return ctx.GetFileState(n.stateKey).State, nil
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

	if !updated.Empty() {
		// Then we failed to adequately clean up the state, so mark as errored.
		file.UpdateStatus(moduletest.Error)
	}

	return updated, diags
}
