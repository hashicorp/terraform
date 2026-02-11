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

// TaintCommand is a cli.Command implementation that manually taints
// a resource, marking it for recreation.
type TaintCommand struct {
	Meta
}

func (c *TaintCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy Ui usage
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	// Parse command flags/args
	args, diags := arguments.ParseTaint(rawArgs)

	// Instantiate the view even if there are parse errors
	view := views.NewTaint(c.View)

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

	if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid resource address",
			fmt.Sprintf("Resource instance %s cannot be tainted", addr),
		))
		view.Diagnostics(diags)
		return 1
	}

	if diags := c.Meta.checkRequiredVersion(); diags != nil {
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
		if diags := stateLocker.Lock(stateMgr, "taint"); diags.HasErrors() {
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

	// Get schemas, if possible, before writing state
	var schemas *terraform.Schemas
	if isCloudMode(b) {
		var schemaDiags tfdiags.Diagnostics
		schemas, schemaDiags = c.MaybeGetSchemas(state, nil)
		diags = diags.Append(schemaDiags)
	}

	ss := state.SyncWrapper()

	// Get the resource and instance we're going to taint
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

	obj.Status = states.ObjectTainted
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

func (c *TaintCommand) Help() string {
	helpText := `
Usage: terraform [global options] taint [options] <address>

  Terraform uses the term "tainted" to describe a resource instance
  which may not be fully functional, either because its creation
  partially failed or because you've manually marked it as such using
  this command.

  This will not modify your infrastructure directly, but subsequent
  Terraform plans will include actions to destroy the remote object
  and create a new object to replace it.

  You can remove the "taint" state from a resource instance using
  the "terraform untaint" command.

  The address is in the usual resource address syntax, such as:
    aws_instance.foo
    aws_instance.bar[1]
    module.foo.module.bar.aws_instance.baz

  Use your shell's quoting or escaping syntax to ensure that the
  address will reach Terraform correctly, without any special
  interpretation.

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

func (c *TaintCommand) Synopsis() string {
	return "Mark a resource instance as not fully functional"
}
