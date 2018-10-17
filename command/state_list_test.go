package command

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/mitchellh/cli"
)

func TestStateList(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	expected := strings.TrimSpace(testStateListOutput) + "\n"
	actual := ui.OutputWriter.String()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
	}
}

func TestStateListWithID(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-id", "bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	expected := strings.TrimSpace(testStateListOutput) + "\n"
	actual := ui.OutputWriter.String()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
	}
}

func TestStateListWithNonExistentID(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-id", "baz",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that output is empty
	if ui.OutputWriter != nil {
		actual := ui.OutputWriter.String()
		if actual != "" {
			t.Fatalf("Expected an empty output but got: %q", actual)
		}
	}
}

func TestStateList_backendState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-list-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	expected := "null_resource.a\n"
	actual := ui.OutputWriter.String()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
	}
}

func TestStateList_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

const testStateListOutput = `
test_instance.foo
`
