// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type QueryCommand struct {
	Meta
}

func (c *QueryCommand) Help() string {
	helpText := `
Usage: terraform [global options] query [options]

  Queries the remote infrastructure for resources.

  Terraform will search for .tfquery.hcl files within the current configuration.
  Terraform will then use the configured providers to query the remote
  infrastructure for resources that match the defined list blocks. The results
  will be printed to the terminal and optionally can be used to generate
  configuration.

Query Customization Options:

  The following options customize how Terraform will run the query.

  -var 'foo=bar'        Set a value for one of the input variables in the query
                        file of the configuration. Use this option more than
                        once to set more than one variable.

  -var-file=filename    Load variable values from the given file, in addition
                        to the default files terraform.tfvars and *.auto.tfvars.
                        Use this option more than once to include more than one
                        variables file.

Other Options:

  -generate-config-out=path  Instructs Terraform to generate import and resource
                             blocks for any found results. The configuration is
                             written to a new file at PATH, which must not
                             already exist. When this option is used with the
                             json option, the generated configuration will be
                             part of the JSON output instead of written to a
                             file.

  -json                      If specified, machine readable output will be
                             printed in JSON format

  -no-color                  If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *QueryCommand) Synopsis() string {
	return "Search and list remote infrastructure with Terraform"
}

func (c *QueryCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy use of Ui.  The remote backend and
	// cloud package use this; it should be removed when/if they are
	// migrated to views.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color
	c.Meta.includeQueryFiles = true

	// Parse and validate flags
	args, diags := arguments.ParseQuery(rawArgs)

	// Instantiate the view, even if there are flag errors, so that we render
	// diagnostics according to the desired view
	view := views.NewQuery(args.ViewType, c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// We currently don't support the paralleism flag in the query command,
	// so we set it to the default value here. This avoids backend errors
	// that check for deviant values.
	c.Meta.parallelism = DefaultParallelism

	// Prepare the backend with the backend-specific arguments
	be, beDiags := c.PrepareBackend(args.State, args.ViewType)
	b, isRemoteBackend := be.(BackendWithRemoteTerraformVersion)
	if isRemoteBackend && !b.IsLocalOperations() {
		diags = diags.Append(c.providerDevOverrideRuntimeWarningsRemoteExecution())
	} else {
		diags = diags.Append(c.providerDevOverrideRuntimeWarnings())
	}
	diags = diags.Append(beDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation request
	opReq, opDiags := c.OperationRequest(be, view, args.ViewType, args.GenerateConfigPath)
	diags = diags.Append(opDiags)
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

	// Collect variable value and add them to the operation request
	var varDiags tfdiags.Diagnostics
	opReq.Variables, varDiags = args.Vars.CollectValues(func(filename string, src []byte) {
		loader.Parser().ForceFileSource(filename, src)
	})
	diags = diags.Append(varDiags)

	// Before we delegate to the backend, we'll print any warning diagnostics
	// we've accumulated here, since the backend will start fresh with its own
	// diagnostics.
	view.Diagnostics(diags)
	diags = nil

	// Perform the operation
	op, err := c.RunOperation(be, opReq)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	if op.Result != backendrun.OperationSuccess {
		return op.Result.ExitStatus()
	}

	return op.Result.ExitStatus()
}

func (c *QueryCommand) PrepareBackend(args *arguments.State, viewType arguments.ViewType) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	// Load the backend
	be, diags := c.backend(".", viewType)
	return be, diags
}

func (c *QueryCommand) OperationRequest(
	be backendrun.OperationsBackend,
	view views.Query,
	viewType arguments.ViewType,
	generateConfigOut string,
) (*backendrun.Operation, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Build the operation
	opReq := c.Operation(be, viewType)
	opReq.Hooks = view.Hooks()
	opReq.ConfigDir = "."
	opReq.Type = backendrun.OperationTypePlan
	opReq.GenerateConfigOut = generateConfigOut
	opReq.View = view.Operation()
	opReq.Query = true

	var err error
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to initialize config loader: %s", err))
		return nil, diags
	}

	return opReq, diags
}
