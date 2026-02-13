// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Taint represents the command-line arguments for the taint command.
type Taint struct {
	// Address is the address of the resource instance to taint.
	Address string

	// AllowMissing, if true, means the command will succeed even if the
	// resource is not found in state.
	AllowMissing bool

	// BackupPath is the path to backup the existing state file before
	// modifying.
	BackupPath string

	// StateLock, if true, locks the state file during operations.
	StateLock bool

	// StateLockTimeout is the duration to retry a state lock.
	StateLockTimeout time.Duration

	// StatePath is the path to the state file to read and modify.
	StatePath string

	// StateOutPath is the path to write the updated state file.
	StateOutPath string

	// IgnoreRemoteVersion, if true, continues even if remote and local
	// Terraform versions are incompatible.
	IgnoreRemoteVersion bool
}

// ParseTaint processes CLI arguments, returning a Taint value and errors.
// If errors are encountered, a Taint value is still returned representing
// the best effort interpretation of the arguments.
func ParseTaint(args []string) (*Taint, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	taint := &Taint{
		StateLock: true,
	}

	cmdFlags := defaultFlagSet("taint")
	cmdFlags.BoolVar(&taint.AllowMissing, "allow-missing", false, "allow missing")
	cmdFlags.StringVar(&taint.BackupPath, "backup", "", "path")
	cmdFlags.BoolVar(&taint.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&taint.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&taint.StatePath, "state", "", "path")
	cmdFlags.StringVar(&taint.StateOutPath, "state-out", "", "path")
	cmdFlags.BoolVar(&taint.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Required argument missing",
			"The taint command expects exactly one argument: the address of the resource to taint.",
		))
	} else if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"The taint command expects exactly one argument: the address of the resource to taint.",
		))
	}

	if len(args) > 0 {
		taint.Address = args[0]
	}

	return taint, diags
}
