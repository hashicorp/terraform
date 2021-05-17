package views

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/colorstring"
)

// Test is the view interface for the "terraform test" command.
type Test interface {
	// Results presents the given test results.
	Results(map[string]*moduletest.Suite) tfdiags.Diagnostics

	// Diagnostics is for reporting warnings or errors that occurred with the
	// mechanics of running tests. For this command in particular, some
	// errors are considered to be test failures rather than mechanism failures,
	// and so those will be reported via Results rather than via Diagnostics.
	Diagnostics(tfdiags.Diagnostics)
}

// NewTest returns an implementation of Test configured to respect the
// settings described in the given arguments.
func NewTest(base *View, args arguments.TestOutput) Test {
	return &testHuman{
		streams:         base.streams,
		showDiagnostics: base.Diagnostics,
		colorize:        base.colorize,
		junitXMLFile:    args.JUnitXMLFile,
	}
}

type testHuman struct {
	// This is the subset of functionality we need from the base view.
	streams         *terminal.Streams
	showDiagnostics func(diags tfdiags.Diagnostics)
	colorize        *colorstring.Colorize

	// If junitXMLFile is not empty then results will be written to
	// the given file path in addition to the usual output.
	junitXMLFile string
}

func (v *testHuman) Results(results map[string]*moduletest.Suite) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// FIXME: Due to how this prototype command evolved concurrently with
	// establishing the idea of command views, the handling of JUnit output
	// as part of the "human" view rather than as a separate view in its
	// own right is a little odd and awkward. We should refactor this
	// prior to making "terraform test" a real supported command to make
	// it be structured more like the other commands that use the views
	// package.

	v.humanResults(results)

	if v.junitXMLFile != "" {
		moreDiags := v.junitXMLResults(results, v.junitXMLFile)
		diags = diags.Append(moreDiags)
	}

	return diags
}

func (v *testHuman) Diagnostics(diags tfdiags.Diagnostics) {
	if len(diags) == 0 {
		return
	}
	v.showDiagnostics(diags)
}

func (v *testHuman) humanResults(results map[string]*moduletest.Suite) {
	failCount := 0
	width := v.streams.Stderr.Columns()

	suiteNames := make([]string, 0, len(results))
	for suiteName := range results {
		suiteNames = append(suiteNames, suiteName)
	}
	sort.Strings(suiteNames)
	for _, suiteName := range suiteNames {
		suite := results[suiteName]

		componentNames := make([]string, 0, len(suite.Components))
		for componentName := range suite.Components {
			componentNames = append(componentNames, componentName)
		}
		for _, componentName := range componentNames {
			component := suite.Components[componentName]

			assertionNames := make([]string, 0, len(component.Assertions))
			for assertionName := range component.Assertions {
				assertionNames = append(assertionNames, assertionName)
			}
			sort.Strings(assertionNames)

			for _, assertionName := range assertionNames {
				assertion := component.Assertions[assertionName]

				fullName := fmt.Sprintf("%s.%s.%s", suiteName, componentName, assertionName)
				if strings.HasPrefix(componentName, "(") {
					// parenthesis-prefixed components are placeholders that
					// the test harness generates to represent problems that
					// prevented checking any assertions at all, so we'll
					// just hide them and show the suite name.
					fullName = suiteName
				}
				headingExtra := fmt.Sprintf("%s (%s)", fullName, assertion.Description)

				switch assertion.Outcome {
				case moduletest.Failed:
					// Failed means that the assertion was successfully
					// excecuted but that the assertion condition didn't hold.
					v.eprintRuleHeading("yellow", "Failed", headingExtra)

				case moduletest.Error:
					// Error means that the system encountered an unexpected
					// error when trying to evaluate the assertion.
					v.eprintRuleHeading("red", "Error", headingExtra)

				default:
					// We don't do anything for moduletest.Passed or
					// moduletest.Skipped. Perhaps in future we'll offer a
					// -verbose option to include information about those.
					continue
				}
				failCount++

				if len(assertion.Message) > 0 {
					dispMsg := format.WordWrap(assertion.Message, width)
					v.streams.Eprintln(dispMsg)
				}
				if len(assertion.Diagnostics) > 0 {
					// We'll do our own writing of the diagnostics in this
					// case, rather than using v.Diagnostics, because we
					// specifically want all of these diagnostics to go to
					// Stderr along with all of the other output we've
					// generated.
					for _, diag := range assertion.Diagnostics {
						diagStr := format.Diagnostic(diag, nil, v.colorize, width)
						v.streams.Eprint(diagStr)
					}
				}
			}
		}
	}

	if failCount > 0 {
		// If we've printed at least one failure then we'll have printed at
		// least one horizontal rule across the terminal, and so we'll balance
		// that with another horizontal rule.
		if width > 1 {
			rule := strings.Repeat("─", width-1)
			v.streams.Eprintln(v.colorize.Color("[dark_gray]" + rule))
		}
	}

	if failCount == 0 {
		if len(results) > 0 {
			// This is not actually an error, but it's convenient if all of our
			// result output goes to the same stream for when this is running in
			// automation that might be gathering this output via a pipe.
			v.streams.Eprint(v.colorize.Color("[bold][green]Success![reset] All of the test assertions passed.\n\n"))
		} else {
			v.streams.Eprint(v.colorize.Color("[bold][yellow]No tests defined.[reset] This module doesn't have any test suites to run.\n\n"))
		}
	}

	// Try to flush any buffering that might be happening. (This isn't always
	// successful, depending on what sort of fd Stderr is connected to.)
	v.streams.Stderr.File.Sync()
}

func (v *testHuman) junitXMLResults(results map[string]*moduletest.Suite, filename string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// "JUnit XML" is a file format that has become a de-facto standard for
	// test reporting tools but that is not formally specified anywhere, and
	// so each producer and consumer implementation unfortunately tends to
	// differ in certain ways from others.
	// With that in mind, this is a best effort sort of thing aimed at being
	// broadly compatible with various consumers, but it's likely that
	// some consumers will present these results better than others.
	// This implementation is based mainly on the pseudo-specification of the
	// format curated here, based on the Jenkins parser implementation:
	//    https://llg.cubic.org/docs/junit/

	// An "Outcome" represents one of the various XML elements allowed inside
	// a testcase element to indicate the test outcome.
	type Outcome struct {
		Message string `xml:"message,omitempty"`
	}

	// TestCase represents an individual test case as part of a suite. Note
	// that a JUnit XML incorporates both the "component" and "assertion"
	// levels of our model: we pretend that component is a class name and
	// assertion is a method name in order to match with the Java-flavored
	// expectations of JUnit XML, which are hopefully close enough to get
	// a test result rendering that's useful to humans.
	type TestCase struct {
		AssertionName string `xml:"name"`
		ComponentName string `xml:"classname"`

		// These fields represent the different outcomes of a TestCase. Only one
		// of these should be populated in each TestCase; this awkward
		// structure is just to make this play nicely with encoding/xml's
		// expecatations.
		Skipped *Outcome `xml:"skipped,omitempty"`
		Error   *Outcome `xml:"error,omitempty"`
		Failure *Outcome `xml:"failure,omitempty"`

		Stderr string `xml:"system-out,omitempty"`
	}

	// TestSuite represents an individual test suite, of potentially many
	// in a JUnit XML document.
	type TestSuite struct {
		Name         string      `xml:"name"`
		TotalCount   int         `xml:"tests"`
		SkippedCount int         `xml:"skipped"`
		ErrorCount   int         `xml:"errors"`
		FailureCount int         `xml:"failures"`
		Cases        []*TestCase `xml:"testcase"`
	}

	// TestSuites represents the root element of the XML document.
	type TestSuites struct {
		XMLName      struct{}     `xml:"testsuites"`
		ErrorCount   int          `xml:"errors"`
		FailureCount int          `xml:"failures"`
		TotalCount   int          `xml:"tests"`
		Suites       []*TestSuite `xml:"testsuite"`
	}

	xmlSuites := TestSuites{}
	suiteNames := make([]string, 0, len(results))
	for suiteName := range results {
		suiteNames = append(suiteNames, suiteName)
	}
	sort.Strings(suiteNames)
	for _, suiteName := range suiteNames {
		suite := results[suiteName]

		xmlSuite := &TestSuite{
			Name: suiteName,
		}
		xmlSuites.Suites = append(xmlSuites.Suites, xmlSuite)

		componentNames := make([]string, 0, len(suite.Components))
		for componentName := range suite.Components {
			componentNames = append(componentNames, componentName)
		}
		for _, componentName := range componentNames {
			component := suite.Components[componentName]

			assertionNames := make([]string, 0, len(component.Assertions))
			for assertionName := range component.Assertions {
				assertionNames = append(assertionNames, assertionName)
			}
			sort.Strings(assertionNames)

			for _, assertionName := range assertionNames {
				assertion := component.Assertions[assertionName]
				xmlSuites.TotalCount++
				xmlSuite.TotalCount++

				xmlCase := &TestCase{
					ComponentName: componentName,
					AssertionName: assertionName,
				}
				xmlSuite.Cases = append(xmlSuite.Cases, xmlCase)

				switch assertion.Outcome {
				case moduletest.Pending:
					// We represent "pending" cases -- cases blocked by
					// upstream errors -- as if they were "skipped" in JUnit
					// terms, because we didn't actually check them and so
					// can't say whether they succeeded or not.
					xmlSuite.SkippedCount++
					xmlCase.Skipped = &Outcome{
						Message: assertion.Message,
					}
				case moduletest.Failed:
					xmlSuites.FailureCount++
					xmlSuite.FailureCount++
					xmlCase.Failure = &Outcome{
						Message: assertion.Message,
					}
				case moduletest.Error:
					xmlSuites.ErrorCount++
					xmlSuite.ErrorCount++
					xmlCase.Error = &Outcome{
						Message: assertion.Message,
					}

					// We'll also include the diagnostics in the "stderr"
					// portion of the output, so they'll hopefully be visible
					// in a test log viewer in JUnit-XML-Consuming CI systems.
					var buf strings.Builder
					for _, diag := range assertion.Diagnostics {
						diagStr := format.DiagnosticPlain(diag, nil, 68)
						buf.WriteString(diagStr)
					}
					xmlCase.Stderr = buf.String()
				}

			}
		}
	}

	xmlOut, err := xml.MarshalIndent(&xmlSuites, "", "  ")
	if err != nil {
		// If marshalling fails then that's a bug in the code above,
		// because we should always be producing a value that is
		// accepted by encoding/xml.
		panic(fmt.Sprintf("invalid values to marshal as JUnit XML: %s", err))
	}

	err = ioutil.WriteFile(filename, xmlOut, 0644)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to write JUnit XML file",
			fmt.Sprintf(
				"Could not create %s to record the test results in JUnit XML format: %s.",
				filename,
				err,
			),
		))
	}

	return diags
}

func (v *testHuman) eprintRuleHeading(color, prefix, extra string) {
	const lineCell string = "─"
	textLen := len(prefix) + len(": ") + len(extra)
	spacingLen := 2
	leftLineLen := 3

	rightLineLen := 0
	width := v.streams.Stderr.Columns()
	if (textLen + spacingLen + leftLineLen) < (width - 1) {
		// (we allow an extra column at the end because some terminals can't
		// print in the final column without wrapping to the next line)
		rightLineLen = width - (textLen + spacingLen + leftLineLen) - 1
	}

	colorCode := "[" + color + "]"

	// We'll prepare what we're going to print in memory first, so that we can
	// send it all to stderr in one write in case other programs are also
	// concurrently trying to write to the terminal for some reason.
	var buf strings.Builder
	buf.WriteString(v.colorize.Color(colorCode + strings.Repeat(lineCell, leftLineLen)))
	buf.WriteByte(' ')
	buf.WriteString(v.colorize.Color("[bold]" + colorCode + prefix + ":"))
	buf.WriteByte(' ')
	buf.WriteString(extra)
	if rightLineLen > 0 {
		buf.WriteByte(' ')
		buf.WriteString(v.colorize.Color(colorCode + strings.Repeat(lineCell, rightLineLen)))
	}
	v.streams.Eprintln(buf.String())
}
