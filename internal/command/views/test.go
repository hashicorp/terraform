package views

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/colorstring"
)

// Test is the view interface for the "terraform test" command.
type Test interface {
	// Results presents the given test results.
	Results(map[string]*moduletest.ScenarioResult) tfdiags.Diagnostics

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

func (v *testHuman) Results(results map[string]*moduletest.ScenarioResult) tfdiags.Diagnostics {
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

func (v *testHuman) humanResults(results map[string]*moduletest.ScenarioResult) {
	scenarioNames := make([]string, 0, len(results))
	for scenarioName := range results {
		scenarioNames = append(scenarioNames, scenarioName)
	}
	sort.Strings(scenarioNames)

	// To make the result easier to scan, we'll first print a short summary
	// of just the test scenario statuses.
	for _, scenarioName := range scenarioNames {
		scenario := results[scenarioName]
		prefix, symbol, colorCode := v.presentationForStatus(scenario.Status)
		v.streams.Eprintf(
			"%s: %s\n",
			v.colorize.Color(fmt.Sprintf("[%s]%s %s", colorCode, symbol, prefix)),
			scenario.Name,
		)
	}

	// If any of the scenarios failed or errored then we'll now repeat them
	// with more detail about exactly what went wrong.
	for _, scenarioName := range scenarioNames {
		scenario := results[scenarioName]
		if scenario.Status != checks.StatusFail && scenario.Status != checks.StatusError {
			continue
		}
		prefix, _, colorCode := v.presentationForStatus(scenario.Status)
		v.streams.Eprintln("")
		v.eprintRuleHeading(colorCode, prefix, scenarioName)

		if len(scenario.Steps) == 0 && scenario.Status == checks.StatusError {
			continue
		}

		for _, step := range scenario.Steps {
			if step.IsImplied() && (step.Status == checks.StatusPass || step.Status == checks.StatusUnknown) {
				// We don't mention implied steps at all if they passed
				// or were skipped. They are implementation details that we
				// mention only if we are describing an error for them.
				continue
			}

			prefix, _, colorCode := v.presentationForStatus(step.Status)
			v.streams.Eprintf(
				"%s: %s\n",
				v.colorize.Color(fmt.Sprintf("[%s]%s", colorCode, prefix)),
				step.Name,
			)
			if step.Status == checks.StatusFail || step.Status == checks.StatusError {
				// In case of problems we'll describe all of the checkable
				// objects individually, to help narrow down the cause of
				// the failure.
				if step.Checks != nil {
					for _, elem := range step.Checks.ConfigResults.Elems {
						configAddr := elem.Key
						configResult := elem.Value
						if configResult.ObjectResults.Len() == 0 {
							// If we don't have any object results then either
							// this is an expanding object that expanded to zero
							// instances or expansion failed altogether. Either
							// way we'll just emit a single entry representing the
							// static config object just so we'll say _something_
							// about each checkable object.
							prefix, _, colorCode := v.presentationForStatus(configResult.Status)
							v.streams.Eprintf(
								"  - %s: %s\n",
								v.colorize.Color(fmt.Sprintf("[%s]%s", colorCode, prefix)),
								configAddr.String(),
							)
						}
						for _, elem := range configResult.ObjectResults.Elems {
							objAddr := elem.Key
							objResult := elem.Value

							prefix, symbol, colorCode := v.presentationForStatus(objResult.Status)
							if isExpectedFail := step.ExpectedFailures.Has(objAddr); isExpectedFail {
								// For an expected failure we use an inverted
								// presentation style where a failure is shown
								// with a checkmark and a success is shown as
								// a cross mark, along with some different
								// messaging to indicate why those are appearing.
								_, symbol, colorCode = v.presentationForStatus(objResult.Status.ForExpectedFailure())
								switch objResult.Status {
								case checks.StatusPass:
									prefix = "Unexpected pass"
								case checks.StatusUnknown:
									prefix = "Error instead of expected failure"
								case checks.StatusFail:
									prefix = "Expected failure"
								}
							}
							v.streams.Eprintf(
								"  %s: %s\n",
								v.colorize.Color(fmt.Sprintf("[%s]%s %s", colorCode, symbol, prefix)),
								objAddr.String(),
							)

							for _, msg := range objResult.FailureMessages {
								v.streams.Eprintf("      %s\n", msg)
							}
						}
					}
				}
				for _, diag := range step.Diagnostics {
					// TEMP: We'll just render the diagnostics in a similar
					// way as an errored checkable object, for now.
					var prefix, symbol, colorCode string
					switch diag.Severity() {
					case tfdiags.Error:
						prefix, symbol, colorCode = v.presentationForStatus(checks.StatusError)
					case tfdiags.Warning:
						prefix = "Warning"
						colorCode = "yellow"
						symbol = "!"
					default:
						prefix = "Problem"
						symbol = "*"
						colorCode = "reset"
					}
					v.streams.Eprintf(
						"  %s: %s\n",
						v.colorize.Color(fmt.Sprintf("[%s]%s %s", colorCode, symbol, prefix)),
						diag.Description().Summary,
					)
				}
			}
		}
	}

	overallStatus := checks.AggregateCheckStatusMap(
		results,
		func(k string, v *moduletest.ScenarioResult) checks.Status {
			return v.Status
		},
	)

	switch overallStatus {
	case checks.StatusPass:
		if len(results) > 0 {
			// This is not actually an error, but it's convenient if all of our
			// result output goes to the same stream for when this is running in
			// automation that might be gathering this output via a pipe.
			v.streams.Eprint(v.colorize.Color("\n[bold][green]Success![reset] All of the test assertions passed.\n\n"))
		} else {
			v.streams.Eprint(v.colorize.Color("\n[bold][yellow]No tests defined.[reset] This module doesn't have any test suites to run.\n\n"))
		}
	case checks.StatusFail:
		v.streams.Eprint(v.colorize.Color("\n[bold][red]Fail:[reset] Not all of the test scenarios passed.\n\n"))
	case checks.StatusError:
		// We won't say anything for error because that'll get described
		// by showing one or more error diagnostics.
	case checks.StatusUnknown:
		v.streams.Eprint(v.colorize.Color("\n[bold][yellow]Partial success:[reset] Some checks were skipped, but none failed.\n\n"))
	}

	// Try to flush any buffering that might be happening. (This isn't always
	// successful, depending on what sort of fd Stderr is connected to.)
	v.streams.Stderr.File.Sync()
}

func (v *testHuman) junitXMLResults(results map[string]*moduletest.ScenarioResult, filename string) tfdiags.Diagnostics {
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
	scenarioNames := make([]string, 0, len(results))
	for scenarioName := range results {
		scenarioNames = append(scenarioNames, scenarioName)
	}
	sort.Strings(scenarioNames)
	for _, scenarioName := range scenarioNames {
		scenario := results[scenarioName]

		xmlSuite := &TestSuite{
			Name: scenarioName,
		}
		xmlSuites.Suites = append(xmlSuites.Suites, xmlSuite)

		for _, step := range scenario.Steps {
			log.Printf("Should add JUnit representation of step %q", step.Name)
		}
		// TODO: Implement the rest of this
	}
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		"JUnit XML report not fully implemented",
		"Support for the JUnit XML format is currently incomplete.",
	))

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

func (v *testHuman) presentationForStatus(status checks.Status) (prefix, symbol, color string) {
	switch status {
	case checks.StatusFail:
		return "Fail", "\u2718", "red"
	case checks.StatusError:
		return "Error", "\u203C", "red"
	case checks.StatusPass:
		return "Pass", "\u2713", "green"
	case checks.StatusUnknown:
		return "Skipped", "…", "dark_gray"
	default:
		// If this is a status we don't recognize then we'll just use reset
		// as a no-op. We shouldn't get here because the above cases should
		// be exhaustive for all of the possible checks.Status values.
		return "Tested", "*", "reset"
	}
}
