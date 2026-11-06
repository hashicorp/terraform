// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePullCommand is a Command implementation that allows downloading
// and outputting state information from remote state.
type StatePullCommand struct {
	Meta
	StateMeta
}

func (c *StatePullCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	parsedArgs, argDiags := arguments.ParseStatePull(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewStatePull(parsedArgs.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	loader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	var varDiags tfdiags.Diagnostics
	c.VariableValues, varDiags = parsedArgs.Vars.CollectValues(func(filename string, src []byte) {
		loader.Parser().ForceFileSource(filename, src)
	})
	if varDiags.HasErrors() {
		diags = diags.Append(varDiags)
		view.Diagnostics(diags)
		return 1
	}

	vDiags := c.Meta.checkRequiredVersion()
	diags = diags.Append(vDiags)
	if vDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, beDiags := c.backend(".", parsedArgs.ViewType)
	diags = diags.Append(beDiags)
	if beDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Get the state manager for the current workspace
	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to select workspace",
			fmt.Sprintf("Error selecting workspace: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}
	stateMgr, sDiags := b.StateMgr(env)
	if sDiags.HasErrors() {
		diags = diags.Append(fmt.Errorf(errStateLoadingState, sDiags.Err()))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to refresh the state",
			fmt.Sprintf("Error refreshing the state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Get a statefile object representing the latest snapshot
	stateFile := statemgr.Export(stateMgr)

	if stateFile != nil { // we produce no output if the statefile is nil
		var buf bytes.Buffer
		err = statefile.Write(stateFile, &buf)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to save the state",
				fmt.Sprintf("Terraform encountered an error when saving state: %s", err),
			))
			view.Diagnostics(diags)
			return 1
		}

		view.State(buf.String())
	}

	return 0
}

func (c *StatePullCommand) Help() string {
	helpText := `
Usage: terraform [global options] state pull [options]

  Pull the state from its location, upgrade the local copy, and output it
  to stdout.

  This command "pulls" the current state and outputs it to stdout.
  As part of this process, Terraform will upgrade the state format of the
  local copy to the current version.

  The primary use of this is for state stored remotely. This command
  will still work with local state but is less useful for this.

Options:

  -compact-warnings   If Terraform produces any warnings that are not
                      accompanied by errors, show them in a more compact
                      form that includes only the summary messages

  -no-color           Disable text coloring in the output.

  -var 'foo=bar'      Set a value for one of the input variables in the root
                      module of the configuration. Use this option more than
                      once to set more than one variable.

  -var-file=filename  Load variable values from the given file, in addition
                      to the default files terraform.tfvars and *.auto.tfvars.
                      Use this option more than once to include more than one
                      variables file.

`
	return strings.TrimSpace(helpText)
}

func (c *StatePullCommand) Synopsis() string {
	return "Pull current state and output to stdout"
}
