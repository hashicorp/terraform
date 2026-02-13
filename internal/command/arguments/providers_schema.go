// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// ProvidersSchema represents the command-line arguments for the providers
// schema command.
type ProvidersSchema struct {
	JSON bool
}

// ParseProvidersSchema processes CLI arguments, returning a ProvidersSchema
// value and errors. If errors are encountered, a ProvidersSchema value is
// still returned representing the best effort interpretation of the arguments.
func ParseProvidersSchema(args []string) (*ProvidersSchema, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providersSchema := &ProvidersSchema{}

	cmdFlags := defaultFlagSet("providers schema")
	cmdFlags.BoolVar(&providersSchema.JSON, "json", false, "produce JSON output")

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

	if !providersSchema.JSON {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -json flag is required",
			"The `terraform providers schema` command requires the `-json` flag.",
		))
	}

	return providersSchema, diags
}
