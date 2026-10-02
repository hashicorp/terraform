// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Login represents the command-line arguments for the login command.
type Login struct {
	Host string

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	CompactWarnings bool
}

func ParseLogin(rawArgs []string) (*Login, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Login{}

	cmdFlags := defaultFlagSet("login")
	cmdFlags.BoolVar(&ret.InputEnabled, "input", true, "input")
	cmdFlags.BoolVar(&ret.CompactWarnings, "compact-warnings", false, "use compact warnings")

	// This flag was accepted by `login` in the past due to using (m *Meta) extendedFlagSet
	// but it was never used in the command implementation.
	// We allow the flag to be passed, but it's still unused. We warn the user if they provide it.
	// TODO: Remove flag.
	var targetFlags []string
	cmdFlags.Var((*FlagStringSlice)(&targetFlags), "target", "resource to target")

	if err := cmdFlags.Parse(rawArgs); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	if len(targetFlags) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"The `target` flag is ignored by the login command.",
			"",
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
