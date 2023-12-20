// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceSelectCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceSelectCommand) Run(args []string) int {
	args = c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	var orCreate bool
	cmdFlags := c.Meta.defaultFlagSet("workspace select")
	cmdFlags.BoolVar(&orCreate, "or-create", false, "create workspace if it does not exist")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("Expected a single argument: NAME.\n")
		return cli.RunResultHelp
	}

	configPath, err := ModulePath(args[1:])
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	current, isOverridden := c.WorkspaceOverridden()
	if isOverridden {
		c.Ui.Error(envIsOverriddenSelectError)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	name := args[0]
	if !validWorkspaceName(name) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, name))
		return 1
	}

	states, err := b.Workspaces()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if name == current {
		// already using this workspace
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
		if orCreate {
			_, err = b.StateMgr(name)
			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
			newState = true
		} else {
			c.Ui.Error(fmt.Sprintf(envDoesNotExist, name))
			return 1
		}
	}

	err = c.SetWorkspace(name)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if newState {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			strings.TrimSpace(envCreated), name)))
	} else {
		c.Ui.Output(
			c.Colorize().Color(
				fmt.Sprintf(envChanged, name),
			),
		)
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

    -or-create=false    Create the Terraform workspace if it doesn't exist.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceSelectCommand) Synopsis() string {
	return "Select a workspace"
}
