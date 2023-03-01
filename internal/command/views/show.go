package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Show interface {
	// Display renders the plan, if it is available. If plan is nil, it renders the statefile.
	Display(config *configs.Config, plan *plans.Plan, stateFile *statefile.File, schemas *terraform.Schemas) int

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewShow(vt arguments.ViewType, view *View) Show {
	switch vt {
	case arguments.ViewJSON:
		return &ShowJSON{view: view}
	case arguments.ViewHuman:
		return &ShowHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type ShowHuman struct {
	view *View
}

var _ Show = (*ShowHuman)(nil)

func (v *ShowHuman) Display(config *configs.Config, plan *plans.Plan, stateFile *statefile.File, schemas *terraform.Schemas) int {
	renderer := jsonformat.Renderer{
		Colorize:            v.view.colorize,
		Streams:             v.view.streams,
		RunningInAutomation: v.view.runningInAutomation,
	}

	if plan != nil {
		outputs, changed, drift, attrs, err := jsonplan.MarshalForRenderer(plan, schemas)
		if err != nil {
			v.view.streams.Eprintf("Failed to marshal plan to json: %s", err)
			return 1
		}

		jplan := jsonformat.Plan{
			PlanFormatVersion:     jsonplan.FormatVersion,
			ProviderFormatVersion: jsonprovider.FormatVersion,
			OutputChanges:         outputs,
			ResourceChanges:       changed,
			ResourceDrift:         drift,
			ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
			RelevantAttributes:    attrs,
		}

		var opts []jsonformat.PlanRendererOpt
		if !plan.CanApply() {
			opts = append(opts, jsonformat.CanNotApply)
		}
		if plan.Errored {
			opts = append(opts, jsonformat.Errored)
		}

		renderer.RenderHumanPlan(jplan, plan.UIMode, opts...)
	} else {
		if stateFile == nil {
			v.view.streams.Println("No state.")
			return 0
		}

		root, outputs, err := jsonstate.MarshalForRenderer(stateFile, schemas)
		if err != nil {
			v.view.streams.Eprintf("Failed to marshal state to json: %s", err)
			return 1
		}

		jstate := jsonformat.State{
			StateFormatVersion:    jsonstate.FormatVersion,
			ProviderFormatVersion: jsonprovider.FormatVersion,
			RootModule:            root,
			RootModuleOutputs:     outputs,
			ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
		}

		renderer.RenderHumanState(jstate)
	}
	return 0
}

func (v *ShowHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type ShowJSON struct {
	view *View
}

var _ Show = (*ShowJSON)(nil)

func (v *ShowJSON) Display(config *configs.Config, plan *plans.Plan, stateFile *statefile.File, schemas *terraform.Schemas) int {
	if plan != nil {
		jsonPlan, err := jsonplan.Marshal(config, plan, stateFile, schemas)

		if err != nil {
			v.view.streams.Eprintf("Failed to marshal plan to json: %s", err)
			return 1
		}
		v.view.streams.Println(string(jsonPlan))
	} else {
		// It is possible that there is neither state nor a plan.
		// That's ok, we'll just return an empty object.
		jsonState, err := jsonstate.Marshal(stateFile, schemas)
		if err != nil {
			v.view.streams.Eprintf("Failed to marshal state to json: %s", err)
			return 1
		}
		v.view.streams.Println(string(jsonState))
	}
	return 0
}

// Diagnostics should only be called if show cannot be executed.
// In this case, we choose to render human-readable diagnostic output,
// primarily for backwards compatibility.
func (v *ShowJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
