// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Untaint represents the command-line arguments for the "untaint" command.
type Untaint struct {
	// Addr is the parsed address of the resource instance to untaint.
	Addr addrs.AbsResourceInstance

	// AllowMissing means the command will succeed (exit code 0) even if the
	// resource instance is not found in state.
	AllowMissing bool

	// StatePath, StateOutPath, and BackupPath are legacy options for the local
	// backend only.
	StatePath    string
	StateOutPath string
	BackupPath   string

	// Lock and LockTimeout control state locking behavior.
	Lock        bool
	LockTimeout time.Duration

	// IgnoreRemoteVersion suppresses the error when the configured Terraform
	// version on the remote workspace does not match the local version.
	IgnoreRemoteVersion bool
}

// ParseUntaint processes CLI arguments, returning an Untaint value and
// diagnostics. If errors are encountered, an Untaint value is still returned
// representing the best effort interpretation of the arguments.
func ParseUntaint(args []string) (*Untaint, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	untaint := &Untaint{
		Lock: true,
	}

	cmdFlags := defaultFlagSet("untaint")
	cmdFlags.BoolVar(&untaint.AllowMissing, "allow-missing", false, "allow missing")
	cmdFlags.StringVar(&untaint.BackupPath, "backup", "", "path")
	cmdFlags.BoolVar(&untaint.Lock, "lock", true, "lock state")
	cmdFlags.DurationVar(&untaint.LockTimeout, "lock-timeout", 0, "lock timeout")
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
	if len(args) != 1 {
		if len(args) == 0 {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Missing required argument",
				"The untaint command expects exactly one argument: the address of the resource instance to untaint.",
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Too many command line arguments",
				"The untaint command expects exactly one argument: the address of the resource instance to untaint.",
			))
		}
		return untaint, diags
	}

	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	diags = diags.Append(addrDiags)
	if !addrDiags.HasErrors() {
		untaint.Addr = addr
	}

	return untaint, diags
}
