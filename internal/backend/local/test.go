// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type TestSuiteRunner struct {
	Config *configs.Config

	TestingDirectory string

	// Global variables comes from the main configuration directory,
	// and the Global Test Variables are loaded from the test directory.
	GlobalVariables     map[string]backendrun.UnparsedVariableValue
	GlobalTestVariables map[string]backendrun.UnparsedVariableValue

	Opts *terraform.ContextOpts

	View  views.Test
	JUnit junit.JUnit

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

	Concurrency int
	semaphore   terraform.Semaphore
}

func (runner *TestSuiteRunner) Stop() {
	runner.Stopped = true
}

func (runner *TestSuiteRunner) IsStopped() bool {
	return runner.Stopped
}

func (runner *TestSuiteRunner) Cancel() {
	runner.Cancelled = true
}

func (runner *TestSuiteRunner) Test() (moduletest.Status, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if runner.Concurrency < 1 {
		runner.Concurrency = 10
	}
	runner.semaphore = terraform.NewSemaphore(runner.Concurrency)

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

	// We have two sets of variables that are available to different test files.
	// Test files in the root directory have access to the GlobalVariables only,
	// while test files in the test directory have access to the union of
	// GlobalVariables and GlobalTestVariables.
	testDirectoryGlobalVariables := make(map[string]backendrun.UnparsedVariableValue)
	for name, value := range runner.GlobalVariables {
		testDirectoryGlobalVariables[name] = value
	}
	for name, value := range runner.GlobalTestVariables {
		// We're okay to overwrite the global variables in case of name
		// collisions, as the test directory variables should take precedence.
		testDirectoryGlobalVariables[name] = value
	}

	suite.Status = moduletest.Pass
	for _, name := range files {
		if runner.Cancelled {
			return suite.Status, diags
		}

		file := suite.Files[name]
		// The eval context inherits the cancelled context from the runner.
		// This allows the eval context to stop the graph walk if the runner
		// requests a hard stop.
		evalCtx := graph.NewEvalContext(runner.CancelledCtx)

		for _, run := range file.Runs {
			// Pre-initialise the prior outputs, so we can easily tell between
			// a run block that doesn't exist and a run block that hasn't been
			// executed yet.
			// (moduletest.EvalContext treats cty.NilVal as "not visited yet")
			evalCtx.SetOutput(run, cty.NilVal)
		}

		currentGlobalVariables := runner.GlobalVariables
		if filepath.Dir(file.Name) == runner.TestingDirectory {
			// If the file is in the test directory, we'll use the union of the
			// global variables and the global test variables.
			currentGlobalVariables = testDirectoryGlobalVariables
		}

		evalCtx.VariableCaches = hcltest.NewVariableCaches(func(vc *hcltest.VariableCaches) {
			for name, value := range currentGlobalVariables {
				vc.GlobalVariables[name] = value
			}
			vc.FileVariables = file.Config.Variables
		})
		fileRunner := &TestFileRunner{
			Suite:       runner,
			EvalContext: evalCtx,
		}

		runner.View.File(file, moduletest.Starting)
		fileRunner.Test(file)
		runner.View.File(file, moduletest.TearDown)
		fileRunner.cleanup(file)
		runner.View.File(file, moduletest.Complete)
		suite.Status = suite.Status.Merge(file.Status)
	}

	runner.View.Conclusion(suite)

	if runner.JUnit != nil {
		artifactDiags := runner.JUnit.Save(suite)
		diags = diags.Append(artifactDiags)
		if artifactDiags.HasErrors() {
			return moduletest.Error, diags
		}
	}

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
						config := runner.Config
						if run.ConfigUnderTest != nil {
							config = run.ConfigUnderTest
						}
						runs = append(runs, moduletest.NewRun(run, config, ix))

					}

					runCount += len(runs)
					files[name] = moduletest.NewFile(name, file, runs)
				}

				return files
			}

			// Otherwise, we'll just do all the tests in the directory!
			for name, file := range runner.Config.Module.Tests {
				fileCount++

				var runs []*moduletest.Run
				for ix, run := range file.Runs {
					config := runner.Config
					if run.ConfigUnderTest != nil {
						config = run.ConfigUnderTest
					}
					runs = append(runs, moduletest.NewRun(run, config, ix))
				}

				runCount += len(runs)
				files[name] = moduletest.NewFile(name, file, runs)
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
	Suite       *TestSuiteRunner
	EvalContext *graph.EvalContext
}

func (runner *TestFileRunner) Test(file *moduletest.File) {
	log.Printf("[TRACE] TestFileRunner: executing test file %s", file.Name)

	// The file validation only returns warnings so we'll just add them without
	// checking anything about them.
	file.Diagnostics = file.Diagnostics.Append(file.Config.Validate(runner.Suite.Config))

	// We'll execute the tests in the file. First, mark the overall status as
	// being skipped. This will ensure that if we've cancelled and the files not
	// going to do anything it'll be marked as skipped.
	file.Status = file.Status.Merge(moduletest.Skip)
	if len(file.Runs) == 0 {
		// If we have zero run blocks then we'll just mark the file as passed.
		file.Status = file.Status.Merge(moduletest.Pass)
	}

	// Build the graph for the file.
	b := graph.TestGraphBuilder{File: file, GlobalVars: runner.EvalContext.VariableCaches.GlobalVariables}
	graph, diags := b.Build()
	file.Diagnostics = file.Diagnostics.Append(diags)
	if diags.HasErrors() {
		file.Status = file.Status.Merge(moduletest.Error)
		return
	}

	// walk and execute the graph
	diags = runner.walkGraph(graph, file)

	// If the graph walk was terminated, we don't want to add the diagnostics.
	// The error the user receives will just be:
	// 			Failure! 0 passed, 1 failed.
	// 			exit status 1
	if runner.EvalContext.Cancelled() {
		file.UpdateStatus(moduletest.Error)
		log.Printf("[TRACE] TestFileRunner: graph walk terminated for %s", file.Name)
		return
	}

	file.Diagnostics = file.Diagnostics.Append(diags)
}

// walkGraph goes through the graph and execute each run it finds.
func (runner *TestFileRunner) walkGraph(g *terraform.Graph, file *moduletest.File) tfdiags.Diagnostics {
	sem := runner.Suite.semaphore
	collectRunStatus, updateFileStatus := runner.trackRunStatuses(file)

	// Walk the graph.
	walkFn := func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		if runner.EvalContext.Cancelled() {
			// If the graph walk has been cancelled, the node should just return immediately.
			// For now, this means a hard stop has been requested, in this case we don't
			// even stop to mark future test runs as having been skipped. They'll
			// just show up as pending in the printed summary. We will quickly
			// just mark the overall file status has having errored to indicate
			// it was interrupted.
			return
		}

		// the walkFn is called asynchronously, and needs to be recovered
		// separately in the case of a panic.
		defer logging.PanicHandler()

		log.Printf("[TRACE] vertex %q: starting visit (%T)", dag.VertexName(v), v)

		defer func() {
			if r := recover(); r != nil {
				// If the walkFn panics, we get confusing logs about how the
				// visit was complete. To stop this, we'll catch the panic log
				// that the vertex panicked without finishing and re-panic.
				log.Printf("[ERROR] vertex %q panicked", dag.VertexName(v))
				panic(r) // re-panic
			}

			if diags.HasErrors() {
				for _, diag := range diags {
					if diag.Severity() == tfdiags.Error {
						desc := diag.Description()
						log.Printf("[ERROR] vertex %q error: %s", dag.VertexName(v), desc.Summary)
					}
				}
				log.Printf("[TRACE] vertex %q: visit complete, with errors", dag.VertexName(v))
			} else {
				log.Printf("[TRACE] vertex %q: visit complete", dag.VertexName(v))
			}
		}()

		// Acquire a lock on the semaphore
		sem.Acquire()
		defer sem.Release()

		switch v := v.(type) {
		case *graph.NodeTestRun: // NodeTestRun is also executable, so it has to be first.
			file := v.File()
			run := v.Run()
			if file.GetStatus() == moduletest.Error {
				// If the overall test file has errored, we don't keep trying to
				// execute tests. Instead, we mark all remaining run blocks as
				// skipped, print the status, and move on.
				run.Status = moduletest.Skip
				runner.Suite.View.Run(run, file, moduletest.Complete, 0)
				return
			}
			if runner.Suite.Stopped {
				// Then the test was requested to be stopped, so we just mark each
				// following test as skipped, print the status, and move on.
				run.Status = moduletest.Skip
				runner.Suite.View.Run(run, file, moduletest.Complete, 0)
				return
			}

			// TODO: The execution of a NodeTestRun is currently split between
			// its Execute method and the continuation of the walk callback.
			// Eventually, we should move all the logic related to a test run into
			// its Execute method, effectively ensuring that the Execute method is
			// enough to execute a test run in the graph.
			diags = v.Execute(runner.EvalContext)
			if diags.HasErrors() {
				return diags
			}

			startTime := time.Now().UTC()
			deferFileStatus := runner.run(run, file, startTime)
			runner.Suite.View.Run(run, file, moduletest.Complete, 0)
			// If the run block is done, but it was due to an expected failure, we
			// don't want to update the file status immediately. We'll collect the
			// status of this run block and update the file status at the end of the
			// file execution.
			collectRunStatus(run, deferFileStatus)
		case graph.GraphNodeExecutable:
			diags = v.Execute(runner.EvalContext)
			return diags
		default:
			// If the vertex isn't a test run or executable, we'll just skip it.
			return
		}
		return
	}

	diags := g.AcyclicGraph.Walk(walkFn)
	updateFileStatus()
	return diags
}

func (runner *TestFileRunner) run(run *moduletest.Run, file *moduletest.File, startTime time.Time) (deferFileStatus bool) {
	log.Printf("[TRACE] TestFileRunner: executing run block %s/%s", file.Name, run.Name)
	defer func() {
		// If we got far enough to actually execute the run then we'll give
		// the view some additional metadata about the execution.
		run.ExecutionMeta = &moduletest.RunExecutionMeta{
			Start:    startTime,
			Duration: time.Since(startTime),
		}

	}()

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
			return
		}
	}
	state := runner.EvalContext.GetFileState(key).State

	config := run.ModuleConfig
	if runner.Suite.Cancelled {
		// Don't do anything, just give up and return immediately.
		// The surrounding functions should stop this even being called, but in
		// case of race conditions or something we can still verify this.
		return
	}

	if runner.Suite.Stopped {
		// Basically the same as above, except we'll be a bit nicer.
		run.Status = moduletest.Skip
		return
	}

	start := time.Now().UTC().UnixMilli()
	runner.Suite.View.Run(run, file, moduletest.Starting, 0)

	run.Diagnostics = run.Diagnostics.Append(run.Config.Validate(config))
	if run.Diagnostics.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	configDiags := graph.TransformConfigForTest(runner.EvalContext, run, file)
	run.Diagnostics = run.Diagnostics.Append(configDiags)
	if configDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	validateDiags := runner.validate(run, file, start)
	run.Diagnostics = run.Diagnostics.Append(validateDiags)
	if validateDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	// already validated during static analysis
	references, _ := run.GetReferences()

	variables, variableDiags := runner.GetVariables(run, references, true)
	run.Diagnostics = run.Diagnostics.Append(variableDiags)
	if variableDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	// FilterVariablesToModule only returns warnings, so we don't check the
	// returned diags for errors.
	setVariables, testOnlyVariables, setVariableDiags := runner.FilterVariablesToModule(run, variables)
	run.Diagnostics = run.Diagnostics.Append(setVariableDiags)

	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
	run.Diagnostics = run.Diagnostics.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return
	}

	planScope, plan, planDiags := runner.plan(tfCtx, config, state, run, file, setVariables, references, start)
	if run.Config.Command == configs.PlanTestCommand {
		planDiags = run.ValidateExpectedFailures(planDiags)
		run.Diagnostics = run.Diagnostics.Append(planDiags)
		// Then we want to assess our conditions and diagnostics differently.
		if planDiags.HasErrors() {
			run.Status = moduletest.Error
			return
		}

		runner.AddVariablesToConfig(run, variables)

		if runner.Suite.Verbose {
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
		newStatus, outputVals, moreDiags := runner.EvalContext.EvaluateRun(run, planScope, testOnlyVariables)
		run.Status = newStatus
		run.Diagnostics = run.Diagnostics.Append(moreDiags)

		// Now we've successfully validated this run block, lets add it into
		// our prior run outputs so future run blocks can access it.
		runner.EvalContext.SetOutput(run, outputVals)
		return
	}

	// Otherwise any error (expected or unexpected) during the planning prevents our apply from
	// continuing which is an error.
	planDiags = run.ExplainExpectedFailures(planDiags)
	run.Diagnostics = run.Diagnostics.Append(planDiags)
	if planDiags.HasErrors() {
		run.Status = moduletest.Error
		// If the plan failed, but all the failures were expected, then we don't
		// want to mark the overall file as a failure, so that subsequent runs can
		// still be executed.
		// We will collect the status of this run instead of updating the file status.
		// At the end of the file execution, we will update the file status based on the
		// statuses of all the runs.
		if !run.ValidateExpectedFailures(planDiags).HasErrors() {
			deferFileStatus = true
		}
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

	applyScope, updated, applyDiags := runner.apply(tfCtx, plan, state, run, file, moduletest.Running, start, variables)

	// Remove expected diagnostics, and add diagnostics in case anything that should have failed didn't.
	applyDiags = run.ValidateExpectedFailures(applyDiags)

	run.Diagnostics = run.Diagnostics.Append(applyDiags)
	if applyDiags.HasErrors() {
		run.Status = moduletest.Error
		// Even though the apply operation failed, the graph may have done
		// partial updates and the returned state should reflect this.
		runner.EvalContext.SetFileState(key, &graph.TestFileState{
			Run:   run,
			State: updated,
		})
		return
	}

	runner.AddVariablesToConfig(run, variables)

	if runner.Suite.Verbose {
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
	newStatus, outputVals, moreDiags := runner.EvalContext.EvaluateRun(run, applyScope, testOnlyVariables)
	run.Status = newStatus
	run.Diagnostics = run.Diagnostics.Append(moreDiags)

	// Now we've successfully validated this run block, lets add it into
	// our prior run outputs so future run blocks can access it.
	runner.EvalContext.SetOutput(run, outputVals)

	// Only update the most recent run and state if the state was
	// actually updated by this change. We want to use the run that
	// most recently updated the tracked state as the cleanup
	// configuration.
	runner.EvalContext.SetFileState(key, &graph.TestFileState{
		Run:   run,
		State: updated,
	})
	return
}

func (runner *TestFileRunner) validate(run *moduletest.Run, file *moduletest.File, start int64) tfdiags.Diagnostics {
	log.Printf("[TRACE] TestFileRunner: called validate for %s/%s", file.Name, run.Name)

	var diags tfdiags.Diagnostics
	config := run.ModuleConfig

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
		validateDiags = tfCtx.Validate(config, nil)
		log.Printf("[DEBUG] TestFileRunner: completed validate for  %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, nil, moduletest.Running, start)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(validateDiags)

	return diags
}

func (runner *TestFileRunner) destroy(state *states.State, run *moduletest.Run, file *moduletest.File) (*states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] TestFileRunner: called destroy for %s/%s", file.Name, run.Name)

	if state.Empty() {
		// Nothing to do!
		return state, nil
	}

	var diags tfdiags.Diagnostics

	variables, variableDiags := runner.GetVariables(run, nil, false)
	diags = diags.Append(variableDiags)

	if diags.HasErrors() {
		return state, diags
	}

	// During the destroy operation, we don't add warnings from this operation.
	// Anything that would have been reported here was already reported during
	// the original plan, and a successful destroy operation is the only thing
	// we care about.
	setVariables, _, _ := runner.FilterVariablesToModule(run, variables)

	planOpts := &terraform.PlanOpts{
		Mode:         plans.DestroyMode,
		SetVariables: setVariables,
		Overrides:    mocking.PackageOverrides(run.Config, file.Config, run.ModuleConfig),
	}

	tfCtx, ctxDiags := terraform.NewContext(runner.Suite.Opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return state, diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	start := time.Now().UTC().UnixMilli()
	runner.Suite.View.Run(run, file, moduletest.TearDown, 0)

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestFileRunner: starting destroy plan for %s/%s", file.Name, run.Name)
		plan, planDiags = tfCtx.Plan(run.ModuleConfig, state, planOpts)
		log.Printf("[DEBUG] TestFileRunner: completed destroy plan for %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, nil, moduletest.TearDown, start)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(planDiags)

	if diags.HasErrors() {
		return state, diags
	}

	_, updated, applyDiags := runner.apply(tfCtx, plan, state, run, file, moduletest.TearDown, start, variables)
	diags = diags.Append(applyDiags)
	return updated, diags
}

func (runner *TestFileRunner) plan(tfCtx *terraform.Context, config *configs.Config, state *states.State, run *moduletest.Run, file *moduletest.File, variables terraform.InputValues, references []*addrs.Reference, start int64) (*lang.Scope, *plans.Plan, tfdiags.Diagnostics) {
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
		ExternalReferences: references,
		Overrides:          mocking.PackageOverrides(run.Config, file.Config, config),
	}

	runningCtx, done := context.WithCancel(context.Background())

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	var planScope *lang.Scope
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestFileRunner: starting plan for %s/%s", file.Name, run.Name)
		plan, planScope, planDiags = tfCtx.PlanAndEval(config, state, planOpts)
		log.Printf("[DEBUG] TestFileRunner: completed plan for %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, nil, moduletest.Running, start)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(planDiags)

	return planScope, plan, diags
}

func (runner *TestFileRunner) apply(tfCtx *terraform.Context, plan *plans.Plan, state *states.State, run *moduletest.Run, file *moduletest.File, progress moduletest.Progress, start int64, variables terraform.InputValues) (*lang.Scope, *states.State, tfdiags.Diagnostics) {
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

	runningCtx, done := context.WithCancel(context.Background())

	var updated *states.State
	var applyDiags tfdiags.Diagnostics
	var newScope *lang.Scope

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

	go func() {
		defer logging.PanicHandler()
		defer done()
		log.Printf("[DEBUG] TestFileRunner: starting apply for %s/%s", file.Name, run.Name)
		updated, newScope, applyDiags = tfCtx.ApplyAndEval(plan, config, applyOpts)
		log.Printf("[DEBUG] TestFileRunner: completed apply for %s/%s", file.Name, run.Name)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, run, file, created, progress, start)

	if cancelled {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	diags = diags.Append(waitDiags)
	diags = diags.Append(applyDiags)

	return newScope, updated, diags
}

func (runner *TestFileRunner) wait(ctx *terraform.Context, runningCtx context.Context, run *moduletest.Run, file *moduletest.File, created []*plans.ResourceInstanceChangeSrc, progress moduletest.Progress, start int64) (diags tfdiags.Diagnostics, cancelled bool) {
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

	// Keep track of when the execution is actually finished.
	finished := false

	// This function handles what happens when the user presses the second
	// interrupt. This is a "hard cancel", we are going to stop doing whatever
	// it is we're doing. This means even if we're halfway through creating or
	// destroying infrastructure we just give up.
	handleCancelled := func() {
		log.Printf("[DEBUG] TestFileRunner: test execution cancelled during %s", identifier)

		states := make(map[*moduletest.Run]*states.State)
		mainKey := moduletest.MainStateIdentifier
		states[nil] = runner.EvalContext.GetFileState(mainKey).State
		for key, module := range runner.EvalContext.FileStates {
			if key == mainKey {
				continue
			}
			states[module.Run] = module.State
		}
		runner.Suite.View.FatalInterruptSummary(run, file, states, created)

		cancelled = true
		go ctx.Stop()

		for !finished {
			select {
			case <-time.After(2 * time.Second):
				// Print an update while we're waiting.
				now := time.Now().UTC().UnixMilli()
				runner.Suite.View.Run(run, file, progress, now-start)
			case <-runningCtx.Done():
				// Just wait for things to finish now, the overall test execution will
				// exit early if this takes too long.
				finished = true
			}
		}

	}

	// This function handles what happens when the user presses the first
	// interrupt. This is essentially a "soft cancel", we're not going to do
	// anything but just wait for things to finish safely. But, we do listen
	// for the crucial second interrupt which will prompt a hard stop / cancel.
	handleStopped := func() {
		log.Printf("[DEBUG] TestFileRunner: test execution stopped during %s", identifier)

		for !finished {
			select {
			case <-time.After(2 * time.Second):
				// Print an update while we're waiting.
				now := time.Now().UTC().UnixMilli()
				runner.Suite.View.Run(run, file, progress, now-start)
			case <-runner.Suite.CancelledCtx.Done():
				// We've been asked again. This time we stop whatever we're doing
				// and abandon all attempts to do anything reasonable.
				handleCancelled()
			case <-runningCtx.Done():
				// Do nothing, we finished safely and skipping the remaining tests
				// will be handled elsewhere.
				finished = true
			}
		}

	}

	for !finished {
		select {
		case <-time.After(2 * time.Second):
			// Print an update while we're waiting.
			now := time.Now().UTC().UnixMilli()
			runner.Suite.View.Run(run, file, progress, now-start)
		case <-runner.Suite.StoppedCtx.Done():
			handleStopped()
		case <-runner.Suite.CancelledCtx.Done():
			handleCancelled()
		case <-runningCtx.Done():
			// The operation exited normally.
			finished = true
		}
	}

	return diags, cancelled
}

func (runner *TestFileRunner) cleanup(file *moduletest.File) {
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)

	if runner.Suite.Cancelled {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return
	}

	var states []*graph.TestFileState
	for key, state := range runner.EvalContext.FileStates {

		empty := true
		for _, module := range state.State.Modules {
			for _, resource := range module.Resources {
				if resource.Addr.Resource.Mode == addrs.ManagedResourceMode {
					empty = false
					break
				}
			}
		}

		if empty {
			// The state can be empty for a run block that just executed a plan
			// command, or a run block that only read data sources. We'll just
			// skip empty run blocks.
			continue
		}

		if state.Run == nil {
			log.Printf("[ERROR] TestFileRunner: found inconsistent run block and state file in %s for module %s", file.Name, key)

			// The state can have a nil run block if it only executed a plan
			// command. In which case, we shouldn't have reached here as the
			// state should also have been empty and this will have been skipped
			// above. If we do reach here, then something has gone badly wrong
			// and we can't really recover from it.

			var diags tfdiags.Diagnostics
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Inconsistent state", fmt.Sprintf("Found inconsistent state while cleaning up %s. This is a bug in Terraform - please report it", file.Name)))
			file.UpdateStatus(moduletest.Error)
			runner.Suite.View.DestroySummary(diags, nil, file, state.State)
			continue
		}

		states = append(states, state)
	}

	slices.SortFunc(states, func(a, b *graph.TestFileState) int {
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

		configDiags := graph.TransformConfigForTest(runner.EvalContext, state.Run, file)
		diags = diags.Append(configDiags)

		updated := state.State
		if !diags.HasErrors() {
			var destroyDiags tfdiags.Diagnostics
			updated, destroyDiags = runner.destroy(state.State, state.Run, file)
			diags = diags.Append(destroyDiags)
		}

		if !updated.Empty() {
			// Then we failed to adequately clean up the state, so mark success
			// as false.
			file.UpdateStatus(moduletest.Error)
		}
		runner.Suite.View.DestroySummary(diags, state.Run, file, updated)
	}
}

// GetVariables builds the terraform.InputValues required for the provided run
// block. It pulls the relevant variables (ie. the variables needed for the
// run block) from the total pool of all available variables, and converts them
// into input values.
//
// As a run block can reference variables defined within the file and are not
// actually defined within the configuration, this function actually returns
// more variables than are required by the config. FilterVariablesToConfig
// should be called before trying to use these variables within a Terraform
// plan, apply, or destroy operation.
func (runner *TestFileRunner) GetVariables(run *moduletest.Run, references []*addrs.Reference, includeWarnings bool) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// relevantVariables contains the variables that are of interest to this
	// run block. This is a combination of the variables declared within the
	// configuration for this run block, and the variables referenced by the
	// run block assertions.
	relevantVariables := make(map[string]bool)

	// First, we'll check to see which variables the run block assertions
	// reference.
	for _, reference := range references {
		if addr, ok := reference.Subject.(addrs.InputVariable); ok {
			relevantVariables[addr.Name] = true
		}
	}

	// And check to see which variables the run block configuration references.
	for name := range run.ModuleConfig.Module.Variables {
		relevantVariables[name] = true
	}

	// We'll put the parsed values into this map.
	values := make(terraform.InputValues)

	// First, let's step through the expressions within the run block and work
	// them out.
	for name, expr := range run.Config.Variables {
		requiredValues := make(map[string]cty.Value)

		refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				cache := runner.EvalContext.GetCache(run)

				value, valueDiags := cache.GetFileVariable(addr.Name)
				diags = diags.Append(valueDiags)
				if value != nil {
					requiredValues[addr.Name] = value.Value
					continue
				}

				// Otherwise, it might be a global variable.
				value, valueDiags = cache.GetGlobalVariable(addr.Name)
				diags = diags.Append(valueDiags)
				if value != nil {
					requiredValues[addr.Name] = value.Value
					continue
				}
			}
		}
		diags = diags.Append(refDiags)

		ctx, ctxDiags := hcltest.EvalContext(hcltest.TargetRunBlock, map[string]hcl.Expression{name: expr}, requiredValues, runner.EvalContext.GetOutputs())
		diags = diags.Append(ctxDiags)

		value := cty.DynamicVal
		if !ctxDiags.HasErrors() {
			var valueDiags hcl.Diagnostics
			value, valueDiags = expr.Value(ctx)
			diags = diags.Append(valueDiags)
		}

		// We do this late on so we still validate whatever it was that the user
		// wrote in the variable expression. But, we don't want to actually use
		// it if it's not actually relevant.
		if _, exists := relevantVariables[name]; !exists {
			// Do not display warnings during cleanup phase
			if includeWarnings {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Value for undeclared variable",
					Detail:   fmt.Sprintf("The module under test does not declare a variable named %q, but it is declared in run block %q.", name, run.Name),
					Subject:  expr.Range().Ptr(),
				})
			}
			continue // Don't add it to our final set of variables.
		}

		values[name] = &terraform.InputValue{
			Value:       value,
			SourceType:  terraform.ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(expr.Range()),
		}
	}

	for variable := range relevantVariables {
		if _, exists := values[variable]; exists {
			// Then we've already got a value for this variable.
			continue
		}

		// Otherwise, we'll get it from the cache as a file-level or global
		// variable.
		cache := runner.EvalContext.GetCache(run)

		value, valueDiags := cache.GetFileVariable(variable)
		diags = diags.Append(valueDiags)
		if value != nil {
			values[variable] = value
			continue
		}

		value, valueDiags = cache.GetGlobalVariable(variable)
		diags = diags.Append(valueDiags)
		if value != nil {
			values[variable] = value
			continue
		}
	}

	// Finally, we check the configuration again. This is where we'll discover
	// if there's any missing variables and fill in any optional variables that
	// don't have a value already.

	for name, variable := range run.ModuleConfig.Module.Variables {
		if _, exists := values[name]; exists {
			// Then we've provided a variable for this. It's all good.
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

			values[name] = &terraform.InputValue{
				Value:       cty.DynamicVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		} else {
			values[name] = &terraform.InputValue{
				Value:       cty.NilVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		}
	}

	return values, diags
}

// FilterVariablesToModule splits the provided values into two disjoint maps:
// moduleVars contains the ones that correspond with declarations in the root
// module of the given configuration, while testOnlyVars contains any others
// that are presumably intended only for use in the test configuration file.
//
// This function is essentially the opposite of AddVariablesToConfig which
// makes the config match the variables rather than the variables match the
// config.
//
// This function can only return warnings, and the callers can rely on this so
// please check the callers of this function if you add any error diagnostics.
func (runner *TestFileRunner) FilterVariablesToModule(run *moduletest.Run, values terraform.InputValues) (moduleVars, testOnlyVars terraform.InputValues, diags tfdiags.Diagnostics) {
	moduleVars = make(terraform.InputValues)
	testOnlyVars = make(terraform.InputValues)
	for name, value := range values {
		_, exists := run.ModuleConfig.Module.Variables[name]
		if !exists {
			// If it's not in the configuration then it's a test-only variable.
			testOnlyVars[name] = value
			continue
		}

		moduleVars[name] = value
	}
	return moduleVars, testOnlyVars, diags
}

// AddVariablesToConfig extends the provided config to ensure it has definitions
// for all specified variables.
//
// This function is essentially the opposite of FilterVariablesToConfig which
// makes the variables match the config rather than the config match the
// variables.
func (runner *TestFileRunner) AddVariablesToConfig(run *moduletest.Run, variables terraform.InputValues) {

	// If we have got variable values from the test file we need to make sure
	// they have an equivalent entry in the configuration. We're going to do
	// that dynamically here.

	// First, take a backup of the existing configuration so we can easily
	// restore it later.
	currentVars := make(map[string]*configs.Variable)
	for name, variable := range run.ModuleConfig.Module.Variables {
		currentVars[name] = variable
	}

	for name, value := range variables {
		if _, exists := run.ModuleConfig.Module.Variables[name]; exists {
			continue
		}

		run.ModuleConfig.Module.Variables[name] = &configs.Variable{
			Name:           name,
			Type:           value.Value.Type(),
			ConstraintType: value.Value.Type(),
			DeclRange:      value.SourceRange.ToHCL(),
		}
	}

}

// trackRunStatuses is a helper function that returns two functions. The first
// function is used to collect the statuses of the runs, and the second function
// is used to update the overall file status based on the statuses of the runs.
func (runner *TestFileRunner) trackRunStatuses(file *moduletest.File) (func(*moduletest.Run, bool), func()) {
	statuses := make([]moduletest.Status, len(file.Runs))
	collector := func(run *moduletest.Run, deferred bool) {
		if deferred {
			statuses[run.Index] = run.Status
		} else {
			file.UpdateStatus(run.Status)
		}
	}

	updater := func() {
		for _, status := range statuses {
			file.UpdateStatus(status)
		}
	}

	// We'll return two functions, one to collect the statuses of the runs, and
	// one to update the overall file status.
	return collector, updater
}
