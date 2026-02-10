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
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateRmCommand is a Command implementation that removes
// a single resource from the state.
type StateRmCommand struct {
	StateMeta
}

func (c *StateRmCommand) Run(rawArgs []string) int {
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
	args, diags := arguments.ParseStateRm(rawArgs)

	// Instantiate the view even if flag parsing produced errors
	view := views.NewStateRm(c.View)

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

	// Get the state
	viewType := arguments.ViewHuman
	stateMgr, err := c.State(viewType)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if diags := stateLocker.Lock(stateMgr, "state-rm"); diags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		defer func() {
			if diags := stateLocker.Unlock(); diags.HasErrors() {
				view.Diagnostics(diags)
			}
		}()
	}

	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh state: %s", err))
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		c.Ui.Error(errStateNotFound)
		return 1
	}

	// This command primarily works with resource instances, though it will
	// also clean up any modules and resources left empty by actions it takes.
	var resolvedAddrs []addrs.AbsResourceInstance
	for _, addrStr := range args.Addrs {
		moreAddrs, moreDiags := c.lookupResourceInstanceAddr(state, true, addrStr)
		resolvedAddrs = append(resolvedAddrs, moreAddrs...)
		diags = diags.Append(moreDiags)
	}
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	prefix := "Removed "
	if args.DryRun {
		prefix = "Would remove "
	}

	var isCount int
	ss := state.SyncWrapper()
	for _, addr := range resolvedAddrs {
		isCount++
		c.Ui.Output(prefix + addr.String())
		if !args.DryRun {
			ss.ForgetResourceInstanceAll(addr)
			ss.RemoveResourceIfEmpty(addr.ContainingResource())
		}
	}

	if args.DryRun {
		if isCount == 0 {
			c.Ui.Output("Would have removed nothing.")
		}
		return 0 // This is as far as we go in dry-run mode
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

	if err := stateMgr.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	if len(diags) > 0 && isCount != 0 {
		view.Diagnostics(diags)
	}

	if isCount == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid target address",
			"No matching objects found. To view the available instances, use \"terraform state list\". Please modify the address to reference a specific instance.",
		))
		view.Diagnostics(diags)
		return 1
	}

	c.Ui.Output(fmt.Sprintf("Successfully removed %d resource instance(s).", isCount))
	return 0
}

func (c *StateRmCommand) Help() string {
	helpText := `
Usage: terraform [global options] state rm [options] ADDRESS...

  Remove one or more items from the Terraform state, causing Terraform to
  "forget" those items without first destroying them in the remote system.

  This command removes one or more resource instances from the Terraform state
  based on the addresses given. You can view and list the available instances
  with "terraform state list".

  If you give the address of an entire module then all of the instances in
  that module and any of its child modules will be removed from the state.

  If you give the address of a resource that has "count" or "for_each" set,
  all of the instances of that resource will be removed from the state.

Options:

  -dry-run                If set, prints out what would've been removed but
                          doesn't actually remove anything.

  -backup=PATH            Path where Terraform should write the backup
                          state.

  -lock=false             Don't hold a state lock during the operation. This is
                          dangerous if others might concurrently run commands
                          against the same workspace.

  -lock-timeout=0s        Duration to retry a state lock.

  -state=PATH             Path to the state file to update. Defaults to the
                          current workspace state.
                          Legacy option for the local backend only. See
                          the local backend's documentation for more
                          information.

  -ignore-remote-version  Continue even if remote and local Terraform versions
                          are incompatible. This may result in an unusable
                          workspace, and should be used with extreme caution.

`
	return strings.TrimSpace(helpText)
}

func (c *StateRmCommand) Synopsis() string {
	return "Remove instances from the state"
}

const errStateRmPersist = `Error saving the state: %s

The state was not saved. No items were removed from the persisted
state. No backup was created since no modification occurred. Please
resolve the issue above and try again.`
