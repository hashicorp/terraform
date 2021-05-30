package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Output represents the command-line arguments for the output command.
type Output struct {
	// Name identifies which root module output to show.  If empty, show all
	// outputs.
	Name string

	// StatePath is an optional path to a state file, from which outputs will
	// be loaded.
	StatePath string

	// ViewType specifies which output format to use: human, JSON, or "raw".
	ViewType ViewType
}

// ParseOutput processes CLI arguments, returning an Output value and errors.
// If errors are encountered, an Output value is still returned representing
// the best effort interpretation of the arguments.
func ParseOutput(args []string) (*Output, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	output := &Output{}

	var jsonOutput, rawOutput bool
	var statePath string
	cmdFlags := defaultFlagSet("output")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&rawOutput, "raw", false, "raw")
	cmdFlags.StringVar(&statePath, "state", "", "path")

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
			"Unexpected argument",
			"The output command expects exactly one argument with the name of an output variable or no arguments to show all outputs.",
		))
	}

	if jsonOutput && rawOutput {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid output format",
			"The -raw and -json options are mutually-exclusive.",
		))

		// Since the desired output format is unknowable, fall back to default
		jsonOutput = false
		rawOutput = false
	}

	output.StatePath = statePath

	if len(args) > 0 {
		output.Name = args[0]
	}

	if rawOutput && output.Name == "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Output name required",
			"You must give the name of a single output value when using the -raw option.",
		))
	}

	switch {
	case jsonOutput:
		output.ViewType = ViewJSON
	case rawOutput:
		output.ViewType = ViewRaw
	default:
		output.ViewType = ViewHuman
	}

	return output, diags
}
