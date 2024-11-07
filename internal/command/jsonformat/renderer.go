// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonformat

import (
	"fmt"
	"strconv"

	"github.com/mitchellh/colorstring"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
)

type JSONLogType string

type JSONLog struct {
	Message    string                 `json:"@message"`
	Type       JSONLogType            `json:"type"`
	Diagnostic *viewsjson.Diagnostic  `json:"diagnostic"`
	Outputs    viewsjson.Outputs      `json:"outputs"`
	Hook       map[string]interface{} `json:"hook"`

	// Special fields for test messages.

	TestRun  string `json:"@testrun,omitempty"`
	TestFile string `json:"@testfile,omitempty"`

	TestFileStatus     *viewsjson.TestFileStatus     `json:"test_file,omitempty"`
	TestRunStatus      *viewsjson.TestRunStatus      `json:"test_run,omitempty"`
	TestFileCleanup    *viewsjson.TestFileCleanup    `json:"test_cleanup,omitempty"`
	TestSuiteSummary   *viewsjson.TestSuiteSummary   `json:"test_summary,omitempty"`
	TestFatalInterrupt *viewsjson.TestFatalInterrupt `json:"test_interrupt,omitempty"`
	TestState          *State                        `json:"test_state,omitempty"`
	TestPlan           *Plan                         `json:"test_plan,omitempty"`
}

const (
	LogApplyComplete     JSONLogType = "apply_complete"
	LogApplyErrored      JSONLogType = "apply_errored"
	LogApplyStart        JSONLogType = "apply_start"
	LogChangeSummary     JSONLogType = "change_summary"
	LogDiagnostic        JSONLogType = "diagnostic"
	LogPlannedChange     JSONLogType = "planned_change"
	LogProvisionComplete JSONLogType = "provision_complete"
	LogProvisionErrored  JSONLogType = "provision_errored"
	LogProvisionProgress JSONLogType = "provision_progress"
	LogProvisionStart    JSONLogType = "provision_start"
	LogOutputs           JSONLogType = "outputs"
	LogRefreshComplete   JSONLogType = "refresh_complete"
	LogRefreshStart      JSONLogType = "refresh_start"
	LogResourceDrift     JSONLogType = "resource_drift"
	LogVersion           JSONLogType = "version"

	// Ephemeral operation messages
	LogEphemeralOpStart    JSONLogType = "ephemeral_op_start"
	LogEphemeralOpComplete JSONLogType = "ephemeral_op_complete"
	LogEphemeralOpErrored  JSONLogType = "ephemeral_op_errored"

	// Test Messages
	LogTestAbstract  JSONLogType = "test_abstract"
	LogTestFile      JSONLogType = "test_file"
	LogTestRun       JSONLogType = "test_run"
	LogTestPlan      JSONLogType = "test_plan"
	LogTestState     JSONLogType = "test_state"
	LogTestSummary   JSONLogType = "test_summary"
	LogTestCleanup   JSONLogType = "test_cleanup"
	LogTestInterrupt JSONLogType = "test_interrupt"
	LogTestStatus    JSONLogType = "test_status"
	LogTestRetry     JSONLogType = "test_retry"
)

func incompatibleVersions(localVersion, remoteVersion string) bool {
	var parsedLocal, parsedRemote float64
	var err error

	if parsedLocal, err = strconv.ParseFloat(localVersion, 64); err != nil {
		return false
	}
	if parsedRemote, err = strconv.ParseFloat(remoteVersion, 64); err != nil {
		return false
	}

	// If the local version is less than the remote version then the remote
	// version might contain things the local version doesn't know about, so
	// we're going to say they are incompatible.
	//
	// So far, we have built the renderer and the json packages to be backwards
	// compatible so if the local version is greater than the remote version
	// then that is okay, we'll still render a complete and correct plan.
	//
	// Note, this might change in the future. For example, if we introduce a
	// new major version in one of the formats the renderer may no longer be
	// backward compatible.
	return parsedLocal < parsedRemote
}

type Renderer struct {
	Streams  *terminal.Streams
	Colorize *colorstring.Colorize

	RunningInAutomation bool
}

func (renderer Renderer) RenderHumanPlan(plan Plan, mode plans.Mode, opts ...plans.Quality) {
	if incompatibleVersions(jsonplan.FormatVersion, plan.PlanFormatVersion) || incompatibleVersions(jsonprovider.FormatVersion, plan.ProviderFormatVersion) {
		renderer.Streams.Println(format.WordWrap(
			renderer.Colorize.Color("\n[bold][red]Warning:[reset][bold] This plan was generated using a different version of Terraform, the diff presented here may be missing representations of recent features."),
			renderer.Streams.Stdout.Columns()))
	}

	plan.renderHuman(renderer, mode, opts...)
}

func (renderer Renderer) RenderHumanState(state State) {
	if incompatibleVersions(jsonstate.FormatVersion, state.StateFormatVersion) || incompatibleVersions(jsonprovider.FormatVersion, state.ProviderFormatVersion) {
		renderer.Streams.Println(format.WordWrap(
			renderer.Colorize.Color("\n[bold][red]Warning:[reset][bold] This state was retrieved using a different version of Terraform, the state presented here maybe missing representations of recent features."),
			renderer.Streams.Stdout.Columns()))
	}

	if state.Empty() {
		renderer.Streams.Println("The state file is empty. No resources are represented.")
		return
	}

	opts := computed.NewRenderHumanOpts(renderer.Colorize)
	opts.ShowUnchangedChildren = true
	opts.HideDiffActionSymbols = true

	state.renderHumanStateModule(renderer, state.RootModule, opts, true)
	state.renderHumanStateOutputs(renderer, opts)
}

func (renderer Renderer) RenderLog(log *JSONLog) error {
	switch log.Type {
	case LogRefreshComplete,
		LogVersion,
		LogPlannedChange,
		LogProvisionComplete,
		LogProvisionErrored,
		LogApplyErrored,
		LogEphemeralOpErrored,
		LogTestAbstract,
		LogTestStatus,
		LogTestRetry,
		LogTestPlan,
		LogTestState,
		LogTestInterrupt:
		// We won't display these types of logs
		return nil

	case LogApplyStart, LogApplyComplete, LogRefreshStart, LogProvisionStart, LogResourceDrift, LogEphemeralOpStart, LogEphemeralOpComplete:
		msg := fmt.Sprintf(renderer.Colorize.Color("[bold]%s[reset]"), log.Message)
		renderer.Streams.Println(msg)

	case LogDiagnostic:
		diag := format.DiagnosticFromJSON(log.Diagnostic, renderer.Colorize, 78)
		renderer.Streams.Print(diag)

	case LogOutputs:
		if len(log.Outputs) > 0 {
			renderer.Streams.Println(renderer.Colorize.Color("[bold][green]Outputs:[reset]"))
			for name, output := range log.Outputs {
				change := structured.FromJsonViewsOutput(output)
				ctype, err := ctyjson.UnmarshalType(output.Type)
				if err != nil {
					return err
				}

				opts := computed.NewRenderHumanOpts(renderer.Colorize)
				opts.ShowUnchangedChildren = true

				outputDiff := differ.ComputeDiffForType(change, ctype)
				outputStr := outputDiff.RenderHuman(0, opts)

				msg := fmt.Sprintf("%s = %s", name, outputStr)
				renderer.Streams.Println(msg)
			}
		}

	case LogProvisionProgress:
		provisioner := log.Hook["provisioner"].(string)
		output := log.Hook["output"].(string)
		resource := log.Hook["resource"].(map[string]interface{})
		resourceAddr := resource["addr"].(string)

		msg := fmt.Sprintf(renderer.Colorize.Color("[bold]%s: (%s):[reset] %s"),
			resourceAddr, provisioner, output)
		renderer.Streams.Println(msg)

	case LogChangeSummary:
		// Normally, we will only render the apply change summary since the renderer
		// generates a plan change summary for us
		msg := fmt.Sprintf(renderer.Colorize.Color("[bold][green]%s[reset]"), log.Message)
		renderer.Streams.Println("\n" + msg + "\n")

	case LogTestFile:
		status := log.TestFileStatus

		var msg string
		switch status.Progress {
		case "starting":
			msg = fmt.Sprintf(renderer.Colorize.Color("%s... [light_gray]in progress[reset]"), status.Path)
		case "teardown":
			msg = fmt.Sprintf(renderer.Colorize.Color("%s... [light_gray]tearing down[reset]"), status.Path)
		case "complete":
			switch status.Status {
			case "error", "fail":
				msg = fmt.Sprintf(renderer.Colorize.Color("%s... [red]fail[reset]"), status.Path)
			case "pass":
				msg = fmt.Sprintf(renderer.Colorize.Color("%s... [green]pass[reset]"), status.Path)
			case "skip", "pending":
				msg = fmt.Sprintf(renderer.Colorize.Color("%s... [light_gray]%s[reset]"), status.Path, string(status.Status))
			}
		case "running":
			// Don't print anything for the running status.
			break
		}

		renderer.Streams.Println(msg)

	case LogTestRun:
		status := log.TestRunStatus

		if status.Progress != "complete" {
			// Don't print anything for status updates, we only report when the
			// run is actually finished.
			break
		}

		var msg string
		switch status.Status {
		case "error", "fail":
			msg = fmt.Sprintf(renderer.Colorize.Color("  %s... [red]fail[reset]"), status.Run)
		case "pass":
			msg = fmt.Sprintf(renderer.Colorize.Color("  %s... [green]pass[reset]"), status.Run)
		case "skip", "pending":
			msg = fmt.Sprintf(renderer.Colorize.Color("  %s... [light_gray]%s[reset]"), status.Run, string(status.Status))
		}

		renderer.Streams.Println(msg)

	case LogTestSummary:
		renderer.Streams.Println() // We start our summary with a line break.

		summary := log.TestSuiteSummary

		switch summary.Status {
		case "pending", "skip":
			renderer.Streams.Print("Executed 0 tests")
			if summary.Skipped > 0 {
				renderer.Streams.Printf(", %d skipped.\n", summary.Skipped)
			} else {
				renderer.Streams.Println(".")
			}
			return nil
		case "pass":
			renderer.Streams.Print(renderer.Colorize.Color("[green]Success![reset] "))
		case "fail", "error":
			renderer.Streams.Print(renderer.Colorize.Color("[red]Failure![reset] "))
		}

		renderer.Streams.Printf("%d passed, %d failed", summary.Passed, summary.Failed+summary.Errored)
		if summary.Skipped > 0 {
			renderer.Streams.Printf(", %d skipped.\n", summary.Skipped)
		} else {
			renderer.Streams.Println(".")
		}

	case LogTestCleanup:
		cleanup := log.TestFileCleanup

		renderer.Streams.Eprintln(format.WordWrap(log.Message, renderer.Streams.Stderr.Columns()))
		for _, resource := range cleanup.FailedResources {
			if len(resource.DeposedKey) > 0 {
				renderer.Streams.Eprintf(" - %s (%s)\n", resource.Instance, resource.DeposedKey)
			} else {
				renderer.Streams.Eprintf(" - %s\n", resource.Instance)
			}
		}

	default:
		// If the log type is not a known log type, we will just print the log message
		renderer.Streams.Println(log.Message)
	}

	return nil
}
