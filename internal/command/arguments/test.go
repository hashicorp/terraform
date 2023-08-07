// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arguments

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Test represents the command-line arguments for the test command.
type Test struct {
	// Filter contains a list of test files to execute. If empty, all test files
	// will be executed.
	Filter []string

	// TestDirectory allows the user to override the directory that the test
	// command will use to discover test files, defaults to "tests". Regardless
	// of the value here, test files within the configuration directory will
	// always be discovered.
	TestDirectory string

	// ViewType specifies which output format to use: human or JSON.
	ViewType ViewType

	// You can specify common variables for all tests from the command line.
	Vars *Vars

	// Verbose tells the test command to print out the plan either in
	// human-readable format or JSON for each run step depending on the
	// ViewType.
	Verbose bool
}

func ParseTest(args []string) (*Test, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	test := Test{
		Vars: new(Vars),
	}

	var jsonOutput bool
	cmdFlags := extendedFlagSet("test", nil, nil, test.Vars)
	cmdFlags.Var((*flagStringSlice)(&test.Filter), "filter", "filter")
	cmdFlags.StringVar(&test.TestDirectory, "test-directory", configs.DefaultTestDirectory, "test-directory")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&test.Verbose, "verbose", false, "verbose")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error()))
	}

	switch {
	case jsonOutput:
		test.ViewType = ViewJSON
	default:
		test.ViewType = ViewHuman
	}

	return &test, diags
}
