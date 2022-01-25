package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Refresh represents the command-line arguments for the apply command.
type Refresh struct {
	// State, Operation, and Vars are the common extended flags
	State     *State
	Operation *Operation
	Vars      *Vars

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// ViewType specifies which output format to use
	ViewType ViewType
}

// ParseRefresh processes CLI arguments, returning a Refresh value and errors.
// If errors are encountered, a Refresh value is still returned representing
// the best effort interpretation of the arguments.
func ParseRefresh(args []string) (*Refresh, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	refresh := &Refresh{
		State:     &State{},
		Operation: &Operation{},
		Vars:      &Vars{},
	}

	cmdFlags := extendedFlagSet("refresh", refresh.State, refresh.Operation, refresh.Vars)
	cmdFlags.BoolVar(&refresh.InputEnabled, "input", true, "input")

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
			"Expected at most one positional argument.",
		))
	}

	diags = diags.Append(refresh.Operation.Parse())

	// JSON view currently does not support input, so we disable it here
	if json {
		refresh.InputEnabled = false
	}

	switch {
	case json:
		refresh.ViewType = ViewJSON
	default:
		refresh.ViewType = ViewHuman
	}

	return refresh, diags
}
