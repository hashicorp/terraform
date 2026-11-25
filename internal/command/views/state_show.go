// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateShow interface {
	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)

	// DisplayResourceInstanceState displays the state for a single resource instance. Diagnostics from the state show command are included in the json output.
	DisplayResourceInstanceState(jsonformat.State, tfdiags.Diagnostics) int
}

func NewStateShow(vt arguments.ViewType, view *View) StateShow {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const formatVersion = "1.0"

	switch vt {
	case arguments.ViewJSON:
		return &StateShowJSON{
			view:          NewJSONStaticView[StateShowOutput](view),
			formatVersion: formatVersion,
		}
	case arguments.ViewHuman:
		return &StateShowHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type StateShowHuman struct {
	view *View
}

var _ StateShow = (*StateShowHuman)(nil)

func (v *StateShowHuman) DisplayResourceInstanceState(state jsonformat.State, diags tfdiags.Diagnostics) int {
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

func (v *StateShowHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type StateShowJSON struct {
	view *JSONStaticView[StateShowOutput]

	formatVersion string
}

var _ StateShow = (*StateShowJSON)(nil)

// JSON output from the `state show` command
type StateShowOutput struct {
	FormatVersion string                  `json:"format_version"`
	Resource      jsonstate.Resource      `json:"resource"`
	Diagnostics   []*viewsjson.Diagnostic `json:"diagnostics"`
}

// Diagnostics should only be called if show cannot be executed.
// In this case, we choose to render human-readable diagnostic output,
// primarily for backwards compatibility.
func (v *StateShowJSON) Diagnostics(diags tfdiags.Diagnostics) {
	output := StateShowOutput{
		FormatVersion: v.formatVersion,
		Diagnostics:   v.view.prepareDiagnostics(diags),
	}

	v.view.Print(output)
}

func (v *StateShowJSON) DisplayResourceInstanceState(jsonState jsonformat.State, diags tfdiags.Diagnostics) int {
	output := StateShowOutput{
		FormatVersion: v.formatVersion,
		Diagnostics:   v.view.prepareDiagnostics(diags),
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

	v.view.Print(output)

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
