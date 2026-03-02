// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/posener/complete"
)

type WorkspaceListCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceListCommand) Run(args []string) int {
	args = c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	parsedArgs, diags := arguments.ParseWorkspaceList(args)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	configPath, err := ModulePath(parsedArgs.Args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the backend
	view := arguments.ViewHuman
	b, diags := c.backend(configPath, view)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		c.Ui.Error(wDiags.Err().Error())
		return 1
	}
	c.showDiagnostics(diags) // output warnings, if any

	env, isOverridden := c.WorkspaceOverridden()

	if len(states) != 0 {
		var out bytes.Buffer
		for _, s := range states {
			if s == env {
				out.WriteString("* ")
			} else {
				out.WriteString("  ")
			}
			out.WriteString(s + "\n")
		}

		c.Ui.Output(out.String())
	} else {
		// Warn that no states exist
		c.showDiagnostics(warnNoEnvsExistDiag(env))
	}

	if isOverridden {
		c.Ui.Output(envIsOverriddenNote)
	}

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

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List Workspaces"
}
