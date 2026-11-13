// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// MetadataFunctions represents the command-line arguments for the metadata
// functions command.
type MetadataFunctions struct {
	ViewType ViewType
}

// ParseMetadataFunctions processes CLI arguments, returning a
// MetadataFunctions value and errors. If errors are encountered, a
// MetadataFunctions value is still returned representing the best effort
// interpretation of the arguments.
func ParseMetadataFunctions(args []string) (*MetadataFunctions, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	metadataFunctions := &MetadataFunctions{}

	cmdFlags := defaultFlagSet("metadata functions")
	var json bool
	cmdFlags.BoolVar(&json, "json", false, "produce JSON output")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// Positional arguments are currently accepted and ignored.
	// Rejecting them would be a breaking change.
	// TODO: Add validation of unexpected arguments in future major version.
	_ = cmdFlags.Args()

	switch {
	case json:
		metadataFunctions.ViewType = ViewJSON

	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -json flag is required",
			"The `terraform metadata functions` command requires the `-json` flag.",
		))
		metadataFunctions.ViewType = ViewHuman // only used to render the error message above.
	}

	return metadataFunctions, diags
}
