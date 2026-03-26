// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePull represents the command-line arguments for the state pull command.
type StatePull struct {
	// Vars are the variable-related flags (-var, -var-file).
	Vars *Vars
}

// ParseStatePull processes CLI arguments, returning a StatePull value and
// diagnostics. If errors are encountered, a StatePull value is still returned
// representing the best effort interpretation of the arguments.
func ParseStatePull(args []string) (*StatePull, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	pull := &StatePull{
		Vars: &Vars{},
	}

	cmdFlags := extendedFlagSet("state pull", nil, nil, pull.Vars)

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	return pull, diags
}
