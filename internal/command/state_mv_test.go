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

func TestStateMv(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, statePath, testStateMvOutput)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvOutputOriginal)

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

func TestStateMv_backupAndBackupOutOptionsWithNonLocalBackend(t *testing.T) {
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
		td := tempDir(t)
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer os.RemoveAll(td)
		defer testChdir(t, td)()

		backupPath := filepath.Join(td, "backup")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMvCommand{
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
		td := tempDir(t)
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer os.RemoveAll(td)
		defer testChdir(t, td)()

		backupOutPath := filepath.Join(td, "backup-out")

		// Set up our backend state using mock state
		dataState, srv := testBackendState(t, state, 200)
		defer srv.Close()
		testStateFileRemote(t, dataState)

		p := testProvider()
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateMvCommand{
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
		td := tempDir(t)
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer os.RemoveAll(td)
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
		c := &StateMvCommand{
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
		td := tempDir(t)
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer os.RemoveAll(td)
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
		c := &StateMvCommand{
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
		testStateOutput(t, statePath, testStateMvBackupAndBackupOutOptionsWithNonLocalBackendOutput)
	})

	t.Run("backup-out option specified with state option", func(t *testing.T) {
		td := tempDir(t)
		testCopyDir(t, testFixturePath("init-backend-http"), td)
		defer os.RemoveAll(td)
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
		c := &StateMvCommand{
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
		testStateOutput(t, statePath, testStateMvBackupAndBackupOutOptionsWithNonLocalBackendOutput)
	})
}

func TestStateMv_resourceToInstance(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, backups[0], testStateMvOutputOriginal)
}

func TestStateMv_resourceToInstanceErr(t *testing.T) {
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

	c := &StateMvCommand{
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

func TestStateMv_resourceToInstanceErrInAutomation(t *testing.T) {
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
	c := &StateMvCommand{
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

func TestStateMv_instanceToResource(t *testing.T) {
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
	c := &StateMvCommand{
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

func TestStateMv_instanceToNewResource(t *testing.T) {
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
	c := &StateMvCommand{
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

func TestStateMv_differentResourceTypes(t *testing.T) {
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
	c := &StateMvCommand{
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
func TestStateMv_explicitWithBackend(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("init-backend"), td)
	defer os.RemoveAll(td)
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
	c := &StateMvCommand{
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
	testStateOutput(t, statePath, testStateMvOutput)
}

func TestStateMv_backupExplicit(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, statePath, testStateMvOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateMvOutputOriginal)
}

func TestStateMv_stateOutNew(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, stateOutPath, testStateMvOutput_stateOut)
	testStateOutput(t, statePath, testStateMvOutput_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvOutput_stateOutOriginal)
}

func TestStateMv_stateOutExisting(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, stateOutPath, testStateMvExisting_stateDst)
	testStateOutput(t, statePath, testStateMvExisting_stateSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvExisting_stateSrcOriginal)

	backups = testStateBackups(t, filepath.Dir(stateOutPath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvExisting_stateDstOriginal)
}

func TestStateMv_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMvCommand{
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

func TestStateMv_stateOutNew_count(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, stateOutPath, testStateMvCount_stateOut)
	testStateOutput(t, statePath, testStateMvCount_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvCount_stateOutOriginal)
}

// Modules with more than 10 resources were sorted lexically, causing the
// indexes in the new location to change.
func TestStateMv_stateOutNew_largeCount(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, stateOutPath, testStateMvLargeCount_stateOut)
	testStateOutput(t, statePath, testStateMvLargeCount_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvLargeCount_stateOutOriginal)
}

func TestStateMv_stateOutNew_nestedModule(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, stateOutPath, testStateMvNestedModule_stateOut)
	testStateOutput(t, statePath, testStateMvNestedModule_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvNestedModule_stateOutOriginal)
}

func TestStateMv_toNewModule(t *testing.T) {
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
	c := &StateMvCommand{
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
	testStateOutput(t, stateOutPath1, testStateMvNewModule_stateOut)
	testStateOutput(t, statePath, testStateMvNestedModule_stateOutSrc)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvNewModule_stateOutOriginal)

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
	testStateOutput(t, stateOutPath2, testStateMvModuleNewModule_stateOut)
}

func TestStateMv_withinBackend(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("backend-unchanged"), td)
	defer os.RemoveAll(td)
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
	c := &StateMvCommand{
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

	testStateOutput(t, statePath, testStateMvOutput)
	testStateOutput(t, backupPath, testStateMvOutputOriginal)
}

func TestStateMv_fromBackendToLocal(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("backend-unchanged"), td)
	defer os.RemoveAll(td)
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
	c := &StateMvCommand{
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

	testStateOutput(t, statePathOut, testStateMvCount_stateOutSrc)

	// the backend state should be left with only baz
	testStateOutput(t, statePath, testStateMvOriginal_backend)
}

// This test covers moving the only resource in a module to a new address in
// that module, which triggers the maybePruneModule functionality. This caused
// a panic report: https://github.com/hashicorp/terraform/issues/25520
func TestStateMv_onlyResourceInModule(t *testing.T) {
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
	testStateOutput(t, statePath, testStateMvOnlyResourceInModule_original)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StateMvCommand{
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
	testStateOutput(t, statePath, testStateMvOnlyResourceInModule_output)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvOnlyResourceInModule_original)
}

func TestStateMvHelp(t *testing.T) {
	c := &StateMvCommand{}
	if strings.ContainsRune(c.Help(), '\t') {
		t.Fatal("help text contains tab character, which will result in poor formatting")
	}
}

const testStateMvOutputOriginal = `
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

const testStateMvOutput = `
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

const testStateMvBackupAndBackupOutOptionsWithNonLocalBackendOutput = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvCount_stateOut = `
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

const testStateMvCount_stateOutSrc = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvCount_stateOutOriginal = `
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

const testStateMvLargeCount_stateOut = `
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

const testStateMvLargeCount_stateOutSrc = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvLargeCount_stateOutOriginal = `
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

const testStateMvNestedModule_stateOut = `
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

const testStateMvNewModule_stateOut = `
<no state>
module.bar:
  test_instance.bar:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMvModuleNewModule_stateOut = `
<no state>
module.foo:
  test_instance.bar:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMvNewModule_stateOutOriginal = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvNestedModule_stateOutSrc = `
<no state>
`

const testStateMvNestedModule_stateOutOriginal = `
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

const testStateMvOutput_stateOut = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvOutput_stateOutSrc = `
<no state>
`

const testStateMvOutput_stateOutOriginal = `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvExisting_stateSrc = `
<no state>
`

const testStateMvExisting_stateDst = `
test_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
test_instance.qux:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
`

const testStateMvExisting_stateSrcOriginal = `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvExisting_stateDstOriginal = `
test_instance.qux:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
`

const testStateMvOriginal_backend = `
test_instance.baz:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/test"]
  bar = value
  foo = value
`

const testStateMvOnlyResourceInModule_original = `
<no state>
module.foo:
  test_instance.foo.0:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`

const testStateMvOnlyResourceInModule_output = `
<no state>
module.foo:
  test_instance.bar.0:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
    bar = value
    foo = value
`
