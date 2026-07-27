// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceListCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceListCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments.
	args, diags := arguments.ParseWorkspaceList(rawArgs)

	// Prepare the view
	view := views.NewWorkspaceList(args.ViewType, c.View)

	// Warn against using `terraform env` commands, if needed
	diags = diags.Append(envCommandWarningDiag(c.LegacyName))

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.List("", nil, diags)
		return 1
	}

	// Load the backend
	configPath := c.WorkingDir.RootModuleDir()
	b, bDiags := c.backend(configPath, args.ViewType)
	diags = diags.Append(bDiags)
	if bDiags.HasErrors() {
		view.List("", nil, diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		view.List("", nil, diags)
		return 1
	}

	env, isOverridden, err := c.WorkspaceOverridden()
	if err != nil {
		diags = diags.Append(err)
		view.List("", nil, diags)
		return 1
	}

	if isOverridden {
		warn := tfdiags.Sourceless(
			tfdiags.Warning,
			envIsOverriddenNote,
			"",
		)
		diags = diags.Append(warn)
	}

	// Print:
	// 1. Diagnostics
	// 2. The list of workspaces, highlighting the current workspace
	view.List(env, states, diags)

	return 0
}

func (c *WorkspaceListCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictDirs("")
}

func (c *WorkspaceListCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceListCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace list

  List Terraform workspaces.

Options:

  -json            If specified, machine readable output will be
                   printed in JSON format.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List Workspaces"
}
