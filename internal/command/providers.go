// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xlab/treeprint"

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

func (c *ProvidersCommand) Run(args []string) int {
	var testsDirectory string

	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("providers")
	cmdFlags.StringVar(&testsDirectory, "test-directory", "tests", "test-directory")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	empty, err := configs.IsEmptyDir(configPath, testsDirectory)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error validating configuration directory",
			fmt.Sprintf("Terraform encountered an unexpected error while verifying that the given configuration directory is valid: %s.", err),
		))
		c.showDiagnostics(diags)
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
		c.showDiagnostics(diags)
		return 1
	}

	config, configDiags := c.loadConfigWithTests(configPath, testsDirectory)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: config.Module.Backend,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Get the state
	env, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}
	s, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}
	if err := s.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	reqs, reqDiags := config.ProviderRequirementsByModule()
	diags = diags.Append(reqDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	state := s.State()
	var stateReqs getproviders.Requirements
	if state != nil {
		stateReqs = state.ProviderRequirements()
	}

	printRoot := treeprint.New()
	c.populateTreeNode(printRoot, reqs)

	c.Ui.Output("\nProviders required by configuration:")
	c.Ui.Output(printRoot.String())

	if len(stateReqs) > 0 {
		c.Ui.Output("Providers required by state:\n")
		for fqn := range stateReqs {
			c.Ui.Output(fmt.Sprintf("    provider[%s]\n", fqn.String()))
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

func (c *ProvidersCommand) populateTreeNode(tree treeprint.Tree, node *configs.ModuleRequirements) {
	for fqn, dep := range node.Requirements {
		versionsStr := getproviders.VersionConstraintsString(dep)
		if versionsStr != "" {
			versionsStr = " " + versionsStr
		}
		tree.AddNode(fmt.Sprintf("provider[%s]%s", fqn.String(), versionsStr))
	}
	for name, testNode := range node.Tests {
		name = strings.TrimSuffix(name, ".tftest.hcl")
		name = strings.ReplaceAll(name, "/", ".")
		branch := tree.AddBranch(fmt.Sprintf("test.%s", name))

		for fqn, dep := range testNode.Requirements {
			versionsStr := getproviders.VersionConstraintsString(dep)
			if versionsStr != "" {
				versionsStr = " " + versionsStr
			}
			branch.AddNode(fmt.Sprintf("provider[%s]%s", fqn.String(), versionsStr))
		}

		for _, run := range testNode.Runs {
			branch := branch.AddBranch(fmt.Sprintf("run.%s", run.Name))
			c.populateTreeNode(branch, run)
		}
	}
	for name, childNode := range node.Children {
		branch := tree.AddBranch(fmt.Sprintf("module.%s", name))
		c.populateTreeNode(branch, childNode)
	}
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
