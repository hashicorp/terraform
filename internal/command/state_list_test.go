// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"strings"
	"testing"

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

func TestStateList_backendDefaultState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-list-backend-default"), td)
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

func TestStateList_backendCustomState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-list-backend-custom"), td)
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

func TestStateList_backendOverrideState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-list-backend-custom"), td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// This test is configured to use a local backend that has
	// a custom path defined. So we test if we can still pass
	// is a user defined state file that will then override the
	// one configured in the backend. As this file does not exist
	// it should exit with a no state found error.
	args := []string{"-state=" + DefaultStateFilename}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d", code)
	}
	if !strings.Contains(ui.ErrorWriter.String(), "No state file was found!") {
		t.Fatalf("expected a no state file error, got: %s", ui.ErrorWriter.String())
	}
}

func TestStateList_noState(t *testing.T) {
	testCwd(t)

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
		t.Fatalf("bad: %d", code)
	}
}

func TestStateList_modules(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-list-nested-modules"), td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateListCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	t.Run("list resources in module and submodules", func(t *testing.T) {
		args := []string{"module.nest"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d", code)
		}

		// resources in the module and any submodules should be included in the outputs
		expected := "module.nest.test_instance.nest\nmodule.nest.module.subnest.test_instance.subnest\n"
		actual := ui.OutputWriter.String()
		if actual != expected {
			t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
		}
	})

	t.Run("submodule has resources only", func(t *testing.T) {
		// now get the state for a module that has no resources, only another nested module
		ui.OutputWriter.Reset()
		args := []string{"module.nonexist"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d", code)
		}
		expected := "module.nonexist.module.child.test_instance.child\n"
		actual := ui.OutputWriter.String()
		if actual != expected {
			t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
		}
	})

	t.Run("expanded module", func(t *testing.T) {
		// finally get the state for a module with an index
		ui.OutputWriter.Reset()
		args := []string{"module.count"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d", code)
		}
		expected := "module.count[0].test_instance.count\nmodule.count[1].test_instance.count\n"
		actual := ui.OutputWriter.String()
		if actual != expected {
			t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
		}
	})

	t.Run("completely nonexistent module", func(t *testing.T) {
		// finally get the state for a module with an index
		ui.OutputWriter.Reset()
		args := []string{"module.notevenalittlebit"}
		if code := c.Run(args); code != 1 {
			t.Fatalf("bad: %d", code)
		}
	})

}

const testStateListOutput = `
test_instance.foo
`
