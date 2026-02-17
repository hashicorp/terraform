// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateList represents the command-line arguments for the state list command.
type StateList struct {
	// StatePath is an optional path to a state file, overriding the default.
	StatePath string

	// ID filters the results to include only instances whose resource types
	// have an attribute named "id" whose value equals this string.
	ID string

	// Addrs are optional resource or module addresses used to filter the
	// listed instances.
	Addrs []string
}

// ParseStateList processes CLI arguments, returning a StateList value and
// diagnostics. If errors are encountered, a StateList value is still returned
// representing the best effort interpretation of the arguments.
func ParseStateList(args []string) (*StateList, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	list := &StateList{}

	var statePath, id string
	cmdFlags := defaultFlagSet("state list")
	cmdFlags.StringVar(&statePath, "state", "", "path")
	cmdFlags.StringVar(&id, "id", "", "Restrict output to paths with a resource having the specified ID.")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	list.StatePath = statePath
	list.ID = id
	list.Addrs = cmdFlags.Args()

	return list, diags
}
