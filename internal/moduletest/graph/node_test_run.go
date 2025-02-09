// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ GraphNodeExecutable = (*NodeTestRun)(nil)

type NodeTestRun struct {
	file *moduletest.File
	run  *moduletest.Run

	// requiredProviders is a map of provider names that the test run depends on.
	requiredProviders map[string]bool

	ctxOpts      *terraform.ContextOpts
	cancelledCtx context.Context
	stoppedCtx   context.Context
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

func (n *NodeTestRun) AttachSuiteContext(cancelCtx, stopCtx context.Context, opts *terraform.ContextOpts) {
	n.cancelledCtx = cancelCtx
	n.stoppedCtx = stopCtx
	n.ctxOpts = opts
}

// Execute adds the providers required by the test run to the context.
// TODO: Eventually, we should move all the logic related to a test run into this method,
// effectively ensuring that the Execute method is enough to execute a test run in the graph.
func (n *NodeTestRun) Execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	log.Printf("[TRACE] TestFileRunner: executing run block %s/%s", n.file.Name, n.run.Name)
	var diags tfdiags.Diagnostics
	start := time.Now().UTC().UnixMilli()
	if evalCtx.Cancelled() {
		// Don't do anything, just give up and return immediately.
		// The surrounding functions should stop this even being called, but in
		// case of race conditions or something we can still verify this.
		return diags
	}

	if evalCtx.Stopped() {
		// Basically the same as above, except we'll be a bit nicer.
		n.run.Status = moduletest.Skip
		return diags
	}

	// Add the providers required by the test run to the context.
	evalCtx.SetProviders(n.run, n.requiredProviders)

	w := NewTestWaiter(nil, n.cancelledCtx, n.stoppedCtx, evalCtx, evalCtx.Renderer(), n.run, n.file, nil, moduletest.Running, start)
	RunAndWait(func() {
		defer logging.PanicHandler()
		diags = n.execute(evalCtx, start, w)
	}, w)

	return diags //n.execute(evalCtx, start, w)
}
func (n *NodeTestRun) execute(ctx *EvalContext, start int64, waiter *testWaiter) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	file := n.file
	run := n.run
	key := run.GetStateKey()
	if run.Config.ConfigUnderTest != nil {
		if key == moduletest.MainStateIdentifier {
			// This is bad. It means somehow the module we're loading has
			// the same key as main state and we're about to corrupt things.
			run.Diagnostics = run.Diagnostics.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module source",
				Detail:   fmt.Sprintf("The source for the selected module evaluated to %s which should not be possible. This is a bug in Terraform - please report it!", key),
				Subject:  run.Config.Module.DeclRange.Ptr(),
			})

			run.Status = moduletest.Error
			file.UpdateStatus(moduletest.Error)
			return diags
		}
	}
	config := run.ModuleConfig
	ctx.Renderer().Run(run, file, moduletest.Starting, 0)
	TransformConfigForRun(ctx, run, file)

	// --- Validate the run block ----------------------------------------------
	log.Printf("[TRACE] TestFileRunner: called validate for %s/%s", file.Name, run.Name)
	tfCtx, ctxDiags := terraform.NewContext(n.ctxOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return diags // TODO: Run status
	}
	waiter.update(tfCtx, moduletest.Running, nil)
	validateDiags := tfCtx.Validate(config, nil)
	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		run.Diagnostics = run.Diagnostics.Append(validateDiags)
		run.Status = moduletest.Error
	}
	// --------------------------------------------------------------------------
	// validateDiags := n.validate(run, file, start)
	// run.Diagnostics = run.Diagnostics.Append(validateDiags)
	// if validateDiags.HasErrors() {
	// 	run.Status = moduletest.Error
	// 	return
	// }
	return diags
}

// func (n *NodeTestRun) validate(run *moduletest.Run, file *moduletest.File, start int64) tfdiags.Diagnostics {
// 	log.Printf("[TRACE] TestFileRunner: called validate for %s/%s", file.Name, run.Name)

// 	var diags tfdiags.Diagnostics
// 	config := run.ModuleConfig

// 	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
// 	diags = diags.Append(ctxDiags)
// 	if ctxDiags.HasErrors() {
// 		return diags
// 	}

// 	var validateDiags tfdiags.Diagnostics
// 	validate := func() {
// 		defer logging.PanicHandler()

// 		log.Printf("[DEBUG] TestFileRunner: starting validate for %s/%s", file.Name, run.Name)
// 		validateDiags = tfCtx.Validate(config, nil)
// 		log.Printf("[DEBUG] TestFileRunner: completed validate for  %s/%s", file.Name, run.Name)
// 	}
// 	waitDiags, cancelled := runner.runAndWait(validate, tfCtx, run, file, nil, moduletest.Running, start)

// 	if cancelled {
// 		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
// 	}

// 	diags = diags.Append(waitDiags)
// 	diags = diags.Append(validateDiags)

// 	return diags
// }

func validateRunConfigs(g *terraform.Graph) error {
	for _, v := range g.Vertices() {
		if node, ok := v.(*NodeTestRun); ok {
			diags := node.run.Config.Validate(node.run.ModuleConfig)
			node.run.Diagnostics = node.run.Diagnostics.Append(diags)
			if diags.HasErrors() {
				node.run.Status = moduletest.Error
			}
		}
	}
	return nil
}
