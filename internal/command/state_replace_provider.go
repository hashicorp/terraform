// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateReplaceProviderCommand is a Command implementation that allows users
// to change the provider associated with existing resources. This is only
// likely to be useful if a provider is forked or changes its fully-qualified
// name.

type StateReplaceProviderCommand struct {
	StateMeta
}

func (c *StateReplaceProviderCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	parsedArgs, parseDiags := arguments.ParseStateReplaceProvider(rawArgs)
	diags = diags.Append(parseDiags)

	// Prepare the view
	view := views.NewStateReplaceProvider(parsedArgs.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	c.backupPath = parsedArgs.BackupPath
	c.Meta.stateLock = parsedArgs.StateLock
	c.Meta.stateLockTimeout = parsedArgs.StateLockTimeout
	c.statePath = parsedArgs.StatePath
	c.Meta.ignoreRemoteVersion = parsedArgs.IgnoreRemoteVersion

	loader, err := c.initConfigLoader()
	if err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	var varDiags tfdiags.Diagnostics
	c.VariableValues, varDiags = parsedArgs.Vars.CollectValues(func(filename string, src []byte) {
		loader.Parser().ForceFileSource(filename, src)
	})
	if varDiags.HasErrors() {
		view.Diagnostics(varDiags)
		return 1
	}

	if diags := c.Meta.checkRequiredVersion(); diags != nil {
		view.Diagnostics(diags)
		return 1
	}

	// Parse from/to arguments into providers
	from, fromDiags := addrs.ParseProviderSourceString(parsedArgs.FromProviderAddr)
	if fromDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf(`Invalid "from" provider %q`, parsedArgs.FromProviderAddr),
			fromDiags.Err().Error(),
		))
	}
	to, toDiags := addrs.ParseProviderSourceString(parsedArgs.ToProviderAddr)
	if toDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf(`Invalid "to" provider %q`, parsedArgs.ToProviderAddr),
			toDiags.Err().Error(),
		))
	}
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Initialize the state manager as configured
	stateMgr, err := c.State(parsedArgs.ViewType)
	if err != nil {
		diags = diags.Append(fmt.Errorf(errStateLoadingState, err))
		view.Diagnostics(diags)
		return 1
	}

	// Acquire lock if requested
	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(parsedArgs.ViewType, c.View))
		if diags := stateLocker.Lock(stateMgr, "state-replace-provider"); diags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		defer func() {
			if diags := stateLocker.Unlock(); diags.HasErrors() {
				view.Diagnostics(diags)
			}
		}()
	}

	// Refresh and load state
	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to refresh the state",
			fmt.Sprintf("Error refreshing the source state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		diags = diags.Append(errStateNotFound)
		view.Diagnostics(diags)
		return 1
	}

	// Fetch all resources from the state
	resources, diags := c.lookupAllResources(state)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	var willReplace []*states.Resource

	// Update all matching resources with new provider
	for _, resource := range resources {
		if resource.ProviderConfig.Provider.Equals(from) {
			willReplace = append(willReplace, resource)
		}
	}
	view.Diagnostics(diags)

	view.DescribeChanges(from, to, willReplace)

	if len(willReplace) == 0 {
		// Nothing more to do
		return 0
	}

	// Confirm
	if !parsedArgs.AutoApprove {
		v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
			Id: "replace-provider",
			Query: "\n[bold]Do you want to make these changes?[reset]\n" +
				"Only 'yes' will be accepted to continue.\n",
		})
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to confirm provider replacement",
				fmt.Sprintf("Error asking for approval: %s", err),
			))
			view.Diagnostics(diags)
			return 1
		}
		if v != "yes" {
			view.Cancelled()
			return 0
		}
	}

	// Update the provider for each resource
	for _, resource := range willReplace {
		resource.ProviderConfig.Provider = to
	}

	// Load the backend
	b, backendDiags := c.backend(".", parsedArgs.ViewType)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Get schemas, if possible, before writing state
	var schemas *terraform.Schemas
	if isCloudMode(b) {
		var schemaDiags tfdiags.Diagnostics
		schemas, schemaDiags = c.MaybeGetSchemas(state, nil)
		diags = diags.Append(schemaDiags)
	}

	// Write the updated state
	if err := stateMgr.WriteState(state); err != nil {
		diags = diags.Append(stateReplaceProviderPersistErrDiag(err))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		diags = diags.Append(stateReplaceProviderPersistErrDiag(err))
		view.Diagnostics(diags)
		return 1
	}

	view.Diagnostics(diags)
	view.Success(len(willReplace))
	return 0
}

func (c *StateReplaceProviderCommand) Help() string {
	helpText := `
Usage: terraform [global options] state replace-provider [options] FROM_PROVIDER_FQN TO_PROVIDER_FQN

  Replace provider for resources in the Terraform state.

Options:

  -auto-approve           Skip interactive approval.

  -lock=false             Don't hold a state lock during the operation. This is
                          dangerous if others might concurrently run commands
                          against the same workspace.

  -lock-timeout=0s        Duration to retry a state lock.

  -ignore-remote-version  A rare option used for the remote backend only. See
                          the remote backend documentation for more information.

  -var 'foo=bar'          Set a value for one of the input variables in the root
                          module of the configuration. Use this option more than
                          once to set more than one variable.

  -var-file=filename      Load variable values from the given file, in addition
                          to the default files terraform.tfvars and *.auto.tfvars.
                          Use this option more than once to include more than one
                          variables file.

  -state, state-out, and -backup are legacy options supported for the local
  backend only. For more information, see the local backend's documentation.

`
	return strings.TrimSpace(helpText)
}

func (c *StateReplaceProviderCommand) Synopsis() string {
	return "Replace provider in the state"
}

func stateReplaceProviderPersistErrDiag(err error) tfdiags.Diagnostic {
	return tfdiags.Sourceless(
		tfdiags.Error,
		"Failed to save the state",
		fmt.Sprintf(`Terraform encountered an error when saving state: %s

The state was not saved. No items were changed in the persisted
state. Please resolve the issue above and try again.`, err),
	)
}
