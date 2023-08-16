// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/exp/slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
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

type TestCommand struct {
	Meta
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform [global options] test [options]

  Executes automated integration tests against the current Terraform 
  configuration.

  Terraform will search for .tftest.hcl files within the current configuration 
  and testing directories. Terraform will then execute the testing run blocks 
  within any testing files in order, and verify conditional checks and 
  assertions against the created infrastructure. 

  This command creates real infrastructure and will attempt to clean up the
  testing infrastructure on completion. Monitor the output carefully to ensure
  this cleanup process is successful.

Options:

  -filter=testfile      If specified, Terraform will only execute the test files
                        specified by this flag. You can use this option multiple
                        times to execute more than one test file.

  -json                 If specified, machine readable output will be printed in
                        JSON format

  -no-color             If specified, output won't contain any color.

  -test-directory=path	Set the Terraform test directory, defaults to "tests".    

  -var 'foo=bar'        Set a value for one of the input variables in the root
                        module of the configuration. Use this option more than
                        once to set more than one variable.

  -var-file=filename    Load variable values from the given file, in addition
                        to the default files terraform.tfvars and *.auto.tfvars.
                        Use this option more than once to include more than one
                        variables file.

  -verbose              Print the plan or state for each test run block as it
                        executes.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Execute integration tests for Terraform modules"
}

func (c *TestCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseTest(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("test")
		return 1
	}

	view := views.NewTest(args.ViewType, c.View)

	config, configDiags := c.loadConfigWithTests(".", args.TestDirectory)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	runCount := 0
	fileCount := 0

	var fileDiags tfdiags.Diagnostics
	suite := moduletest.Suite{
		Files: func() map[string]*moduletest.File {
			files := make(map[string]*moduletest.File)

			if len(args.Filter) > 0 {
				for _, name := range args.Filter {
					file, ok := config.Module.Tests[name]
					if !ok {
						// If the filter is invalid, we'll simply skip this
						// entry and print a warning. But we could still execute
						// any other tests within the filter.
						fileDiags.Append(tfdiags.Sourceless(
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
			for name, file := range config.Module.Tests {
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

	log.Printf("[DEBUG] TestCommand: found %d files with %d run blocks", fileCount, runCount)

	diags = diags.Append(fileDiags)
	if fileDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	// Users can also specify variables via the command line, so we'll parse
	// all that here.
	var items []rawFlag
	for _, variable := range args.Vars.All() {
		items = append(items, rawFlag{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	c.variableArgs = rawFlags{items: &items}

	variables, variableDiags := c.collectVariableValues()
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	opts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	// Print out all the diagnostics we have from the setup. These will just be
	// warnings, and we want them out of the way before we start the actual
	// testing.
	view.Diagnostics(nil, nil, diags)

	// We have two levels of interrupt here. A 'stop' and a 'cancel'. A 'stop'
	// is a soft request to stop. We'll finish the current test, do the tidy up,
	// but then skip all remaining tests and run blocks. A 'cancel' is a hard
	// request to stop now. We'll cancel the current operation immediately
	// even if it's a delete operation, and we won't clean up any infrastructure
	// if we're halfway through a test. We'll print details explaining what was
	// stopped so the user can do their best to recover from it.

	runningCtx, done := context.WithCancel(context.Background())
	stopCtx, stop := context.WithCancel(runningCtx)
	cancelCtx, cancel := context.WithCancel(context.Background())

	runner := &TestSuiteRunner{
		command: c,

		Suite:  &suite,
		Config: config,
		View:   view,

		GlobalVariables: variables,
		Opts:            opts,

		CancelledCtx: cancelCtx,
		StoppedCtx:   stopCtx,

		// Just to be explicit, we'll set the following fields even though they
		// default to these values.
		Cancelled: false,
		Stopped:   false,

		Verbose: args.Verbose,
	}

	view.Abstract(&suite)

	go func() {
		defer logging.PanicHandler()
		defer done()
		defer stop()
		defer cancel()

		runner.Start()
	}()

	// Wait for the operation to complete, or for an interrupt to occur.
	select {
	case <-c.ShutdownCh:
		// Nice request to be cancelled.

		view.Interrupted()
		runner.Stopped = true
		stop()

		select {
		case <-c.ShutdownCh:
			// The user pressed it again, now we have to get it to stop as
			// fast as possible.

			view.FatalInterrupt()
			runner.Cancelled = true
			cancel()

			// We'll wait 5 seconds for this operation to finish now, regardless
			// of whether it finishes successfully or not.
			select {
			case <-runningCtx.Done():
			case <-time.After(5 * time.Second):
			}

		case <-runningCtx.Done():
			// The application finished nicely after the request was stopped.
		}
	case <-runningCtx.Done():
		// tests finished normally with no interrupts.
	}

	if runner.Cancelled {
		// Don't print out the conclusion if the test was cancelled.
		return 1
	}

	view.Conclusion(&suite)

	if suite.Status != moduletest.Pass {
		return 1
	}
	return 0
}

// TestSuiteRunner executes an entire set of Terraform test files.
//
// It contains all shared information needed by all the test files, like the
// main configuration and the global variable values.
type TestSuiteRunner struct {
	command *TestCommand

	Suite  *moduletest.Suite
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

	// Verbose tells the runner to print out plan files during each test run.
	Verbose bool
}

func (runner *TestSuiteRunner) Start() {
	var files []string
	for name := range runner.Suite.Files {
		files = append(files, name)
	}
	sort.Strings(files) // execute the files in alphabetical order

	runner.Suite.Status = moduletest.Pass
	for _, name := range files {
		if runner.Cancelled {
			return
		}

		file := runner.Suite.Files[name]

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

		fileRunner.ExecuteTestFile(file)
		fileRunner.Cleanup(file)
		runner.Suite.Status = runner.Suite.Status.Merge(file.Status)
	}
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

func (runner *TestFileRunner) ExecuteTestFile(file *moduletest.File) {
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

		state, updatedState := runner.ExecuteTestRun(run, file, runner.RelevantStates[key].State, config)
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

func (runner *TestFileRunner) ExecuteTestRun(run *moduletest.Run, file *moduletest.File, state *states.State, config *configs.Config) (*states.State, bool) {
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

	planCtx, plan, planDiags := runner.plan(config, state, run, file)
	if run.Config.Command == configs.PlanTestCommand {
		// Then we want to assess our conditions and diagnostics differently.
		planDiags = run.ValidateExpectedFailures(planDiags)
		run.Diagnostics = run.Diagnostics.Append(planDiags)
		if planDiags.HasErrors() {
			run.Status = moduletest.Error
			return state, false
		}

		variables, resetVariables, variableDiags := runner.prepareInputVariablesForAssertions(config, run, file)
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

	variables, resetVariables, variableDiags := runner.prepareInputVariablesForAssertions(config, run, file)
	defer resetVariables()

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

func (runner *TestFileRunner) plan(config *configs.Config, state *states.State, run *moduletest.Run, file *moduletest.File) (*terraform.Context, *plans.Plan, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called plan for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics

	targets, targetDiags := run.GetTargets()
	diags = diags.Append(targetDiags)

	replaces, replaceDiags := run.GetReplaces()
	diags = diags.Append(replaceDiags)

	references, referenceDiags := run.GetReferences()
	diags = diags.Append(referenceDiags)

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

func (runner *TestFileRunner) Cleanup(file *moduletest.File) {
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)

	if runner.Suite.Cancelled {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return
	}

	// First, we'll clean up the main state.
	main := runner.RelevantStates[MainStateIdentifier]

	var diags tfdiags.Diagnostics
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
			runner.Suite.View.DestroySummary(diags, nil, file, state.State)
			continue
		}

		states = append(states, state)
	}

	slices.SortFunc(states, func(a, b *TestFileState) bool {
		// We want to clean up later run blocks first. So, we'll sort this in
		// reverse according to index. This means larger indices first.
		return a.Run.Index > b.Run.Index
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
		runner.Suite.View.DestroySummary(diags, state.Run, file, updated)

		reset()
	}
}

// buildInputVariablesForTest creates a terraform.InputValues mapping for
// variable values that are relevant to the config being tested.
//
// Crucially, it differs from prepareInputVariablesForAssertions in that it only
// includes variables that are reference by the config and not everything that
// is defined within the test run block and test file.
func (runner *TestFileRunner) buildInputVariablesForTest(run *moduletest.Run, file *moduletest.File, config *configs.Config) (terraform.InputValues, tfdiags.Diagnostics) {
	variables := make(map[string]backend.UnparsedVariableValue)
	for name := range config.Module.Variables {
		if run != nil {
			if expr, exists := run.Config.Variables[name]; exists {
				// Local variables take precedence.
				variables[name] = unparsedTestVariableValue{
					expr: expr,
					ctx:  runner.EvalCtx(),
				}
				continue
			}
		}

		if file != nil {
			if expr, exists := file.Config.Variables[name]; exists {
				// If it's not set locally, it maybe set for the entire file.
				variables[name] = unparsedVariableValueExpression{
					expr:       expr,
					sourceType: terraform.ValueFromConfig,
				}
				continue
			}
		}

		if runner.Suite.GlobalVariables != nil {
			// If it's not set locally or at the file level, maybe it was
			// defined globally.
			if variable, exists := runner.Suite.GlobalVariables[name]; exists {
				variables[name] = variable
			}
		}

		// If it's not set at all that might be okay if the variable is optional
		// so we'll just not add anything to the map.
	}

	return backend.ParseVariableValues(variables, config.Module.Variables)
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
func (runner *TestFileRunner) prepareInputVariablesForAssertions(config *configs.Config, run *moduletest.Run, file *moduletest.File) (terraform.InputValues, func(), tfdiags.Diagnostics) {
	variables := make(map[string]backend.UnparsedVariableValue)

	if run != nil {
		for name, expr := range run.Config.Variables {
			variables[name] = unparsedTestVariableValue{
				expr: expr,
				ctx:  runner.EvalCtx(),
			}
		}
	}

	if file != nil {
		for name, expr := range file.Config.Variables {
			if _, exists := variables[name]; exists {
				// Then this variable was defined at the run level and we want
				// that value to take precedence.
				continue
			}
			variables[name] = unparsedVariableValueExpression{
				expr:       expr,
				sourceType: terraform.ValueFromConfig,
			}
		}
	}

	for name, variable := range runner.Suite.GlobalVariables {
		if _, exists := variables[name]; exists {
			// Then this value was already defined at either the run level
			// or the file level, and we want those values to take
			// precedence.
			continue
		}
		variables[name] = variable
	}

	// We've gathered all the values we have, let's convert them into
	// terraform.InputValues so they can be passed into the Terraform graph.

	inputs := make(terraform.InputValues, len(variables))
	var diags tfdiags.Diagnostics
	for name, variable := range variables {
		value, valueDiags := variable.ParseVariableValue(configs.VariableParseLiteral)
		diags = diags.Append(valueDiags)
		inputs[name] = value
	}

	// Next, we're going to apply any default values from the configuration.
	// We do this after the conversion into terraform.InputValues, as the
	// defaults have already been converted into cty.Value objects.

	for name, variable := range config.Module.Variables {
		if _, exists := variables[name]; exists {
			// Then we don't want to apply the default for this variable as we
			// already have a value.
			continue
		}

		if variable.Default != cty.NilVal {
			inputs[name] = &terraform.InputValue{
				Value:       variable.Default,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		}
	}

	// Finally, we're going to do a some modifications to the config.
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
	for name, value := range inputs {
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

	return inputs, func() {
		config.Module.Variables = currentVars
	}, diags
}

// EvalCtx returns an hcl.EvalContext that allows the variables blocks within
// run blocks to evaluate references to the outputs from other run blocks.
func (runner *TestFileRunner) EvalCtx() *hcl.EvalContext {
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

					if value.Sensitive {
						outputs[output.Name] = value.Value.Mark(marks.Sensitive)
						continue
					}

					outputs[output.Name] = value.Value
				}

				blocks[run] = cty.ObjectVal(outputs)
			}

			return map[string]cty.Value{
				"run": cty.ObjectVal(blocks),
			}
		}(),
	}
}
