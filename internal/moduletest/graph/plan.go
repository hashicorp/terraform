// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (n *NodeTestRun) testPlan(ctx *EvalContext, variables terraform.InputValues, waiter *operationWaiter) {
	file, run := n.File(), n.run
	config := run.ModuleConfig

	// FilterVariablesToModule only returns warnings, so we don't check the
	// returned diags for errors.
	setVariables, testOnlyVariables, setVariableDiags := n.FilterVariablesToModule(variables)
	run.Diagnostics = run.Diagnostics.Append(setVariableDiags)

	// ignore diags because validate has covered it
	tfCtx, _ := terraform.NewContext(n.opts.ContextOpts)

	// execute the terraform plan operation
	planScope, plan, planDiags := n.plan(ctx, tfCtx, setVariables, waiter)
	// We exclude the diagnostics that are expected to fail from the plan
	// diagnostics, and if an expected failure is not found, we add a new error diagnostic.
	planDiags = run.ValidateExpectedFailures(planDiags)
	run.Diagnostics = run.Diagnostics.Append(planDiags)
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
	newStatus, outputVals, moreDiags := ctx.EvaluateRun(run, planScope, testOnlyVariables)
	run.Status = newStatus
	run.Diagnostics = run.Diagnostics.Append(moreDiags)

	// Now we've successfully validated this run block, lets add it into
	// our prior run outputs so future run blocks can access it.
	ctx.SetOutput(run, outputVals)
}

func (n *NodeTestRun) plan(ctx *EvalContext, tfCtx *terraform.Context, variables terraform.InputValues, waiter *operationWaiter) (*lang.Scope, *plans.Plan, tfdiags.Diagnostics) {
	file, run := n.File(), n.run
	config := run.ModuleConfig
	log.Printf("[TRACE] TestFileRunner: called plan for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics

	targets, targetDiags := run.GetTargets()
	diags = diags.Append(targetDiags)

	replaces, replaceDiags := run.GetReplaces()
	diags = diags.Append(replaceDiags)

	if diags.HasErrors() {
		return nil, nil, diags
	}

	planOpts := &terraform.PlanOpts{
		Mode: func() plans.Mode {
			switch run.Config.Options.Mode {
			case configs.RefreshOnlyTestMode:
				return plans.RefreshOnlyMode
			default:
				return plans.NormalMode
			}
		}(),
		Targets:            targets,
		ForceReplace:       replaces,
		SkipRefresh:        !run.Config.Options.Refresh,
		SetVariables:       variables,
		ExternalReferences: n.References(),
		Overrides:          mocking.PackageOverrides(run.Config, file.Config, config),
		ExternalProviders:  n.ExternalProviders(),
	}

	waiter.update(tfCtx, moduletest.Running, nil)
	log.Printf("[DEBUG] TestFileRunner: starting plan for %s/%s", file.Name, run.Name)
	state := ctx.GetFileState(run.GetStateKey()).State
	plan, planScope, planDiags := tfCtx.PlanAndEval(config, state, planOpts)
	log.Printf("[DEBUG] TestFileRunner: completed plan for %s/%s", file.Name, run.Name)
	diags = diags.Append(planDiags)

	return planScope, plan, diags
}

// ExternalProviders returns the external providers that are available for this test run's module configuration.
// Such providers are expected to already be pre-configured. The test framework does not currently
// configure providers, rather it monkey-patches the module configuration with the providers
// in the test file (see TransformConfigForRun). In future, we may want to
// configure providers in the test framework, but for now we only do this within go tests.
func (n *NodeTestRun) ExternalProviders() map[addrs.RootProviderConfig]providers.Interface {
	ret := make(map[addrs.RootProviderConfig]providers.Interface)
	// Copy the external providers that are actually used in the test run's config.
	for _, provider := range n.Run().ModuleConfig.EffectiveRequiredProviderConfigs().Elements() {
		prov := provider.Value.Local
		addr := addrs.RootProviderConfig{
			Provider: addrs.NewDefaultProvider(prov.LocalName),
			Alias:    prov.Alias,
		}
		if prov, ok := n.opts.Providers[addr]; ok {
			ret[addr] = prov
		}

	}
	return ret

}
