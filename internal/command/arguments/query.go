// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Query represents the command-line arguments for the query command.
type Query struct {
	// State, Operation, and Vars are the common extended flags
	State     *State
	Operation *Operation
	Vars      *Vars

	// ViewType specifies which output format to use: human or JSON.
	ViewType ViewType
}

func ParseQuery(args []string) (*Query, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	query := &Query{
		State:     &State{},
		Operation: &Operation{},
		Vars:      &Vars{},
	}

	cmdFlags := defaultFlagSet("query")

	varsFlags := NewFlagNameValueSlice("-var")
	varFilesFlags := varsFlags.Alias("-var-file")
	query.Vars.vars = &varsFlags
	query.Vars.varFiles = &varFilesFlags
	cmdFlags.Var(query.Vars.vars, "var", "var")
	cmdFlags.Var(query.Vars.varFiles, "var-file", "var-file")

	var json bool
	cmdFlags.BoolVar(&json, "json", false, "json")

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
			"To specify a working directory for the plan, use the global -chdir flag.",
		))
	}

	diags = diags.Append(query.Operation.Parse())

	switch {
	case json:
		query.ViewType = ViewJSON
	default:
		query.ViewType = ViewHuman
	}

	return query, diags
}
