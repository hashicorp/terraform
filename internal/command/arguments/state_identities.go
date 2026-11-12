// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateIdentities represents the command-line arguments for the state identities command.
type StateIdentities struct {
	ViewType ViewType

	// StatePath is an optional path to a state file, overriding the default.
	StatePath string

	// ID filters the results to include only instances whose resource types
	// have an attribute named "id" whose value equals this string.
	ID string

	// Addrs are optional resource or module addresses used to filter the
	// listed instances.
	Addrs []string
}

func ParseStateIdentities(args []string) (*StateIdentities, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &StateIdentities{}

	cmdFlags := defaultFlagSet("state identities")
	cmdFlags.StringVar(&ret.StatePath, "state", "", "path")
	cmdFlags.StringVar(&ret.ID, "id", "", "Restrict output to paths with a resource having the specified ID.")
	var jsonOutput bool
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// All positional arguments are assumed to be resource addresses
	addrs := cmdFlags.Args()
	if len(addrs) > 0 {
		ret.Addrs = addrs
	}

	if jsonOutput {
		ret.ViewType = ViewJSON
	} else {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Missing required -json flag",
			"The `terraform state identities` command requires the `-json` flag.",
		))
		ret.ViewType = ViewHuman
	}

	return ret, diags
}
