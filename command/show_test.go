package command

import (
	"path/filepath"
	"testing"

	"github.com/mitchellh/cli"
)

func TestShow(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_noArgs(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	defer testChdir(t, filepath.Dir(statePath))()

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_noArgsNoState(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	defer testChdir(t, filepath.Dir(statePath))()

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_plan(t *testing.T) {
	planPath := testPlanFileNoop(t)

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestShow_state(t *testing.T) {
	originalState := testState()
	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}
