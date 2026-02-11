// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// UntaintCommand is a cli.Command implementation that manually untaints
// a resource, marking it as primary and ready for service.
type UntaintCommand struct {
	Meta
}

func (c *UntaintCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy Ui usage
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	// Parse command flags/args
	args, diags := arguments.ParseUntaint(rawArgs)

	// Instantiate the view even if there are parse errors
	view := views.NewUntaint(c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// Copy parsed values to Meta for backend/state operations
	c.Meta.statePath = args.StatePath
	c.Meta.stateOutPath = args.StateOutPath
	c.Meta.backupPath = args.BackupPath
	c.Meta.stateLock = args.Lock
	c.Meta.stateLockTimeout = args.LockTimeout
	c.Meta.ignoreRemoteVersion = args.IgnoreRemoteVersion

	addr := args.Addr

	// Load the backend
	b, backendDiags := c.backend(".", arguments.ViewHuman)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Determine the workspace name
	workspace, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Check remote Terraform version is compatible
	remoteVersionDiags := c.remoteVersionCheck(b, workspace)
	diags = diags.Append(remoteVersionDiags)
	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	// Get the state
	stateMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", sDiags.Err()))
		view.Diagnostics(diags)
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if lockDiags := stateLocker.Lock(stateMgr, "untaint"); lockDiags.HasErrors() {
			view.Diagnostics(lockDiags)
			return 1
		}
		defer func() {
			if unlockDiags := stateLocker.Unlock(); unlockDiags.HasErrors() {
				view.Diagnostics(unlockDiags)
			}
		}()
	}

	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Get the actual state structure
	state := stateMgr.State()
	if state.Empty() {
		if args.AllowMissing {
			view.AllowMissingWarning(addr)
			return 0
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No such resource instance",
			"The state currently contains no resource instances whatsoever. This may occur if the configuration has never been applied or if it has recently been destroyed.",
		))
		view.Diagnostics(diags)
		return 1
	}

	ss := state.SyncWrapper()

	// Get the resource and instance we're going to untaint
	rs := ss.Resource(addr.ContainingResource())
	is := ss.ResourceInstance(addr)
	if is == nil {
		if args.AllowMissing {
			view.AllowMissingWarning(addr)
			return 0
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No such resource instance",
			fmt.Sprintf("There is no resource instance in the state with the address %s. If the resource configuration has just been added, you must run \"terraform apply\" once to create the corresponding instance(s) before they can be tainted.", addr),
		))
		view.Diagnostics(diags)
		return 1
	}

	obj := is.Current
	if obj == nil {
		if len(is.Deposed) != 0 {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No such resource instance",
				fmt.Sprintf("Resource instance %s is currently part-way through a create_before_destroy replacement action. Run \"terraform apply\" to complete its replacement before tainting it.", addr),
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No such resource instance",
				fmt.Sprintf("Resource instance %s does not currently have a remote object associated with it, so it cannot be tainted.", addr),
			))
		}
		view.Diagnostics(diags)
		return 1
	}

	if obj.Status != states.ObjectTainted {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource instance is not tainted",
			fmt.Sprintf("Resource instance %s is not currently tainted, and so it cannot be untainted.", addr),
		))
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

	obj.Status = states.ObjectReady
	ss.SetResourceInstanceCurrent(addr, obj, rs.ProviderConfig)

	if err := stateMgr.WriteState(state); err != nil {
		diags = diags.Append(fmt.Errorf("Error writing state file: %s", err))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		diags = diags.Append(fmt.Errorf("Error writing state file: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	view.Diagnostics(diags)
	view.Success(addr)
	return 0
}

func (c *UntaintCommand) Help() string {
	helpText := `
Usage: terraform [global options] untaint [options] name

  Terraform uses the term "tainted" to describe a resource instance
  which may not be fully functional, either because its creation
  partially failed or because you've manually marked it as such using
  the "terraform taint" command.

  This command removes that state from a resource instance, causing
  Terraform to see it as fully-functional and not in need of
  replacement.

  This will not modify your infrastructure directly. It only avoids
  Terraform planning to replace a tainted instance in a future operation.

Options:

  -allow-missing          If specified, the command will succeed (exit code 0)
                          even if the resource is missing.

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

func (c *UntaintCommand) Synopsis() string {
	return "Remove the 'tainted' state from a resource instance"
}
