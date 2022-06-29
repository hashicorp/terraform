package views

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Test is the view interface for the "terraform test" command.
type Test interface {
	// Results displays all of the given test results and runner diagnostics.
	//
	// For testing in particular, we split the usual single set of diagnostics
	// into multiple smaller sets of diagnostics attached to particular
	// test steps or test cases that encountered problems, and so the
	// runnerDiags are just the subset of diagnostics that describe problems
	// with the test runner itself, and does not include any diagnostics
	// related to the behavior of the individual test scenarios.
	Results(results map[addrs.ModuleTestScenario]*moduletest.ScenarioResult, runnerDiags tfdiags.Diagnostics, sources map[string][]byte) tfdiags.Diagnostics

	// Diagnostics displays just some arbitrary diagnostics, which are not
	// associated with any particular test results.
	//
	// The test command uses this method only for its own early init errors
	// if it's not able to get far enough to even start running tests. Once
	// it's run at least some of the tests it will use the Results method
	// instead, not call this method at all in that case.
	Diagnostics(tfdiags.Diagnostics)
}

// NewTest returns an implementation of Test configured to respect the
// settings described in the given arguments.
func NewTest(base *View, args arguments.TestOutput) Test {
	main := &testHuman{
		streams:         base.streams,
		showDiagnostics: base.Diagnostics,
		colorize:        base.colorize,
		junitXMLFile:    args.JUnitXMLFile,
		showAll:         args.ShowAll,
	}

	if args.JUnitXMLFile != "" {
		// CI systems that consume JUnit XML conventionally expect the results
		// to be emitted into a separate file on disk in addition to
		// terminal-friendly output on stdout, and so unlike many other commands
		// in this case we'll be using two views at once, where the JUnit XML
		// view writes to the designated filename and the main view will be
		// the one writing to the stdio streams.
		return testMulti{
			main,
			&testJUnitXML{args.JUnitXMLFile},
		}
	}

	return main
}

type testHuman struct {
	// This is the subset of functionality we need from the base view.
	streams         *terminal.Streams
	showDiagnostics func(diags tfdiags.Diagnostics)
	colorize        *colorstring.Colorize
	showAll         bool

	// If junitXMLFile is not empty then results will be written to
	// the given file path in addition to the usual output.
	junitXMLFile string
}

func (v *testHuman) Results(results map[addrs.ModuleTestScenario]*moduletest.ScenarioResult, runnerDiags tfdiags.Diagnostics, sources map[string][]byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(runnerDiags) != 0 {
		v.showDiagnostics(runnerDiags)
	}

	for scenarioAddr, scenarioResult := range results {
		aggrStatus := scenarioResult.AggregateStatus()

		// We'll always show at least the scenario's aggregate result. We
		// might also show details about the individual test cases inside,
		// depending on the aggregate status and the view settings.
		switch aggrStatus {
		case checks.StatusPass:
			v.streams.Printf(v.colorize.Color("[green]*[reset] [bold]%s[reset] passed\n"), scenarioAddr)
		case checks.StatusFail:
			v.streams.Printf(v.colorize.Color("[red]*[reset] [bold]%s[reset] failed\n"), scenarioAddr)
		case checks.StatusError:
			v.streams.Printf(v.colorize.Color("[red]*[reset] [bold]%s[reset] errored\n"), scenarioAddr)
		default:
			v.streams.Printf(v.colorize.Color("[yellow]*[reset] [bold]%s[reset] has unknown status\n"), scenarioAddr)
		}

		switch aggrStatus {
		case checks.StatusPass:
			if !v.showAll {
				continue
			}
		case checks.StatusUnknown:
			// It's unexpected for an entire scenario to have unknown
			// status, because that could only happen if we skipped some
			// of its test cases when there wasn't an error.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Test scenario with unknown result",
				fmt.Sprintf("Test scenario %s produced an unknown result. This is a bug in Terraform; please report it!", scenarioResult.Scenario.Addr()),
			))
			// We don't treat an unknown at this level as "skipped", since it
			// is weird to skip a whole scenario, so we'll let this fall
			// through to show details below in case that's useful to understand
			// why we ended up in this unexpected codepath.
		default:
			// all other statuses we'll just fall through and handle below
		}

		for _, stepResult := range scenarioResult.StepResults {
			aggrStatus := stepResult.AggregateStatus
			stepAddr := stepResult.Step.Addr()

			if aggrStatus == checks.StatusPass || aggrStatus == checks.StatusUnknown {
				if !v.showAll {
					continue // Don't mention passed/skipped at all unless asked to
				}
			}

			switch aggrStatus {
			case checks.StatusPass:
				v.streams.Printf(v.colorize.Color("  [green]*[reset] step %s passed\n"), stepAddr)
			case checks.StatusFail:
				v.streams.Printf(v.colorize.Color("  [red]*[reset] step %s failed\n"), stepAddr)
			case checks.StatusError:
				v.streams.Printf(v.colorize.Color("  [red]*[reset] step %s errored\n"), stepAddr)
			case checks.StatusUnknown:
				v.streams.Printf(v.colorize.Color("  [dark_gray]* step %s was skipped[reset]\n"), stepAddr)
			default:
				v.streams.Printf(v.colorize.Color("  [yellow]*[reset] step %s has unknown status\n"), stepAddr)
			}

			if len(stepResult.Diagnostics) != 0 {
				v.streams.Print(formatIndentedDiagnostics(
					stepResult.Diagnostics,
					2, v.streams, v.colorize,
					sources,
				))
			}

			// TODO: Sort the TestCaseResults by the addrs.ConfigCheckable and
			// the inner ObjectResults by the addrs.Checkable, so that we will
			// always produce the same results in the same order.

			for _, elem := range stepResult.TestCaseResults.Elems {
				configAddr := elem.Key
				caseResult := elem.Value

				showCheckResult := func(addrStr string, status checks.Status, failMsgs []string) {
					if status == checks.StatusPass || status == checks.StatusUnknown {
						if !v.showAll {
							return // Don't mention passed/skipped at all unless asked to
						}
					}

					switch status {
					case checks.StatusPass:
						v.streams.Printf(v.colorize.Color("    [green]*[reset] %s passed\n"), addrStr)
					case checks.StatusFail:
						if len(failMsgs) > 0 {
							v.streams.Printf(v.colorize.Color("    [red]*[reset] %s failed:\n"), addrStr)
						} else {
							v.streams.Printf(v.colorize.Color("    [red]*[reset] %s failed\n"), addrStr)
						}
					case checks.StatusError:
						v.streams.Printf(v.colorize.Color("    [red]*[reset] %s errored\n"), addrStr)
					case checks.StatusUnknown:
						v.streams.Printf(v.colorize.Color("    [dark_gray]* %s was skipped[reset]\n"), addrStr)
					default:
						v.streams.Printf(v.colorize.Color("    [yellow]*[reset] %s has unknown status\n"), addrStr)
					}

					for _, msg := range failMsgs {
						v.streams.Print(formatFailureMessageBullet(msg, 6, v.streams, v.colorize))
					}
				}

				// Our treatment of the test case itself depends on whether
				// it has any associated objects. If it does then we'll not
				// mention the overall test case at all and prefer to report
				// the objects instead, but the test case as a whole serves
				// as a fallback so we can say at least _something_ about
				// this test case even if it didn't have any dynamic objects.
				if caseResult.ObjectResults.Len() != 0 {
					for _, elem := range caseResult.ObjectResults.Elems {
						objectAddr := elem.Key
						objectResult := elem.Value
						showCheckResult(objectAddr.String(), objectResult.Status, objectResult.FailureMessages)
					}
				} else {
					showCheckResult(configAddr.String(), caseResult.AggregateStatus, nil)
				}

				if len(caseResult.Diagnostics) != 0 {
					v.streams.Print(formatIndentedDiagnostics(
						stepResult.Diagnostics,
						4, v.streams, v.colorize,
						sources,
					))
				}
			}
		}
	}

	return diags
}

func formatIndentedDiagnostics(diags tfdiags.Diagnostics, indent int, streams *terminal.Streams, colorize *colorstring.Colorize, sources map[string][]byte) string {
	var buf strings.Builder

	for _, diag := range diags {
		var orig string
		if colorize.Disable {
			orig = format.DiagnosticPlain(diag, sources, streams.Stdout.Columns()-indent)
		} else {
			orig = format.Diagnostic(diag, sources, colorize, streams.Stdout.Columns()-indent)
		}

		sc := bufio.NewScanner(strings.NewReader(orig))
		for sc.Scan() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(sc.Text())
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

func formatFailureMessageBullet(msg string, indent int, streams *terminal.Streams, colorize *colorstring.Colorize) string {
	wrapped := format.WordWrap(msg, streams.Stdout.Columns()-indent-2)

	var buf strings.Builder
	sc := bufio.NewScanner(strings.NewReader(wrapped))
	sc.Scan()
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(colorize.Color("[red]- "))
	buf.WriteString(sc.Text())
	buf.WriteByte('\n')
	for sc.Scan() {
		buf.WriteString(strings.Repeat(" ", indent+2))
		buf.WriteString(sc.Text())
		buf.WriteByte('\n')
	}
	return buf.String()
}

func (v *testHuman) Diagnostics(diags tfdiags.Diagnostics) {
	if len(diags) != 0 {
		v.showDiagnostics(diags)
	}
}

type testJUnitXML struct {
	filename string
}

func (v *testJUnitXML) Results(results map[addrs.ModuleTestScenario]*moduletest.ScenarioResult, runnerDiags tfdiags.Diagnostics, sources map[string][]byte) tfdiags.Diagnostics {
	// TODO: Implement
	return nil
}

func (v *testJUnitXML) Diagnostics(diags tfdiags.Diagnostics) {
	// For non-test-run related errors we don't emit any JUnit XML form at
	// all, since JUnit XML is only for describing test results.
}

// testMulti is a wrapper around multiple other implementations of Test
// that calls each of the wrapped implementions in turn.
//
// We use this whenever the user opts in to JUnit XML test reporting, because
// in that case we need to produce the JUnit XML report in addition to the
// primary view, rather than instead of.
type testMulti []Test

func (v testMulti) Results(results map[addrs.ModuleTestScenario]*moduletest.ScenarioResult, runnerDiags tfdiags.Diagnostics, sources map[string][]byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, innerV := range v {
		diags = diags.Append(
			innerV.Results(results, runnerDiags, sources),
		)
	}
	return diags
}

func (v testMulti) Diagnostics(diags tfdiags.Diagnostics) {
	for _, innerV := range v {
		innerV.Diagnostics(diags)
	}
}
