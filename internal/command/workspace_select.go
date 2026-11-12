// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceSelectCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceSelectCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	diags = diags.Append(envCommandWarningDiag(c.LegacyName))

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	args, parseDiags := arguments.ParseWorkspaceSelect(rawArgs)
	diags = diags.Append(parseDiags)

	// Prepare the view
	view := views.NewWorkspaceSelect(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Block selecting a workspace if an environment variable will override the new selection anyway.
	//
	// We ignore errors raised about the value of the override ENV, or the value of the currently selected
	// workspace; this command should help users recover from those errors, not block them.
	current, isOverridden, _ := c.WorkspaceOverridden()
	if isOverridden {
		diags = diags.Append(errors.New(envIsOverriddenSelectError))
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	c.bypassWorkspaceNameValidityCheck = true // allow selecting a new workspace when the current one is invalid.
	configPath := c.WorkingDir.RootModuleDir()
	b, bDiags := c.backend(configPath, args.ViewType)
	diags = diags.Append(bDiags)
	if bDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		view.Diagnostics(wDiags)
		return 1
	}

	name := args.Name
	if name == current {
		// already using this workspace
		view.Diagnostics(diags) // output warnings, if any
		return 0
	}

	found := false
	for _, s := range states {
		if name == s {
			found = true
			break
		}
	}

	var newState bool

	if !found {
		if args.OrCreate {
			_, sDiags := b.StateMgr(name)
			diags = diags.Append(sDiags)
			if sDiags.HasErrors() {
				view.Diagnostics(diags)
				return 1
			}
			newState = true
		} else {
			diags = diags.Append(fmt.Errorf(envDoesNotExist, name))
			view.Diagnostics(diags)
			return 1
		}
	}

	err := c.SetWorkspace(name)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	if newState {
		view.CreatedWorkspace(name, diags)
	} else {
		view.SelectedWorkspace(name, diags)
	}

	return 0
}

func (c *WorkspaceSelectCommand) AutocompleteArgs() complete.Predictor {
	return completePredictSequence{
		c.completePredictWorkspaceName(),
		complete.PredictDirs(""),
	}
}

func (c *WorkspaceSelectCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceSelectCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace select NAME

  Select a different Terraform workspace.

Options:

    -compact-warnings    If Terraform produces any warnings that are not
                         accompanied by errors, show them in a more compact
                         form that includes only the summary messages

    -no-color            If specified, output won't contain any color.

    -or-create=false     Create the Terraform workspace if it doesn't exist.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceSelectCommand) Synopsis() string {
	return "Select a workspace"
}
