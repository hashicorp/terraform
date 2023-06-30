// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		LogApplyErrored:
		// We won't display these types of logs
		return nil

	case LogApplyStart, LogApplyComplete, LogRefreshStart, LogProvisionStart, LogResourceDrift:
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

	default:
		// If the log type is not a known log type, we will just print the log message
		renderer.Streams.Println(log.Message)
	}

	return nil
}
