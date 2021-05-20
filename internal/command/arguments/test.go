package arguments

import (
	"flag"
	"io/ioutil"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Test represents the command line arguments for the "terraform test" command.
type Test struct {
	Output TestOutput
}

// TestOutput represents a subset of the arguments for "terraform test"
// related to how it presents its results. That is, it's the arguments that
// are relevant to the command's view rather than its controller.
type TestOutput struct {
	// If not an empty string, JUnitXMLFile gives a filename where JUnit-style
	// XML test result output should be written, in addition to the normal
	// output printed to the standard output and error streams.
	// (The typical usage pattern for tools that can consume this file format
	// is to configure them to look for a separate test result file on disk
	// after running the tests.)
	JUnitXMLFile string
}

// ParseTest interprets a slice of raw command line arguments into a
// Test value.
func ParseTest(args []string) (Test, tfdiags.Diagnostics) {
	var ret Test
	var diags tfdiags.Diagnostics

	// NOTE: ParseTest should still return at least a partial
	// Test even on error, containing enough information for the
	// command to report error diagnostics in a suitable way.

	f := flag.NewFlagSet("test", flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)
	f.Usage = func() {}
	f.StringVar(&ret.Output.JUnitXMLFile, "junit-xml", "", "Write a JUnit XML file describing the results")

	err := f.Parse(args)
	if err != nil {
		diags = diags.Append(err)
		return ret, diags
	}

	// We'll now discard all of the arguments that the flag package handled,
	// and focus only on the positional arguments for the rest of the function.
	args = f.Args()

	if len(args) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid command arguments",
			"The test command doesn't expect any positional command-line arguments.",
		))
		return ret, diags
	}

	return ret, diags
}
