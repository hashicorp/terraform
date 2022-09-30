package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Plan represents the command-line arguments for the plan command.
type Plan struct {
	// State, Operation, and Vars are the common extended flags
	State     *State
	Operation *Operation
	Vars      *Vars

	// DetailedExitCode enables different exit codes for error, success with
	// changes, and success with no changes.
	DetailedExitCode bool

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// OutPath contains an optional path to store the plan file, while
	// AlwaysOut means that we'll write to OutPath even if the plan is
	// incomplete, so that it's still possible to inspect it with
	// "terraform show". AlwaysOut is irrelevant if OutPath isn't set.
	OutPath   string
	AlwaysOut bool

	// ViewType specifies which output format to use
	ViewType ViewType
}

// ParsePlan processes CLI arguments, returning a Plan value and errors.
// If errors are encountered, a Plan value is still returned representing
// the best effort interpretation of the arguments.
func ParsePlan(args []string) (*Plan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	plan := &Plan{
		State:     &State{},
		Operation: &Operation{},
		Vars:      &Vars{},
	}

	var outPath, alwaysOutPath string

	cmdFlags := extendedFlagSet("plan", plan.State, plan.Operation, plan.Vars)
	cmdFlags.BoolVar(&plan.DetailedExitCode, "detailed-exitcode", false, "detailed-exitcode")
	cmdFlags.BoolVar(&plan.InputEnabled, "input", true, "input")
	cmdFlags.StringVar(&outPath, "out", "", "out")
	cmdFlags.StringVar(&alwaysOutPath, "always-out", "", "always-out")

	var json bool
	cmdFlags.BoolVar(&json, "json", false, "json")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	switch {
	case outPath != "":
		if alwaysOutPath != "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Incompatible command line options",
				"The -out=... and -always-out=... options are mutually-exclusive.",
			))
		}
		plan.OutPath = outPath
		plan.AlwaysOut = false
	case alwaysOutPath != "":
		plan.OutPath = alwaysOutPath
		plan.AlwaysOut = true
	}

	args = cmdFlags.Args()

	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"To specify a working directory for the plan, use the global -chdir flag.",
		))
	}

	diags = diags.Append(plan.Operation.Parse())

	// JSON view currently does not support input, so we disable it here
	if json {
		plan.InputEnabled = false
	}

	switch {
	case json:
		plan.ViewType = ViewJSON
	default:
		plan.ViewType = ViewHuman
	}

	return plan, diags
}
