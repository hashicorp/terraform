package arguments

import (
	"github.com/hashicorp/terraform/tfdiags"
)

// Apply represents the command-line arguments for the apply command.
type Apply struct {
	// State, Operation, and Vars are the common extended flags
	State     *State
	Operation *Operation
	Vars      *Vars

	// AutoApprove skips the manual verification step for the apply operation.
	AutoApprove bool

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// PlanPath contains an optional path to a stored plan file
	PlanPath string

	// ViewType specifies which output format to use
	ViewType ViewType
}

// ParseApply processes CLI arguments, returning an Apply value and errors.
// If errors are encountered, an Apply value is still returned representing
// the best effort interpretation of the arguments.
func ParseApply(args []string) (*Apply, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	apply := &Apply{
		State:     &State{},
		Operation: &Operation{},
		Vars:      &Vars{},
	}

	cmdFlags := extendedFlagSet("apply", apply.State, apply.Operation, apply.Vars)
	cmdFlags.BoolVar(&apply.AutoApprove, "auto-approve", false, "auto-approve")
	cmdFlags.BoolVar(&apply.InputEnabled, "input", true, "input")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		apply.PlanPath = args[0]
		args = args[1:]
	}

	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected at most one positional argument.",
		))
	}

	diags = diags.Append(apply.Operation.Parse())

	switch {
	default:
		apply.ViewType = ViewHuman
	}

	return apply, diags
}
