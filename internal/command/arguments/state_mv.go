// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateMv represents the command-line arguments for the "state mv" command.
type StateMv struct {
	DryRun              bool
	BackupPath          string
	BackupPathOut       string
	StateLock           bool
	StateLockTimeout    time.Duration
	StatePath           string
	StatePathOut        string
	IgnoreRemoteVersion bool

	// Positional arguments: source and destination addresses
	Source      string
	Destination string
}

// ParseStateMv processes CLI arguments, returning a StateMv value and
// diagnostics. If there are any diagnostics present, a StateMv value is still
// returned representing the best effort interpretation of the arguments.
func ParseStateMv(args []string) (*StateMv, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &StateMv{
		BackupPath:    "-",
		BackupPathOut: "-",
		StateLock:     true,
	}

	cmdFlags := defaultFlagSet("state mv")
	cmdFlags.BoolVar(&result.DryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&result.BackupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&result.BackupPathOut, "backup-out", "-", "backup")
	cmdFlags.BoolVar(&result.StateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&result.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&result.StatePath, "state", "", "path")
	cmdFlags.StringVar(&result.StatePathOut, "state-out", "", "path")
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
			"The state mv command requires a source and destination address.",
		))
	}

	if len(args) > 0 {
		result.Source = args[0]
	}
	if len(args) > 1 {
		result.Destination = args[1]
	}

	return result, diags
}
