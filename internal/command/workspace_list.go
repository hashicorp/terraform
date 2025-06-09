// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	tfversion "github.com/hashicorp/terraform/version"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceListCommand struct {
	Meta
	LegacyName bool
}

type WorkspaceList struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type WorkspaceListOutput struct {
	TerraformVersion string          `json:"terraform_version"`
	WorkspaceList    []WorkspaceList `json:"workspaces"`
	IsOverridden     bool            `json:"is_overridden"`
	OverriddenNote   string          `json:"overridden_note"`
}

func (c *WorkspaceListCommand) Run(args []string) int {
	args = c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	cmdFlags := c.Meta.defaultFlagSet("workspace list")
	var jsonOutput bool
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")
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

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
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

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, err := b.Workspaces()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	env, isOverridden := c.WorkspaceOverridden()

	// If json
	if jsonOutput {
		var wsOutput WorkspaceListOutput
		wsOutput.TerraformVersion = tfversion.String()
		wsOutput.IsOverridden = isOverridden
		if isOverridden {
			wsOutput.OverriddenNote = envIsOverriddenNote
		}

		for _, s := range states {
			ws := WorkspaceList{Name: s, Selected: s == env}
			wsOutput.WorkspaceList = append(wsOutput.WorkspaceList, ws)
		}

		jsonOutput, err := json.Marshal(wsOutput)

		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to marshal workspace list to json: %s", err))
			return 1
		}
		c.Ui.Output(string(jsonOutput))
		return 0
	}

	// If not json
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
Usage: terraform [global options] workspace list [options] 

  List Terraform workspaces.

Options:

  -json               If specified, output to a machine-readable form.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List Workspaces"
}
