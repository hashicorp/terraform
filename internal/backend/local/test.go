// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
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
		evalCtx := graph.NewEvalContext(&graph.EvalContextOpts{
			CancelCtx: runner.CancelledCtx,
			StopCtx:   runner.StoppedCtx,
			Verbose:   runner.Verbose,
			Render:    runner.View,
		})

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
		return
	}

	// Build the graph for the file.
	b := graph.TestGraphBuilder{
		File:        file,
		GlobalVars:  runner.EvalContext.VariableCaches.GlobalVariables,
		ContextOpts: runner.Suite.Opts,
	}
	g, diags := b.Build()
	file.Diagnostics = file.Diagnostics.Append(diags)
	if walkCancelled := runner.renderPreWalkDiags(file); walkCancelled {
		return
	}

	// walk and execute the graph
	diags = runner.walkGraph(g)

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
func (runner *TestFileRunner) walkGraph(g *terraform.Graph) tfdiags.Diagnostics {
	sem := runner.Suite.semaphore

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

		if executable, ok := v.(graph.GraphNodeExecutable); ok {
			diags = executable.Execute(runner.EvalContext)
		}
		return
	}

	return g.AcyclicGraph.Walk(walkFn)
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
