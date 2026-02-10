// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateRm represents the command-line arguments for the "state rm" command.
type StateRm struct {
	DryRun              bool
	BackupPath          string
	StateLock           bool
	StateLockTimeout    time.Duration
	StatePath           string
	IgnoreRemoteVersion bool

	// Addrs contains the positional arguments: one or more resource addresses.
	Addrs []string
}

// ParseStateRm processes CLI arguments, returning a StateRm value and
// diagnostics. If there are any diagnostics present, a StateRm value is still
// returned representing the best effort interpretation of the arguments.
func ParseStateRm(args []string) (*StateRm, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &StateRm{
		BackupPath: "-",
		StateLock:  true,
	}

	cmdFlags := defaultFlagSet("state rm")
	cmdFlags.BoolVar(&result.DryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&result.BackupPath, "backup", "-", "backup")
	cmdFlags.BoolVar(&result.StateLock, "lock", true, "lock state")
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
	if len(args) < 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"At least one address required",
			"The state rm command requires one or more resource addresses as arguments.",
		))
	}

	result.Addrs = args

	return result, diags
}
