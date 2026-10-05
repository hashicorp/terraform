// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Unlock represents the command-line arguments for the unlock command.
type Unlock struct {
	Force  bool
	LockID string
}

func ParseUnlock(args []string) (*Unlock, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Unlock{}

	cmdFlags := defaultFlagSet("force-unlock")
	cmdFlags.BoolVar(&ret.Force, "force", false, "force")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	switch {
	case len(args) == 1:
		ret.LockID = args[0]
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid number of arguments",
			"Expected a single argument: LOCK_ID",
		))
	}

	return ret, diags
}
