package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Checks represents the command-line arguments for the checks command.
type Checks struct {
	ShowAll bool
}

// ParseChecks processes CLI arguments for the "checks" command, returning a
// Checks value and errors.
//
// If errors are encountered, a Checks value is still returned representing
// the best effort interpretation of the arguments.
func ParseChecks(args []string) (*Checks, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Checks{}

	var showAll bool
	cmdFlags := defaultFlagSet("checks")
	cmdFlags.BoolVar(&showAll, "all", false, "all")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line arguments",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unexpected argument",
			"The 'checks' command does not expect any arguments aside from its options.",
		))
	}

	ret.ShowAll = showAll
	return ret, diags
}
