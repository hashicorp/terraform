// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeExecutable = (*NodeTestRun)(nil)
	_ BindContextOpts     = (*NodeTestRun)(nil)
)

type NodeTestRun struct {
	file    *moduletest.File
	run     *moduletest.Run
	ctxOpts *terraform.ContextOpts

	// requiredProviders is a map of provider names that the test run depends on.
	requiredProviders map[string]bool
}

func (n *NodeTestRun) Run() *moduletest.Run {
	return n.run
}

func (n *NodeTestRun) File() *moduletest.File {
	return n.file
}

func (n *NodeTestRun) Name() string {
	return fmt.Sprintf("%s.%s", n.file.Name, n.run.Name)
}

func (n *NodeTestRun) References() []*addrs.Reference {
	references, _ := n.run.GetReferences()
	return references
}

func (n *NodeTestRun) BindContextOpts(opts *terraform.ContextOpts) {
	n.ctxOpts = opts
}

// Execute executes the test run block and update the status of the run block
// based on the result of the execution.
func (n *NodeTestRun) Execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	log.Printf("[TRACE] TestFileRunner: executing run block %s/%s", n.file.Name, n.run.Name)
	startTime := time.Now().UTC()
	var diags tfdiags.Diagnostics
	file, run := n.file, n.run

	if file.GetStatus() == moduletest.Error {
		// If the overall test file has errored, we don't keep trying to
		// execute tests. Instead, we mark all remaining run blocks as
		// skipped, print the status, and move on.
		run.Status = moduletest.Skip
		return diags
	}
	if evalCtx.Cancelled() {
		// A cancellation signal has been received.
		// Don't do anything, just give up and return immediately.
		// The surrounding functions should stop this even being called, but in
		// case of race conditions or something we can still verify this.
		return diags
	}

	if evalCtx.Stopped() {
		// Then the test was requested to be stopped, so we just mark each
		// following test as skipped, print the status, and move on.
		run.Status = moduletest.Skip
		return diags
	}

	// Add the providers required by the test run to the context.
	evalCtx.SetProviders(n.run, n.requiredProviders)

	// Create a waiter which handles waiting for terraform operations to complete.
	// While waiting, the wait will also respond to cancellation signals, and
	// handle them appropriately.
	// The test progress is updated periodically, and the progress status
	// depends on the async operation being waited on.
	// Before the terraform operation is started, the operation updates the
	// waiter with the cleanup context on cancellation, as well as the
	// progress status.
	waiter := NewOperationWaiter(nil, evalCtx, n, moduletest.Running, startTime.UnixMilli())
	cancelled := waiter.Run(func() {
		defer logging.PanicHandler()
		n.execute(evalCtx, waiter)
	})

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	// If we got far enough to actually attempt to execute the run then
	// we'll give the view some additional metadata about the execution.
	n.run.ExecutionMeta = &moduletest.RunExecutionMeta{
		Start:    startTime,
		Duration: time.Since(startTime),
	}
	return diags
}

func (n *NodeTestRun) execute(ctx *EvalContext, waiter *operationWaiter) {
	file, run := n.file, n.run
	ctx.Renderer().Run(run, file, moduletest.Starting, 0)
	if run.Config.ConfigUnderTest != nil && run.GetStateKey() == moduletest.MainStateIdentifier {
		// This is bad, and should not happen because the state key is derived from the custom module source.
		panic(fmt.Sprintf("TestFileRunner: custom module %s has the same key as main state", file.Name))
	}

	n.testValidate(ctx, waiter)
	if run.Diagnostics.HasErrors() {
		return
	}

	variables, variableDiags := n.GetVariables(ctx, true)
	run.Diagnostics = run.Diagnostics.Append(variableDiags)
	if variableDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	if run.Config.Command == configs.PlanTestCommand {
		n.testPlan(ctx, variables, waiter)
	} else {
		n.testApply(ctx, variables, waiter)
	}
	return
}

// Validating the module config which the run acts on
func (n *NodeTestRun) testValidate(ctx *EvalContext, waiter *operationWaiter) {
	run := n.run
	file := n.file
	config := run.ModuleConfig

	TransformConfigForRun(ctx, run, file)
	log.Printf("[TRACE] TestFileRunner: called validate for %s/%s", file.Name, run.Name)
	tfCtx, ctxDiags := terraform.NewContext(n.ctxOpts)
	if ctxDiags.HasErrors() {
		return
	}
	waiter.update(tfCtx, moduletest.Running, nil)
	validateDiags := tfCtx.Validate(config, nil)
	run.Diagnostics = run.Diagnostics.Append(validateDiags)
	if validateDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}
}
