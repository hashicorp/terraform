// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"os"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// getwd is a package-level variable that defaults to os.Getwd.
// It can be overridden in tests to provide a mock implementation.
var getwd = os.Getwd

// Import represents the command-line arguments for the import command.
type Import struct {
	// State, Vars are the common extended flags
	State *State
	Vars  *Vars

	// ConfigPath is the path to a directory of Terraform configuration files
	// to use to configure the provider. An empty string means the caller
	// should use the current working directory.
	ConfigPath string

	// Parallelism is the limit Terraform places on total parallel operations
	// as it walks the dependency graph.
	Parallelism int

	// IgnoreRemoteVersion controls whether to suppress the error when the
	// configured Terraform version on the remote workspace does not match the
	// local Terraform version.
	IgnoreRemoteVersion bool

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// CompactWarnings enables compact warning output.
	CompactWarnings bool

	// TargetFlags are the raw -target flag values.
	TargetFlags []string

	// Addr is the resource address to import into.
	Addr string

	// ID is the provider-specific ID of the resource to import.
	ID string
}

// ParseImport processes CLI arguments, returning an Import value and errors.
// If errors are encountered, an Import value is still returned representing
// the best effort interpretation of the arguments.
func ParseImport(args []string) (*Import, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	imp := &Import{
		State: &State{
			Lock: true,
		},
		Vars: &Vars{},
	}
	// Get the pwd since its our default -config flag value
	pwd, err := getwd()
	if err != nil {
		return nil, diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error getting pwd",
			err.Error(),
		))
	}

	cmdFlags := extendedFlagSet("import", imp.State, nil, imp.Vars)
	cmdFlags.BoolVar(&imp.IgnoreRemoteVersion, "ignore-remote-version", false, "ignore-remote-version")
	cmdFlags.IntVar(&imp.Parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&imp.ConfigPath, "config", pwd, "config")
	cmdFlags.BoolVar(&imp.InputEnabled, "input", true, "input")
	cmdFlags.BoolVar(&imp.CompactWarnings, "compact-warnings", false, "compact-warnings")
	cmdFlags.Var((*FlagStringSlice)(&imp.TargetFlags), "target", "target")

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
			"Wrong number of arguments",
			"The import command expects two arguments: ADDR and ID.",
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
