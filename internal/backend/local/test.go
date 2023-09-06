// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"log"
	"path"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"golang.org/x/exp/slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	MainStateIdentifier = ""
)

type TestSuiteRunner struct {
	Config *configs.Config

	GlobalVariables map[string]backend.UnparsedVariableValue
	Opts            *terraform.ContextOpts

	View views.Test

	// Stopped and Cancelled track whether the user requested the testing
	// process to be interrupted. Stopped is a nice graceful exit, we'll still
	// tidy up any state that was created and mark the tests with relevant
	// `skipped` status updates. Cancelled is a hard stop right now exit, we
	// won't attempt to clean up any state left hanging, and tests will just
	// be left showing `pending` as the status. We will still print out the
	// destroy summary diagnostics that tell the user what state has been left
	// behind and needs manual clean up.
	Stopped   bool
	Cancelled bool

	// StoppedCtx and CancelledCtx allow in progress Terraform operations to
	// respond to external calls from the test command.
	StoppedCtx   context.Context
	CancelledCtx context.Context

	// Filter restricts exactly which test files will be executed.
	Filter []string

	// Verbose tells the runner to print out plan files during each test run.
	Verbose bool
}

func (runner *TestSuiteRunner) Stop() {
	runner.Stopped = true
}

func (runner *TestSuiteRunner) Cancel() {
	runner.Cancelled = true
}

func (runner *TestSuiteRunner) Test() (moduletest.Status, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	suite, suiteDiags := runner.collectTests()
	diags = diags.Append(suiteDiags)
	if suiteDiags.HasErrors() {
		return moduletest.Error, diags
	}

	runner.View.Abstract(suite)

	var files []string
	for name := range suite.Files {
		files = append(files, name)
	}
	sort.Strings(files) // execute the files in alphabetical order

	suite.Status = moduletest.Pass
	for _, name := range files {
		if runner.Cancelled {
			return suite.Status, diags
		}

		file := suite.Files[name]

		fileRunner := &TestFileRunner{
			Suite: runner,
			RelevantStates: map[string]*TestFileState{
				MainStateIdentifier: {
					Run:   nil,
					State: states.NewState(),
				},
			},
			PriorStates: make(map[string]*terraform.TestContext),
		}

		fileRunner.Test(file)
		fileRunner.cleanup(file)
		suite.Status = suite.Status.Merge(file.Status)
	}

	runner.View.Conclusion(suite)

	return suite.Status, diags
}

func (runner *TestSuiteRunner) collectTests() (*moduletest.Suite, tfdiags.Diagnostics) {
	runCount := 0
	fileCount := 0

	var diags tfdiags.Diagnostics
	suite := &moduletest.Suite{
		Files: func() map[string]*moduletest.File {
			files := make(map[string]*moduletest.File)

			if len(runner.Filter) > 0 {
				for _, name := range runner.Filter {
					file, ok := runner.Config.Module.Tests[name]
					if !ok {
						// If the filter is invalid, we'll simply skip this
						// entry and print a warning. But we could still execute
						// any other tests within the filter.
						diags.Append(tfdiags.Sourceless(
							tfdiags.Warning,
							"Unknown test file",
							fmt.Sprintf("The specified test file, %s, could not be found.", name)))
						continue
					}

					fileCount++

					var runs []*moduletest.Run
					for ix, run := range file.Runs {
						runs = append(runs, &moduletest.Run{
							Config: run,
							Index:  ix,
							Name:   run.Name,
						})
					}

					runCount += len(runs)
					files[name] = &moduletest.File{
						Config: file,
						Name:   name,
						Runs:   runs,
					}
				}

				return files
			}

			// Otherwise, we'll just do all the tests in the directory!
			for name, file := range runner.Config.Module.Tests {
				fileCount++

				var runs []*moduletest.Run
				for ix, run := range file.Runs {
					runs = append(runs, &moduletest.Run{
						Config: run,
						Index:  ix,
						Name:   run.Name,
					})
				}

				runCount += len(runs)
				files[name] = &moduletest.File{
					Config: file,
					Name:   name,
					Runs:   runs,
				}
			}
			return files
		}(),
	}

	log.Printf("[DEBUG] TestSuiteRunner: found %d files with %d run blocks", fileCount, runCount)

	return suite, diags
}

type TestFileRunner struct {
	// Suite contains all the helpful metadata about the test that we need
	// during the execution of a file.
	Suite *TestSuiteRunner

	// RelevantStates is a mapping of module keys to it's last applied state
	// file.
	//
	// This is used to clean up the infrastructure created during the test after
	// the test has finished.
	RelevantStates map[string]*TestFileState

	// PriorStates is mapping from run block names to the TestContexts that were
	// created when that run block executed.
	//
	// This is used to allow run blocks to refer back to the output values of
	// previous run blocks. It is passed into the Evaluate functions that
	// validate the test assertions, and used when calculating values for
	// variables within run blocks.
	PriorStates map[string]*terraform.TestContext
}

// TestFileState is a helper struct that just maps a run block to the state that
// was produced by the execution of that run block.
type TestFileState struct {
	Run   *moduletest.Run
	State *states.State
}

func (runner *TestFileRunner) Test(file *moduletest.File) {
	log.Printf("[TRACE] TestFileRunner: executing test file %s", file.Name)

	file.Status = file.Status.Merge(moduletest.Pass)
	for _, run := range file.Runs {
		if runner.Suite.Cancelled {
			// This means a hard stop has been requested, in this case we don't
			// even stop to mark future tests as having been skipped. They'll
			// just show up as pending in the printed summary.
			return
		}

		if runner.Suite.Stopped {
			// Then the test was requested to be stopped, so we just mark each
			// following test as skipped and move on.
			run.Status = moduletest.Skip
			continue
		}

		if file.Status == moduletest.Error {
			// If the overall test file has errored, we don't keep trying to
			// execute tests. Instead, we mark all remaining run blocks as
			// skipped.
			run.Status = moduletest.Skip
			continue
		}

		key := MainStateIdentifier
		config := runner.Suite.Config
		if run.Config.ConfigUnderTest != nil {
			config = run.Config.ConfigUnderTest
			// Then we need to load an alternate state and not the main one.

			key = run.Config.Module.Source.String()
			if key == MainStateIdentifier {
				// This is bad. It means somehow the module we're loading has
				// the same key as main state and we're about to corrupt things.

				run.Diagnostics = run.Diagnostics.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid module source",
					Detail:   fmt.Sprintf("The source for the selected module evaluated to %s which should not be possible. This is a bug in Terraform - please report it!", key),
					Subject:  run.Config.Module.DeclRange.Ptr(),
				})

				run.Status = moduletest.Error
				file.Status = moduletest.Error
				continue // Abort!
			}

			if _, exists := runner.RelevantStates[key]; !exists {
				runner.RelevantStates[key] = &TestFileState{
					Run:   nil,
					State: states.NewState(),
				}
			}
		}

		state, updatedState := runner.run(run, file, runner.RelevantStates[key].State, config)
		if updatedState {
			// Only update the most recent run and state if the state was
			// actually updated by this change. We want to use the run that
			// most recently updated the tracked state as the cleanup
			// configuration.
			runner.RelevantStates[key].State = state
			runner.RelevantStates[key].Run = run
		}

		file.Status = file.Status.Merge(run.Status)
	}

	runner.Suite.View.File(file)
	for _, run := range file.Runs {
		runner.Suite.View.Run(run, file)
	}
}

func (runner *TestFileRunner) run(run *moduletest.Run, file *moduletest.File, state *states.State, config *configs.Config) (*states.State, bool) {
	log.Printf("[TRACE] TestFileRunner: executing run block %s/%s", file.Name, run.Name)

	if runner.Suite.Cancelled {
		// Don't do anything, just give up and return immediately.
		// The surrounding functions should stop this even being called, but in
		// case of race conditions or something we can still verify this.
		return state, false
	}

	if runner.Suite.Stopped {
		// Basically the same as above, except we'll be a bit nicer.
		run.Status = moduletest.Skip
		return state, false
	}

	run.Diagnostics = run.Diagnostics.Append(run.Config.Validate())
	if run.Diagnostics.HasErrors() {
		run.Status = moduletest.Error
		return state, false
	}

	resetConfig, configDiags := config.TransformForTest(run.Config, file.Config)
	defer resetConfig()

	run.Diagnostics = run.Diagnostics.Append(configDiags)
	if configDiags.HasErrors() {
		run.Status = moduletest.Error
		return state, false
	}

	validateDiags := runner.validate(config, run, file)
	run.Diagnostics = run.Diagnostics.Append(validateDiags)
	if validateDiags.HasErrors() {
		run.Status = moduletest.Error
		return state, false
	}

	references, referenceDiags := run.GetReferences()
	run.Diagnostics = run.Diagnostics.Append(referenceDiags)
	if referenceDiags.HasErrors() {
		run.Status = moduletest.Error
		return state, false
	}

	planCtx, plan, planDiags := runner.plan(config, state, run, file, references)
	if run.Config.Command == configs.PlanTestCommand {
		// Then we want to assess our conditions and diagnostics differently.
		planDiags = run.ValidateExpectedFailures(planDiags)
		run.Diagnostics = run.Diagnostics.Append(planDiags)
		if planDiags.HasErrors() {
			run.Status = moduletest.Error
			return state, false
		}

		variables, resetVariables, variableDiags := runner.prepareInputVariablesForAssertions(config, run, file, references)
		defer resetVariables()

		run.Diagnostics = run.Diagnostics.Append(variableDiags)
		if variableDiags.HasErrors() {
			run.Status = moduletest.Error
			return state, false
		}

		if runner.Suite.Verbose {
			schemas, diags := planCtx.Schemas(config, plan.PlannedState)

			// If we're going to fail to render the plan, let's not fail the overall
			// test. It can still have succeeded. So we'll add the diagnostics, but
			// still report the test status as a success.
			if diags.HasErrors() {
				// This is very unlikely.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to print verbose output",
					fmt.Sprintf("Terraform failed to print the verbose output for %s, other diagnostics will contain more details as to why.", path.Join(file.Name, run.Name))))
			} else {
				run.Verbose = &moduletest.Verbose{
					Plan:         plan,
					State:        plan.PlannedState,
					Config:       config,
					Providers:    schemas.Providers,
					Provisioners: schemas.Provisioners,
				}
			}

			run.Diagnostics = run.Diagnostics.Append(diags)
		}

		// First, make the test context we can use to validate the assertions
		// of the
		ctx := planCtx.TestContext(run, config, plan.PlannedState, plan, variables)

		// Second, evaluate the run block directly. We also pass in all the
		// previous contexts so this run block can refer to outputs from
		// previous run blocks.
		ctx.Evaluate(runner.PriorStates)

		// Now we've successfully validated this run block, lets add it into
		// our prior states so future run blocks can access it.
		runner.PriorStates[run.Name] = ctx

		return state, false
	}

	// Otherwise any error during the planning prevents our apply from
	// continuing which is an error.
	run.Diagnostics = run.Diagnostics.Append(planDiags)
	if planDiags.HasErrors() {
		run.Status = moduletest.Error
		return state, false
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

	applyCtx, updated, applyDiags := runner.apply(plan, state, config, run, file)

	// Remove expected diagnostics, and add diagnostics in case anything that should have failed didn't.
	applyDiags = run.ValidateExpectedFailures(applyDiags)

	run.Diagnostics = run.Diagnostics.Append(applyDiags)
	if applyDiags.HasErrors() {
		run.Status = moduletest.Error
		// Even though the apply operation failed, the graph may have done
		// partial updates and the returned state should reflect this.
		return updated, true
	}

	variables, resetVariables, variableDiags := runner.prepareInputVariablesForAssertions(config, run, file, references)
	if resetVariables != nil {
		defer resetVariables()
	}

	run.Diagnostics = run.Diagnostics.Append(variableDiags)
	if variableDiags.HasErrors() {
		run.Status = moduletest.Error
		return updated, true
	}

	if runner.Suite.Verbose {
		schemas, diags := planCtx.Schemas(config, plan.PlannedState)

		// If we're going to fail to render the plan, let's not fail the overall
		// test. It can still have succeeded. So we'll add the diagnostics, but
		// still report the test status as a success.
		if diags.HasErrors() {
			// This is very unlikely.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Failed to print verbose output",
				fmt.Sprintf("Terraform failed to print the verbose output for %s, other diagnostics will contain more details as to why.", path.Join(file.Name, run.Name))))
		} else {
			run.Verbose = &moduletest.Verbose{
				Plan:         plan,
				State:        updated,
				Config:       config,
				Providers:    schemas.Providers,
				Provisioners: schemas.Provisioners,
			}
		}

		run.Diagnostics = run.Diagnostics.Append(diags)
	}

	// First, make the test context we can use to validate the assertions
	// of the
	ctx := applyCtx.TestContext(run, config, updated, plan, variables)

	// Second, evaluate the run block directly. We also pass in all the
	// previous contexts so this run block can refer to outputs from
	// previous run blocks.
	ctx.Evaluate(runner.PriorStates)

	// Now we've successfully validated this run block, lets add it into
	// our prior states so future run blocks can access it.
	runner.PriorStates[run.Name] = ctx

	return updated, true
}

func (runner *TestFileRunner) validate(config *configs.Config, run *moduletest.Run, file *moduletest.File) tfdiags.Diagnostics {
	log.Printf("[TRACE] TestFileRunner: called validate for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics

	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	var validateDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestFileRunner: starting validate for %s/%s", file.Name, run.Name)
		validateDiags = tfCtx.Validate(config)
		log.Printf("[DEBUG] TestFileRunner: completed validate for  %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, nil)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(validateDiags)

	return diags
}

func (runner *TestFileRunner) destroy(config *configs.Config, state *states.State, run *moduletest.Run, file *moduletest.File) (*states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called destroy for %s/%s", file.Name, run.Name)

	if state.Empty() {
		// Nothing to do!
		return state, nil
	}

	var diags tfdiags.Diagnostics

	variables, variableDiags := runner.buildInputVariablesForTest(run, file, config)
	diags = diags.Append(variableDiags)

	if diags.HasErrors() {
		return state, diags
	}

	planOpts := &terraform.PlanOpts{
		Mode:         plans.DestroyMode,
		SetVariables: variables,
	}

	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return state, diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestFileRunner: starting destroy plan for %s/%s", file.Name, run.Name)
		plan, planDiags = tfCtx.Plan(config, state, planOpts)
		log.Printf("[DEBUG] TestFileRunner: completed destroy plan for %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, nil)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(planDiags)

	if diags.HasErrors() {
		return state, diags
	}

	_, updated, applyDiags := runner.apply(plan, state, config, run, file)
	diags = diags.Append(applyDiags)
	return updated, diags
}

func (runner *TestFileRunner) plan(config *configs.Config, state *states.State, run *moduletest.Run, file *moduletest.File, references []*addrs.Reference) (*terraform.Context, *plans.Plan, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called plan for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics

	targets, targetDiags := run.GetTargets()
	diags = diags.Append(targetDiags)

	replaces, replaceDiags := run.GetReplaces()
	diags = diags.Append(replaceDiags)

	variables, variableDiags := runner.buildInputVariablesForTest(run, file, config)
	diags = diags.Append(variableDiags)

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
		ExternalReferences: references,
	}

	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, nil, diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestFileRunner: starting plan for %s/%s", file.Name, run.Name)
		plan, planDiags = tfCtx.Plan(config, state, planOpts)
		log.Printf("[DEBUG] TestFileRunner: completed plan for %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, nil)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(planDiags)

	return tfCtx, plan, diags
}

func (runner *TestFileRunner) apply(plan *plans.Plan, state *states.State, config *configs.Config, run *moduletest.Run, file *moduletest.File) (*terraform.Context, *states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called apply for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics

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

	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, state, diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	var updated *states.State
	var applyDiags tfdiags.Diagnostics

	go func() {
		defer logging.PanicHandler()
		defer done()
		log.Printf("[DEBUG] TestFileRunner: starting apply for %s/%s", file.Name, run.Name)
		updated, applyDiags = tfCtx.Apply(plan, config)
		log.Printf("[DEBUG] TestFileRunner: completed apply for %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, created)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(applyDiags)

	return tfCtx, updated, diags
}

func (runner *TestFileRunner) wait(ctx *terraform.Context, runningCtx context.Context, run *moduletest.Run, file *moduletest.File, created []*plans.ResourceInstanceChangeSrc) (diags tfdiags.Diagnostics, cancelled bool) {
	var identifier string
	if file == nil {
		identifier = "validate"
	} else {
		identifier = file.Name
		if run != nil {
			identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
		}
	}
	log.Printf("[TRACE] TestFileRunner: waiting for execution during %s", identifier)

	// This function handles what happens when the user presses the second
	// interrupt. This is a "hard cancel", we are going to stop doing whatever
	// it is we're doing. This means even if we're halfway through creating or
	// destroying infrastructure we just give up.
	handleCancelled := func() {
		log.Printf("[DEBUG] TestFileRunner: test execution cancelled during %s", identifier)

		states := make(map[*moduletest.Run]*states.State)
		states[nil] = runner.RelevantStates[MainStateIdentifier].State
		for key, module := range runner.RelevantStates {
			if key == MainStateIdentifier {
				continue
			}
			states[module.Run] = module.State
		}
		runner.Suite.View.FatalInterruptSummary(run, file, states, created)

		cancelled = true
		go ctx.Stop()

		// Just wait for things to finish now, the overall test execution will
		// exit early if this takes too long.
		<-runningCtx.Done()
	}

	// This function handles what happens when the user presses the first
	// interrupt. This is essentially a "soft cancel", we're not going to do
	// anything but just wait for things to finish safely. But, we do listen
	// for the crucial second interrupt which will prompt a hard stop / cancel.
	handleStopped := func() {
		log.Printf("[DEBUG] TestFileRunner: test execution stopped during %s", identifier)

		select {
		case <-runner.Suite.CancelledCtx.Done():
			// We've been asked again. This time we stop whatever we're doing
			// and abandon all attempts to do anything reasonable.
			handleCancelled()
		case <-runningCtx.Done():
			// Do nothing, we finished safely and skipping the remaining tests
			// will be handled elsewhere.
		}

	}

	select {
	case <-runner.Suite.StoppedCtx.Done():
		handleStopped()
	case <-runner.Suite.CancelledCtx.Done():
		handleCancelled()
	case <-runningCtx.Done():
		// The operation exited normally.
	}

	return diags, cancelled
}

func (runner *TestFileRunner) cleanup(file *moduletest.File) {
	var diags tfdiags.Diagnostics

	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)

	if runner.Suite.Cancelled {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return
	}

	// First, we'll clean up the main state.
	main := runner.RelevantStates[MainStateIdentifier]

	updated := main.State
	if main.Run == nil {
		if !main.State.Empty() {
			log.Printf("[ERROR] TestFileRunner: found inconsistent run block and state file in %s", file.Name)
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Inconsistent state", fmt.Sprintf("Found inconsistent state while cleaning up %s. This is a bug in Terraform - please report it", file.Name)))
		}
	} else {
		reset, configDiags := runner.Suite.Config.TransformForTest(main.Run.Config, file.Config)
		diags = diags.Append(configDiags)

		if !configDiags.HasErrors() {
			var destroyDiags tfdiags.Diagnostics
			updated, destroyDiags = runner.destroy(runner.Suite.Config, main.State, main.Run, file)
			diags = diags.Append(destroyDiags)
		}

		reset()
	}

	if !updated.Empty() {
		// Then we failed to adequately clean up the state, so mark success
		// as false.
		file.Status = moduletest.Error
	}
	runner.Suite.View.DestroySummary(diags, main.Run, file, updated)

	if runner.Suite.Cancelled {
		// In case things were cancelled during the last execution.
		return
	}

	var states []*TestFileState
	for key, state := range runner.RelevantStates {
		if key == MainStateIdentifier {
			// We processed the main state above.
			continue
		}

		if state.Run == nil {
			if state.State.Empty() {
				// We can see a run block being empty when the state is empty if
				// a module was only used to execute plan commands. So this is
				// okay, and means we have nothing to cleanup so we'll just
				// skip it.
				continue
			}
			log.Printf("[ERROR] TestFileRunner: found inconsistent run block and state file in %s for module %s", file.Name, key)

			// Otherwise something bad has happened, and we have no way to
			// recover from it. This shouldn't happen in reality, but we'll
			// print a diagnostic instead of panicking later.

			var diags tfdiags.Diagnostics
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Inconsistent state", fmt.Sprintf("Found inconsistent state while cleaning up %s. This is a bug in Terraform - please report it", file.Name)))
			file.Status = moduletest.Error
			runner.Suite.View.DestroySummary(diags, nil, file, state.State)
			continue
		}

		states = append(states, state)
	}

	slices.SortFunc(states, func(a, b *TestFileState) int {
		// We want to clean up later run blocks first. So, we'll sort this in
		// reverse according to index. This means larger indices first.
		return b.Run.Index - a.Run.Index
	})

	// Then we'll clean up the additional states for custom modules in reverse
	// order.
	for _, state := range states {
		log.Printf("[DEBUG] TestStateManager: cleaning up state for %s/%s", file.Name, state.Run.Name)

		if runner.Suite.Cancelled {
			// In case the cancellation came while a previous state was being
			// destroyed.
			log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s/%s due to cancellation", file.Name, state.Run.Name)
			return
		}

		var diags tfdiags.Diagnostics

		reset, configDiags := state.Run.Config.ConfigUnderTest.TransformForTest(state.Run.Config, file.Config)
		diags = diags.Append(configDiags)

		updated := state.State
		if !diags.HasErrors() {
			var destroyDiags tfdiags.Diagnostics
			updated, destroyDiags = runner.destroy(state.Run.Config.ConfigUnderTest, state.State, state.Run, file)
			diags = diags.Append(destroyDiags)
		}

		if !updated.Empty() {
			// Then we failed to adequately clean up the state, so mark success
			// as false.
			file.Status = moduletest.Error
		}
		runner.Suite.View.DestroySummary(diags, state.Run, file, updated)

		reset()
	}

	return
}

// buildInputVariablesForTest creates a terraform.InputValues mapping for
// variable values that are relevant to the config being tested.
//
// Crucially, it differs from prepareInputVariablesForAssertions in that it only
// includes variables that are reference by the config and not everything that
// is defined within the test run block and test file.
func (runner *TestFileRunner) buildInputVariablesForTest(run *moduletest.Run, file *moduletest.File, config *configs.Config) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// configVariables keeps track of the variables that will actually be given
	// to the terraform graph to provide values to the configuration.
	configVariables := make(terraform.InputValues)

	// ctxVariables contains all the possible variables we have definitions for
	// and is used to build the context that is used to evaluate variables.
	ctxVariables := make(terraform.InputValues)

	// First, we process all the global variables.
	for name, value := range runner.Suite.GlobalVariables {
		var variableDiags tfdiags.Diagnostics
		if variable, exists := config.Module.Variables[name]; exists {
			ctxVariables[name], variableDiags = value.ParseVariableValue(variable.ParsingMode)
			configVariables[name] = ctxVariables[name]
		} else {
			// Since we don't have the config here to parse the variable value
			// we just blanket parse it as an HCL expression. We don't include
			// this in the configVariables, as we only want variables that are
			// defined in the context.
			ctxVariables[name], variableDiags = value.ParseVariableValue(configs.VariableParseHCL)
		}
		diags = diags.Append(variableDiags)
	}

	// Second, we process the variables defined at the file level
	//
	// We're happy for anything here to override any values from the global
	// variables.
	if file != nil {
		for name, expr := range file.Config.Variables {

			value := unparsedTestVariableValue{
				expr: expr,
			}

			var variableDiags tfdiags.Diagnostics
			if variable, exists := config.Module.Variables[name]; exists {
				ctxVariables[name], variableDiags = value.ParseVariableValue(variable.ParsingMode)
				configVariables[name] = ctxVariables[name]
			} else {
				// As above, we don't have this defined in the config so we
				// parse it as an expression and don't include it in
				// configVariables.
				ctxVariables[name], variableDiags = value.ParseVariableValue(configs.VariableParseHCL)
			}
			diags = diags.Append(variableDiags)
		}
	}

	// Thirdly, we process the variables defined at the run level and pull out
	// any that are relevant to the config under test.
	//
	// We're happy for anything here to override any values from the global or
	// file level variables
	if run != nil {
		skipVars := false

		ctx, ctxDiags := runner.ctx(run, file, ctxVariables)
		diags = diags.Append(ctxDiags)
		if ctxDiags.HasErrors() {
			// We still want to validate all the right variables are being
			// declared. So we don't return early, but we note that we shouldn't
			// eval vars from this block.
			skipVars = true
		}

		for name, expr := range run.Config.Variables {
			variable, exists := config.Module.Variables[name]
			if !exists {
				// At this point we are going to add a warning if a variable
				// is defined within a run block and not referenced by the
				// configuration under test.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Value for undeclared variable",
					Detail:   fmt.Sprintf("The module under test does not declare a variable named %q, but it is declared in run block %q.", name, run.Name),
					Subject:  expr.Range().Ptr(),
				})

				continue
			}

			if skipVars {
				// Then we don't have a valid evaluation context, so we won't
				// actually process these variables. We'll put in a dummy value
				// knowing that we have errors in the diags so these won't be
				// processed.
				//
				// We still want to track this variable has a value, even if we
				// don't know what it is, because we have some validations later
				// that we don't want to trigger because this variable is
				// missing.

				configVariables[name] = &terraform.InputValue{
					Value:       cty.NilVal,
					SourceType:  terraform.ValueFromConfig,
					SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
				}

				continue
			}

			value := unparsedTestVariableValue{
				expr: expr,
				ctx:  ctx,
			}

			var variableDiags tfdiags.Diagnostics
			configVariables[name], variableDiags = value.ParseVariableValue(variable.ParsingMode)
			diags = diags.Append(variableDiags)
		}

	}

	// Finally, we'll do something about any variables defined in the
	// configuration that we haven't given values for.

	for name, variable := range config.Module.Variables {

		if _, exists := configVariables[name]; exists {
			// Then we have a value for this variable already.
			continue
		}

		// Otherwise, we're going to give these variables a value. They'll be
		// processed by the Terraform graph and provided a default value later
		// if they have one.

		if variable.Required() {

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "No value for required variable",
				Detail: fmt.Sprintf("The module under test for run block %q has a required variable %q with no set value. Use a -var or -var-file command line argument or add this variable into a \"variables\" block within the test file or run block.",
					run.Name, variable.Name),
				Subject: variable.DeclRange.Ptr(),
			})

			configVariables[name] = &terraform.InputValue{
				Value:       cty.DynamicVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		} else {
			configVariables[name] = &terraform.InputValue{
				Value:       cty.NilVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		}

	}

	return configVariables, diags
}

// prepareInputVariablesForAssertions creates a terraform.InputValues mapping
// that contains all the variables defined for a given run and file, alongside
// any unset variables that have defaults within the provided config.
//
// Crucially, it differs from buildInputVariablesForTest in that the returned
// input values include all variables available even if they are not defined
// within the config. This allows the assertions to refer to variables defined
// solely within the test file, and not only those within the configuration.
//
// In addition, it modifies the provided config so that any variables that are
// available are also defined in the config. It returns a function that resets
// the config which must be called so the config can be reused going forward.
func (runner *TestFileRunner) prepareInputVariablesForAssertions(config *configs.Config, run *moduletest.Run, file *moduletest.File, references []*addrs.Reference) (terraform.InputValues, func(), tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// process is a helper function that converts an unparsed variable into an
	// input value. All the various input formats share this logic so we extract
	// it out here.
	process := func(name string, value backend.UnparsedVariableValue, reference *addrs.Reference) (*terraform.InputValue, tfdiags.Diagnostics) {
		if config, exists := config.Module.Variables[name]; exists {
			variable, diags := value.ParseVariableValue(config.ParsingMode)
			if diags.HasErrors() {
				return variable, diags
			}

			// Normally, variable values would be converted during the Terraform
			// graph processing. But, `terraform test` assertions are not
			// executed during the graph but after. This means the variables we
			// create for use in the assertions must be converted here.

			converted, err := convert.Convert(variable.Value, config.Type)
			if err != nil {
				var subject *hcl.Range
				if reference != nil {
					subject = reference.SourceRange.ToHCL().Ptr()
				}

				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid value for input variable",
					Detail:   fmt.Sprintf("The given value is not suitable for var.%s declared at %s: %s.", name, config.DeclRange.String(), err),
					Subject:  subject,
				})
				return variable, diags
			}

			variable.Value = converted
			return variable, diags
		} else {

			// If the variable isn't defined in the config, then we don't know
			// what type it is supposed to be. So we'll just parse it as HCL and
			// we can deduce the type that way.

			return value.ParseVariableValue(configs.VariableParseHCL)
		}
	}

	// relevant keeps track of the variables that are actually referenced by
	// this set of assertions.
	relevant := make(map[string]*addrs.Reference)
	for _, reference := range references {
		addr, ok := reference.Subject.(addrs.InputVariable)
		if !ok {
			// We only care about variables.
			continue
		}

		relevant[addr.Name] = reference
	}

	variables := make(terraform.InputValues)

	// Now, we're going to process the various different sources of variables
	// and turn them into input values that our test context can read.

	// First, we'll process the global variables.

	for name, value := range runner.Suite.GlobalVariables {
		variable, variableDiags := process(name, value, relevant[name])
		diags = diags.Append(variableDiags)
		if variable != nil {
			variables[name] = variable
		}
	}

	// Second, we'll process the variables from the file.

	if file != nil {
		for name, expr := range file.Config.Variables {
			value := unparsedTestVariableValue{
				expr: expr,
			}

			variable, variableDiags := process(name, value, relevant[name])
			diags = diags.Append(variableDiags)
			if variable != nil {
				variables[name] = variable
			}
		}
	}

	// Third, we'll process the variables from the run block. We pass in the
	// variables from the global and file level into the eval context here so
	// that users can set run variables from file and global variables.

	if run != nil {
		skipVars := false

		ctx, ctxDiags := runner.ctx(run, file, variables)
		diags = diags.Append(ctxDiags)
		if ctxDiags.HasErrors() {
			// Then we won't try and actually evaluate run variables but we do
			// keep note of them.
			skipVars = true
		}

		for name, expr := range run.Config.Variables {

			if skipVars {

				// Then we had a problem with the evaluation context.
				//
				// We'll just make a placeholder input value so we can finish
				// evaluating everything else. We won't end up using the
				// placeholder values as the test will fail due to the errored
				// diags when we build the context.

				variables[name] = &terraform.InputValue{
					Value:       cty.NilVal,
					SourceType:  terraform.ValueFromConfig,
					SourceRange: tfdiags.SourceRangeFromHCL(expr.Range()),
				}

				continue
			}

			value := unparsedTestVariableValue{
				expr: expr,
				ctx:  ctx,
			}

			variable, variableDiags := process(name, value, relevant[name])
			diags = diags.Append(variableDiags)
			if variable != nil {
				variables[name] = variable
			}
		}
	}

	// Finally, we look for any default values from the configuration for
	// variables that we haven't assigned a value to yet.

	for name, variable := range config.Module.Variables {
		if _, exists := variables[name]; exists {
			// Then we don't want to apply the default for this variable as we
			// already have a value.
			continue
		}

		if variable.Default != cty.NilVal {
			variables[name] = &terraform.InputValue{
				Value:       variable.Default,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		}
	}

	// Now we're going to do a some modifications to the config.
	//
	// If we have got variable values from the test file we need to make sure
	// they have an equivalent entry in the configuration. We're going to do
	// that dynamically here.

	// First, take a backup of the existing configuration so we can easily
	// restore it later.
	currentVars := make(map[string]*configs.Variable)
	for name, variable := range config.Module.Variables {
		currentVars[name] = variable
	}

	// Next, let's go through our entire inputs and add any that aren't already
	// defined into the config.
	for name, value := range variables {
		if _, exists := config.Module.Variables[name]; exists {
			continue
		}

		config.Module.Variables[name] = &configs.Variable{
			Name:           name,
			Type:           value.Value.Type(),
			ConstraintType: value.Value.Type(),
			DeclRange:      value.SourceRange.ToHCL(),
		}
	}

	// We return our input values, a function that will reset the variables
	// within the config so it can be used again, and any diagnostics reporting
	// variables that we couldn't parse.

	return variables, func() {
		config.Module.Variables = currentVars
	}, diags
}

// EvalCtx returns an hcl.EvalContext that allows the variables blocks within
// run blocks to evaluate references to the outputs from other run blocks.
func (runner *TestFileRunner) ctx(run *moduletest.Run, file *moduletest.File, availableVariables terraform.InputValues) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	availableRunBlocks := make(map[string]bool)
	for _, run := range file.Runs {
		name := run.Name

		if _, exists := runner.PriorStates[name]; exists {
			// We have executed this run block previously, therefore it is
			// available as a reference at this point in time.
			availableRunBlocks[name] = true
			continue
		}

		// We haven't executed this run block yet, therefore it is not available
		// as a reference at this point in time.
		availableRunBlocks[name] = false
	}

	for _, value := range run.Config.Variables {
		refs, refDiags := lang.ReferencesInExpr(addrs.ParseRefFromTestingScope, value)
		diags = diags.Append(refDiags)
		if refDiags.HasErrors() {
			continue
		}

		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.Run); ok {
				available, exists := availableRunBlocks[addr.Name]

				if !exists {
					// Then this is a made up run block.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unknown run block",
						Detail:   fmt.Sprintf("The run block %q does not exist within this test file. You can only reference run blocks that are in the same test file and will execute before the current run block.", addr.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				if !available {
					// This run block exists, but it is after the current run block.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable run block",
						Detail:   fmt.Sprintf("The run block %q is not available to the current run block. You can only reference run blocks that are in the same test file and will execute before the current run block.", addr.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				// Otherwise, we're good. This is an acceptable reference.
				continue
			}

			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				if _, exists := availableVariables[addr.Name]; !exists {
					// This variable reference doesn't exist.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable variable",
						Detail:   fmt.Sprintf("The input variable %q is not available to the current run block. You can only reference variables defined at the file or global levels when populating the variables block within a run block.", addr.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				// Otherwise, we're good. This is an acceptable reference.
				continue
			}

			// You can only reference run blocks and variables from the run
			// block variables.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "You can only reference earlier run blocks, file level, and global variables while defining variables from inside a run block.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	return &hcl.EvalContext{
		Variables: func() map[string]cty.Value {
			blocks := make(map[string]cty.Value)
			for run, ctx := range runner.PriorStates {

				outputs := make(map[string]cty.Value)
				for _, output := range ctx.Config.Module.Outputs {
					value := ctx.State.OutputValue(addrs.AbsOutputValue{
						Module: addrs.RootModuleInstance,
						OutputValue: addrs.OutputValue{
							Name: output.Name,
						},
					})

					if value == nil {
						// Then this output returned null when the configuration
						// executed. For now, we'll just skip this output.
						//
						// There are several things we could try to do, like
						// figure out the type based on the variable that it
						// is referencing and wrap it up as cty.Val(...) or we
						// could not try and work anything out and return it as
						// a cty.NilVal.
						//
						// Both of these mean the error would be raised later
						// as non-optional variables would say they don't have
						// a value. By just ignoring it here, we get an error
						// quicker that says this output doesn't exist. I think
						// that would prompt users to go look at the output and
						// realise it might be returning null and make the
						// connection. With the other approaches they'd look at
						// their variable definitions and think they are
						// assigning it a value since we would be telling them
						// the output does exist.
						//
						// Let's do the simple thing now, and see what the
						// future holds.
						continue
					}

					if value.Sensitive || output.Sensitive {
						outputs[output.Name] = value.Value.Mark(marks.Sensitive)
						continue
					}

					outputs[output.Name] = value.Value
				}

				blocks[run] = cty.ObjectVal(outputs)
			}

			variables := make(map[string]cty.Value)
			for name, variable := range availableVariables {
				variables[name] = variable.Value
			}

			return map[string]cty.Value{
				"run": cty.ObjectVal(blocks),
				"var": cty.ObjectVal(variables),
			}
		}(),
	}, diags
}
