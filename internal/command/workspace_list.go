// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceListCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceListCommand) Run(args []string) int {
	args = c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	cmdFlags := c.Meta.defaultFlagSet("workspace list")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	configPath, err := ModulePath(args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.prepareBackend(configPath)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		diags = diags.Append(errors.New("Failed to load backend"))
		c.showDiagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, err := b.Workspaces()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	env, isOverridden := c.WorkspaceOverridden()

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

// prepareBackend returns an operations backend that may use a backend, cloud, or state_store block for state storage.
func (c *WorkspaceListCommand) prepareBackend(configPath string) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	if configPath == "" {
		configPath = "."
	}

	mod, diags := c.loadSingleModule(configPath)
	if diags.HasErrors() {
		return nil, diags
	}

	return c.Meta.prepareBackend(mod)
}
