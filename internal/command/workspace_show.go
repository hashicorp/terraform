// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceShowCommand struct {
	Meta
}

func (c *WorkspaceShowCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments.
	args, parseDiags := arguments.ParseWorkspaceShow(rawArgs)
	diags = diags.Append(parseDiags)

	// Prepare the view
	view := views.NewWorkspaceShow(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Show("", diags)
		return 1
	}

	workspace, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
		view.Show("", diags)
		return 1
	}

	view.Show(workspace, diags)

	return 0
}

func (c *WorkspaceShowCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *WorkspaceShowCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceShowCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace show

  Show the name of the current workspace.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceShowCommand) Synopsis() string {
	return "Show the name of the current workspace"
}
