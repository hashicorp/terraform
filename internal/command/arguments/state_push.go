// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePush represents the command-line arguments for the "state push" command.
type StatePush struct {
	Force               bool
	StateLock           bool
	StateLockTimeout    time.Duration
	IgnoreRemoteVersion bool

	// Path is the positional argument: the path to the state file to push,
	// or "-" for stdin.
	Path string
}

// ParseStatePush processes CLI arguments, returning a StatePush value and
// diagnostics. If there are any diagnostics present, a StatePush value is still
// returned representing the best effort interpretation of the arguments.
func ParseStatePush(args []string) (*StatePush, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &StatePush{
		StateLock: true,
	}

	cmdFlags := defaultFlagSet("state push")
	cmdFlags.BoolVar(&result.Force, "force", false, "")
	cmdFlags.BoolVar(&result.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&result.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.BoolVar(&result.IgnoreRemoteVersion, "ignore-remote-version", false, "ignore remote version")

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
			"Exactly one argument expected",
			"The state push command requires a path to a local state file to push. Use \"-\" to read from stdin.",
		))
	}

	if len(args) > 0 {
		result.Path = args[0]
	}

	return result, diags
}
