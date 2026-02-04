// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	teststates "github.com/hashicorp/terraform/internal/moduletest/states"
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
}

func (n *NodeStateCleanup) Name() string {
	return fmt.Sprintf("cleanup.%s", n.stateKey)
}

// Execute destroys the resources created in the state file.
func (n *NodeStateCleanup) Execute(evalCtx *EvalContext) {
	file := n.opts.File
	state := evalCtx.GetState(n.stateKey)
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)

	if evalCtx.Cancelled() {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return
	}

	if emptyState(state.State) {
		// The state can be empty for a run block that just executed a plan
		// command, or a run block that only read data sources. We'll just
		// skip empty run blocks. We do reset the state reason to None for this
		// just to make sure we are indicating externally this state file
		// doesn't need to be saved.
		evalCtx.SetFileState(n.stateKey, state.Run, state.State, teststates.StateReasonNone)
		return
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
		evalCtx.Renderer().DestroySummary(diags, nil, file, state.State)
		return
	}

	updated := state.State
	startTime := time.Now().UTC()
	waiter := NewOperationWaiter(nil, evalCtx, file, state.Run, moduletest.Running, startTime.UnixMilli())
	var destroyDiags tfdiags.Diagnostics
	evalCtx.Renderer().Run(state.Run, file, moduletest.TearDown, 0)
	cancelled := waiter.Run(func() {
		if state.RestoreState {
			updated, destroyDiags = n.restore(evalCtx, file.Config, state.Run.Config, state.Run.ModuleConfig, updated, waiter)
		} else {
			updated, destroyDiags = n.destroy(evalCtx, file.Config, state.Run.Config, state.Run.ModuleConfig, updated, waiter)
			updated.RootOutputValues = state.State.RootOutputValues // we're going to preserve the output values in case we need to tidy up
		}
	})
	if cancelled {
		destroyDiags = destroyDiags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	switch {
	case destroyDiags.HasErrors():
		file.UpdateStatus(moduletest.Error)
		evalCtx.SetFileState(n.stateKey, state.Run, updated, teststates.StateReasonError)
	case state.Run.Config.Backend != nil:
		evalCtx.SetFileState(n.stateKey, state.Run, updated, teststates.StateReasonNone)
	case state.RestoreState:
		evalCtx.SetFileState(n.stateKey, state.Run, updated, teststates.StateReasonSkip)
	case !emptyState(updated):
		file.UpdateStatus(moduletest.Error)
		evalCtx.SetFileState(n.stateKey, state.Run, updated, teststates.StateReasonError)
	default:
		evalCtx.SetFileState(n.stateKey, state.Run, updated, teststates.StateReasonNone)
	}
	evalCtx.Renderer().DestroySummary(destroyDiags, state.Run, file, updated)
}

func (n *NodeStateCleanup) restore(ctx *EvalContext, file *configs.TestFile, run *configs.TestRun, module *configs.Config, state *states.State, waiter *operationWaiter) (*states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called restore for %s", run.Name)

	variables, diags := GetVariables(ctx, run, module, false)
	if diags.HasErrors() {
		return state, diags
	}

	// we ignore the diagnostics from here, because we will have reported them
	// during the initial execution of the run block and we would not have
	// executed the run block if there were any errors.
	providers, _, _ := getProviders(ctx, file, run, module)

	// During the destroy operation, we don't add warnings from this operation.
	// Anything that would have been reported here was already reported during
	// the original plan, and a successful destroy operation is the only thing
	// we care about.
	setVariables, _, _ := FilterVariablesToModule(module, variables)

	planOpts := &terraform.PlanOpts{
		Mode:                      plans.NormalMode,
		SetVariables:              setVariables,
		Overrides:                 ctx.GetOverrides(run.Name),
		ExternalProviders:         providers,
		SkipRefresh:               true,
		OverridePreventDestroy:    true,
		DeferralAllowed:           ctx.deferralAllowed,
		AllowRootEphemeralOutputs: true,
	}

	tfCtx, _ := terraform.NewContext(n.opts.ContextOpts)

	waiter.update(tfCtx, moduletest.TearDown, nil)
	plan, planDiags := tfCtx.Plan(module, state, planOpts)
	diags = diags.Append(planDiags)
	if diags.HasErrors() || plan.Errored {
		return state, diags
	}

	if !plan.Complete {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Incomplete restore plan",
			fmt.Sprintf("The restore plan for %s was reported as incomplete."+
				" This means some of the cleanup operations were deferred due to unknown values, please check the rest of the output to see which resources could not be reverted.", run.Name)))
	}

	_, updated, applyDiags := apply(tfCtx, run, module, plan, moduletest.TearDown, variables, providers, waiter)
	diags = diags.Append(applyDiags)
	return updated, diags
}

func (n *NodeStateCleanup) destroy(ctx *EvalContext, file *configs.TestFile, run *configs.TestRun, module *configs.Config, state *states.State, waiter *operationWaiter) (*states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called destroy for %s", run.Name)

	variables, diags := GetVariables(ctx, run, module, false)
	if diags.HasErrors() {
		return state, diags
	}

	// we ignore the diagnostics from here, because we will have reported them
	// during the initial execution of the run block and we would not have
	// executed the run block if there were any errors.
	providers, _, _ := getProviders(ctx, file, run, module)

	// During the destroy operation, we don't add warnings from this operation.
	// Anything that would have been reported here was already reported during
	// the original plan, and a successful destroy operation is the only thing
	// we care about.
	setVariables, _, _ := FilterVariablesToModule(module, variables)

	planOpts := &terraform.PlanOpts{
		Mode:                      plans.DestroyMode,
		SetVariables:              setVariables,
		Overrides:                 ctx.GetOverrides(run.Name),
		ExternalProviders:         providers,
		SkipRefresh:               true,
		OverridePreventDestroy:    true,
		DeferralAllowed:           ctx.deferralAllowed,
		AllowRootEphemeralOutputs: true,
	}

	tfCtx, _ := terraform.NewContext(n.opts.ContextOpts)

	waiter.update(tfCtx, moduletest.TearDown, nil)
	plan, planDiags := tfCtx.Plan(module, state, planOpts)
	diags = diags.Append(planDiags)
	if diags.HasErrors() || plan.Errored {
		return state, diags
	}

	if !plan.Complete {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Incomplete destroy plan",
			fmt.Sprintf("The destroy plan for %s was reported as incomplete."+
				" This means some of the cleanup operations were deferred due to unknown values, please check the rest of the output to see which resources could not be destroyed.", run.Name)))
	}

	_, updated, applyDiags := apply(tfCtx, run, module, plan, moduletest.TearDown, variables, providers, waiter)
	diags = diags.Append(applyDiags)
	return updated, diags
}

func emptyState(state *states.State) bool {
	if state.Empty() {
		return true
	}
	for _, module := range state.Modules {
		for _, resource := range module.Resources {
			if resource.Addr.Resource.Mode == addrs.ManagedResourceMode {
				return false
			}
		}
	}
	return true
}
