// Copyright IBM Corp. 2014, 2026
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
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments.
	parsedArgs, diags := arguments.ParseProviders(rawArgs)

	// Prepare the view
	view := views.NewProviders(parsedArgs.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	configPath := c.Meta.WorkingDir.RootModuleDir()

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
	diags = diags.Append(varDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	empty, err := configs.IsEmptyDir(configPath, parsedArgs.TestsDirectory)
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

	diags = diags.Append(c.resolveConstVariables(configPath, arguments.ViewHuman))
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	config, configDiags := c.loadConfigWithTests(configPath, parsedArgs.TestsDirectory)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.backend(".", parsedArgs.ViewType)
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
			"Error determining the current workspace",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}
	s, sDiags := b.StateMgr(env)
	if sDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error accessing state",
			fmt.Sprintf("Failed to access state: %s", sDiags.Err()),
		))
		view.Diagnostics(diags)
		return 1
	}
	if err := s.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error loading state",
			fmt.Sprintf("Failed to load state: %s", err),
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

	view.LogProviderRequirements(reqs, stateReqs, diags)
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

  -test-directory=path  Set the Terraform test directory, defaults to "tests".

  -var 'foo=bar'        Set a value for one of the input variables in the root
                        module of the configuration. Use this option more than
                        once to set more than one variable.

  -var-file=filename    Load variable values from the given file, in addition
                        to the default files terraform.tfvars and *.auto.tfvars.
                        Use this option more than once to include more than one
                        variables file.
`
