// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (n *NodeTestRun) testApply(ctx *EvalContext, variables terraform.InputValues, waiter *operationWaiter) {
	file, run := n.File(), n.run
	config := run.ModuleConfig
	key := n.run.GetStateKey()

	// FilterVariablesToModule only returns warnings, so we don't check the
	// returned diags for errors.
	setVariables, testOnlyVariables, setVariableDiags := n.FilterVariablesToModule(variables)
	run.Diagnostics = run.Diagnostics.Append(setVariableDiags)

	// ignore diags because validate has covered it
	tfCtx, _ := terraform.NewContext(n.opts.ContextOpts)

	// execute the terraform plan operation
	_, plan, planDiags := n.plan(ctx, tfCtx, setVariables, waiter)

	// Any error during the planning prevents our apply from
	// continuing which is an error.
	planDiags = run.ExplainExpectedFailures(planDiags)
	run.Diagnostics = run.Diagnostics.Append(planDiags)
	if planDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	// Since we're carrying on an executing the apply operation as well, we're
	// just going to do some post processing of the diagnostics. We remove the
	// warnings generated from check blocks, as the apply operation will either
	// reproduce them or fix them and we don't want fixed diagnostics to be
	// reported and we don't want duplicates either.
	var filteredDiags tfdiags.Diagnostics
	for _, diag := range run.Diagnostics {
		if rule, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok && rule.Container.CheckableKind() == addrs.CheckableCheck {
			continue
		}
		filteredDiags = filteredDiags.Append(diag)
	}
	run.Diagnostics = filteredDiags

	// execute the apply operation
	applyScope, updated, applyDiags := n.apply(tfCtx, plan, moduletest.Running, variables, waiter)

	// Remove expected diagnostics, and add diagnostics in case anything that should have failed didn't.
	// We'll also update the run status based on the presence of errors or missing expected failures.
	failOrErr := n.checkForMissingExpectedFailures(run, applyDiags)
	if failOrErr {
		// Even though the apply operation failed, the graph may have done
		// partial updates and the returned state should reflect this.
		ctx.SetFileState(key, &TestFileState{
			Run:   run,
			State: updated,
		})
		return
	}

	n.AddVariablesToConfig(variables)

	if ctx.Verbose() {
		schemas, diags := tfCtx.Schemas(config, updated)

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
				Plan:         nil, // We don't have a plan to show in apply mode.
				State:        updated,
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
	newStatus, outputVals, moreDiags := ctx.EvaluateRun(run, applyScope, testOnlyVariables)
	run.Status = newStatus
	run.Diagnostics = run.Diagnostics.Append(moreDiags)

	// Now we've successfully validated this run block, lets add it into
	// our prior run outputs so future run blocks can access it.
	ctx.SetOutput(run, outputVals)

	// Only update the most recent run and state if the state was
	// actually updated by this change. We want to use the run that
	// most recently updated the tracked state as the cleanup
	// configuration.
	ctx.SetFileState(key, &TestFileState{
		Run:   run,
		State: updated,
	})
}

func (n *NodeTestRun) apply(tfCtx *terraform.Context, plan *plans.Plan, progress moduletest.Progress, variables terraform.InputValues, waiter *operationWaiter) (*lang.Scope, *states.State, tfdiags.Diagnostics) {
	run := n.run
	file := n.File()
	log.Printf("[TRACE] TestFileRunner: called apply for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics
	config := run.ModuleConfig

	// If things get cancelled while we are executing the apply operation below
	// we want to print out all the objects that we were creating so the user
	// can verify we managed to tidy everything up possibly.
	//
	// Unfortunately, this creates a race condition as the apply operation can
	// edit the plan (by removing changes once they are applied) while at the
	// same time our cancellation process will try to read the plan.
	//
	// We take a quick copy of the changes we care about here, which will then
	// be used in place of the plan when we print out the objects to be created
	// as part of the cancellation process.
	var created []*plans.ResourceInstanceChangeSrc
	for _, change := range plan.Changes.Resources {
		if change.Action != plans.Create {
			continue
		}
		created = append(created, change)
	}

	// We only need to pass ephemeral variables to the apply operation, as the
	// plan has already been evaluated with the full set of variables.
	ephemeralVariables := make(terraform.InputValues)
	for k, v := range config.Root.Module.Variables {
		if v.EphemeralSet {
			if value, ok := variables[k]; ok {
				ephemeralVariables[k] = value
			}
		}
	}

	applyOpts := &terraform.ApplyOpts{
		SetVariables: ephemeralVariables,
	}

	waiter.update(tfCtx, progress, created)
	log.Printf("[DEBUG] TestFileRunner: starting apply for %s/%s", file.Name, run.Name)
	updated, newScope, applyDiags := tfCtx.ApplyAndEval(plan, config, applyOpts)
	log.Printf("[DEBUG] TestFileRunner: completed apply for %s/%s", file.Name, run.Name)
	diags = diags.Append(applyDiags)

	return newScope, updated, diags
}

// checkForMissingExpectedFailures checks for missing expected failures in the diagnostics.
// It updates the run status based on the presence of errors or missing expected failures.
func (n *NodeTestRun) checkForMissingExpectedFailures(run *moduletest.Run, diags tfdiags.Diagnostics) (failOrErr bool) {
	// Retrieve and append diagnostics that are either unrelated to expected failures
	// or report missing expected failures.
	unexpectedDiags := run.ValidateExpectedFailures(diags)
	run.Diagnostics = run.Diagnostics.Append(unexpectedDiags)
	for _, diag := range unexpectedDiags {
		// // If any diagnostic indicates a missing expected failure, set the run status to fail.
		if ok := moduletest.DiagnosticFromMissingExpectedFailure(diag); ok {
			run.Status = run.Status.Merge(moduletest.Fail)
			continue
		}

		// upgrade the run status to error if there still are other errors in the diagnostics
		if diag.Severity() == tfdiags.Error {
			run.Status = run.Status.Merge(moduletest.Error)
			break
		}
	}
	return run.Status > moduletest.Pass
}
