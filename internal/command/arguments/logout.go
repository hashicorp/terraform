// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Logout represents the command-line arguments for the logout command.
type Logout struct {
	Host string
}

const defaultHost = "app.terraform.io"

func ParseLogout(rawArgs []string) (*Logout, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Logout{}

	cmdFlags := defaultFlagSet("logout")
	// No command-specific flags for logout.

	if err := cmdFlags.Parse(rawArgs); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args := cmdFlags.Args()

	switch len(args) {
	case 0:
		// No host provided, use default.
		ret.Host = defaultHost
	case 1:
		ret.Host = args[0]
	default:
		ret.Host = args[0] // Best-effort attempt at parsing despite errors

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The logout command expects at most one argument: the host to log out of.",
			"",
		))
	}

	return ret, diags
}
