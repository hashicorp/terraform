package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Validate represents the command-line arguments for the validate command.
type Validate struct {
	// Path is the directory containing the configuration to be validated. If
	// unspecified, validate will use the current directory.
	Path string

	// ViewType specifies which output format to use: human, JSON, or "raw".
	ViewType ViewType

	// Hints enables generation of additional hints that may make a module
	// easier to understand or avoid common pitfalls, but that may not apply
	// in all situations.
	Hints bool
}

// ParseValidate processes CLI arguments, returning a Validate value and errors.
// If errors are encountered, a Validate value is still returned representing
// the best effort interpretation of the arguments.
func ParseValidate(args []string) (*Validate, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	validate := &Validate{
		Path: ".",
	}

	var jsonOutput bool
	var hints bool
	cmdFlags := defaultFlagSet("validate")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&hints, "hints", false, "hints")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected at most one positional argument.",
		))
	}

	if len(args) > 0 {
		validate.Path = args[0]
	}

	switch {
	case jsonOutput:
		validate.ViewType = ViewJSON
	default:
		validate.ViewType = ViewHuman
	}

	validate.Hints = hints

	return validate, diags
}
