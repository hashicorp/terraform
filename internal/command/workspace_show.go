// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/posener/complete"
)

type WorkspaceShowCommand struct {
	Meta
}

func (c *WorkspaceShowCommand) Run(rawArgs []string) int {
	// Process global flags and configure the view/UI.
	rawArgs = c.Meta.process(rawArgs)

	// Process command-specific arguments.
	// Currently there are no arguments for this command, so ignore the returned value for now.
	_, diags := arguments.ParseWorkspaceShow(rawArgs)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return cli.RunResultHelp
	}

	workspace, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}
	c.Ui.Output(workspace)

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
