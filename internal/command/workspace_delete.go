// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceDeleteCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceDeleteCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics
	diags = diags.Append(envCommandWarningDiag(c.LegacyName))

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	args, argDiags := arguments.ParseWorkspaceDelete(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewWorkspaceDelete(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	configPath := c.WorkingDir.RootModuleDir()
	b, backendDiags := c.backend(configPath, args.ViewType)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	workspaces, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Is the user attempting to delete a workspace that doesn't exist?
	workspace := args.Name
	exists := false
	for _, ws := range workspaces {
		if workspace == ws {
			exists = true
			break
		}
	}

	if !exists {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(envDoesNotExist), workspace))
		view.Diagnostics(diags)
		return 1
	}

	// Is the user attempting to delete the currently selected workspace?
	currentWorkspace, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
		view.Diagnostics(diags)
		return 1
	}
	if workspace == currentWorkspace {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(envDelCurrent), workspace))
		view.Diagnostics(diags)
		return 1
	}

	// Is the user attempting to delete the default workspace?
	if workspace == backend.DefaultStateName {
		diags = diags.Append(fmt.Errorf("Cannot delete the default workspace"))
		view.Diagnostics(diags)
		return 1
	}

	// Check if the workspace's state is empty or not
	stateMgr, sDiags := b.StateMgr(workspace)
	diags = diags.Append(sDiags)
	if sDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	var stateLocker clistate.Locker
	if args.Lock {
		stateLocker = clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(args.ViewType, c.View))
		if lDiags := stateLocker.Lock(stateMgr, "state-replace-provider"); lDiags.HasErrors() {
			diags = diags.Append(lDiags)
			view.Diagnostics(diags)
			return 1
		}
	} else {
		stateLocker = clistate.NewNoopLocker()
	}

	if err := stateMgr.RefreshState(); err != nil {
		// We need to release the lock before exit
		stateLocker.Unlock()
		diags = diags.Append(fmt.Errorf("Error refreshing state: %w", err))
		view.Diagnostics(diags)
		return 1
	}

	hasResources := stateMgr.State().HasManagedResourceInstanceObjects()

	if hasResources && !args.Force {
		// We'll collect a list of what's being managed here as extra context
		// for the message.
		var buf strings.Builder
		for _, obj := range stateMgr.State().AllManagedResourceInstanceObjectAddrs() {
			if obj.DeposedKey == states.NotDeposed {
				fmt.Fprintf(&buf, "\n  - %s", obj.ResourceInstance.String())
			} else {
				fmt.Fprintf(&buf, "\n  - %s (deposed object %s)", obj.ResourceInstance.String(), obj.DeposedKey)
			}
		}

		// We need to release the lock before exit
		stateLocker.Unlock()

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Workspace is not empty",
			fmt.Sprintf(
				"Workspace %q is currently tracking the following resource instances:%s\n\nDeleting this workspace would cause Terraform to lose track of any associated remote objects, which would then require you to delete them manually outside of Terraform. You should destroy these objects with Terraform before deleting the workspace.\n\nIf you want to delete this workspace anyway, and have Terraform forget about these managed objects, use the -force option to disable this safety check.",
				workspace, buf.String(),
			),
		))
		view.Diagnostics(diags)
		return 1
	}

	// We need to release the lock just before deleting the state, in case
	// the backend can't remove the resource while holding the lock. This
	// is currently true for Windows local files.
	//
	// TODO: While there is little safety in locking while deleting the
	// state, it might be nice to be able to coordinate processes around
	// state deletion, i.e. in a CI environment. Adding Delete() as a
	// required method of States would allow the removal of the resource to
	// be delegated from the Backend to the State itself.
	stateLocker.Unlock()

	dwDiags := b.DeleteWorkspace(workspace, args.Force)
	diags = diags.Append(dwDiags)
	if dwDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	if hasResources {
		// A user forced this deletion, but lets warn them of the consequence of the force flag.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			fmt.Sprintf("Workspace %q was non-empty", workspace),
			`The resources managed by the deleted workspace may still exist, but are no longer manageable by Terraform since the state has been deleted.`,
		))
	}

	view.LogWorkspaceDeletionSuccess(workspace, diags)
	return 0
}

func (c *WorkspaceDeleteCommand) AutocompleteArgs() complete.Predictor {
	return completePredictSequence{
		c.completePredictWorkspaceName(),
		complete.PredictDirs(""),
	}
}

func (c *WorkspaceDeleteCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-force": complete.PredictNothing,
	}
}

func (c *WorkspaceDeleteCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace delete [OPTIONS] NAME

  Delete a Terraform workspace


Options:

  -force             Remove a workspace even if it is managing resources.
                     Terraform can no longer track or manage the workspace's
                     infrastructure.

  -lock=false        Don't hold a state lock during the operation. This is
                     dangerous if others might concurrently run commands
                     against the same workspace.

  -lock-timeout=0s   Duration to retry a state lock.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceDeleteCommand) Synopsis() string {
	return "Delete a workspace"
}
