// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// testPlan defines how to execute a run block representing a plan command
//
// See also: (n *NodeTestRun).testApply
func (n *NodeTestRun) testPlan(ctx *EvalContext, variables terraform.InputValues, providers map[addrs.RootProviderConfig]providers.Interface, waiter *operationWaiter) {
	file, run := n.File(), n.run
	config := run.ModuleConfig

	// FilterVariablesToModule only returns warnings, so we don't check the
	// returned diags for errors.
	setVariables, testOnlyVariables, setVariableDiags := FilterVariablesToModule(run.ModuleConfig, variables)
	run.Diagnostics = run.Diagnostics.Append(setVariableDiags)

	// ignore diags because validate has covered it
	tfCtx, _ := terraform.NewContext(n.opts.ContextOpts)

	// execute the terraform plan operation
	planScope, plan, originalDiags := plan(ctx, tfCtx, file.Config, run.Config, run.ModuleConfig, setVariables, providers, waiter)
	// We exclude the diagnostics that are expected to fail from the plan
	// diagnostics, and if an expected failure is not found, we add a new error diagnostic.
	planDiags := moduletest.ValidateExpectedFailures(run.Config, originalDiags)

	if ctx.Verbose() {
		// in verbose mode, we still add all the original diagnostics for
		// display.
		run.Diagnostics = run.Diagnostics.Append(originalDiags)
	} else {
		run.Diagnostics = run.Diagnostics.Append(planDiags)
	}

	if planDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	if ctx.Verbose() {
		schemas, diags := tfCtx.Schemas(config, plan.PriorState)

		// If we're going to fail to render the plan, let's not fail the overall
		// test. It can still have succeeded. So we'll add the diagnostics, but
		// still report the test status as a success.
		if diags.HasErrors() {
			// This is very unlikely.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Failed to print verbose output",
				fmt.Sprintf("Terraform failed to print the verbose output for %s, other diagnostics will contain more details as to why.", filepath.Join(file.Name, run.Name))))
		} else {
			run.Verbose = &moduletest.Verbose{
				Plan:         plan,
				State:        nil, // We don't have a state to show in plan mode.
				Config:       config,
				Providers:    schemas.Providers,
				Provisioners: schemas.Provisioners,
			}
		}

		run.Diagnostics = run.Diagnostics.Append(diags)
	}

	// Evaluate the run block directly in the graph context to validate the assertions
	// of the run. We also pass in all the
	// previous contexts so this run block can refer to outputs from
	// previous run blocks.
	status, outputVals, moreDiags := ctx.EvaluateRun(run.Config, run.ModuleConfig.Module, planScope, testOnlyVariables)
	run.Status = run.Status.Merge(status)
	run.Diagnostics = run.Diagnostics.Append(moreDiags)
	run.Outputs = outputVals
}

func plan(ctx *EvalContext, tfCtx *terraform.Context, file *configs.TestFile, run *configs.TestRun, module *configs.Config, variables terraform.InputValues, providers map[addrs.RootProviderConfig]providers.Interface, waiter *operationWaiter) (*lang.Scope, *plans.Plan, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called plan for %s", run.Name)

	var diags tfdiags.Diagnostics

	targets, targetDiags := moduletest.GetRunTargets(run)
	diags = diags.Append(targetDiags)

	replaces, replaceDiags := moduletest.GetRunReplaces(run)
	diags = diags.Append(replaceDiags)

	references, referenceDiags := moduletest.GetRunReferences(run)
	diags = diags.Append(referenceDiags)

	state, err := ctx.LoadState(run)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to load state",
			Detail:   fmt.Sprintf("Could not retrieve state for run %s: %s.", run.Name, err),
			Subject:  run.Backend.DeclRange.Ptr(),
		})
	}

	if diags.HasErrors() {
		return nil, nil, diags
	}

	planOpts := &terraform.PlanOpts{
		Mode: func() plans.Mode {
			switch run.Options.Mode {
			case configs.RefreshOnlyTestMode:
				return plans.RefreshOnlyMode
			default:
				return plans.NormalMode
			}
		}(),
		Targets:                   targets,
		ForceReplace:              replaces,
		SkipRefresh:               !run.Options.Refresh,
		SetVariables:              variables,
		ExternalReferences:        references,
		ExternalProviders:         providers,
		Overrides:                 ctx.GetOverrides(run.Name),
		DeferralAllowed:           ctx.deferralAllowed,
		AllowRootEphemeralOutputs: true,
	}

	waiter.update(tfCtx, moduletest.Running, nil)
	log.Printf("[DEBUG] TestFileRunner: starting plan for %s", run.Name)
	plan, scope, planDiags := tfCtx.PlanAndEval(module, state, planOpts)
	log.Printf("[DEBUG] TestFileRunner: completed plan for %s", run.Name)
	diags = diags.Append(planDiags)

	return scope, plan, diags
}
