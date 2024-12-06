// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/modsdir"
	"github.com/hashicorp/terraform/internal/moduleref"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ModulesCommand is a Command implementation that prints out information
// about the modules declared by the current configuration.
type ModulesCommand struct {
	Meta
	viewType arguments.ViewType
}

func (c *ModulesCommand) Help() string {
	return modulesCommandHelp
}

func (c *ModulesCommand) Synopsis() string {
	return "Show all declared modules in a working directory"
}

func (c *ModulesCommand) Run(rawArgs []string) int {
	// Parse global view arguments
	rawArgs = c.Meta.process(rawArgs)
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command specific flags
	args, diags := arguments.ParseModules(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("modules")
		return 1
	}
	c.viewType = args.ViewType

	// Set up the command's view
	view := views.NewModules(c.viewType, c.View)

	rootModPath, err := ModulePath([]string{})
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// Read the root module path so we can then traverse the tree
	rootModEarly, earlyConfDiags := c.loadSingleModule(rootModPath)
	if rootModEarly == nil {
		diags = diags.Append(errors.New("root module not found. Please run terraform init"), earlyConfDiags)
		view.Diagnostics(diags)
		return 1
	}

	config, confDiags := c.loadConfig(rootModPath)
	// Here we check if there are any uninstalled dependencies
	versionDiags := terraform.CheckCoreVersionRequirements(config)
	if versionDiags.HasErrors() {
		view.Diagnostics(versionDiags)
		return 1
	}

	diags = diags.Append(earlyConfDiags)
	if earlyConfDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	diags = diags.Append(confDiags)
	if confDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Fetch the module manifest
	internalManifest, diags := c.internalManifest()
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Create a module reference resolver
	resolver := moduleref.NewResolver(internalManifest)

	// Crawl the Terraform config and find entries with references
	manifestWithRef := resolver.Resolve(config)

	// Render the new manifest with references
	return view.Display(*manifestWithRef)
}

// internalManifest will use the configuration loader to refresh and load the
// internal manifest.
func (c *ModulesCommand) internalManifest() (modsdir.Manifest, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	loader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to initialize config loader: %w", err))
		return nil, diags
	}

	if err = loader.RefreshModules(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to refresh module manifest: %w", err))
		return nil, diags
	}

	return loader.ModuleManifest(), diags
}

const modulesCommandHelp = `
Usage: terraform [global options] modules [options]

  Prints out a list of all declared Terraform modules and their resolved versions
  in a Terraform working directory.

Options:

  -json            If specified, output declared Terraform modules and
                   their resolved versions in a machine-readable format.
`
