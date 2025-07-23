// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"slices"

	"github.com/google/go-dap"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
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
}

func (runner *TestSuiteRunner) Stop() {
	runner.Stopped = true
}

// Debug is a special function that allows the test suite to be run in a
// debug mode. The suite is returned immediately, and the tests are
// executed asynchronously. The suite's scope is used to communicate
// the results of the tests back to the caller. The caller can then
// use the scope to wait for the tests to complete and retrieve the
// results of the tests.
func (runner *TestSuiteRunner) Debug() *graph.DebugContext {
	var diags tfdiags.Diagnostics
	ctx := &graph.DebugContext{
		RunCh:             make(chan *moduletest.Run),
		ErrCh:             make(chan tfdiags.Diagnostics, 1),
		BeforeBreakpoints: make(map[string]dap.Breakpoint),
		Breakpoints:       make(map[string]dap.Breakpoint),
	}

	// TODO: If debug mode does not run tests sequentially, functions like
	// `next` will be non-deterministic.
	runner.Concurrency = 1 // Debug mode always runs tests sequentially.

	// run the test suite in a goroutine and
	// send the results back to the caller via the debug context's RunCh channel.
	go func() {
		suite, suiteDiags := runner.collectTests()
		diags = diags.Append(suiteDiags)
		if suiteDiags.HasErrors() {
			ctx.ErrCh <- diags
			close(ctx.RunCh)
			return
		}
		ctx.Suite = suite

		if diags.HasErrors() {
			ctx.ErrCh <- diags
			close(ctx.RunCh)
			return
		}

		_, diags := runner.test(ctx, suite)
		close(ctx.RunCh)
		if diags.HasErrors() {
			ctx.ErrCh <- diags
		}
	}()

	return ctx
}

func (runner *TestSuiteRunner) IsStopped() bool {
	return runner.Stopped
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
	return runner.test(nil, suite)
}

func (runner *TestSuiteRunner) test(dbgCtx *graph.DebugContext, suite *moduletest.Suite) (moduletest.Status, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	runner.View.Abstract(suite)

	// We have two sets of variables that are available to different test files.
	// Test files in the root directory have access to the GlobalVariables only,
	// while test files in the test directory have access to the union of
	// GlobalVariables and GlobalTestVariables.
	testDirectoryGlobalVariables := make(map[string]backendrun.UnparsedVariableValue)
	maps.Copy(testDirectoryGlobalVariables, runner.GlobalVariables)
	// We're okay to overwrite the global variables in case of name
	// collisions, as the test directory variables should take precedence.
	maps.Copy(testDirectoryGlobalVariables, runner.GlobalTestVariables)

	suite.Status = moduletest.Pending
	for _, name := range slices.Sorted(maps.Keys(suite.Files)) {
		if runner.Cancelled {
			return suite.Status, diags
		}

		file := suite.Files[name]

		// Attach the source code to each run in the suite.
		diags = diags.Append(file.WithSourceCode())
		if diags.HasErrors() {
			return moduletest.Error, diags
		}

		currentGlobalVariables := runner.GlobalVariables
		if filepath.Dir(file.Name) == runner.TestingDirectory {
			// If the file is in the test directory, we'll use the union of the
			// global variables and the global test variables.
			currentGlobalVariables = testDirectoryGlobalVariables
		}

		evalCtx := graph.NewEvalContext(graph.EvalContextOpts{
			Config:            runner.Config,
			CancelCtx:         runner.CancelledCtx,
			StopCtx:           runner.StoppedCtx,
			Verbose:           runner.Verbose,
			Render:            runner.View,
			UnparsedVariables: currentGlobalVariables,
			Concurrency:       runner.Concurrency,
			DebugContext:      dbgCtx,
		})
		evalCtx.File = file

		if dbgCtx != nil {
			// set the current file runner's eval context as the active eval context, so that the caller
			// can resume the test execution within this eval context.
			dbgCtx.ActiveEvalContext = evalCtx

			// Pause immediately if the debugger is active. TODO: Pause outside of here
			evalCtx.Pause(true)
		}

		fileRunner := &TestFileRunner{
			Suite:        runner,
			EvalContext:  evalCtx,
			DebugContext: dbgCtx,
		}

		runner.View.File(file, moduletest.Starting)
		fileRunner.Test(file)
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
	Suite        *TestSuiteRunner
	EvalContext  *graph.EvalContext
	TestSuite    *moduletest.Suite
	DebugContext *graph.DebugContext
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
		return
	}

	// Build the graph for the file.
	b := graph.TestGraphBuilder{
		Config:      runner.Suite.Config,
		File:        file,
		ContextOpts: runner.Suite.Opts,
		DebugMode:   runner.DebugContext != nil,
	}
	g, diags := b.Build()
	file.Diagnostics = file.Diagnostics.Append(diags)
	if walkCancelled := runner.renderPreWalkDiags(file); walkCancelled {
		return
	}

	// walk and execute the graph
	diags = diags.Append(graph.Walk(g, runner.EvalContext))

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

func (runner *TestFileRunner) renderPreWalkDiags(file *moduletest.File) (walkCancelled bool) {
	errored := file.Diagnostics.HasErrors()
	// Some runs may have errored during the graph build, but we didn't fail immediately
	// as we still wanted to gather all the diagnostics.
	// Now we go through the runs and if there are any errors, we'll update the
	// file status to be errored.
	for _, run := range file.Runs {
		if run.Status == moduletest.Error {
			errored = true
			runner.Suite.View.Run(run, file, moduletest.Complete, 0)
		}
	}
	if errored {
		// print a teardown message even though there was no teardown to run
		runner.Suite.View.File(file, moduletest.TearDown)
		file.Status = file.Status.Merge(moduletest.Error)
		return true
	}

	return false
}
