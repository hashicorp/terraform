// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const fmtStdinArg = "-"

// Fmt represents the command-line arguments for the fmt command.
type Fmt struct {
	List      bool
	Write     bool
	Diff      bool
	Check     bool
	Recursive bool
	Paths     []string
}

func ParseFmt(args []string) (*Fmt, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	fmt := &Fmt{
		List:  true,
		Write: true,
	}

	cmdFlags := defaultFlagSet("fmt")
	cmdFlags.BoolVar(&fmt.List, "list", true, "list")
	cmdFlags.BoolVar(&fmt.Write, "write", true, "write")
	cmdFlags.BoolVar(&fmt.Diff, "diff", false, "diff")
	cmdFlags.BoolVar(&fmt.Check, "check", false, "check")
	cmdFlags.BoolVar(&fmt.Recursive, "recursive", false, "recursive")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	fmt.Paths = cmdFlags.Args()
	switch {
	case len(fmt.Paths) == 0:
		// We're parsing the current directory by default
		fmt.Paths = []string{"."}
	case fmt.Paths[0] == fmtStdinArg:
		// TODO: Reject additional targets after "-" with a diagnostic. This
		// would make stdin validation more explicit, but is a breaking change
		// because the legacy command currently ignores them.
		fmt.Paths = nil
		fmt.List = false
		fmt.Write = false
	}

	return fmt, diags
}
