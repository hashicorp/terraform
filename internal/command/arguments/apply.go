package arguments

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// JSON view currently does not support input, so we disable it here.
	if json {
		apply.InputEnabled = false
	}

	// JSON view cannot confirm apply, so we require either a plan file or
	// auto-approve to be specified. We intentionally fail here rather than
	// override auto-approve, which would be dangerous.
	if json && apply.PlanPath == "" && !apply.AutoApprove {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Plan file or auto-approve required",
			"Terraform cannot ask for interactive approval when -json is set. You can either apply a saved plan file, or enable the -auto-approve option.",
		))
	}

	diags = diags.Append(apply.Operation.Parse())

	switch {
	case json:
		apply.ViewType = ViewJSON
	default:
		apply.ViewType = ViewHuman
	}

	return apply, diags
}

// ParseApplyDestroy is a special case of ParseApply that deals with the
// "terraform destroy" command, which is effectively an alias for
// "terraform apply -destroy".
func ParseApplyDestroy(args []string) (*Apply, tfdiags.Diagnostics) {
	apply, diags := ParseApply(args)

	// So far ParseApply was using the command line options like -destroy
	// and -refresh-only to determine the plan mode. For "terraform destroy"
	// we expect neither of those arguments to be set, and so the plan mode
	// should currently be set to NormalMode, which we'll replace with
	// DestroyMode here. If it's already set to something else then that
	// suggests incorrect usage.
	switch apply.Operation.PlanMode {
	case plans.NormalMode:
		// This indicates that the user didn't specify any mode options at
		// all, which is correct, although we know from the command that
		// they actually intended to use DestroyMode here.
		apply.Operation.PlanMode = plans.DestroyMode
	case plans.DestroyMode:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid mode option",
			"The -destroy option is not valid for \"terraform destroy\", because this command always runs in destroy mode.",
		))
	case plans.RefreshOnlyMode:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid mode option",
			"The -refresh-only option is not valid for \"terraform destroy\".",
		))
	default:
		// This is a non-ideal error message for if we forget to handle a
		// newly-handled plan mode in Operation.Parse. Ideally they should all
		// have cases above so we can produce better error messages.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid mode option",
			fmt.Sprintf("The \"terraform destroy\" command doesn't support %s.", apply.Operation.PlanMode),
		))
	}

	// NOTE: It's also invalid to have apply.PlanPath set in this codepath,
	// but we don't check that in here because we'll return a different error
	// message depending on whether the given path seems to refer to a saved
	// plan file or to a configuration directory. The apply command
	// implementation itself therefore handles this situation.

	return apply, diags
}
