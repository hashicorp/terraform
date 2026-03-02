// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"slices"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
	teststates "github.com/hashicorp/terraform/internal/moduletest/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type TestSuiteRunner struct {
	Config *configs.Config

	// BackendFactory is used to enable initializing multiple backend types,
	// depending on which backends are used in a test suite.
	//
	// Note: This is currently necessary because the source of the init functions,
	// the backend/init package, experiences import cycles if used in other test-related
	// packages. We set this field on a TestSuiteRunner when making runners in the
	// command package, which is the main place where backend/init has previously been used.
	BackendFactory func(string) backend.InitFn

	TestingDirectory string

	// Global variables comes from the main configuration directory,
	// and the Global Test Variables are loaded from the test directory.
	GlobalVariables     map[string]arguments.UnparsedVariableValue
	GlobalTestVariables map[string]arguments.UnparsedVariableValue

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

	Concurrency     int
	DeferralAllowed bool

	CommandMode moduletest.CommandMode

	// Strict tells the runner to fail tests that produce warnings.
	Strict bool

	// Repair is used to indicate whether the test cleanup command should run in
	// "repair" mode. In this mode, the cleanup command will only remove state
	// files that are a result of failed destroy operations, leaving any
	// state due to skip_cleanup in place.
	Repair bool
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

func (runner *TestSuiteRunner) Test(experimentsAllowed bool) (moduletest.Status, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if runner.Concurrency < 1 {
		runner.Concurrency = 10
	}

	suite, suiteDiags := runner.collectTests()
	diags = diags.Append(suiteDiags)
	if suiteDiags.HasErrors() {
		return moduletest.Error, diags
	}

	manifest, err := teststates.LoadManifest(".", experimentsAllowed)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to open state manifest",
			fmt.Sprintf("The test state manifest file could not be opened: %s.", err)))
	}

	runner.View.Abstract(suite)

	// We have two sets of variables that are available to different test files.
	// Test files in the root directory have access to the GlobalVariables only,
	// while test files in the test directory have access to the union of
	// GlobalVariables and GlobalTestVariables.
	testDirectoryGlobalVariables := make(map[string]arguments.UnparsedVariableValue)
	maps.Copy(testDirectoryGlobalVariables, runner.GlobalVariables)
	// We're okay to overwrite the global variables in case of name
	// collisions, as the test directory variables should take precedence.
	maps.Copy(testDirectoryGlobalVariables, runner.GlobalTestVariables)

	suite.Status = moduletest.Pass
	for _, name := range slices.Sorted(maps.Keys(suite.Files)) {
		if runner.Cancelled {
			return moduletest.Error, diags
		}
		file := suite.Files[name]
		fileRunner := &TestFileRunner{
			Suite:                        runner,
			TestDirectoryGlobalVariables: testDirectoryGlobalVariables,
			Manifest:                     manifest,
		}
		runner.View.File(file, moduletest.Starting)
		fileRunner.Test(file)
		runner.View.File(file, moduletest.Complete)
		suite.Status = suite.Status.Merge(file.Status)
	}

	if err := manifest.Save(experimentsAllowed); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to save state manifest",
			fmt.Sprintf("The test state manifest file could not be saved: %s.", err)))
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
		Status:      moduletest.Pending,
		CommandMode: runner.CommandMode,
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
	Suite                        *TestSuiteRunner
	TestDirectoryGlobalVariables map[string]arguments.UnparsedVariableValue
	Manifest                     *teststates.TestManifest
}

func (runner *TestFileRunner) Test(file *moduletest.File) {
	log.Printf("[TRACE] TestFileRunner: executing test file %s", file.Name)

	// The file validation only returns warnings so we'll just add them without
	// checking anything about them.
	file.Diagnostics = file.Diagnostics.Append(file.Config.Validate(runner.Suite.Config))

	states, stateDiags := runner.Manifest.LoadStates(file, runner.Suite.BackendFactory)
	file.Diagnostics = file.Diagnostics.Append(stateDiags)
	if stateDiags.HasErrors() {
		file.Status = moduletest.Error
	}

	if runner.Suite.CommandMode != moduletest.CleanupMode {
		// then we can't have any state files pending cleanup
		for _, state := range states {
			if state.Manifest.Reason != teststates.StateReasonNone {
				file.Diagnostics = file.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"State manifest not empty",
					fmt.Sprintf("The state manifest for %s should be empty before running tests. This could be due to a previous test run not cleaning up after itself. Please ensure that all state files are cleaned up before running tests.", file.Name)))
				file.Status = moduletest.Error
			}
		}
	}

	// We'll execute the tests in the file. First, mark the overall status as
	// being skipped. This will ensure that if we've cancelled and the files not
	// going to do anything it'll be marked as skipped.
	file.Status = file.Status.Merge(moduletest.Skip)
	if len(file.Runs) == 0 {
		// If we have zero run blocks then we'll just mark the file as passed.
		file.Status = file.Status.Merge(moduletest.Pass)
		return
	} else if runner.Suite.CommandMode == moduletest.CleanupMode {
		// In cleanup mode, we don't actually execute the run blocks so we'll
		// start with the assumption they have all passed.
		file.Status = file.Status.Merge(moduletest.Pass)
	}

	currentGlobalVariables := runner.Suite.GlobalVariables
	if filepath.Dir(file.Name) == runner.Suite.TestingDirectory {
		// If the file is in the test directory, we'll use the union of the
		// global variables and the global test variables.
		currentGlobalVariables = runner.TestDirectoryGlobalVariables
	}

	evalCtx := graph.NewEvalContext(graph.EvalContextOpts{
		Config:            runner.Suite.Config,
		CancelCtx:         runner.Suite.CancelledCtx,
		StopCtx:           runner.Suite.StoppedCtx,
		Verbose:           runner.Suite.Verbose,
		Strict:            runner.Suite.Strict,
		Render:            runner.Suite.View,
		UnparsedVariables: currentGlobalVariables,
		FileStates:        states,
		Concurrency:       runner.Suite.Concurrency,
		DeferralAllowed:   runner.Suite.DeferralAllowed,
		Mode:              runner.Suite.CommandMode,
		Repair:            runner.Suite.Repair,
	})

	// Build the graph for the file.
	b := graph.TestGraphBuilder{
		Config:      runner.Suite.Config,
		File:        file,
		ContextOpts: runner.Suite.Opts,
		CommandMode: runner.Suite.CommandMode,
	}
	g, diags := b.Build()
	file.Diagnostics = file.Diagnostics.Append(diags)
	if walkCancelled := runner.renderPreWalkDiags(file); walkCancelled {
		return
	}

	// walk and execute the graph
	diags = diags.Append(graph.Walk(g, evalCtx))

	// save any dangling state files. we'll check all the states we have in
	// memory, and if any are skipped or errored it means we might want to do
	// a cleanup command in the future. this means we need to save the other
	// state files as dependencies in case they are needed during the cleanup.

	saveDependencies := false
	for _, state := range states {
		if state.Manifest.Reason == teststates.StateReasonSkip || state.Manifest.Reason == teststates.StateReasonError {
			saveDependencies = true // at least one state file does have resources left over
			break
		}
	}
	if saveDependencies {
		for _, state := range states {
			if state.Manifest.Reason == teststates.StateReasonNone {
				// any states that have no reason to be saved, will be updated
				// to the dependency reason and this will tell the manifest to
				// save those state files as well.
				state.Manifest.Reason = teststates.StateReasonDep
			}
		}
	}
	diags = diags.Append(runner.Manifest.SaveStates(file, states))

	// If the graph walk was terminated, we don't want to add the diagnostics.
	// The error the user receives will just be:
	// 			Failure! 0 passed, 1 failed.
	// 			exit status 1
	if evalCtx.Cancelled() {
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
