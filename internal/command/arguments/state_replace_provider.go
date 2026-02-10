// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateReplaceProvider represents the command-line arguments for the
// "state replace-provider" command.
type StateReplaceProvider struct {
	AutoApprove         bool
	BackupPath          string
	StateLock           bool
	StateLockTimeout    time.Duration
	StatePath           string
	IgnoreRemoteVersion bool

	// Positional arguments: from and to provider FQNs
	From string
	To   string
}

// ParseStateReplaceProvider processes CLI arguments, returning a
// StateReplaceProvider value and diagnostics. If there are any diagnostics
// present, a StateReplaceProvider value is still returned representing the
// best effort interpretation of the arguments.
func ParseStateReplaceProvider(args []string) (*StateReplaceProvider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &StateReplaceProvider{
		BackupPath: "-",
		StateLock:  true,
	}

	cmdFlags := defaultFlagSet("state replace-provider")
	cmdFlags.BoolVar(&result.AutoApprove, "auto-approve", false, "skip interactive approval of replacements")
	cmdFlags.StringVar(&result.BackupPath, "backup", "-", "backup")
	cmdFlags.BoolVar(&result.StateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&result.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&result.StatePath, "state", "", "path")
	cmdFlags.BoolVar(&result.IgnoreRemoteVersion, "ignore-remote-version", false, "ignore remote version")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Exactly two arguments expected",
			"The state replace-provider command requires a from and to provider FQN.",
		))
	}

	if len(args) > 0 {
		result.From = args[0]
	}
	if len(args) > 1 {
		result.To = args[1]
	}

	return result, diags
}
