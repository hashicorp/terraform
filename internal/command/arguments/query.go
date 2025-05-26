// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Query represents the command-line arguments for the query command.
type Query struct {
	// ViewType specifies which output format to use: human or JSON.
	ViewType ViewType

	// You can specify common variables for all tests from the command line.
	Vars *Vars

	// Verbose tells the test command to print out the plan either in
	// human-readable format or JSON for each run step depending on the
	// ViewType.
	Verbose bool
}

func ParseQuery(args []string) (*Query, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	query := Query{
		Vars: new(Vars),
	}

	var jsonOutput bool
	cmdFlags := extendedFlagSet("query", nil, nil, query.Vars)
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&query.Verbose, "verbose", false, "verbose")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error()))
	}

	switch {
	case jsonOutput:
		query.ViewType = ViewJSON
	default:
		query.ViewType = ViewHuman
	}

	return &query, diags
}
