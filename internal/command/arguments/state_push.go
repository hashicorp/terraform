// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePush represents the command-line arguments for the state push command.
type StatePush struct {
	// Force writes the state even if lineages don't match or the remote
	// serial is higher.
	Force bool

	// StateLock, if true, requests that the backend lock the state for this
	// operation.
	StateLock bool

	// StateLockTimeout is the duration to retry a state lock.
	StateLockTimeout time.Duration

	// IgnoreRemoteVersion, if true, continues even if remote and local
	// Terraform versions are incompatible.
	IgnoreRemoteVersion bool

	// Path is the path to the state file to push, or "-" for stdin.
	Path string
}

// ParseStatePush processes CLI arguments, returning a StatePush value and
// diagnostics. If errors are encountered, a StatePush value is still returned
// representing the best effort interpretation of the arguments.
func ParseStatePush(args []string) (*StatePush, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	push := &StatePush{
		StateLock: true,
	}

	cmdFlags := defaultFlagSet("state push")
	cmdFlags.BoolVar(&push.Force, "force", false, "")
	cmdFlags.BoolVar(&push.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&push.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.BoolVar(&push.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Required argument missing",
			"Exactly one argument expected: the path to a Terraform state file.",
		))
		return push, diags
	}

	push.Path = args[0]

	return push, diags
}
