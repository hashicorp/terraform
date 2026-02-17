// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateMv represents the command-line arguments for the state mv command.
type StateMv struct {
	// DryRun, if true, prints out what would be moved without actually
	// moving anything.
	DryRun bool

	// BackupPath is the path where Terraform should write the backup state.
	BackupPath string

	// BackupOutPath is the path where Terraform should write the backup of
	// the destination state.
	BackupOutPath string

	// StateLock, if true, requests that the backend lock the state for this
	// operation.
	StateLock bool

	// StateLockTimeout is the duration to retry a state lock.
	StateLockTimeout time.Duration

	// StatePath is an optional path to a local state file.
	StatePath string

	// StateOutPath is an optional path to write the destination state.
	StateOutPath string

	// IgnoreRemoteVersion, if true, continues even if remote and local
	// Terraform versions are incompatible.
	IgnoreRemoteVersion bool

	// SourceAddr is the source resource address.
	SourceAddr string

	// DestAddr is the destination resource address.
	DestAddr string
}

// ParseStateMv processes CLI arguments, returning a StateMv value and
// diagnostics. If errors are encountered, a StateMv value is still returned
// representing the best effort interpretation of the arguments.
func ParseStateMv(args []string) (*StateMv, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	mv := &StateMv{}

	cmdFlags := defaultFlagSet("state mv")
	cmdFlags.BoolVar(&mv.DryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&mv.BackupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&mv.BackupOutPath, "backup-out", "-", "backup")
	cmdFlags.BoolVar(&mv.StateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&mv.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&mv.StatePath, "state", "", "path")
	cmdFlags.StringVar(&mv.StateOutPath, "state-out", "", "path")
	cmdFlags.BoolVar(&mv.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

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
			"Required argument missing",
			"Exactly two arguments expected: the source and destination addresses.",
		))
	}

	if len(args) > 0 {
		mv.SourceAddr = args[0]
	}
	if len(args) > 1 {
		mv.DestAddr = args[1]
	}

	return mv, diags
}
