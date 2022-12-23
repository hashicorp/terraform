package jsonformat

import (
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type RendererOpt int

const (
	detectedDrift  string = "drift"
	proposedChange string = "change"

	Errored RendererOpt = iota
	CanNotApply
)

type JSONLogType string

type JSONLog struct {
	Message    string                `json:"@message"`
	Type       JSONLogType           `json:"type"`
	Diagnostic *viewsjson.Diagnostic `json:"diagnostic"`
	Outputs    viewsjson.Outputs     `json:"outputs"`
}

const (
	LogVersion         JSONLogType = "version"
	LogDiagnostic      JSONLogType = "diagnostic"
	LogPlannedChange   JSONLogType = "planned_change"
	LogRefreshStart    JSONLogType = "refresh_start"
	LogRefreshComplete JSONLogType = "refresh_complete"
	LogApplyStart      JSONLogType = "apply_start"
	LogApplyComplete   JSONLogType = "apply_complete"
	LogChangeSummary   JSONLogType = "change_summary"
	LogOutputs         JSONLogType = "outputs"
)

type Plan struct {
	PlanFormatVersion  string                     `json:"plan_format_version"`
	OutputChanges      map[string]jsonplan.Change `json:"output_changes"`
	ResourceChanges    []jsonplan.ResourceChange  `json:"resource_changes"`
	ResourceDrift      []jsonplan.ResourceChange  `json:"resource_drift"`
	RelevantAttributes []jsonplan.ResourceAttr    `json:"relevant_attributes"`

	ProviderFormatVersion string                            `json:"provider_format_version"`
	ProviderSchemas       map[string]*jsonprovider.Provider `json:"provider_schemas"`
}

func (plan Plan) GetSchema(change jsonplan.ResourceChange) *jsonprovider.Schema {
	switch change.Mode {
	case jsonplan.ManagedResourceMode:
		return plan.ProviderSchemas[change.ProviderName].ResourceSchemas[change.Type]
	case jsonplan.DataResourceMode:
		return plan.ProviderSchemas[change.ProviderName].DataSourceSchemas[change.Type]
	default:
		panic("found unrecognized resource mode: " + change.Mode)
	}
}

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
			renderer.Colorize.Color("\n[bold][red]Warning:[reset][bold] This plan was generated using a different version of Terraform, the diff presented here maybe missing representations of recent features."),
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

	opts: = computed.RenderHumanOpts{
		ShowUnchangedChildren: true,
		HideDiffActionSymbols: true,
	}

	state.renderHumanStateModule(renderer, state.RootModule, opts, true)
	state.renderHumanStateOutputs(renderer, opts)
}

func (r Renderer) RenderLog(log *JSONLog) error {
	switch log.Type {
	case LogApplyStart, LogApplyComplete, LogRefreshStart:
		msg := fmt.Sprintf("[bold]%s[reset]", log.Message)
		r.Streams.Println(r.Colorize.Color(msg))

	case LogDiagnostic:
		diag := format.DiagnosticFromJSON(log.Diagnostic, r.Colorize, 78)
		r.Streams.Print(diag)

	case LogOutputs:
		if len(log.Outputs) > 0 {
			r.Streams.Println(r.Colorize.Color("[bold][green]Outputs:[reset]"))
			for name, output := range log.Outputs {
				change := differ.FromJsonOutput(output)
				ctype, err := ctyjson.UnmarshalType(output.Type)
				if err != nil {
					return err
				}

				outputDiff := change.ComputeDiffForType(ctype)
				outputStr := outputDiff.RenderHuman(0, computed.RenderHumanOpts{
					Colorize:              r.Colorize,
					ShowUnchangedChildren: true,
				})

				msg := fmt.Sprintf("%s = %s", name, outputStr)
				r.Streams.Println(r.Colorize.Color(msg))
			}
		}

	case LogChangeSummary:
		// We will only render the apply change summary since the renderer
		// generates a plan change summary for us
		if !strings.Contains(log.Message, "Plan") {
			msg := fmt.Sprintf("[bold][green]%s[reset]", log.Message)
			r.Streams.Println("\n" + r.Colorize.Color(msg) + "\n")
		}
	}

	return nil
}