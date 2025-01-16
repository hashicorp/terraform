// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Test renders outputs for test executions.
type Test interface {
	// Abstract should print an early summary of the tests that will be
	// executed. This will be called before the tests have been executed so
	// the status for everything within suite will be test.Pending.
	//
	// This should be used to state what is going to be tested.
	Abstract(suite *moduletest.Suite)

	// Conclusion should print out a summary of the tests including their
	// completed status.
	Conclusion(suite *moduletest.Suite)

	// File prints out the summary for an entire test file.
	File(file *moduletest.File, progress moduletest.Progress)

	// Run prints out the summary for a single test run block.
	Run(run *moduletest.Run, file *moduletest.File, progress moduletest.Progress, elapsed int64)

	// DestroySummary prints out the summary of the destroy step of each test
	// file. If everything goes well, this should be empty.
	DestroySummary(diags tfdiags.Diagnostics, run *moduletest.Run, file *moduletest.File, state *states.State)

	// Diagnostics prints out the provided diagnostics.
	Diagnostics(run *moduletest.Run, file *moduletest.File, diags tfdiags.Diagnostics)

	// Interrupted prints out a message stating that an interrupt has been
	// received and testing will stop.
	Interrupted()

	// FatalInterrupt prints out a message stating that a hard interrupt has
	// been received and testing will stop and cleanup will be skipped.
	FatalInterrupt()

	// FatalInterruptSummary prints out the resources that were held in state
	// and were being created at the time the FatalInterrupt was received.
	//
	// This will typically be called in place of DestroySummary, as there is no
	// guarantee that this function will be called during a FatalInterrupt. In
	// addition, this function prints additional details about the current
	// operation alongside the current state as the state will be missing newly
	// created resources that also need to be handled manually.
	FatalInterruptSummary(run *moduletest.Run, file *moduletest.File, states map[*moduletest.Run]*states.State, created []*plans.ResourceInstanceChangeSrc)

	// TFCStatusUpdate prints a reassuring update, letting users know the latest
	// status of their ongoing remote test run.
	TFCStatusUpdate(status tfe.TestRunStatus, elapsed time.Duration)

	// TFCRetryHook prints an update if a request failed and is being retried.
	TFCRetryHook(attemptNum int, resp *http.Response)
}

func NewTest(vt arguments.ViewType, view *View) Test {
	switch vt {
	case arguments.ViewJSON:
		return &TestJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &TestHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type TestHuman struct {
	CloudHooks

	view *View
}

var _ Test = (*TestHuman)(nil)

func (t *TestHuman) Abstract(_ *moduletest.Suite) {
	// Do nothing, we don't print an abstract for the human view.
}

func (t *TestHuman) Conclusion(suite *moduletest.Suite) {
	t.view.streams.Println()

	counts := make(map[moduletest.Status]int)
	for _, file := range suite.Files {
		for _, run := range file.Runs {
			count := counts[run.Status]
			counts[run.Status] = count + 1
		}
	}

	if suite.Status <= moduletest.Skip {
		// Then no tests.
		t.view.streams.Print("Executed 0 tests")
		if counts[moduletest.Skip] > 0 {
			t.view.streams.Printf(", %d skipped.\n", counts[moduletest.Skip])
		} else {
			t.view.streams.Println(".")
		}
		return
	}

	if suite.Status == moduletest.Pass {
		t.view.streams.Print(t.view.colorize.Color("[green]Success![reset]"))
	} else {
		t.view.streams.Print(t.view.colorize.Color("[red]Failure![reset]"))
	}

	t.view.streams.Printf(" %d passed, %d failed", counts[moduletest.Pass], counts[moduletest.Fail]+counts[moduletest.Error])
	if counts[moduletest.Skip] > 0 {
		t.view.streams.Printf(", %d skipped.\n", counts[moduletest.Skip])
	} else {
		t.view.streams.Println(".")
	}
}

func (t *TestHuman) File(file *moduletest.File, progress moduletest.Progress) {
	switch progress {
	case moduletest.Starting, moduletest.Running:
		t.view.streams.Printf(t.view.colorize.Color("%s... [light_gray]in progress[reset]\n"), file.Name)
	case moduletest.TearDown:
		t.view.streams.Printf(t.view.colorize.Color("%s... [light_gray]tearing down[reset]\n"), file.Name)
	case moduletest.Complete:
		t.view.streams.Printf("%s... %s\n", file.Name, colorizeTestStatus(file.Status, t.view.colorize))
		t.Diagnostics(nil, file, file.Diagnostics)
	default:
		panic("unrecognized test progress: " + progress.String())
	}
}

func (t *TestHuman) Run(run *moduletest.Run, file *moduletest.File, progress moduletest.Progress, _ int64) {
	switch progress {
	case moduletest.Starting, moduletest.Running, moduletest.TearDown:
		return // We don't print progress updates in human mode
	case moduletest.Complete:
		// Do nothing, the rest of the function handles this.
	default:
		panic("unrecognized test progress: " + progress.String())
	}

	t.view.streams.Printf("  run %q... %s\n", run.Name, colorizeTestStatus(run.Status, t.view.colorize))

	if run.Verbose != nil {
		// We're going to be more verbose about what we print, here's the plan
		// or the state depending on the type of run we did.

		schemas := &terraform.Schemas{
			Providers:    run.Verbose.Providers,
			Provisioners: run.Verbose.Provisioners,
		}

		renderer := jsonformat.Renderer{
			Streams:             t.view.streams,
			Colorize:            t.view.colorize,
			RunningInAutomation: t.view.runningInAutomation,
		}

		if run.Config.Command == configs.ApplyTestCommand {
			// Then we'll print the state.
			root, outputs, err := jsonstate.MarshalForRenderer(statefile.New(run.Verbose.State, file.Name, uint64(run.Index)), schemas)
			if err != nil {
				run.Diagnostics = run.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to render test state",
					fmt.Sprintf("Terraform could not marshal the state for display: %v", err)))
			} else {
				state := jsonformat.State{
					StateFormatVersion:    jsonstate.FormatVersion,
					ProviderFormatVersion: jsonprovider.FormatVersion,
					RootModule:            root,
					RootModuleOutputs:     outputs,
					ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
				}

				t.view.streams.Println() // Separate the state from any previous statements.
				renderer.RenderHumanState(state)
				t.view.streams.Println() // Separate the state from any future statements.
			}
		} else {
			// We'll print the plan.
			outputs, changed, drift, attrs, err := jsonplan.MarshalForRenderer(run.Verbose.Plan, schemas)
			if err != nil {
				run.Diagnostics = run.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to render test plan",
					fmt.Sprintf("Terraform could not marshal the plan for display: %v", err)))
			} else {
				plan := jsonformat.Plan{
					PlanFormatVersion:     jsonplan.FormatVersion,
					ProviderFormatVersion: jsonprovider.FormatVersion,
					OutputChanges:         outputs,
					ResourceChanges:       changed,
					ResourceDrift:         drift,
					ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
					RelevantAttributes:    attrs,
				}

				var opts []plans.Quality
				if run.Verbose.Plan.Errored {
					opts = append(opts, plans.Errored)
				} else if !run.Verbose.Plan.Applyable {
					opts = append(opts, plans.NoChanges)
				}

				renderer.RenderHumanPlan(plan, run.Verbose.Plan.UIMode, opts...)
				t.view.streams.Println() // Separate the plan from any future statements.
			}
		}
	}

	// Finally we'll print out a summary of the diagnostics from the run.
	t.Diagnostics(run, file, run.Diagnostics)

	var warnings bool
	for _, diag := range run.Diagnostics {
		switch diag.Severity() {
		case tfdiags.Error:
			// do nothing
		case tfdiags.Warning:
			warnings = true
		}

		if warnings {
			// We only care about checking if we printed any warnings in the
			// previous output.
			break
		}
	}

	if warnings {
		// warnings are printed to stdout, so we'll put a new line into stdout
		// to separate any future statements info statements.
		t.view.streams.Println()
	}
}

func (t *TestHuman) DestroySummary(diags tfdiags.Diagnostics, run *moduletest.Run, file *moduletest.File, state *states.State) {
	identifier := file.Name
	if run != nil {
		identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
	}

	if diags.HasErrors() {
		t.view.streams.Eprint(format.WordWrap(fmt.Sprintf("Terraform encountered an error destroying resources created while executing %s.\n", identifier), t.view.errorColumns()))
	}
	t.Diagnostics(run, file, diags)

	if state.HasManagedResourceInstanceObjects() {
		// FIXME: This message says "resources" but this is actually a list
		// of resource instance objects.
		t.view.streams.Eprint(format.WordWrap(fmt.Sprintf("\nTerraform left the following resources in state after executing %s, and they need to be cleaned up manually:\n", identifier), t.view.errorColumns()))
		for _, resource := range addrs.SetSortedNatural(state.AllManagedResourceInstanceObjectAddrs()) {
			if resource.DeposedKey != states.NotDeposed {
				t.view.streams.Eprintf("  - %s (%s)\n", resource.ResourceInstance, resource.DeposedKey)
				continue
			}
			t.view.streams.Eprintf("  - %s\n", resource.ResourceInstance)
		}
	}
}

func (t *TestHuman) Diagnostics(_ *moduletest.Run, _ *moduletest.File, diags tfdiags.Diagnostics) {
	t.view.Diagnostics(diags)
}

func (t *TestHuman) Interrupted() {
	t.view.streams.Eprintln(format.WordWrap(interrupted, t.view.errorColumns()))
}

func (t *TestHuman) FatalInterrupt() {
	t.view.streams.Eprintln(format.WordWrap(fatalInterrupt, t.view.errorColumns()))
}

func (t *TestHuman) FatalInterruptSummary(run *moduletest.Run, file *moduletest.File, existingStates map[*moduletest.Run]*states.State, created []*plans.ResourceInstanceChangeSrc) {
	t.view.streams.Eprint(format.WordWrap(fmt.Sprintf("\nTerraform was interrupted while executing %s, and may not have performed the expected cleanup operations.\n", file.Name), t.view.errorColumns()))

	// Print out the main state first, this is the state that isn't associated
	// with a run block.
	if state, exists := existingStates[nil]; exists && !state.Empty() {
		t.view.streams.Eprint(format.WordWrap("\nTerraform has already created the following resources from the module under test:\n", t.view.errorColumns()))
		for _, resource := range addrs.SetSortedNatural(state.AllManagedResourceInstanceObjectAddrs()) {
			if resource.DeposedKey != states.NotDeposed {
				t.view.streams.Eprintf("  - %s (%s)\n", resource.ResourceInstance, resource.DeposedKey)
				continue
			}
			t.view.streams.Eprintf("  - %s\n", resource.ResourceInstance)
		}
	}

	// Then print out the other states in order.
	for _, run := range file.Runs {
		state, exists := existingStates[run]
		if !exists || state.Empty() {
			continue
		}

		t.view.streams.Eprint(format.WordWrap(fmt.Sprintf("\nTerraform has already created the following resources for %q from %q:\n", run.Name, run.Config.Module.Source), t.view.errorColumns()))
		for _, resource := range addrs.SetSortedNatural(state.AllManagedResourceInstanceObjectAddrs()) {
			if resource.DeposedKey != states.NotDeposed {
				t.view.streams.Eprintf("  - %s (%s)\n", resource.ResourceInstance, resource.DeposedKey)
				continue
			}
			t.view.streams.Eprintf("  - %s\n", resource.ResourceInstance)
		}
	}

	if len(created) == 0 {
		// No planned changes, so we won't print anything.
		return
	}

	var resources []string
	for _, change := range created {
		resources = append(resources, change.Addr.String())
	}

	if len(resources) > 0 {
		module := "the module under test"
		if run.Config.ConfigUnderTest != nil {
			module = fmt.Sprintf("%q", run.Config.Module.Source.String())
		}

		t.view.streams.Eprint(format.WordWrap(fmt.Sprintf("\nTerraform was in the process of creating the following resources for %q from %s, and they may not have been destroyed:\n", run.Name, module), t.view.errorColumns()))
		for _, resource := range resources {
			t.view.streams.Eprintf("  - %s\n", resource)
		}
	}
}

func (t *TestHuman) TFCStatusUpdate(status tfe.TestRunStatus, elapsed time.Duration) {
	switch status {
	case tfe.TestRunQueued:
		t.view.streams.Printf("Waiting for the tests to start... (%s elapsed)\n", elapsed.Truncate(30*time.Second))
	case tfe.TestRunRunning:
		t.view.streams.Printf("Waiting for the tests to complete... (%s elapsed)\n", elapsed.Truncate(30*time.Second))
	}
}

func (t *TestHuman) TFCRetryHook(attemptNum int, resp *http.Response) {
	t.view.streams.Println(t.view.colorize.Color(t.RetryLogHook(attemptNum, resp, true)))
}

type TestJSON struct {
	CloudHooks

	view *JSONView
}

var _ Test = (*TestJSON)(nil)

func (t *TestJSON) Abstract(suite *moduletest.Suite) {
	var fileCount, runCount int

	abstract := json.TestSuiteAbstract{}
	for name, file := range suite.Files {
		fileCount++
		var runs []string
		for _, run := range file.Runs {
			runCount++
			runs = append(runs, run.Name)
		}
		abstract[name] = runs
	}

	files := "files"
	runs := "run blocks"

	if fileCount == 1 {
		files = "file"
	}

	if runCount == 1 {
		runs = "run block"
	}

	t.view.log.Info(
		fmt.Sprintf("Found %d %s and %d %s", fileCount, files, runCount, runs),
		"type", json.MessageTestAbstract,
		json.MessageTestAbstract, abstract)
}

func (t *TestJSON) Conclusion(suite *moduletest.Suite) {
	summary := json.TestSuiteSummary{
		Status: json.ToTestStatus(suite.Status),
	}
	for _, file := range suite.Files {
		for _, run := range file.Runs {
			switch run.Status {
			case moduletest.Skip:
				summary.Skipped++
			case moduletest.Pass:
				summary.Passed++
			case moduletest.Error:
				summary.Errored++
			case moduletest.Fail:
				summary.Failed++
			}
		}
	}

	var message bytes.Buffer
	if suite.Status <= moduletest.Skip {
		// Then no tests.
		message.WriteString("Executed 0 tests")
		if summary.Skipped > 0 {
			message.WriteString(fmt.Sprintf(", %d skipped.", summary.Skipped))
		} else {
			message.WriteString(".")
		}
	} else {
		if suite.Status == moduletest.Pass {
			message.WriteString("Success!")
		} else {
			message.WriteString("Failure!")
		}

		message.WriteString(fmt.Sprintf(" %d passed, %d failed", summary.Passed, summary.Failed+summary.Errored))
		if summary.Skipped > 0 {
			message.WriteString(fmt.Sprintf(", %d skipped.", summary.Skipped))
		} else {
			message.WriteString(".")
		}
	}

	t.view.log.Info(
		message.String(),
		"type", json.MessageTestSummary,
		json.MessageTestSummary, summary)
}

func (t *TestJSON) File(file *moduletest.File, progress moduletest.Progress) {
	switch progress {
	case moduletest.Starting, moduletest.Running:
		t.view.log.Info(
			fmt.Sprintf("%s... in progress", file.Name),
			"type", json.MessageTestFile,
			json.MessageTestFile, json.TestFileStatus{
				Path:     file.Name,
				Progress: json.ToTestProgress(moduletest.Starting),
			},
			"@testfile", file.Name)
	case moduletest.TearDown:
		t.view.log.Info(
			fmt.Sprintf("%s... tearing down", file.Name),
			"type", json.MessageTestFile,
			json.MessageTestFile, json.TestFileStatus{
				Path:     file.Name,
				Progress: json.ToTestProgress(moduletest.TearDown),
			},
			"@testfile", file.Name)
	case moduletest.Complete:
		t.view.log.Info(
			fmt.Sprintf("%s... %s", file.Name, testStatus(file.Status)),
			"type", json.MessageTestFile,
			json.MessageTestFile, json.TestFileStatus{
				Path:     file.Name,
				Progress: json.ToTestProgress(moduletest.Complete),
				Status:   json.ToTestStatus(file.Status),
			},
			"@testfile", file.Name)
		t.Diagnostics(nil, file, file.Diagnostics)
	default:
		panic("unrecognized test progress: " + progress.String())
	}
}

func (t *TestJSON) Run(run *moduletest.Run, file *moduletest.File, progress moduletest.Progress, elapsed int64) {
	switch progress {
	case moduletest.Starting, moduletest.Running:
		t.view.log.Info(
			fmt.Sprintf("  %q... in progress", run.Name),
			"type", json.MessageTestRun,
			json.MessageTestRun, json.TestRunStatus{
				Path:     file.Name,
				Run:      run.Name,
				Progress: json.ToTestProgress(progress),
				Elapsed:  &elapsed,
			},
			"@testfile", file.Name,
			"@testrun", run.Name)
		return
	case moduletest.TearDown:
		t.view.log.Info(
			fmt.Sprintf("  %q... tearing down", run.Name),
			"type", json.MessageTestRun,
			json.MessageTestRun, json.TestRunStatus{
				Path:     file.Name,
				Run:      run.Name,
				Progress: json.ToTestProgress(progress),
				Elapsed:  &elapsed,
			},
			"@testfile", file.Name,
			"@testrun", run.Name)
		return
	case moduletest.Complete:
		// Do nothing, the rest of the function handles this case.
	default:
		panic("unrecognized test progress: " + progress.String())
	}

	t.view.log.Info(
		fmt.Sprintf("  %q... %s", run.Name, testStatus(run.Status)),
		"type", json.MessageTestRun,
		json.MessageTestRun, json.TestRunStatus{
			Path:     file.Name,
			Run:      run.Name,
			Progress: json.ToTestProgress(progress),
			Status:   json.ToTestStatus(run.Status)},
		"@testfile", file.Name,
		"@testrun", run.Name)

	if run.Verbose != nil {

		schemas := &terraform.Schemas{
			Providers:    run.Verbose.Providers,
			Provisioners: run.Verbose.Provisioners,
		}

		if run.Config.Command == configs.ApplyTestCommand {
			// Then we'll print the state.
			root, outputs, err := jsonstate.MarshalForRenderer(statefile.New(run.Verbose.State, file.Name, uint64(run.Index)), schemas)
			if err != nil {
				run.Diagnostics = run.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to render test state",
					fmt.Sprintf("Terraform could not marshal the state for display: %v", err)))
			} else {
				state := jsonformat.State{
					StateFormatVersion:    jsonstate.FormatVersion,
					ProviderFormatVersion: jsonprovider.FormatVersion,
					RootModule:            root,
					RootModuleOutputs:     outputs,
					ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
				}

				t.view.log.Info(
					"-verbose flag enabled, printing state",
					"type", json.MessageTestState,
					json.MessageTestState, state,
					"@testfile", file.Name,
					"@testrun", run.Name)
			}
		} else {
			outputs, changed, drift, attrs, err := jsonplan.MarshalForRenderer(run.Verbose.Plan, schemas)
			if err != nil {
				run.Diagnostics = run.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to render test plan",
					fmt.Sprintf("Terraform could not marshal the plan for display: %v", err)))
			} else {
				plan := jsonformat.Plan{
					PlanFormatVersion:     jsonplan.FormatVersion,
					ProviderFormatVersion: jsonprovider.FormatVersion,
					OutputChanges:         outputs,
					ResourceChanges:       changed,
					ResourceDrift:         drift,
					ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
					RelevantAttributes:    attrs,
				}

				t.view.log.Info(
					"-verbose flag enabled, printing plan",
					"type", json.MessageTestPlan,
					json.MessageTestPlan, plan,
					"@testfile", file.Name,
					"@testrun", run.Name)
			}
		}
	}

	t.Diagnostics(run, file, run.Diagnostics)
}

func (t *TestJSON) DestroySummary(diags tfdiags.Diagnostics, run *moduletest.Run, file *moduletest.File, state *states.State) {
	if state.HasManagedResourceInstanceObjects() {
		cleanup := json.TestFileCleanup{}
		for _, resource := range addrs.SetSortedNatural(state.AllManagedResourceInstanceObjectAddrs()) {
			cleanup.FailedResources = append(cleanup.FailedResources, json.TestFailedResource{
				Instance:   resource.ResourceInstance.String(),
				DeposedKey: resource.DeposedKey.String(),
			})
		}

		if run != nil {
			t.view.log.Error(
				fmt.Sprintf("Terraform left some resources in state after executing %s/%s, they need to be cleaned up manually.", file.Name, run.Name),
				"type", json.MessageTestCleanup,
				json.MessageTestCleanup, cleanup,
				"@testfile", file.Name,
				"@testrun", run.Name)
		} else {
			t.view.log.Error(
				fmt.Sprintf("Terraform left some resources in state after executing %s, they need to be cleaned up manually.", file.Name),
				"type", json.MessageTestCleanup,
				json.MessageTestCleanup, cleanup,
				"@testfile", file.Name)
		}

	}

	t.Diagnostics(run, file, diags)
}

func (t *TestJSON) Diagnostics(run *moduletest.Run, file *moduletest.File, diags tfdiags.Diagnostics) {
	var metadata []interface{}
	if file != nil {
		metadata = append(metadata, "@testfile", file.Name)
	}
	if run != nil {
		metadata = append(metadata, "@testrun", run.Name)
	}
	t.view.Diagnostics(diags, metadata...)
}

func (t *TestJSON) Interrupted() {
	t.view.Log(interrupted)
}

func (t *TestJSON) FatalInterrupt() {
	t.view.Log(fatalInterrupt)
}

func (t *TestJSON) FatalInterruptSummary(run *moduletest.Run, file *moduletest.File, existingStates map[*moduletest.Run]*states.State, created []*plans.ResourceInstanceChangeSrc) {

	message := json.TestFatalInterrupt{
		States: make(map[string][]json.TestFailedResource),
	}

	for run, state := range existingStates {
		if state.Empty() {
			continue
		}

		var resources []json.TestFailedResource
		for _, resource := range addrs.SetSortedNatural(state.AllManagedResourceInstanceObjectAddrs()) {
			resources = append(resources, json.TestFailedResource{
				Instance:   resource.ResourceInstance.String(),
				DeposedKey: resource.DeposedKey.String(),
			})
		}

		if run == nil {
			message.State = resources
		} else {
			message.States[run.Name] = resources
		}
	}

	if len(created) > 0 {
		for _, change := range created {
			message.Planned = append(message.Planned, change.Addr.String())
		}
	}

	if len(message.States) == 0 && len(message.State) == 0 && len(message.Planned) == 0 {
		// Then we don't have any information to share with the user.
		return
	}

	if run != nil {
		t.view.log.Error(
			"Terraform was interrupted during test execution, and may not have performed the expected cleanup operations.",
			"type", json.MessageTestInterrupt,
			json.MessageTestInterrupt, message,
			"@testfile", file.Name,
			"@testrun", run.Name)
	} else {
		t.view.log.Error(
			"Terraform was interrupted during test execution, and may not have performed the expected cleanup operations.",
			"type", json.MessageTestInterrupt,
			json.MessageTestInterrupt, message,
			"@testfile", file.Name)
	}
}

func (t *TestJSON) TFCStatusUpdate(status tfe.TestRunStatus, elapsed time.Duration) {
	var message string
	switch status {
	case tfe.TestRunQueued:
		message = fmt.Sprintf("Waiting for the tests to start... (%s elapsed)\n", elapsed.Truncate(30*time.Second))
	case tfe.TestRunRunning:
		message = fmt.Sprintf("Waiting for the tests to complete... (%s elapsed)\n", elapsed.Truncate(30*time.Second))
	default:
		// Don't care about updates for other statuses.
		return
	}

	t.view.log.Info(
		message,
		"type", json.MessageTestStatus,
		json.MessageTestStatus, json.TestStatusUpdate{
			Status:   string(status),
			Duration: elapsed.Seconds(),
		})
}

func (t *TestJSON) TFCRetryHook(attemptNum int, resp *http.Response) {
	t.view.log.Error(
		t.RetryLogHook(attemptNum, resp, false),
		"type", json.MessageTestRetry,
	)
}

func colorizeTestStatus(status moduletest.Status, color *colorstring.Colorize) string {
	switch status {
	case moduletest.Error, moduletest.Fail:
		return color.Color("[red]fail[reset]")
	case moduletest.Pass:
		return color.Color("[green]pass[reset]")
	case moduletest.Skip:
		return color.Color("[light_gray]skip[reset]")
	case moduletest.Pending:
		return color.Color("[light_gray]pending[reset]")
	default:
		panic("unrecognized status: " + status.String())
	}
}

func testStatus(status moduletest.Status) string {
	switch status {
	case moduletest.Error, moduletest.Fail:
		return "fail"
	case moduletest.Pass:
		return "pass"
	case moduletest.Skip:
		return "skip"
	case moduletest.Pending:
		return "pending"
	default:
		panic("unrecognized status: " + status.String())
	}
}
