// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateReplaceProvider represents the command-line arguments for the state
// replace-provider command.
type StateReplaceProvider struct {
	// AutoApprove, if true, skips the interactive approval step.
	AutoApprove bool

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

	// FromProviderAddr is the provider address to replace.
	FromProviderAddr string

	// ToProviderAddr is the replacement provider address.
	ToProviderAddr string
}

// ParseStateReplaceProvider processes CLI arguments, returning a
// StateReplaceProvider value and diagnostics. If errors are encountered, a
// StateReplaceProvider value is still returned representing the best effort
// interpretation of the arguments.
func ParseStateReplaceProvider(args []string) (*StateReplaceProvider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rp := &StateReplaceProvider{}

	cmdFlags := defaultFlagSet("state replace-provider")
	cmdFlags.BoolVar(&rp.AutoApprove, "auto-approve", false, "skip interactive approval of replacements")
	cmdFlags.StringVar(&rp.BackupPath, "backup", "-", "backup")
	cmdFlags.BoolVar(&rp.StateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&rp.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&rp.StatePath, "state", "", "path")
	cmdFlags.BoolVar(&rp.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

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
			"Exactly two arguments expected: the from and to provider addresses.",
		))
		return rp, diags
	}
	rp.FromProviderAddr = args[0]
	rp.ToProviderAddr = args[1]

	return rp, diags
}
