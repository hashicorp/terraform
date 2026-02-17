// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePull represents the command-line arguments for the state pull command.
type StatePull struct {
}

// ParseStatePull processes CLI arguments, returning a StatePull value and
// diagnostics. If errors are encountered, a StatePull value is still returned
// representing the best effort interpretation of the arguments.
func ParseStatePull(args []string) (*StatePull, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	pull := &StatePull{}

	cmdFlags := defaultFlagSet("state pull")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	return pull, diags
}
