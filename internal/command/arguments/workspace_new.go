// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WorkspaceNew represent flags and arguments specific to the `terraform workspace new` command.
type WorkspaceNew struct {
	Workspace

	// Flags
	Lock        bool
	LockTimeout time.Duration
	StatePath   string

	// Positional arguments
	Name string
}

// ParseWorkspaceNew processes CLI arguments, returning a WorkspaceNew value and errors.
// If errors are encountered, an WorkspaceNew value is still returned representing
// the best effort interpretation of the arguments.
func ParseWorkspaceNew(args []string) (*WorkspaceNew, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var stateLock bool
	var stateLockTimeout time.Duration
	var statePath string
	cmdFlags := defaultFlagSet("workspace new")
	cmdFlags.BoolVar(&stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&statePath, "state", "", "terraform state file")
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// `workspace new` takes only one positional argument: workspace name.
	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(errors.New("Expected a single argument: NAME.")) // Recreating pre-existing error from command package
	}

	// Obtain and validate name argument, but only if there is the expected number of arguments.
	var name string
	if len(args) == 1 {
		name = args[0]
		if !ValidWorkspaceName(name) {
			diags = diags.Append(fmt.Errorf(EnvInvalidName, name))
		}
	}

	return &WorkspaceNew{
		Workspace:   Workspace{ViewType: ViewHuman},
		Name:        name,
		Lock:        stateLock,
		LockTimeout: stateLockTimeout,
		StatePath:   statePath,
	}, diags
}
