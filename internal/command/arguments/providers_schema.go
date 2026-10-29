// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// ProvidersSchema represents the command-line arguments for the providers
// schema command.
type ProvidersSchema struct {
	// Vars are the variable-related flags (-var, -var-file).
	Vars *Vars

	ViewType ViewType
}

// ParseProvidersSchema processes CLI arguments, returning a ProvidersSchema
// value and errors. If errors are encountered, a ProvidersSchema value is
// still returned representing the best effort interpretation of the arguments.
func ParseProvidersSchema(args []string) (*ProvidersSchema, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providersSchema := &ProvidersSchema{
		Vars:     &Vars{},
		ViewType: ViewHuman,
	}

	var json bool
	cmdFlags := extendedFlagSet("providers schema", nil, nil, providersSchema.Vars)
	cmdFlags.BoolVar(&json, "json", false, "produce JSON output")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected no positional arguments.",
		))
	}

	switch {
	case json:
		providersSchema.ViewType = ViewJSON
	default:
		providersSchema.ViewType = ViewHuman

		// The human view will only be used to render this error message
		// after argument parsing.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -json flag is required",
			"The `terraform providers schema` command requires the `-json` flag.",
		))
	}

	return providersSchema, diags
}
