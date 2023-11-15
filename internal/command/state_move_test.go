// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestStateMove(t *testing.T) {
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
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
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
	c := &StateMoveCommand{
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
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateMoveOutput)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveOutputOriginal)

	// Change the single instance to a counted instance
	args = []string{
		"-state", statePath,
		"test_instance.bar",
		"test_instance.bar[0]",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// extract the resource and verify the mode
	s := testStateRead(t, statePath)
	addr, diags := addrs.ParseAbsResourceStr("test_instance.bar")
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	for key := range s.Resource(addr).Instances {
		if _, ok := key.(addrs.IntKey); !ok {
			t.Fatalf("expected each mode List, got key %q", key)
		}
	}

	// change from list to map
	args = []string{
		"-state", statePath,
		"test_instance.bar[0]",
		"test_instance.bar[\"baz\"]",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// extract the resource and verify the mode
	s = testStateRead(t, statePath)
	addr, diags = addrs.ParseAbsResourceStr("test_instance.bar")
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	for key := range s.Resource(addr).Instances {
		if _, ok := key.(addrs.StringKey); !ok {
			t.Fatalf("expected each mode map, found key %q", key)
		}
	}

	// change from from map back to single
	args = []string{
		"-state", statePath,
		"test_instance.bar[\"baz\"]",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// extract the resource and verify the mode
	s = testStateRead(t, statePath)
	addr, diags = addrs.ParseAbsResourceStr("test_instance.bar")
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	for key := range s.Resource(addr).Instances {
		if key != addrs.NoKey {
			t.Fatalf("expected no each mode, found key %q", key)
		}
	}

}

func TestStateMove_backupAndBackupOutOptionsWithNonLocalBackend(t *testing.T) {
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
	})

	t.Run("backup option specified", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer testChdir(t, td)()

		backupPath := filepath.Join(td, "backup")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMoveCommand{
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
			"test_instance.bar",
		}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
		}

		gotErr := ui.ErrorWriter.String()
		wantErr := `
Error: Invalid command line options: -backup

Command line options -backup and -backup-out are legacy options that operate
on a local state file only. You must specify a local state file with the
-state option or switch to the local backend.

`
		if gotErr != wantErr {
			t.Fatalf("expected error\ngot:%s\n\nwant:%s", gotErr, wantErr)
		}
	})

	t.Run("backup-out option specified", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer testChdir(t, td)()

		backupOutPath := filepath.Join(td, "backup-out")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMoveCommand{
			StateMeta{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					Ui:               ui,
					View:             view,
				},
			},
		}

		args := []string{
			"-backup-out", backupOutPath,
			"test_instance.foo",
			"test_instance.bar",
		}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
		}

		gotErr := ui.ErrorWriter.String()
		wantErr := `
Error: Invalid command line options: -backup-out

Command line options -backup and -backup-out are legacy options that operate
on a local state file only. You must specify a local state file with the
-state option or switch to the local backend.

`
		if gotErr != wantErr {
			t.Fatalf("expected error\ngot:%s\n\nwant:%s", gotErr, wantErr)
		}
	})

	t.Run("backup and backup-out options specified", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer testChdir(t, td)()

		backupPath := filepath.Join(td, "backup")
		backupOutPath := filepath.Join(td, "backup-out")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMoveCommand{
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
			"-backup-out", backupOutPath,
			"test_instance.foo",
			"test_instance.bar",
		}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
		}

		gotErr := ui.ErrorWriter.String()
		wantErr := `
Error: Invalid command line options: -backup, -backup-out

Command line options -backup and -backup-out are legacy options that operate
on a local state file only. You must specify a local state file with the
-state option or switch to the local backend.

`
		if gotErr != wantErr {
			t.Fatalf("expected error\ngot:%s\n\nwant:%s", gotErr, wantErr)
		}
	})

	t.Run("backup option specified with state option", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer testChdir(t, td)()

		statePath := testStateFile(t, state)
		backupPath := filepath.Join(td, "backup")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMoveCommand{
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
			"-backup", backupPath,
			"test_instance.foo",
			"test_instance.bar",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		// Test it is correct
		testStateOutput(t, statePath, testStateMoveBackupAndBackupOutOptionsWithNonLocalBackendOutput)
	})

	t.Run("backup-out option specified with state option", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer testChdir(t, td)()

		statePath := testStateFile(t, state)
		backupOutPath := filepath.Join(td, "backup-out")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMoveCommand{
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
			"-backup-out", backupOutPath,
			"test_instance.foo",
			"test_instance.bar",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		// Test it is correct
		testStateOutput(t, statePath, testStateMoveBackupAndBackupOutOptionsWithNonLocalBackendOutput)
	})
}

func TestStateMove_resourceToInstance(t *testing.T) {
	// A single resource (no count defined)
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
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceProvider(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Absolute(addrs.RootModuleInstance),
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
	c := &StateMoveCommand{
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
		"test_instance.bar[0]",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, `
test_instance.bar.0:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.baz:
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
	testStateOutput(t, backups[0], testStateMoveOutputOriginal)
}

func TestStateMove_resourceToInstanceErr(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceProvider(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Absolute(addrs.RootModuleInstance),
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := cli.NewMockUi()
	view, _ := testView(t)

	c := &StateMoveCommand{
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
		"test_instance.bar[0]",
	}

	if code := c.Run(args); code == 0 {
		t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
	}

	expectedErr := `
Error: Invalid target address

Cannot move test_instance.foo to test_instance.bar[0]: the source is a whole
resource (not a resource instance) so the target must also be a whole
resource.

`
	errOutput := ui.ErrorWriter.String()
	if errOutput != expectedErr {
		t.Errorf("wrong output\n%s", cmp.Diff(errOutput, expectedErr))
	}
}

func TestStateMove_resourceToInstanceErrInAutomation(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceProvider(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Absolute(addrs.RootModuleInstance),
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
	c := &StateMoveCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides:    metaOverridesForProvider(p),
				Ui:                  ui,
				View:                view,
				RunningInAutomation: true,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"test_instance.bar[0]",
	}

	if code := c.Run(args); code == 0 {
		t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
	}

	expectedErr := `
Error: Invalid target address

Cannot move test_instance.foo to test_instance.bar[0]: the source is a whole
resource (not a resource instance) so the target must also be a whole
resource.

`
	errOutput := ui.ErrorWriter.String()
	if errOutput != expectedErr {
		t.Errorf("Unexpected diff.\ngot:\n%s\nwant:\n%s\n", errOutput, expectedErr)
		t.Errorf("%s", cmp.Diff(errOutput, expectedErr))
	}
}

func TestStateMove_instanceToResource(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
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
				Name: "baz",
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
	c := &StateMoveCommand{
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
		"test_instance.foo[0]",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.baz:
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
test_instance.baz:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.0:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`)
}

func TestStateMove_instanceToNewResource(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
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
	c := &StateMoveCommand{
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
		"test_instance.foo[0]",
		"test_instance.bar[\"new\"]",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, `
test_instance.bar["new"]:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`)

	// now move the instance to a new resource in a new module
	args = []string{
		"-state", statePath,
		"test_instance.bar[\"new\"]",
		"module.test.test_instance.baz[\"new\"]",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, `
<no state>
module.test:
  test_instance.baz["new"]:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`)
}

func TestStateMove_differentResourceTypes(t *testing.T) {
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
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"test_network.bar",
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
	}

	gotErr := ui.ErrorWriter.String()
	wantErr := `
Error: Invalid state move request

Cannot move test_instance.foo to test_network.bar: resource types don't
match.

`
	if gotErr != wantErr {
		t.Fatalf("expected initialization error\ngot:\n%s\n\nwant:%s", gotErr, wantErr)
	}
}

// don't modify backend state is we supply a -state flag
func TestStateMove_explicitWithBackend(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	defer testChdir(t, td)()

	backupPath := filepath.Join(td, "backup")

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
				Name: "baz",
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

	// init our backend
	ui := new(cli.MockUi)
	view, _ := testView(t)
	ic := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := ic.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// only modify statePath
	p := testProvider()
	ui = new(cli.MockUi)
	c := &StateMoveCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args = []string{
		"-backup", backupPath,
		"-state", statePath,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateMoveOutput)
}

func TestStateMove_backupExplicit(t *testing.T) {
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
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
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
	c := &StateMoveCommand{
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
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateMoveOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateMoveOutputOriginal)
}

func TestStateMove_stateOutNew(t *testing.T) {
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
	})
	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"-state-out", stateOutPath,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, stateOutPath, testStateMoveOutput_stateOut)
	testStateOutput(t, statePath, testStateMoveOutput_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveOutput_stateOutOriginal)
}

func TestStateMove_stateOutExisting(t *testing.T) {
	stateSrc := states.BuildState(func(s *states.SyncState) {
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
	})
	statePath := testStateFile(t, stateSrc)

	stateDst := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "qux",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	stateOutPath := testStateFile(t, stateDst)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"-state-out", stateOutPath,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, stateOutPath, testStateMoveExisting_stateDst)
	testStateOutput(t, statePath, testStateMoveExisting_stateSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveExisting_stateSrcOriginal)

	backups = testStateBackups(t, filepath.Dir(stateOutPath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveExisting_stateDstOriginal)
}

func TestStateMove_noState(t *testing.T) {
	testCwd(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{"from", "to"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestStateMove_stateOutNew_count(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
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
				Name: "foo",
			}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
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
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"-state-out", stateOutPath,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, stateOutPath, testStateMoveCount_stateOut)
	testStateOutput(t, statePath, testStateMoveCount_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveCount_stateOutOriginal)
}

// Modules with more than 10 resources were sorted lexically, causing the
// indexes in the new location to change.
func TestStateMove_stateOutNew_largeCount(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		// test_instance.foo has 11 instances, all the same except for their ids
		for i := 0; i < 11; i++ {
			s.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_instance",
					Name: "foo",
				}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance),
				&states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(fmt.Sprintf(`{"id":"foo%d","foo":"value","bar":"value"}`, i)),
					Status:    states.ObjectReady,
				},
				addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
			)
		}
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
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
	})
	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"-state-out", stateOutPath,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, stateOutPath, testStateMoveLargeCount_stateOut)
	testStateOutput(t, statePath, testStateMoveLargeCount_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveLargeCount_stateOutOriginal)
}

func TestStateMove_stateOutNew_nestedModule(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("foo", addrs.NoKey).Child("child1", addrs.NoKey)),
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
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("foo", addrs.NoKey).Child("child2", addrs.NoKey)),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"-state-out", stateOutPath,
		"module.foo",
		"module.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, stateOutPath, testStateMoveNestedModule_stateOut)
	testStateOutput(t, statePath, testStateMoveNestedModule_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveNestedModule_stateOutOriginal)
}

func TestStateMove_toNewModule(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
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
	})

	statePath := testStateFile(t, state)
	stateOutPath1 := statePath + ".out1"
	stateOutPath2 := statePath + ".out2"

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"-state-out", stateOutPath1,
		"test_instance.bar",
		"module.bar.test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, stateOutPath1, testStateMoveNewModule_stateOut)
	testStateOutput(t, statePath, testStateMoveNestedModule_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveNewModule_stateOutOriginal)

	// now verify we can move the module itself
	args = []string{
		"-state", stateOutPath1,
		"-state-out", stateOutPath2,
		"module.bar",
		"module.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	testStateOutput(t, stateOutPath2, testStateMoveModuleNewModule_stateOut)
}

func TestStateMove_withinBackend(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unchanged"), td)
	defer testChdir(t, td)()

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
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	// the local backend state file is "foo"
	statePath := "local-state.tfstate"
	backupPath := "local-state.backup"

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := writeStateForTesting(state, f); err != nil {
		t.Fatal(err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testStateMoveOutput)
	testStateOutput(t, backupPath, testStateMoveOutputOriginal)
}

func TestStateMove_fromBackendToLocal(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unchanged"), td)
	defer testChdir(t, td)()

	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetResourceInstanceCurrent(
		mustResourceAddr("test_instance.foo").Resource.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
			Status:    states.ObjectReady,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	state.Module(addrs.RootModuleInstance).SetResourceInstanceCurrent(
		mustResourceAddr("test_instance.baz").Resource.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
			Status:    states.ObjectReady,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)

	// the local backend state file is "foo"
	statePath := "local-state.tfstate"

	// real "local" state file
	statePathOut := "real-local.tfstate"

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := writeStateForTesting(state, f); err != nil {
		t.Fatal(err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
			},
		},
	}

	args := []string{
		"-state-out", statePathOut,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePathOut, testStateMoveCount_stateOutSrc)

	// the backend state should be left with only baz
	testStateOutput(t, statePath, testStateMoveOriginal_backend)
}

// This test covers moving the only resource in a module to a new address in
// that module, which triggers the maybePruneModule functionality. This caused
// a panic report: https://github.com/hashicorp/terraform/issues/25520
func TestStateMove_onlyResourceInModule(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance.Child("foo", addrs.NoKey)),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	statePath := testStateFile(t, state)
	testStateOutput(t, statePath, testStateMoveOnlyResourceInModule_original)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"module.foo.test_instance.foo",
		"module.foo.test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateMoveOnlyResourceInModule_output)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMoveOnlyResourceInModule_original)
}

func TestStateMoveHelp(t *testing.T) {
	c := &StateMoveCommand{}
	if strings.ContainsRune(c.Help(), '\t') {
		t.Fatal("help text contains tab character, which will result in poor formatting")
	}
}

func TestStateMoveInvalidSourceAddress(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMoveCommand{
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
		"foo.bar1",
		"foo.bar2",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected error code 1, got:\n%d", code)
	}
}

func TestStateMove_checkRequiredVersion(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("command-check-required-version"), td)
	defer testChdir(t, td)()

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
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
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
	c := &StateMoveCommand{
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
		"test_instance.bar",
	}

	if code := c.Run(args); code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, ui.ErrorWriter.String(), ui.OutputWriter.String())
	}

	// State is unchanged
	testStateOutput(t, statePath, testStateMoveOutputOriginal)

	// Required version diags are correct
	errStr := ui.ErrorWriter.String()
	if !strings.Contains(errStr, `required_version = "~> 0.9.0"`) {
		t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
	}
	if strings.Contains(errStr, `required_version = ">= 0.13.0"`) {
		t.Fatalf("output should not point to met version constraint, but is:\n\n%s", errStr)
	}
}

const testStateMoveOutputOriginal = `
test_instance.baz:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value

  Dependencies:
    test_instance.foo
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveOutput = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.baz:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveBackupAndBackupOutOptionsWithNonLocalBackendOutput = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveCount_stateOut = `
test_instance.bar.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.1:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveCount_stateOutSrc = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveCount_stateOutOriginal = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.1:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveLargeCount_stateOut = `
test_instance.bar.0:
  ID = foo0
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.1:
  ID = foo1
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.2:
  ID = foo2
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.3:
  ID = foo3
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.4:
  ID = foo4
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.5:
  ID = foo5
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.6:
  ID = foo6
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.7:
  ID = foo7
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.8:
  ID = foo8
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.9:
  ID = foo9
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.bar.10:
  ID = foo10
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveLargeCount_stateOutSrc = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveLargeCount_stateOutOriginal = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.0:
  ID = foo0
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.1:
  ID = foo1
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.2:
  ID = foo2
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.3:
  ID = foo3
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.4:
  ID = foo4
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.5:
  ID = foo5
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.6:
  ID = foo6
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.7:
  ID = foo7
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.8:
  ID = foo8
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.9:
  ID = foo9
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.foo.10:
  ID = foo10
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveNestedModule_stateOut = `
<no state>
module.bar.child1:
  test_instance.foo:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
module.bar.child2:
  test_instance.foo:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMoveNewModule_stateOut = `
<no state>
module.bar:
  test_instance.bar:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMoveModuleNewModule_stateOut = `
<no state>
module.foo:
  test_instance.bar:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMoveNewModule_stateOutOriginal = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveNestedModule_stateOutSrc = `
<no state>
`

const testStateMoveNestedModule_stateOutOriginal = `
<no state>
module.foo.child1:
  test_instance.foo:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
module.foo.child2:
  test_instance.foo:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMoveOutput_stateOut = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveOutput_stateOutSrc = `
<no state>
`

const testStateMoveOutput_stateOutOriginal = `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveExisting_stateSrc = `
<no state>
`

const testStateMoveExisting_stateDst = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.qux:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
`

const testStateMoveExisting_stateSrcOriginal = `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveExisting_stateDstOriginal = `
test_instance.qux:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
`

const testStateMoveOriginal_backend = `
test_instance.baz:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMoveOnlyResourceInModule_original = `
<no state>
module.foo:
  test_instance.foo.0:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMoveOnlyResourceInModule_output = `
<no state>
module.foo:
  test_instance.bar.0:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`
