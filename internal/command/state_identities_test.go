// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/cli"
)

func TestStateIdentities(t *testing.T) {
	state := testStateWithIdentity()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	expected := `{
		"test_instance.foo": {"id": "my-foo-id"},
		"test_instance.bar": {"id": "my-bar-id"}
	}`
	actual := ui.OutputWriter.String()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %s", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %s", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
	}
}

func TestStateIdentitiesWithNoIdentityInfo(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that an empty output is displayed with no error
	expected := `{}`
	actual := ui.OutputWriter.String()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %s", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %s", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
	}
}

func TestStateIdentitiesFilterByID(t *testing.T) {
	state := testStateWithIdentity()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
		"-id", "foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	expected := `{
		"test_instance.foo": {"id": "my-foo-id"}
	}`
	actual := ui.OutputWriter.String()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %s", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %s", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
	}
}

func TestStateIdentitiesWithNonExistentID(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
		"-id", "baz",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that output is empty
	if ui.OutputWriter != nil {
		actual := ui.OutputWriter.String()
		if actual != "{}\n" {
			t.Fatalf("Expected an empty output but got: %q", actual)
		}
	}
}

func TestStateIdentitiesWithNoJsonFlag(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
	}
	// Should return an error because the -json flag is required
	if code := c.Run(args); code != 1 {
		t.Fatalf("expected error: \n%s", ui.OutputWriter.String())
	}
}

func TestStateIdentities_backendDefaultState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-identities-backend-default"), td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{"-json"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	expected := `{
		"null_resource.a": {
			"project": "my-project",
			"role": "roles/viewer",
			"member": "user:peter@example.com"
		}
	}`
	actual := ui.OutputWriter.String()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %s", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %s", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
	}
}

func TestStateIdentities_backendOverrideState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-identities-backend-default"), td)

	// Rename the state file to a custom name to simulate a custom state file
	err := os.Rename(filepath.Join(td, "terraform.tfstate"), filepath.Join(td, "custom.tfstate"))
	if err != nil {
		t.Fatalf("Failed to rename state file: %s", err)
	}
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the command with a custom state file
	args := []string{"-state=custom.tfstate", "-json"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d", code)
	}

	// Test that outputs were displayed
	expected := `{
		"null_resource.a": {
			"project": "my-project",
			"role": "roles/viewer",
			"member": "user:peter@example.com"
		}
	}`
	actual := ui.OutputWriter.String()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %s", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %s", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
	}
}

func TestStateIdentities_noState(t *testing.T) {
	testCwd(t)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
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

func TestStateIdentities_modules(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-identities-nested-modules"), td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StateIdentitiesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	t.Run("list resources in module and submodules", func(t *testing.T) {
		args := []string{"-json", "module.nest"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d", code)
		}

		// resources in the module and any submodules should be included in the outputs
		expected := `{
			"module.nest.test_instance.nest": {
				"project": "my-project-nest",
				"role": "roles/viewer-nest"
			},
			"module.nest.module.subnest.test_instance.subnest": {
				"project": "my-project-subnest",
				"role": "roles/viewer-subnest"
			}
		}`
		actual := ui.OutputWriter.String()

		// Normalize JSON strings
		var expectedJSON, actualJSON map[string]interface{}
		if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
			t.Fatalf("Failed to unmarshal expected JSON: %s", err)
		}
		if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
			t.Fatalf("Failed to unmarshal actual JSON: %s", err)
		}

		if !reflect.DeepEqual(expectedJSON, actualJSON) {
			t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
		}
	})

	t.Run("submodule has resources only", func(t *testing.T) {
		// now get the state for a module that has no resources, only another nested module
		ui.OutputWriter.Reset()
		args := []string{"-json", "module.nonexist"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d", code)
		}
		expected := `{
			"module.nonexist.module.child.test_instance.child": {
				"project": "my-project-child",
				"role": "roles/viewer-child"
			}
		}`
		actual := ui.OutputWriter.String()

		// Normalize JSON strings
		var expectedJSON, actualJSON map[string]interface{}
		if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
			t.Fatalf("Failed to unmarshal expected JSON: %s", err)
		}
		if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
			t.Fatalf("Failed to unmarshal actual JSON: %s", err)
		}

		if !reflect.DeepEqual(expectedJSON, actualJSON) {
			t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
		}
	})

	t.Run("expanded module", func(t *testing.T) {
		// finally get the state for a module with an index
		ui.OutputWriter.Reset()
		args := []string{"-json", "module.count"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d: %s", code, ui.ErrorWriter.String())
		}
		expected := `{
			"module.count[0].test_instance.count": {
				"project": "my-project-count-0",
				"role": "roles/viewer-count-0"
			},
			"module.count[1].test_instance.count": {
				"project": "my-project-count-1",
				"role": "roles/viewer-count-1"
			}
		}`
		actual := ui.OutputWriter.String()

		// Normalize JSON strings
		var expectedJSON, actualJSON map[string]interface{}
		if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
			t.Fatalf("Failed to unmarshal expected JSON: %s", err)
		}
		if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
			t.Fatalf("Failed to unmarshal actual JSON: %s", err)
		}

		if !reflect.DeepEqual(expectedJSON, actualJSON) {
			t.Fatalf("Expected:\n%q\n\nTo equal: %q", expected, actual)
		}
	})

	t.Run("completely nonexistent module", func(t *testing.T) {
		// finally get the state for a module with an index
		ui.OutputWriter.Reset()
		args := []string{"-json", "module.notevenalittlebit"}
		if code := c.Run(args); code != 1 {
			t.Fatalf("bad: %d: %s", code, ui.OutputWriter.String())
		}
	})

}
