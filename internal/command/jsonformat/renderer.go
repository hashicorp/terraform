package jsonformat

import (
	"fmt"

	"github.com/mitchellh/colorstring"

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
	ctyjson "github.com/zclconf/go-cty/cty/json"
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

type Renderer struct {
	Streams  *terminal.Streams
	Colorize *colorstring.Colorize

	RunningInAutomation bool
}

func (renderer Renderer) RenderHumanPlan(plan Plan, mode plans.Mode, opts ...PlanRendererOpt) {
	// TODO(liamcervante): Tidy up this detection of version differences, we
	// should only report warnings when the plan is generated using a newer
	// version then we are executing. We could also look into major vs minor
	// version differences. This should work for alpha testing in the meantime.
	if plan.PlanFormatVersion != jsonplan.FormatVersion || plan.ProviderFormatVersion != jsonprovider.FormatVersion {
		renderer.Streams.Println(format.WordWrap(
			renderer.Colorize.Color("\n[bold][red]Warning:[reset][bold] This plan was generated using a different version of Terraform, the diff presented here may be missing representations of recent features."),
			renderer.Streams.Stdout.Columns()))
	}

	plan.renderHuman(renderer, mode, opts...)
}

func (renderer Renderer) RenderHumanState(state State) {
	// TODO(liamcervante): Tidy up this detection of version differences, we
	// should only report warnings when the plan is generated using a newer
	// version then we are executing. We could also look into major vs minor
	// version differences. This should work for alpha testing in the meantime.
	if state.StateFormatVersion != jsonstate.FormatVersion || state.ProviderFormatVersion != jsonprovider.FormatVersion {
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

func (r Renderer) RenderLog(log *JSONLog) error {
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
		msg := fmt.Sprintf(r.Colorize.Color("[bold]%s[reset]"), log.Message)
		r.Streams.Println(msg)

	case LogDiagnostic:
		diag := format.DiagnosticFromJSON(log.Diagnostic, r.Colorize, 78)
		r.Streams.Print(diag)

	case LogOutputs:
		if len(log.Outputs) > 0 {
			r.Streams.Println(r.Colorize.Color("[bold][green]Outputs:[reset]"))
			for name, output := range log.Outputs {
				change := structured.FromJsonViewsOutput(output)
				ctype, err := ctyjson.UnmarshalType(output.Type)
				if err != nil {
					return err
				}

				opts := computed.NewRenderHumanOpts(r.Colorize)
				opts.ShowUnchangedChildren = true

				outputDiff := differ.ComputeDiffForType(change, ctype)
				outputStr := outputDiff.RenderHuman(0, opts)

				msg := fmt.Sprintf("%s = %s", name, outputStr)
				r.Streams.Println(msg)
			}
		}

	case LogProvisionProgress:
		provisioner := log.Hook["provisioner"].(string)
		output := log.Hook["output"].(string)
		resource := log.Hook["resource"].(map[string]interface{})
		resourceAddr := resource["addr"].(string)

		msg := fmt.Sprintf(r.Colorize.Color("[bold]%s: (%s):[reset] %s"),
			resourceAddr, provisioner, output)
		r.Streams.Println(msg)

	case LogChangeSummary:
		// Normally, we will only render the apply change summary since the renderer
		// generates a plan change summary for us
		msg := fmt.Sprintf(r.Colorize.Color("[bold][green]%s[reset]"), log.Message)
		r.Streams.Println("\n" + msg + "\n")

	default:
		// If the log type is not a known log type, we will just print the log message
		r.Streams.Println(log.Message)
	}

	return nil
}
