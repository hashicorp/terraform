// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersCommand is a Command implementation that prints out information
// about the providers used in the current configuration/state.
type ProvidersCommand struct {
	Meta
}

func (c *ProvidersCommand) Help() string {
	return providersCommandHelp
}

func (c *ProvidersCommand) Synopsis() string {
	return "Show the providers required for this configuration"
}

func (c *ProvidersCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy use of Ui. The remote backend and
	// cloud package use this; it should be removed when/if they are
	// migrated to views.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	// Parse and validate flags
	args, diags := arguments.ParseProviders(rawArgs)

	// Instantiate the view, even if there are flag errors, so that we render
	// diagnostics according to the desired view
	view := views.NewProviders(c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	configPath := args.Path

	empty, err := configs.IsEmptyDir(configPath, args.TestDirectory)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error validating configuration directory",
			fmt.Sprintf("Terraform encountered an unexpected error while verifying that the given configuration directory is valid: %s.", err),
		))
		view.Diagnostics(diags)
		return 1
	}
	if empty {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			absPath = configPath
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files",
			fmt.Sprintf("The directory %s contains no Terraform configuration files.", absPath),
		))
		view.Diagnostics(diags)
		return 1
	}

	config, configDiags := c.loadConfigWithTests(configPath, args.TestDirectory)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.backend(".", arguments.ViewHuman)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Get the state
	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error selecting workspace",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}
	s, sDiags := b.StateMgr(env)
	if sDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to load state",
			sDiags.Err().Error(),
		))
		view.Diagnostics(diags)
		return 1
	}
	if err := s.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to load state",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	reqs, reqDiags := config.ProviderRequirementsByModule()
	diags = diags.Append(reqDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	state := s.State()
	var stateReqs getproviders.Requirements
	if state != nil {
		stateReqs = state.ProviderRequirements()
	}

	// Render the output through the view
	view.Output(reqs, stateReqs)

	// Show any warnings that were collected
	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

const providersCommandHelp = `
Usage: terraform [global options] providers [options] [DIR]

  Prints out a tree of modules in the referenced configuration annotated with
  their provider requirements.

  This provides an overview of all of the provider requirements across all
  referenced modules, as an aid to understanding why particular provider
  plugins are needed and why particular versions are selected.

Options:

  -test-directory=path	Set the Terraform test directory, defaults to "tests".
`
