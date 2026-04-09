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
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
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

	// DisplayResourceInstanceState displays the state for a single resource instance. Diagnostics from the state show command are included in the json output.
	DisplayResourceInstanceState(jsonformat.State, tfdiags.Diagnostics) int
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

func (v *ShowHuman) DisplayResourceInstanceState(state jsonformat.State, diags tfdiags.Diagnostics) int {
	renderer := jsonformat.Renderer{
		Streams:             v.view.streams,
		Colorize:            v.view.colorize,
		RunningInAutomation: v.view.runningInAutomation,
	}

	if diags.HasErrors() {
		v.Diagnostics(diags)
		return 1
	}

	renderer.RenderHumanState(state)
	return 0
}

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
			v.view.streams.Eprintf("Didn't get renderable JSON plan format for human display")
			return 1
		}
		// The redacted json plan format can be decoded into a jsonformat.Plan
		p := jsonformat.Plan{}
		r := bytes.NewReader(planJSON.JSONBytes)
		if err := json.NewDecoder(r).Decode(&p); err != nil {
			v.view.streams.Eprintf("Couldn't decode renderable JSON plan format: %s", err)
		}

		v.view.streams.Print(v.view.colorize.Color(planJSON.RunHeader + "\n"))
		renderer.RenderHumanPlan(p, planJSON.Mode, planJSON.Qualities...)
		v.view.streams.Print(v.view.colorize.Color("\n" + planJSON.RunFooter + "\n"))
	} else if plan != nil {
		outputs, changed, drift, attrs, actions, err := jsonplan.MarshalForRenderer(plan, schemas)
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
			ActionInvocations:     actions,
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

func (v *ShowJSON) Display(config *configs.Config, plan *plans.Plan, planJSON *cloudplan.RemotePlanJSON, stateFile *statefile.File, schemas *terraform.Schemas) int {
	// Prefer to display a pre-built JSON plan, if we got one; then, fall back
	// to building one ourselves.
	if planJSON != nil {
		if planJSON.Redacted {
			v.view.streams.Eprintf("Didn't get external JSON plan format")
			return 1
		}
		v.view.streams.Println(string(planJSON.JSONBytes))
	} else if plan != nil {
		planJSON, err := jsonplan.Marshal(config, plan, stateFile, schemas)

		if err != nil {
			v.view.streams.Eprintf("Failed to marshal plan to json: %s", err)
			return 1
		}
		v.view.streams.Println(string(planJSON))
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

func (v *ShowJSON) DisplayResourceInstanceState(jsonState jsonformat.State, diags tfdiags.Diagnostics) int {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const FormatVersion = "1.0"

	type Output struct {
		FormatVersion string                  `json:"format_version"`
		Resource      jsonstate.Resource      `json:"resource"`
		Diagnostics   []*viewsjson.Diagnostic `json:"diagnostics"`
	}

	output := Output{
		FormatVersion: FormatVersion,
	}

	for _, diag := range diags {
		output.Diagnostics = append(output.Diagnostics, viewsjson.NewDiagnostic(diag, v.view.configSources()))
	}
	if output.Diagnostics == nil {
		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		output.Diagnostics = []*viewsjson.Diagnostic{}
	}

	var rs jsonstate.Resource
	if !jsonState.Empty() {
		// we know there's only one resource instance, but we need to find it.
		if len(jsonState.RootModule.Resources) > 0 {
			rs = jsonState.RootModule.Resources[0]
		} else {
			rs = findResourceInChildModules(jsonState.RootModule)
		}
	}

	output.Resource = rs

	j, err := json.MarshalIndent(&output, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}
	v.view.streams.Println(string(j))

	if diags.HasErrors() {
		return 1
	}
	return 0
}

func findResourceInChildModules(mod jsonstate.Module) jsonstate.Resource {
	for _, cm := range mod.ChildModules {
		if len(cm.Resources) == 1 {
			return cm.Resources[0]
		}
	}
	for _, child := range mod.ChildModules {
		return findResourceInChildModules(child)
	}
	// this shouldn't be possible; we would have returned an error earlier if
	// the resource wasn't found.
	panic("resource not found in state")
}
