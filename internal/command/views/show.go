// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
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
	Display(config *configs.Config, plan *plans.Plan, planJSON *cloudplan.RemotePlanJSON, stateFile *statefile.File, schemas *terraform.Schemas) int

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

func (v *ShowHuman) Display(config *configs.Config, plan *plans.Plan, planJSON *cloudplan.RemotePlanJSON, stateFile *statefile.File, schemas *terraform.Schemas) int {
	renderer := jsonformat.Renderer{
		Colorize:            v.view.colorize,
		Streams:             v.view.streams,
		RunningInAutomation: v.view.runningInAutomation,
	}

	// Prefer to display a pre-built JSON plan, if we got one; then, fall back
	// to building one ourselves.
	if planJSON != nil {
		if !planJSON.Redacted {
			diags := tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to display JSON plan",
					"Didn't get renderable JSON plan format for human display",
				),
			}
			v.Diagnostics(diags)
			return 1
		}
		// The redacted json plan format can be decoded into a jsonformat.Plan
		p := jsonformat.Plan{}
		r := bytes.NewReader(planJSON.JSONBytes)
		if err := json.NewDecoder(r).Decode(&p); err != nil {
			diags := tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to display JSON plan",
					fmt.Sprintf("Couldn't decode renderable JSON plan format: %s", err),
				),
			}
			v.Diagnostics(diags)
			return 1
		}

		v.view.streams.Print(v.view.colorize.Color(planJSON.RunHeader + "\n"))
		renderer.RenderHumanPlan(p, planJSON.Mode, planJSON.Qualities...)
		v.view.streams.Print(v.view.colorize.Color("\n" + planJSON.RunFooter + "\n"))
	} else if plan != nil {
		jsonPlan, err := jsonplan.MarshalForRenderer(plan, schemas)
		if err != nil {
			diags := tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to prepare plan for display",
					fmt.Sprintf("Failed to marshal plan to json when preparing for human display: %s", err),
				),
			}
			v.Diagnostics(diags)
			return 1
		}

		jplan := jsonformat.Plan{
			PlanFormatVersion:     jsonplan.FormatVersion,
			ProviderFormatVersion: jsonprovider.FormatVersion,
			OutputChanges:         jsonPlan.OutputChanges,
			ResourceChanges:       jsonPlan.ResourceChanges,
			ResourceDrift:         jsonPlan.ResourceDrift,
			ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
			RelevantAttributes:    jsonPlan.RelevantAttributes,
			ActionInvocations:     jsonPlan.ActionInvocations,
			DeferredChanges:       jsonPlan.DeferredChanges,
		}

		var opts []plans.Quality
		if plan.Errored {
			opts = append(opts, plans.Errored)
		} else if !plan.Applyable {
			opts = append(opts, plans.NoChanges)
		}

		renderer.RenderHumanPlan(jplan, plan.UIMode, opts...)
	} else {
		if stateFile == nil {
			v.view.streams.Println("No state.")
			return 0
		}

		root, outputs, err := jsonstate.MarshalForRenderer(stateFile, schemas)
		if err != nil {
			diags := tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to prepare state for display",
					fmt.Sprintf("Failed to marshal state to json when preparing for human display: %s", err),
				),
			}
			v.Diagnostics(diags)
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
	// Wrap a human view for access to streams when creating static JSON output.
	//
	// We cannot use the JSONStaticView approach here because the Display method
	// outputs different JSON objects depending on whether a plan or state is available
	view *View
}

var _ Show = (*ShowJSON)(nil)

func (v *ShowJSON) Display(config *configs.Config, plan *plans.Plan, planJSON *cloudplan.RemotePlanJSON, stateFile *statefile.File, schemas *terraform.Schemas) int {
	var diags tfdiags.Diagnostics

	// Prefer to display a pre-built JSON plan, if we got one; then, fall back
	// to building one ourselves.
	if planJSON != nil {
		if planJSON.Redacted {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to display JSON plan",
				"Didn't get external JSON plan format",
			))
			v.Diagnostics(diags)
			return 1
		}
		v.view.streams.Println(string(planJSON.JSONBytes))
	} else if plan != nil {
		planJSON, err := jsonplan.Marshal(config, plan, stateFile, schemas)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to display plan",
				fmt.Sprintf("Failed to marshal plan to json: %s", err),
			))
			v.Diagnostics(diags)
			return 1
		}
		v.view.streams.Println(string(planJSON))
	} else {
		// It is possible that there is neither state nor a plan.
		// That's ok, we'll just return an empty object.
		jsonState, err := jsonstate.Marshal(stateFile, schemas)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to display state",
				fmt.Sprintf("Failed to marshal state to json: %s", err),
			))
			v.Diagnostics(diags)
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
