// Copyright IBM Corp. 2014, 2026
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

// UntaintCommand is a cli.Command implementation that manually untaints
// a resource, marking it as primary and ready for service.
type UntaintCommand struct {
	Meta
}

func (c *UntaintCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments.
	parsedArgs, parseDiags := arguments.ParseUntaint(rawArgs)
	diags = diags.Append(parseDiags)

	// Prepare the view
	view := views.NewUntaint(parsedArgs.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Copy parsed flags to Meta
	c.Meta.backupPath = parsedArgs.BackupPath
	c.Meta.stateLock = parsedArgs.StateLock
	c.Meta.stateLockTimeout = parsedArgs.StateLockTimeout
	c.Meta.statePath = parsedArgs.StatePath
	c.Meta.stateOutPath = parsedArgs.StateOutPath
	c.Meta.ignoreRemoteVersion = parsedArgs.IgnoreRemoteVersion

	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(parsedArgs.Address)
	diags = diags.Append(addrDiags)
	if addrDiags.HasErrors() {
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

	// Determine the workspace name
	workspace, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error determining the current workspace",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Check remote Terraform version is compatible
	remoteVersionDiags := c.remoteVersionCheck(b, workspace)
	diags = diags.Append(remoteVersionDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Get a state manager for the current workspace
	stateMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error accessing state",
			fmt.Sprintf("Failed to access state: %s", sDiags.Err()),
		))
		view.Diagnostics(diags)
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if lockDiags := stateLocker.Lock(stateMgr, "untaint"); lockDiags.HasErrors() {
			diags = diags.Append(lockDiags)
			view.Diagnostics(diags)
			return 1
		}
		defer func() {
			if unlockDiags := stateLocker.Unlock(); unlockDiags.HasErrors() {
				diags = diags.Append(unlockDiags)
				view.Diagnostics(diags)
			}
		}()
	}

	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error loading state",
			fmt.Sprintf("Failed to load state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Get the actual state structure
	state := stateMgr.State()
	if state.Empty() {
		if parsedArgs.AllowMissing {
			diags = diags.Append(c.allowMissingWarning(addr))
			view.Diagnostics(diags)
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

	// Get the resource and instance we're going to taint
	rs := ss.Resource(addr.ContainingResource())
	is := ss.ResourceInstance(addr)
	if is == nil {
		if parsedArgs.AllowMissing {
			diags = diags.Append(c.allowMissingWarning(addr))
			view.Diagnostics(diags)
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
			// Don't know why we're here, but we'll produce a generic error message anyway.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No such resource instance",
				fmt.Sprintf("Resource instance %s does not currently have a remote object associated with it, so it cannot be untainted.", addr),
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
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error writing state file",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error writing state file",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	view.Untaint(addr, diags)
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

  -compact-warnings       If Terraform produces any warnings that are not
                          accompanied by errors, show them in a more compact
                          form that includes only the summary messages

  -ignore-remote-version  A rare option used for the remote backend only. See
                          the remote backend documentation for more information.

  -lock=false             Don't hold a state lock during the operation. This is
                          dangerous if others might concurrently run commands
                          against the same workspace.

  -lock-timeout=0s        Duration to retry a state lock.

  -no-color               Disable text coloring in the output.

  -state, state-out, and -backup are legacy options supported for the local
  backend only. For more information, see the local backend's documentation.

`
	return strings.TrimSpace(helpText)
}

func (c *UntaintCommand) Synopsis() string {
	return "Remove the 'tainted' state from a resource instance"
}

func (c *UntaintCommand) allowMissingWarning(name addrs.AbsResourceInstance) tfdiags.Diagnostic {
	return tfdiags.Sourceless(
		tfdiags.Warning,
		"No such resource instance",
		fmt.Sprintf("Resource instance %s was not found, but this is not an error because -allow-missing was set.", name),
	)
}
