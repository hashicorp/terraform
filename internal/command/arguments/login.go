// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Login represents the command-line arguments for the login command.
type Login struct {
	Host string
}

func ParseLogin(rawArgs []string) (*Login, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Login{}

	cmdFlags := defaultFlagSet("login")
	// No command-specific flags for login.

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
			"The login command expects at most one argument: the host to log in to.",
			"",
		))
	}

	return ret, diags
}
