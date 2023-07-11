package views

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
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
	File(file *moduletest.File)

	// Run prints out the summary for a single test run block.
	Run(run *moduletest.Run, file *moduletest.File)

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
		t.view.streams.Printf("Executed 0 tests")
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

func (t *TestHuman) File(file *moduletest.File) {
	t.view.streams.Printf("%s... %s\n", file.Name, colorizeTestStatus(file.Status, t.view.colorize))
}

func (t *TestHuman) Run(run *moduletest.Run, file *moduletest.File) {
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

				renderer.RenderHumanState(state)
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

				var opts []jsonformat.PlanRendererOpt
				if !run.Verbose.Plan.CanApply() {
					opts = append(opts, jsonformat.CanNotApply)
				}
				if run.Verbose.Plan.Errored {
					opts = append(opts, jsonformat.Errored)
				}

				renderer.RenderHumanPlan(plan, run.Verbose.Plan.UIMode, opts...)
			}
		}
	}

	// Finally we'll print out a summary of the diagnostics from the run.
	t.Diagnostics(run, file, run.Diagnostics)
}

func (t *TestHuman) DestroySummary(diags tfdiags.Diagnostics, run *moduletest.Run, file *moduletest.File, state *states.State) {
	identifier := file.Name
	if run != nil {
		identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
	}

	if diags.HasErrors() {
		t.view.streams.Eprintf("Terraform encountered an error destroying resources created while executing %s.\n", identifier)
	}
	t.Diagnostics(run, file, diags)

	if state.HasManagedResourceInstanceObjects() {
		t.view.streams.Eprintf("\nTerraform left the following resources in state after executing %s, they need to be cleaned up manually:\n", identifier)
		for _, resource := range state.AllResourceInstanceObjectAddrs() {
			if resource.DeposedKey != states.NotDeposed {
				t.view.streams.Eprintf("  - %s (%s)\n", resource.Instance, resource.DeposedKey)
				continue
			}
			t.view.streams.Eprintf("  - %s\n", resource.Instance)
		}
	}
}

func (t *TestHuman) Diagnostics(_ *moduletest.Run, _ *moduletest.File, diags tfdiags.Diagnostics) {
	t.view.Diagnostics(diags)
}

func (t *TestHuman) Interrupted() {
	t.view.streams.Print(interrupted)
}

func (t *TestHuman) FatalInterrupt() {
	t.view.streams.Print(fatalInterrupt)
}

type TestJSON struct {
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

func (t *TestJSON) File(file *moduletest.File) {
	t.view.log.Info(
		fmt.Sprintf("%s... %s", file.Name, testStatus(file.Status)),
		"type", json.MessageTestFile,
		json.MessageTestFile, json.TestFileStatus{file.Name, json.ToTestStatus(file.Status)},
		"@testfile", file.Name)
}

func (t *TestJSON) Run(run *moduletest.Run, file *moduletest.File) {
	t.view.log.Info(
		fmt.Sprintf("  %q... %s", run.Name, testStatus(run.Status)),
		"type", json.MessageTestRun,
		json.MessageTestRun, json.TestRunStatus{file.Name, run.Name, json.ToTestStatus(run.Status)},
		"@testfile", file.Name,
		"@testrun", run.Name)

	if run.Verbose != nil {

		schemas := &terraform.Schemas{
			Providers:    run.Verbose.Providers,
			Provisioners: run.Verbose.Provisioners,
		}

		if run.Config.Command == configs.ApplyTestCommand {
			state, err := jsonstate.MarshalForLog(statefile.New(run.Verbose.State, file.Name, uint64(run.Index)), schemas)
			if err != nil {
				run.Diagnostics = run.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to render test state",
					fmt.Sprintf("Terraform could not marshal the state for display: %v", err)))
			} else {
				t.view.log.Info(
					fmt.Sprintf("-verbose flag enabled, printing state"),
					"type", json.MessageTestState,
					json.MessageTestState, state,
					"@testfile", file.Name,
					"@testrun", run.Name)
			}
		} else {
			plan, err := jsonplan.MarshalForLog(run.Verbose.Config, run.Verbose.Plan, nil, schemas)
			if err != nil {
				run.Diagnostics = run.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Failed to render test plan",
					fmt.Sprintf("Terraform could not marshal the plan for display: %v", err)))
			} else {
				t.view.log.Info(
					fmt.Sprintf("-verbose flag enabled, printing plan"),
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
		for _, resource := range state.AllResourceInstanceObjectAddrs() {
			cleanup.FailedResources = append(cleanup.FailedResources, json.TestFailedResource{
				Instance:   resource.Instance.String(),
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
