package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/tools/equivalence-tests/internal"
)

var Ui cli.Ui

func init() {
	Ui = &cli.BasicUi{
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
		Reader:      os.Stdin,
	}
}

// Args contains the arguments that we accept as part of the equivalence testing
// framework.
//
// We are expecting three arguments:
//   -goldens=/path/to/golden/files
//   -tests=/path/to/tests
//   -binary=/path/to/terraform/binary
//
// And an optional fourth argument: -update
//
// We do not have sensible defaults for the goldens or tests directory, but we
// can default to just trying to run `terraform` for the binary.
type Args struct {
	GoldensPath string
	TestsPath   string
	BinaryPath  string

	Update bool
}

func main() {
	args, err := parseArgs()
	if err != nil {
		Ui.Error(err.Error())
		os.Exit(1)
	}

	Ui.Info(fmt.Sprintf("reading tests from %s", args.TestsPath))
	tests, err := internal.ReadTests(args.TestsPath)
	if err != nil {
		Ui.Error(fmt.Sprintf("failed to read tests from %s, error: %v", args.TestsPath, err))
		os.Exit(1)
	}
	Ui.Info(fmt.Sprintf("read %d tests from %s", len(tests), args.TestsPath))
	Ui.Info(fmt.Sprintf("executing tests from %s", args.TestsPath))

	var failedTests []string
	for _, test := range tests {
		Ui.Info(fmt.Sprintf("executing test %s", test.Name))

		output, err, tfErr := test.Run(args.BinaryPath)
		if err != nil {
			failedTests = append(failedTests, test.Name)
			Ui.Warn(fmt.Sprintf("could not execute test %s, error: %v, tfErr:\n%v", test.Name, err, tfErr))
			continue
		}

		if args.Update {
			Ui.Info(fmt.Sprintf("updating golden files for test %s", test.Name))
			if err := output.Update(args.GoldensPath); err != nil {
				failedTests = append(failedTests, test.Name)
				Ui.Warn(fmt.Sprintf("could not update golden files for test %s, error: %v", test.Name, err))
				continue
			}
			Ui.Info(fmt.Sprintf("updated golden files for test %s", test.Name))
		} else {
			Ui.Info(fmt.Sprintf("comparing diffs for test %s", test.Name))
			diffs, err := output.Diff(args.GoldensPath)
			if err != nil {
				failedTests = append(failedTests, test.Name)
				Ui.Warn(fmt.Sprintf("could not compare diffs for test %s, error: %v", test.Name, err))
				continue
			}

			Ui.Info(fmt.Sprintf("the following diffs for test %s were found", test.Name))
			for file, diff := range diffs {
				Ui.Output(fmt.Sprintf("\nfile: %s\ndiff: %s\n", file, diff))
			}
		}
		Ui.Info(fmt.Sprintf("successfully executed test %s", test.Name))
	}

	if len(failedTests) > 0 {
		if len(failedTests) == len(tests) {
			Ui.Info("all tests failed execution")
			os.Exit(1)
		}
		Ui.Info(fmt.Sprintf("the following tests failed, please check the stderr logs for errors: [%s]", strings.Join(failedTests, ", ")))
		Ui.Info(fmt.Sprintf("every other test executed correctly"))
	} else {
		Ui.Info("all tests executed successfully")
	}
}

func parseArgs() (Args, error) {
	args := Args{
		BinaryPath: "terraform",
		Update:     false,
	}

	for _, arg := range os.Args[1:] {
		parts := strings.Split(arg, "=")
		if len(parts) > 2 {
			return args, fmt.Errorf("could not parse arguments, invalid format: %s", arg)
		}

		if len(parts) == 1 {
			if parts[0] == "-update" {
				args.Update = true
				continue
			}
			return args, fmt.Errorf("could not parse arguments, invalid format: %s", arg)
		}

		switch parts[0] {
		case "-goldens":
			args.GoldensPath = parts[1]
		case "-tests":
			args.TestsPath = parts[1]
		case "-binary":
			args.BinaryPath = parts[1]
		default:
			return args, fmt.Errorf("could not parse arguments, unrecognized argument: %s", arg)
		}
	}

	if len(args.GoldensPath) == 0 {
		return args, errors.New("could not parse arguments, must specify the -goldens argument")
	}
	if len(args.TestsPath) == 0 {
		return args, errors.New("could not parse arguments, must specify the -tests argument")
	}

	return args, nil
}
