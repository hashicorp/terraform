// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestStateIdentities(t *testing.T) {
	state := testStateWithIdentity()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
	}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that outputs were displayed
	expected := `{
		"test_instance.foo": {"id": "my-foo-id"},
		"test_instance.bar": {"id": "my-bar-id"}
	}`
	actual := out.Stdout()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]any
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
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
	}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that an empty output is displayed with no error
	expected := `{}`
	actual := out.Stdout()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]any
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
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
		"-id", "foo",
	}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that outputs were displayed
	expected := `{
		"test_instance.foo": {"id": "my-foo-id"}
	}`
	actual := out.Stdout()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]any
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
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
		"-id", "baz",
	}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that output is empty
	actual := out.Stdout()
	if actual != "{}\n" {
		t.Fatalf("Expected an empty output but got: %q", actual)
	}
}

func TestStateIdentitiesWithNoJsonFlag(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
	}
	// Should return an error because the -json flag is required
	code := c.Run(args)
	out := done(t)
	if code != 1 {
		t.Fatalf("expected error: \n%s", out.Stderr())
	}
}

func TestStateIdentities_backendDefaultState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-identities-backend-default"), td)
	t.Chdir(td)

	p := testProvider()
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{"-json"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that outputs were displayed
	expected := `{
		"null_resource.a": {
			"project": "my-project",
			"role": "roles/viewer",
			"member": "user:peter@example.com"
		}
	}`
	actual := out.Stdout()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]any
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
	t.Chdir(td)

	p := testProvider()
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	// Run the command with a custom state file
	args := []string{"-state=custom.tfstate", "-json"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that outputs were displayed
	expected := `{
		"null_resource.a": {
			"project": "my-project",
			"role": "roles/viewer",
			"member": "user:peter@example.com"
		}
	}`
	actual := out.Stdout()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]any
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
	tmp := t.TempDir()
	t.Chdir(tmp)

	p := testProvider()
	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		},
	}

	args := []string{}
	code := c.Run(args)
	out := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}
}

func TestStateIdentities_modules(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-identities-nested-modules"), td)
	t.Chdir(td)

	p := testProvider()

	t.Run("list resources in module and submodules", func(t *testing.T) {
		view, done := testView(t)
		c := &StateIdentitiesCommand{
			StateMeta: StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			},
		}

		args := []string{"-json", "module.nest"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
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
		actual := out.Stdout()

		// Normalize JSON strings
		var expectedJSON, actualJSON map[string]any
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
		view, done := testView(t)
		c := &StateIdentitiesCommand{
			StateMeta: StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			},
		}

		args := []string{"-json", "module.nonexist"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
		}
		expected := `{
			"module.nonexist.module.child.test_instance.child": {
				"project": "my-project-child",
				"role": "roles/viewer-child"
			}
		}`
		actual := out.Stdout()

		// Normalize JSON strings
		var expectedJSON, actualJSON map[string]any
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
		view, done := testView(t)
		c := &StateIdentitiesCommand{
			StateMeta: StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			},
		}

		args := []string{"-json", "module.count"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("bad: %d: %s", code, out.Stderr())
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
		actual := out.Stdout()

		// Normalize JSON strings
		var expectedJSON, actualJSON map[string]any
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
		view, done := testView(t)
		c := &StateIdentitiesCommand{
			StateMeta: StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			},
		}

		args := []string{"-json", "module.notevenalittlebit"}
		code := c.Run(args)
		out := done(t)
		if code != 1 {
			t.Fatalf("bad: %d: %s", code, out.Stdout())
		}
	})

}

func TestStateIdentities_stateStore(t *testing.T) {
	// We need configuration present to force pluggable state storage to be used
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged/provider-managed-by-terraform"), td)
	t.Chdir(td)

	// Get a state file, that contains identity information,as bytes
	state := testStateWithIdentity()
	var stateBuf bytes.Buffer
	if err := statefile.Write(statefile.New(state, "", 1), &stateBuf); err != nil {
		t.Fatalf("error during test setup: %s", err)
	}
	stateBytes := stateBuf.Bytes()

	// Create a mock that contains a persisted "default" state that uses the bytes from above.
	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithSingleState(
		"test_store",
		"default",
		stateBytes,
	)
	mockProviderAddress := addrs.NewDefaultProvider("test")

	view, done := testView(t)
	c := &StateIdentitiesCommand{
		StateMeta: StateMeta{
			Meta: Meta{
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				View: view,
			},
		},
	}

	args := []string{"-json"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, out.Stderr())
	}

	// Test that outputs were displayed
	expected := `{
  "test_instance.bar": {
    "id": "my-bar-id"
  },
  "test_instance.foo": {
    "id": "my-foo-id"
  }
}
`
	actual := out.Stdout()

	// Normalize JSON strings
	var expectedJSON, actualJSON map[string]any
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
