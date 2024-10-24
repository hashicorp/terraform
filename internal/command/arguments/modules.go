// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Modules represents the command-line arguments for the modules command
type Modules struct {
	// ViewType specifies which output format to use: human, JSON, or "raw"
	ViewType ViewType
}

// ParseModules processes CLI arguments, returning a Modules value and error
// diagnostics. If there are any diagnostics present, a Modules value is still
// returned representing the best effort interpretation of the arguments.
func ParseModules(args []string) (*Modules, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var jsonOutput bool

	modules := &Modules{}
	cmdFlags := defaultFlagSet("modules")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")

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
			"Expected no positional arguments",
		))
	}

	switch {
	case jsonOutput:
		modules.ViewType = ViewJSON
	default:
		modules.ViewType = ViewHuman
	}

	return modules, diags
}
