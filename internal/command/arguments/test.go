// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Test represents the command-line arguments for the test command.
type Test struct {
	// CloudRunSource specifies the remote private module that this test run
	// should execute against in a remote HCP Terraform run.
	CloudRunSource string

	// Filter contains a list of test files to execute. If empty, all test files
	// will be executed.
	Filter []string

	// OperationParallelism is the limit Terraform places on total parallel operations
	// during the plan or apply command within a single test run.
	OperationParallelism int

	// TestDirectory allows the user to override the directory that the test
	// command will use to discover test files, defaults to "tests". Regardless
	// of the value here, test files within the configuration directory will
	// always be discovered.
	TestDirectory string

	// ViewType specifies which output format to use: human or JSON.
	ViewType ViewType

	// JUnitXMLFile specifies an optional filename to write a JUnit XML test
	// result report to, in addition to the information written to the selected
	// view type.
	JUnitXMLFile string

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
	cmdFlags.Var((*FlagStringSlice)(&test.Filter), "filter", "filter")
	cmdFlags.StringVar(&test.TestDirectory, "test-directory", configs.DefaultTestDirectory, "test-directory")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.StringVar(&test.JUnitXMLFile, "junit-xml", "", "junit-xml")
	cmdFlags.BoolVar(&test.Verbose, "verbose", false, "verbose")
	cmdFlags.IntVar(&test.OperationParallelism, "parallelism", DefaultParallelism, "parallelism")

	// TODO: Finalise the name of this flag.
	cmdFlags.StringVar(&test.CloudRunSource, "cloud-run", "", "cloud-run")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error()))
	}

	if len(test.JUnitXMLFile) > 0 && len(test.CloudRunSource) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Incompatible command-line flags",
			"The -junit-xml option is currently not compatible with remote test execution via the -cloud-run flag. If you are interested in JUnit XML output for remotely-executed tests please open an issue in GitHub."))
	}

	// Only set the default parallelism if this is not a cloud-run test.
	// A cloud-run test will eventually run its own local test, and if the
	// user still hasn't set the parallelism, that run will use the default.
	if test.OperationParallelism < 1 && len(test.CloudRunSource) == 0 {
		test.OperationParallelism = DefaultParallelism
	}

	switch {
	case jsonOutput:
		test.ViewType = ViewJSON
	default:
		test.ViewType = ViewHuman
	}

	return &test, diags
}
