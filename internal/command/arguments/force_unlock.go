// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// ForceUnlock represents the command-line arguments for the force-unlock
// command.
type ForceUnlock struct {
	Force  bool
	LockID string
	Args   []string
}

// ParseForceUnlock processes CLI arguments, returning a ForceUnlock value
// and errors. If errors are encountered, a ForceUnlock value is still
// returned representing the best effort interpretation of the arguments.
func ParseForceUnlock(args []string) (*ForceUnlock, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	forceUnlock := &ForceUnlock{}

	cmdFlags := defaultFlagSet("force-unlock")
	cmdFlags.BoolVar(&forceUnlock.Force, "force", false, "force")

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
			"Invalid arguments",
			"Expected a single argument: LOCK_ID",
		))
		return forceUnlock, diags
	}

	forceUnlock.LockID = args[0]
	forceUnlock.Args = args[1:]

	return forceUnlock, diags
}
