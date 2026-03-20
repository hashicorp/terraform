// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestStateRm(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateRmOutputOriginal)
}

func TestStateRm_stateStore(t *testing.T) {
	// Create a temporary working directory
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged"), td)
	t.Chdir(td)

	// Get bytes describing a state containing resources
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	var stateBuf bytes.Buffer
	if err := statefile.Write(statefile.New(state, "", 1), &stateBuf); err != nil {
		t.Fatalf("error during test setup: %s", err)
	}

	// Create a mock that contains a persisted "default" state that uses the bytes from above.
	mockProvider := mockPluggableStateStorageProvider()
	mockProvider.MockStates = map[string]interface{}{
		"default": stateBuf.Bytes(),
	}
	mockProviderAddress := addrs.NewDefaultProvider("test")

	// Make the mock assert that the removed resource is not present when the new state is persisted
	keptResource := "test_instance.bar"
	removedResource := "test_instance.foo"
	mockProvider.WriteStateBytesFn = func(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
		r := bytes.NewReader(req.Bytes)
		file, err := statefile.Read(r)
		if err != nil {
			t.Fatal(err)
		}

		root := file.State.Modules[""]
		if _, ok := root.Resources[keptResource]; !ok {
			t.Fatalf("expected the new state to keep the %s resource, but it couldn't be found", keptResource)
		}
		if _, ok := root.Resources[removedResource]; ok {
			t.Fatalf("expected the %s resource to be removed from the state, but it is present", removedResource)
		}
		return providers.WriteStateBytesResponse{}
	}

	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				Ui: ui,
			},
		},
	}

	args := []string{
		removedResource,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// See the mock definition above for logic that asserts what the new state will look like after removing the resource.
}

func TestStateRmNotChildModule(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		// This second instance has the same local address as the first but
		// is in a child module. Older versions of Terraform would incorrectly
		// remove this one too, since they failed to check the module address.
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey)),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, `
<no state>
module.child:
  test_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value

module.child:
  test_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`)
}

func TestStateRmNoArgs(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code == 0 {
		t.Errorf("expected non-zero exit code, got: %d", code)
	}

	if msg := ui.ErrorWriter.String(); !strings.Contains(msg, "At least one address") {
		t.Errorf("not the error we were looking for:\n%s", msg)
	}

}

func TestStateRmNonExist(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.baz", // doesn't exist in the state constructed above
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("expected exit status %d, got: %d", 1, code)
	}
}

func TestStateRm_backupExplicit(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)
	backupPath := statePath + ".backup.test"

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-backup", backupPath,
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateRmOutputOriginal)
}

func TestStateRm_noState(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{"foo"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestStateRm_constVariable(t *testing.T) {
	t.Run("missing value", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		view, _ := testView(t)
		c := &StateRmCommand{
			StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
					WorkingDir:       wd,
				},
			},
		}

		args := []string{"module.child.test_instance.test"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error, got 0")
		}

		errStr := ui.ErrorWriter.String()
		if !strings.Contains(errStr, "No value for required variable") {
			t.Fatalf("expected missing variable error, got: %s", errStr)
		}
	})

	t.Run("value via cli", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		view, _ := testView(t)
		c := &StateRmCommand{
			StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
					WorkingDir:       wd,
				},
			},
		}

		args := []string{"-var", "module_name=child", "module.child.test_instance.test"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		actual := strings.TrimSpace(testStateRead(t, "terraform.tfstate").String())
		expected := strings.TrimSpace(`<no state>`)
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Fatalf("unexpected state output\n%s", diff)
		}
	})

	t.Run("value via backend", func(t *testing.T) {
		mockBackend := TestNewVariableBackend(map[string]string{
			"module_name": "child",
		})
		backendInit.Set("local-vars", func() backend.Backend { return mockBackend })
		defer backendInit.Set("local-vars", nil)

		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var-backend")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		view, _ := testView(t)
		c := &StateRmCommand{
			StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
					WorkingDir:       wd,
				},
			},
		}

		args := []string{"module.child.test_instance.test"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		actual := strings.TrimSpace(testStateRead(t, "terraform.tfstate").String())
		expected := strings.TrimSpace(`<no state>`)
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Fatalf("unexpected state output\n%s", diff)
		}
	})
}

func TestStateRm_needsInit(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change"), td)
	t.Chdir(td)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{"foo"}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "Backend initialization") {
		t.Fatalf("expected initialization error, got:\n%s", ui.ErrorWriter.String())
	}
}

func TestStateRm_backendState(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unchanged"), td)
	t.Chdir(td)

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	statePath := "local-state.tfstate"
	backupPath := "local-state.backup"

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatalf("failed to create state file %s: %s", statePath, err)
	}
	defer f.Close()

	err = writeStateForTesting(state, f)
	if err != nil {
		t.Fatalf("failed to write state to file %s: %s", statePath, err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-backup", backupPath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateRmOutputOriginal)
}

func TestStateRm_checkRequiredVersion(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("command-check-required-version"), td)
	t.Chdir(td)

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, ui.ErrorWriter.String(), ui.OutputWriter.String())
	}

	// State is unchanged
	testStateOutput(t, statePath, testStateRmOutputOriginal)

	// Required version diags are correct
	errStr := ui.ErrorWriter.String()
	if !strings.Contains(errStr, `required_version = "~> 0.9.0"`) {
		t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
	}
	if strings.Contains(errStr, `required_version = ">= 0.13.0"`) {
		t.Fatalf("output should not point to met version constraint, but is:\n\n%s", errStr)
	}
}

const testStateRmOutputOriginal = `
test_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateRmOutput = `
test_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`
