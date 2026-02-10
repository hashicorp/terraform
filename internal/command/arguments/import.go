// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Import represents the command-line arguments for the import command.
type Import struct {
	// State, and Vars are the common extended flags
	State *State
	Vars  *Vars

	// ConfigPath is the path to a directory of Terraform configuration files
	// to use to configure the provider. Defaults to pwd.
	ConfigPath string

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// Parallelism is the limit Terraform places on total parallel operations
	// as it walks the dependency graph.
	Parallelism int

	// IgnoreRemoteVersion continues even if remote and local Terraform
	// versions are incompatible.
	IgnoreRemoteVersion bool

	// Addr is the resource address to import into.
	Addr string

	// ID is the provider-specific resource ID.
	ID string
}

// ParseImport processes CLI arguments, returning an Import value and errors.
// If errors are encountered, an Import value is still returned representing
// the best effort interpretation of the arguments.
func ParseImport(args []string) (*Import, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	imp := &Import{
		State: &State{},
		Vars:  &Vars{},
	}

	cmdFlags := extendedFlagSet("import", imp.State, nil, imp.Vars)
	cmdFlags.StringVar(&imp.ConfigPath, "config", "", "config")
	cmdFlags.BoolVar(&imp.InputEnabled, "input", true, "input")
	cmdFlags.IntVar(&imp.Parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.BoolVar(&imp.IgnoreRemoteVersion, "ignore-remote-version", false, "ignore-remote-version")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid number of arguments",
			"The import command expects two arguments.",
		))
	}

	if len(args) > 0 {
		imp.Addr = args[0]
	}
	if len(args) > 1 {
		imp.ID = args[1]
	}

	return imp, diags
}
