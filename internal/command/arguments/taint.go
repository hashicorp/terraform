// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Taint represents the command-line arguments for the "taint" command.
type Taint struct {
	// Addr is the parsed address of the resource instance to taint.
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

// ParseTaint processes CLI arguments, returning a Taint value and diagnostics.
// If errors are encountered, a Taint value is still returned representing the
// best effort interpretation of the arguments.
func ParseTaint(args []string) (*Taint, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	taint := &Taint{
		Lock: true,
	}

	cmdFlags := defaultFlagSet("taint")
	cmdFlags.BoolVar(&taint.AllowMissing, "allow-missing", false, "allow missing")
	cmdFlags.StringVar(&taint.BackupPath, "backup", "", "path")
	cmdFlags.BoolVar(&taint.Lock, "lock", true, "lock state")
	cmdFlags.DurationVar(&taint.LockTimeout, "lock-timeout", 0, "lock timeout")
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
	if len(args) != 1 {
		if len(args) == 0 {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Missing required argument",
				"The taint command expects exactly one argument: the address of the resource instance to taint.",
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Too many command line arguments",
				"The taint command expects exactly one argument: the address of the resource instance to taint.",
			))
		}
		return taint, diags
	}

	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	diags = diags.Append(addrDiags)
	if !addrDiags.HasErrors() {
		taint.Addr = addr
	}

	return taint, diags
}
