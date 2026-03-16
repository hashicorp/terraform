// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateShow represents the command-line arguments for the state show command.
type StateShow struct {
	// StatePath is an optional path to a state file, overriding the default.
	StatePath string

	// Address is the resource instance address to show.
	Address string
}

// ParseStateShow processes CLI arguments, returning a StateShow value and
// diagnostics. If errors are encountered, a StateShow value is still returned
// representing the best effort interpretation of the arguments.
func ParseStateShow(args []string) (*StateShow, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	show := &StateShow{}

	var statePath string
	cmdFlags := defaultFlagSet("state show")
	cmdFlags.StringVar(&statePath, "state", "", "path")

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
			"Exactly one argument expected: the address of a resource instance to show.",
		))
	}

	show.StatePath = statePath

	if len(args) > 0 {
		show.Address = args[0]
	}

	return show, diags
}
