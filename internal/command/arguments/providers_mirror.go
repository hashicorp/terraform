// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// ProvidersMirror represents the command-line arguments for the providers
// mirror command.
type ProvidersMirror struct {
	Platforms FlagStringSlice
	LockFile  bool
	OutputDir string
}

// ParseProvidersMirror processes CLI arguments, returning a ProvidersMirror
// value and errors. If errors are encountered, a ProvidersMirror value is
// still returned representing the best effort interpretation of the arguments.
func ParseProvidersMirror(args []string) (*ProvidersMirror, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providersMirror := &ProvidersMirror{}

	cmdFlags := defaultFlagSet("providers mirror")
	cmdFlags.Var(&providersMirror.Platforms, "platform", "target platform")
	cmdFlags.BoolVar(&providersMirror.LockFile, "lock-file", true, "use lock file")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()

	if len(args) != 1 {
		return providersMirror, diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No output directory specified",
			"The providers mirror command requires an output directory as a command-line argument.",
		))
	}

	providersMirror.OutputDir = args[0]

	return providersMirror, diags
}
