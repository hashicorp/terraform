// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Console represents the command-line arguments for the console command.
type Console struct {
	// Vars are the variable-related flags (-var, -var-file).
	Vars *Vars

	// StatePath is the path to the state file.
	StatePath string

	// EvalFromPlan controls whether to evaluate expressions against a plan
	// instead of the current state.
	EvalFromPlan bool

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// CompactWarnings enables compact warning output.
	CompactWarnings bool

	// TargetFlags are the raw -target flag values.
	TargetFlags []string

	// ConfigPath is the path to a directory of Terraform configuration files.
	ConfigPath string
}

// ParseConsole processes CLI arguments, returning a Console value and
// diagnostics. If errors are encountered, a Console value is still returned
// representing the best effort interpretation of the arguments.
func ParseConsole(args []string) (*Console, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	con := &Console{
		Vars: &Vars{},
	}

	pwd, err := getwd()
	if err != nil {
		return nil, diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error getting pwd",
			err.Error(),
		))
	}

	cmdFlags := extendedFlagSet("console", nil, nil, con.Vars)
	cmdFlags.StringVar(&con.StatePath, "state", "", "path")
	cmdFlags.BoolVar(&con.EvalFromPlan, "plan", false, "evaluate from plan")
	cmdFlags.BoolVar(&con.InputEnabled, "input", true, "input")
	cmdFlags.BoolVar(&con.CompactWarnings, "compact-warnings", false, "compact-warnings")
	cmdFlags.Var((*FlagStringSlice)(&con.TargetFlags), "target", "target")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()

	if len(args) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"The console command does not expect any positional arguments. Did you mean to use -chdir?",
		))
	}

	con.ConfigPath = pwd

	return con, diags
}
