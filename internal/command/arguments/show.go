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

	// FormatVersion specifies a major version number for the wire format
	// the user has requested. Not all view types support multiple selectable
	// versions, and so this is always zero for those which don't.
	FormatVersion int
}

// ParseShow processes CLI arguments, returning a Show value and errors.
// If errors are encountered, a Show value is still returned representing
// the best effort interpretation of the arguments.
func ParseShow(args []string) (*Show, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	show := &Show{
		Path: "",
	}

	var jsonOutput1 bool
	var jsonOutput2 bool
	cmdFlags := defaultFlagSet("show")
	cmdFlags.BoolVar(&jsonOutput1, "json", false, "json")
	cmdFlags.BoolVar(&jsonOutput2, "json2", false, "json version 2")

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

	if jsonOutput1 && jsonOutput2 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Incompatible options",
			"Cannot use both -json and -json2 together: they select different versions of the same JSON format.",
		))
	}

	switch {
	case jsonOutput1:
		show.ViewType = ViewJSON
		show.FormatVersion = 1
	case jsonOutput2:
		show.ViewType = ViewJSON
		show.FormatVersion = 2

		if show.Path != "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Incompatible options",
				"The -json2 option is not yet available for describing plan files; it's supported only for state snapshots.",
			))
		}
	default:
		show.ViewType = ViewHuman
	}

	return show, diags
}
