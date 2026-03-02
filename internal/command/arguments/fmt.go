// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Fmt represents the command-line arguments for the fmt command.
type Fmt struct {
	List      bool
	Write     bool
	Diff      bool
	Check     bool
	Recursive bool
	Paths     []string
}

// ParseFmt processes CLI arguments, returning a Fmt value and errors.
// If errors are encountered, a Fmt value is still returned representing
// the best effort interpretation of the arguments.
func ParseFmt(args []string) (*Fmt, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	f := &Fmt{}

	cmdFlags := defaultFlagSet("fmt")
	cmdFlags.BoolVar(&f.List, "list", true, "list")
	cmdFlags.BoolVar(&f.Write, "write", true, "write")
	cmdFlags.BoolVar(&f.Diff, "diff", false, "diff")
	cmdFlags.BoolVar(&f.Check, "check", false, "check")
	cmdFlags.BoolVar(&f.Recursive, "recursive", false, "recursive")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	f.Paths = cmdFlags.Args()

	return f, diags
}
