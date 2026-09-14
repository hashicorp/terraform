// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceList view is used for the `workspace list` subcommand.
type WorkspaceList interface {
	List(selected string, list []string, diags tfdiags.Diagnostics)
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewWorkspaceList(viewType arguments.ViewType, view *View) WorkspaceList {
	switch viewType {
	case arguments.ViewHuman:
		return &WorkspaceListHuman{
			view: view,
		}
	case arguments.ViewJSON:
		return &WorkspaceListJSON{
			view: NewJSONStaticView[WorkspaceListOutput](view),
		}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

// The WorkspaceListHuman implementation renders human-readable logs, suitable for direct consumption by a human user.
type WorkspaceListHuman struct {
	view *View
}

var _ WorkspaceList = (*WorkspaceListHuman)(nil)

// List is used to assemble the list of Workspaces and log it via Output
func (v *WorkspaceListHuman) List(selected string, list []string, diags tfdiags.Diagnostics) {
	// Print diags above output
	v.view.Diagnostics(diags)

	// Print list
	if len(list) > 0 {
		var out bytes.Buffer
		for _, s := range list {
			if s == selected {
				out.WriteString("* ")
			} else {
				out.WriteString("  ")
			}
			out.WriteString(s)
			out.WriteString("\n")
		}
		v.view.streams.Println(out.String())
	} else {
		// Warn that no states exist
		v.view.Diagnostics(warnNoEnvsExistDiag(selected))
	}
}

func (v *WorkspaceListHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// The WorkspaceListJSON implementation renders machine-readable logs, suitable for
// integrating with other software.
//
// This JSON output is a 'static log'; the command should produce a single JSON object containing all the available information.
type WorkspaceListJSON struct {
	view *JSONStaticView[WorkspaceListOutput]
}

var _ WorkspaceList = (*WorkspaceListJSON)(nil)

type WorkspaceListOutput struct {
	FormatVersion string                  `json:"format_version"`
	Workspaces    []WorkspaceOutput       `json:"workspaces"`
	Diagnostics   []*viewsjson.Diagnostic `json:"diagnostics"`
}

type WorkspaceOutput struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current,omitempty"`
}

// List is used to log the list of present workspaces and indicate which is currently selected
//
// If the `workspace list` command succeeds then the list will be populated and the diagnostics list will be either empty or contain warnings.
func (v *WorkspaceListJSON) List(current string, list []string, diags tfdiags.Diagnostics) {
	v.print(current, list, diags)
}

// Diagnostics logs only the diagnostics without listing any workspaces, and is intended to be used
// only when the `workspace list` command returns early with errors.
// The rendered diagnostics are presented in the same JSON object schema as the output when the command
// completes successfully.
func (v *WorkspaceListJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.print("", []string{}, diags)
}

func (v *WorkspaceListJSON) print(current string, list []string, diags tfdiags.Diagnostics) {
	output := WorkspaceListOutput{
		FormatVersion: FormatVersion,
		// Make sure these fields always appear as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		Diagnostics: []*viewsjson.Diagnostic{},
		Workspaces:  []WorkspaceOutput{},
	}

	for _, item := range list {
		workspace := WorkspaceOutput{
			Name:      item,
			IsCurrent: item == current,
		}
		output.Workspaces = append(output.Workspaces, workspace)
	}

	configSources := v.view.view.configSources()
	for _, diag := range diags {
		output.Diagnostics = append(output.Diagnostics, viewsjson.NewDiagnostic(diag, configSources))
	}

	v.view.Print(output)
}

// warnNoEnvsExistDiag creates a warning diagnostic saying that no workspaces exist,
// and provides guidance about how to create the workspace based on whether the workspace is
// custom or not.
func warnNoEnvsExistDiag(currentWorkspace string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	summary := "Terraform cannot find any existing workspaces."

	if currentWorkspace == backend.DefaultStateName {
		// Recommended actions for the user includes running `init` if they're using the default workspace.
		msg := fmt.Sprintf(
			"The %q workspace is selected in your working directory. You can create this workspace by running \"terraform init\", by using the \"terraform workspace new\" subcommand or by including the \"-or-create\" flag with the \"terraform workspace select\" subcommand.",
			currentWorkspace,
		)
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			summary,
			msg,
		))
	}

	msg := fmt.Sprintf(
		"The %q workspace is selected in your working directory. You can create this workspace by using the \"terraform workspace new\" subcommand or including the \"-or-create\" flag with the \"terraform workspace select\" subcommand.",
		currentWorkspace,
	)
	return diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		summary,
		msg,
	))
}
