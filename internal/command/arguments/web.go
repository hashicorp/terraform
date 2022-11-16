package arguments

import (
	"github.com/hashicorp/terraform/internal/command/webcommand"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Web represents the command-line arguments for the "web" command.
type Web struct {
	// TargetObject represents the object that the user wishes to view on the
	// web.
	TargetObject webcommand.TargetObject
}

// ParseWeb processes CLI arguments for the "web" command, returning a Web value
// and diagnostics.
//
// In case of errors, the Web object may still be partially populated with
// a subset of the settings that were parsable, but some fields may be
// incomplete or invalid.
func ParseWeb(args []string) (*Web, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Web{}

	// The structure of this command is a bit different than most others in
	// that it expects zero or one of its "object selection" options, which
	// we'll then reduce into a single target object to return.
	//
	// The "flag" package's approach is a bit awkward for this design but at
	// least we can encapsulate all of this awkwardness in here.

	cmdFlags := defaultFlagSet("web")
	pLatestRun := cmdFlags.Bool("latest-run", false, "")
	pRun := cmdFlags.String("run", "", "")
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
		return ret, diags
	}

	if *pLatestRun && *pRun != "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid combination of options",
			"Cannot use multiple object selection options in the same command.",
		))
		return ret, diags
	}
	if len(cmdFlags.Args()) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unexpected argument",
			"The 'web' command does not expect any positional arguments.",
		))
	}

	switch {
	case *pLatestRun:
		ret.TargetObject = webcommand.TargetObjectLatestRun
	case *pRun != "":
		ret.TargetObject = webcommand.TargetObjectRun{RunID: *pRun}
	default:
		ret.TargetObject = webcommand.TargetObjectCurrentWorkspace
	}

	return ret, diags
}
