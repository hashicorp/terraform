package views

import (
	"fmt"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/moduletest"
)

// Test renders outputs for test executions.
type Test interface {
	// Abstract should print an early summary of the tests that will be
	// executed. This will be called before the tests have been executed so
	// the status for everything within suite will be test.Pending.
	//
	// This should be used to state what is going to be tested.
	Abstract(suite *moduletest.Suite)

	// Conclusion should print out a summary of the tests including their
	// completed status.
	Conclusion(suite *moduletest.Suite)

	// File prints out the summary for an entire test file.
	File(file *moduletest.File)

	// Run prints out the summary for a single test run block.
	Run(run *moduletest.Run)
}

func NewTest(vt arguments.ViewType, view *View) Test {
	switch vt {
	case arguments.ViewJSON:
		// TODO(liamcervante): Add support for JSON outputs.
		panic("not supported yet")
	case arguments.ViewHuman:
		return &TestHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type TestHuman struct {
	view *View
}

var _ Test = (*TestHuman)(nil)

func (t *TestHuman) Abstract(_ *moduletest.Suite) {
	// Do nothing, we don't print an abstract for the human view.
}

func (t *TestHuman) Conclusion(suite *moduletest.Suite) {
	t.view.streams.Println()

	counts := make(map[moduletest.Status]int)
	for _, file := range suite.Files {
		for _, run := range file.Runs {
			count := counts[run.Status]
			counts[run.Status] = count + 1
		}
	}

	if suite.Status <= moduletest.Skip {
		// Then no tests.
		t.view.streams.Printf("Executed 0 tests")
		if counts[moduletest.Skip] > 0 {
			t.view.streams.Printf(", %d skipped.\n", counts[moduletest.Skip])
		} else {
			t.view.streams.Println(".")
		}
		return
	}

	if suite.Status == moduletest.Pass {
		t.view.streams.Print(t.view.colorize.Color("[green]Success![reset]"))
	} else {
		t.view.streams.Print(t.view.colorize.Color("[red]Failure![reset]"))
	}

	t.view.streams.Printf(" %d passed, %d failed", counts[moduletest.Pass], counts[moduletest.Fail]+counts[moduletest.Error])
	if counts[moduletest.Skip] > 0 {
		t.view.streams.Printf(", %d skipped.\n", counts[moduletest.Skip])
	} else {
		t.view.streams.Println(".")
	}
}

func (t *TestHuman) File(file *moduletest.File) {
	t.view.streams.Printf("%s... %s\n", file.Name, colorizeTestStatus(file.Status, t.view.colorize))
}

func (t *TestHuman) Run(run *moduletest.Run) {
	t.view.streams.Printf("  run %q... %s\n", run.Name, colorizeTestStatus(run.Status, t.view.colorize))

	// Finally we'll print out a summary of the diagnostics from the run.
	t.view.Diagnostics(run.Diagnostics)
}

func colorizeTestStatus(status moduletest.Status, color *colorstring.Colorize) string {
	switch status {
	case moduletest.Error, moduletest.Fail:
		return color.Color("[red]fail[reset]")
	case moduletest.Pass:
		return color.Color("[green]pass[reset]")
	case moduletest.Skip:
		return color.Color("[light_gray]skip[reset]")
	case moduletest.Pending:
		return color.Color("[light_gray]pending[reset]")
	default:
		panic("unrecognized status: " + status.String())
	}
}
