// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states/statemgr"

	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// UnlockCommand is a cli.Command implementation that manually unlocks
// the state.
type UnlockCommand struct {
	Meta
}

func (c *UnlockCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	args, argDiags := arguments.ParseUnlock(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewUnlock(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	diags = diags.Append(argDiags)
	if argDiags.HasErrors() {
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

	// This is a read-only operation with respect to the state contents
	c.ignoreRemoteVersionConflict(b)

	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to determine the current workspace",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}
	stateMgr, sDiags := b.StateMgr(env)
	if sDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to access state",
			fmt.Sprintf("Failed to access state: %s", sDiags.Err()),
		))
		view.Diagnostics(diags)
		return 1
	}

	_, isLocal := stateMgr.(*statemgr.Filesystem)

	if !args.Force {
		// Forcing this doesn't do anything, but doesn't break anything either,
		// and allows us to run the basic command test too.
		if isLocal {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to unlock local state",
				"Local state cannot be unlocked by another process"))
			view.Diagnostics(diags)
			return 1
		}

		desc := "Terraform will remove the lock on the remote state.\n" +
			"This will allow local Terraform commands to modify this state, even though it\n" +
			"may still be in use. Only 'yes' will be accepted to confirm."

		v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
			Id:          "force-unlock",
			Query:       "Do you really want to force-unlock?",
			Description: desc,
		})
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to confirm forced unlock",
				fmt.Sprintf("Error asking for confirmation: %s", err),
			))
			view.Diagnostics(diags)
			return 1
		}
		if v != "yes" {
			view.ForceUnlockCancelled(diags)
			return 1
		}
	}

	if err := stateMgr.Unlock(args.LockID); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to unlock state",
			fmt.Sprintf("Failed to unlock state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	view.UnlockSuccessful(diags)
	return 0
}

func (c *UnlockCommand) Help() string {
	helpText := `
Usage: terraform [global options] force-unlock LOCK_ID

  Manually unlock the state for the defined configuration.

  This will not modify your infrastructure. This command removes the lock on the
  state for the current workspace. The behavior of this lock is dependent
  on the backend being used. Local state files cannot be unlocked by another
  process.

Options:

  -force                 Don't ask for input for unlock confirmation.
`
	return strings.TrimSpace(helpText)
}

func (c *UnlockCommand) Synopsis() string {
	return "Release a stuck lock on the current workspace"
}
