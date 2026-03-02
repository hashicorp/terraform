// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Logout represents the command-line arguments for the logout command.
type Logout struct {
	Hostname string
}

// ParseLogout processes CLI arguments, returning a Logout value and errors.
// If errors are encountered, a Logout value is still returned representing
// the best effort interpretation of the arguments.
func ParseLogout(args []string) (*Logout, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	logout := &Logout{}

	cmdFlags := defaultFlagSet("logout")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid arguments",
			"The logout command expects at most one argument: the host to log out of.",
		))
		return logout, diags
	}

	logout.Hostname = "app.terraform.io"
	if len(args) == 1 {
		logout.Hostname = args[0]
	}

	return logout, diags
}
