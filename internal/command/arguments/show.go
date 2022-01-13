package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Show represents the command-line arguments for the show command.
type Show struct {
	// Path is the path to the state file or plan file to be displayed. If
	// unspecified, show will display the latest state snapshot.
	Path string

	// ViewType specifies which output format to use: human, JSON, or "raw".
	ViewType ViewType
}

// ParseShow processes CLI arguments, returning a Show value and errors.
// If errors are encountered, a Show value is still returned representing
// the best effort interpretation of the arguments.
func ParseShow(args []string) (*Show, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	show := &Show{
		Path: "",
	}

	var jsonOutput bool
	cmdFlags := defaultFlagSet("show")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")

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
		show.Path = args[0]
	}

	switch {
	case jsonOutput:
		show.ViewType = ViewJSON
	default:
		show.ViewType = ViewHuman
	}

	return show, diags
}
