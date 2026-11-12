// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// MetadataFunctions represents the command-line arguments for the metadata
// functions command.
type MetadataFunctions struct {
	JSON bool
}

// ParseMetadataFunctions processes CLI arguments, returning a
// MetadataFunctions value and errors. If errors are encountered, a
// MetadataFunctions value is still returned representing the best effort
// interpretation of the arguments.
func ParseMetadataFunctions(args []string) (*MetadataFunctions, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	metadataFunctions := &MetadataFunctions{}

	cmdFlags := defaultFlagSet("metadata functions")
	cmdFlags.BoolVar(&metadataFunctions.JSON, "json", false, "produce JSON output")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))

		return metadataFunctions, diags
	}

	// Positional arguments are currently accepted and ignored.
	// Rejecting them would be a breaking change.
	// TODO: Add validation of unexpected arguments in future major version.
	_ = cmdFlags.Args()

	if !metadataFunctions.JSON {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -json flag is required",
			"The `terraform metadata functions` command requires the `-json` flag.",
		))
	}

	return metadataFunctions, diags
}
