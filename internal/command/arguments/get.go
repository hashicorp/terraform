// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Get represents the command-line arguments for the get command.
type Get struct {
	Update         bool
	TestsDirectory string
	Args           []string
}

// ParseGet processes CLI arguments, returning a Get value and errors.
// If errors are encountered, a Get value is still returned representing
// the best effort interpretation of the arguments.
func ParseGet(args []string) (*Get, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	get := &Get{}

	cmdFlags := defaultFlagSet("get")
	cmdFlags.BoolVar(&get.Update, "update", false, "update")
	cmdFlags.StringVar(&get.TestsDirectory, "test-directory", "tests", "test-directory")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	get.Args = cmdFlags.Args()

	return get, diags
}
