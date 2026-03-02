// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/posener/complete"
)

type WorkspaceSelectCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceSelectCommand) Run(args []string) int {
	args = c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	parsedArgs, diags := arguments.ParseWorkspaceSelect(args)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	configPath, err := ModulePath(parsedArgs.Args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	current, isOverridden := c.WorkspaceOverridden()
	if isOverridden {
		c.Ui.Error(envIsOverriddenSelectError)
		return 1
	}

	// Load the backend
	view := arguments.ViewHuman
	b, bDiags := c.backend(configPath, view)
	if bDiags.HasErrors() {
		c.showDiagnostics(bDiags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	name := parsedArgs.Name
	if !validWorkspaceName(name) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, name))
		return 1
	}

	states, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		c.Ui.Error(wDiags.Err().Error())
		return 1
	}
	c.showDiagnostics(bDiags) // output warnings, if any

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
		if parsedArgs.OrCreate {
			_, sDiags := b.StateMgr(name)
			if sDiags.HasErrors() {
				c.Ui.Error(sDiags.Err().Error())
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
