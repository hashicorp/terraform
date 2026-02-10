// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
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
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)

	// Propagate -no-color for legacy Ui usage
	if common.NoColor {
		c.Meta.Color = false
	}
	c.Meta.color = c.Meta.Color

	// Configure the view with the reconciled color setting
	c.View.Configure(&arguments.View{
		NoColor:         !c.Meta.color,
		CompactWarnings: common.CompactWarnings,
	})

	// Parse command flags/args
	args, diags := arguments.ParseStateReplaceProvider(rawArgs)

	// Instantiate the view even if flag parsing produced errors
	view := views.NewStateReplaceProvider(c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// Copy parsed args to appropriate Meta/StateMeta fields
	c.backupPath = args.BackupPath
	c.Meta.stateLock = args.StateLock
	c.Meta.stateLockTimeout = args.StateLockTimeout
	c.statePath = args.StatePath
	c.Meta.ignoreRemoteVersion = args.IgnoreRemoteVersion

	if diags := c.Meta.checkRequiredVersion(); diags != nil {
		view.Diagnostics(diags)
		return 1
	}

	// Parse from/to arguments into providers
	from, fromDiags := addrs.ParseProviderSourceString(args.From)
	if fromDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf(`Invalid "from" provider %q`, args.From),
			fromDiags.Err().Error(),
		))
	}
	to, toDiags := addrs.ParseProviderSourceString(args.To)
	if toDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf(`Invalid "to" provider %q`, args.To),
			toDiags.Err().Error(),
		))
	}
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Initialize the state manager as configured
	viewType := arguments.ViewHuman
	stateMgr, err := c.State(viewType)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}

	// Acquire lock if requested
	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if lockDiags := stateLocker.Lock(stateMgr, "state-replace-provider"); lockDiags.HasErrors() {
			view.Diagnostics(lockDiags)
			return 1
		}
		defer func() {
			if unlockDiags := stateLocker.Unlock(); unlockDiags.HasErrors() {
				view.Diagnostics(unlockDiags)
			}
		}()
	}

	// Refresh and load state
	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh source state: %s", err))
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		c.Ui.Error(errStateNotFound)
		return 1
	}

	// Fetch all resources from the state
	resources, resourceDiags := c.lookupAllResources(state)
	if resourceDiags.HasErrors() {
		view.Diagnostics(resourceDiags)
		return 1
	}

	var willReplace []*states.Resource

	// Update all matching resources with new provider
	for _, resource := range resources {
		if resource.ProviderConfig.Provider.Equals(from) {
			willReplace = append(willReplace, resource)
		}
	}
	view.Diagnostics(resourceDiags)

	if len(willReplace) == 0 {
		c.Ui.Output("No matching resources found.")
		return 0
	}

	// Explain the changes
	colorize := c.Colorize()
	c.Ui.Output("Terraform will perform the following actions:\n")
	c.Ui.Output(colorize.Color("  [yellow]~[reset] Updating provider:"))
	c.Ui.Output(colorize.Color(fmt.Sprintf("    [red]-[reset] %s", from)))
	c.Ui.Output(colorize.Color(fmt.Sprintf("    [green]+[reset] %s\n", to)))

	c.Ui.Output(colorize.Color(fmt.Sprintf("[bold]Changing[reset] %d resources:\n", len(willReplace))))
	for _, resource := range willReplace {
		c.Ui.Output(colorize.Color(fmt.Sprintf("  %s", resource.Addr)))
	}

	// Confirm
	if !args.AutoApprove {
		c.Ui.Output(colorize.Color(
			"\n[bold]Do you want to make these changes?[reset]\n" +
				"Only 'yes' will be accepted to continue.\n",
		))
		v, err := c.Ui.Ask("Enter a value:")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error asking for approval: %s", err))
			return 1
		}
		if v != "yes" {
			c.Ui.Output("Cancelled replacing providers.")
			return 0
		}
	}

	// Update the provider for each resource
	for _, resource := range willReplace {
		resource.ProviderConfig.Provider = to
	}

	// Load the backend
	b, backendDiags := c.backend(".", viewType)
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
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	view.Diagnostics(diags)
	c.Ui.Output(fmt.Sprintf("\nSuccessfully replaced provider for %d resources.", len(willReplace)))
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

  -state, state-out, and -backup are legacy options supported for the local
  backend only. For more information, see the local backend's documentation.

`
	return strings.TrimSpace(helpText)
}

func (c *StateReplaceProviderCommand) Synopsis() string {
	return "Replace provider in the state"
}
