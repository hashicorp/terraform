// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WorkspaceDelete represent flags and arguments specific to the `terraform workspace delete` command.
type WorkspaceDelete struct {
	Workspace

	// Flags
	Lock        bool
	LockTimeout time.Duration
	Force       bool

	// Positional arguments
	Name string
}

// ParseWorkspaceDelete processes CLI arguments, returning a WorkspaceDelete value and errors.
// If errors are encountered, an WorkspaceDelete value is still returned representing
// the best effort interpretation of the arguments.
func ParseWorkspaceDelete(args []string) (*WorkspaceDelete, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var force bool
	var stateLock bool
	var stateLockTimeout time.Duration
	cmdFlags := defaultFlagSet("workspace delete")
	cmdFlags.BoolVar(&force, "force", false, "force removal of a non-empty workspace")
	cmdFlags.BoolVar(&stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&stateLockTimeout, "lock-timeout", 0, "lock timeout")
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// `workspace delete` takes only one positional argument: workspace name.
	args = cmdFlags.Args()
	var name string
	if len(args) == 0 {
		diags = diags.Append(errors.New("Expected a single argument: NAME.")) // Recreating pre-existing error from command package
	} else {

		// Obtain and validate name argument
		//
		// We purposefully don't use ValidWorkspaceName here; if a user
		// creates a workspace with an invalid name they should be able to
		// delete it easily.
		name = args[0]
		if name == "" {
			diags = diags.Append(fmt.Errorf("Expected a workspace name as an argument, instead got an empty string: %q\n", args[0]))
		}

		args = args[1:]
		if len(args) != 0 {
			diags = diags.Append(errors.New("Expected a single argument: NAME."))
		}
	}

	return &WorkspaceDelete{
		Workspace:   Workspace{ViewType: ViewHuman},
		Name:        name,
		Lock:        stateLock,
		LockTimeout: stateLockTimeout,
		Force:       force,
	}, diags
}
