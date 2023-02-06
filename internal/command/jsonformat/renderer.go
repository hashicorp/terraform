package jsonformat

import (
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
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

	opts := computed.RenderHumanOpts{
		ShowUnchangedChildren: true,
		HideDiffActionSymbols: true,
	}

	state.renderHumanStateModule(renderer, state.RootModule, opts, true)
	state.renderHumanStateOutputs(renderer, opts)
}

func (renderer Renderer) RenderLog(message map[string]interface{}) {
	panic("not implemented")
}
