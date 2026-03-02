// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WorkspaceNew represents the command-line arguments for the workspace new
// command.
type WorkspaceNew struct {
	StateLock        bool
	StateLockTimeout time.Duration
	StatePath        string
	Name             string
	Args             []string
}

// ParseWorkspaceNew processes CLI arguments, returning a WorkspaceNew value
// and errors. If errors are encountered, a WorkspaceNew value is still
// returned representing the best effort interpretation of the arguments.
func ParseWorkspaceNew(args []string) (*WorkspaceNew, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	wn := &WorkspaceNew{}

	cmdFlags := defaultFlagSet("workspace new")
	cmdFlags.BoolVar(&wn.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&wn.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&wn.StatePath, "state", "", "terraform state file")

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
		return wn, diags
	}

	wn.Name = args[0]
	wn.Args = args[1:]

	return wn, diags
}
