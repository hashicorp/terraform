// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateRm represents the command-line arguments for the state rm command.
type StateRm struct {
	// DryRun, if true, prints out what would be removed without actually
	// removing anything.
	DryRun bool

	// BackupPath is the path where Terraform should write the backup state.
	BackupPath string

	// StateLock, if true, requests that the backend lock the state for this
	// operation.
	StateLock bool

	// StateLockTimeout is the duration to retry a state lock.
	StateLockTimeout time.Duration

	// StatePath is an optional path to a local state file.
	StatePath string

	// IgnoreRemoteVersion, if true, continues even if remote and local
	// Terraform versions are incompatible.
	IgnoreRemoteVersion bool

	// Addrs are the resource instance addresses to remove.
	Addrs []string
}

// ParseStateRm processes CLI arguments, returning a StateRm value and
// diagnostics. If errors are encountered, a StateRm value is still returned
// representing the best effort interpretation of the arguments.
func ParseStateRm(args []string) (*StateRm, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rm := &StateRm{}

	cmdFlags := defaultFlagSet("state rm")
	cmdFlags.BoolVar(&rm.DryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&rm.BackupPath, "backup", "-", "backup")
	cmdFlags.BoolVar(&rm.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&rm.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&rm.StatePath, "state", "", "path")
	cmdFlags.BoolVar(&rm.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

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
			"Required argument missing",
			"At least one address is required.",
		))
	}

	rm.Addrs = args

	return rm, diags
}
