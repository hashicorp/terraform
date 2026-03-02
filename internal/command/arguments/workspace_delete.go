// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WorkspaceDelete represents the command-line arguments for the workspace
// delete command.
type WorkspaceDelete struct {
	Force            bool
	StateLock        bool
	StateLockTimeout time.Duration
	Name             string
	Args             []string
}

// ParseWorkspaceDelete processes CLI arguments, returning a WorkspaceDelete
// value and errors. If errors are encountered, a WorkspaceDelete value is still
// returned representing the best effort interpretation of the arguments.
func ParseWorkspaceDelete(args []string) (*WorkspaceDelete, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	wd := &WorkspaceDelete{}

	cmdFlags := defaultFlagSet("workspace delete")
	cmdFlags.BoolVar(&wd.Force, "force", false, "force removal of a non-empty workspace")
	cmdFlags.BoolVar(&wd.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&wd.StateLockTimeout, "lock-timeout", 0, "lock timeout")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid arguments",
			"Expected a single argument: NAME.",
		))
		return wd, diags
	}

	if args[0] == "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid arguments",
			"Expected a workspace name as an argument, instead got an empty string.",
		))
		return wd, diags
	}

	wd.Name = args[0]
	wd.Args = args[1:]

	return wd, diags
}
