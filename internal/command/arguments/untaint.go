// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Untaint represents the command-line arguments for the untaint command.
type Untaint struct {
	// Address is the address of the resource instance to untaint.
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

// ParseUntaint processes CLI arguments, returning an Untaint value and errors.
// If errors are encountered, an Untaint value is still returned representing
// the best effort interpretation of the arguments.
func ParseUntaint(args []string) (*Untaint, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	untaint := &Untaint{
		StateLock: true,
	}

	cmdFlags := defaultFlagSet("untaint")
	cmdFlags.BoolVar(&untaint.AllowMissing, "allow-missing", false, "allow missing")
	cmdFlags.StringVar(&untaint.BackupPath, "backup", "", "path")
	cmdFlags.BoolVar(&untaint.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&untaint.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&untaint.StatePath, "state", "", "path")
	cmdFlags.StringVar(&untaint.StateOutPath, "state-out", "", "path")
	cmdFlags.BoolVar(&untaint.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

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
			"The untaint command expects exactly one argument: the address of the resource to untaint.",
		))
	} else if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"The untaint command expects exactly one argument: the address of the resource to untaint.",
		))
	}

	if len(args) > 0 {
		untaint.Address = args[0]
	}

	return untaint, diags
}
