// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Version represents the command-line arguments for the version command.
type Version struct {
	// ViewType specifies which output format to use: human, JSON, or "raw".
	ViewType ViewType
}

func ParseVersion(args []string, usageFunc func()) (*Version, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var jsonOutput bool
	cmdFlags := defaultFlagSet("version")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")

	// Enable but ignore the global version flags. In main.go, if any of the
	// arguments are -v, -version, or --version, the version command will be called
	// with the rest of the arguments, so we need to be able to cope with
	// those.
	cmdFlags.Bool("v", true, "version")
	cmdFlags.Bool("version", true, "version")

	cmdFlags.Usage = usageFunc
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	_ = cmdFlags.Args()
	// If this method call returned >0 values it means a user has supplied unexpected positional arguments
	// But we don't return an error here to preserve the original behavior of the version command.
	// In future we could return a warning, or an error if we're able to introduce breaking changes.

	var viewType ViewType
	if jsonOutput {
		viewType = ViewJSON
	} else {
		viewType = ViewHuman
	}

	return &Version{
		ViewType: viewType,
	}, diags
}
