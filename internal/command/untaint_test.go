package command

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/mitchellh/cli"
)

func TestUntaint(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expected := strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`)
	testStateOutput(t, statePath, expected)
}

func TestUntaint_lockedState(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)
	unlock, err := testLockState(testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code == 0 {
		t.Fatal("expected error")
	}

	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "lock") {
		t.Fatal("command output does not look like a lock error:", output)
	}
}

func TestUntaint_backup(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Backup is still tainted
	testStateOutput(t, DefaultStateFilename+".backup", strings.TrimSpace(`
test_instance.foo: (tainted)
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))

	// State is untainted
	testStateOutput(t, DefaultStateFilename, strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}

func TestUntaint_backupDisable(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-backup", "-",
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(DefaultStateFilename + ".backup"); err == nil {
		t.Fatal("backup path should not exist")
	}

	testStateOutput(t, DefaultStateFilename, strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}

func TestUntaint_badState(t *testing.T) {
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-state", "i-should-not-exist-ever",
		"foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestUntaint_defaultState(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, DefaultStateFilename, strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}

func TestUntaint_defaultWorkspaceState(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	testWorkspace := "development"
	path := testStateFileWorkspaceDefault(t, testWorkspace, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	meta := Meta{Ui: ui, View: view}
	meta.SetWorkspace(testWorkspace)
	c := &UntaintCommand{
		Meta: meta,
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, path, strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}

func TestUntaint_missing(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.bar",
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.OutputWriter.String())
	}
}

func TestUntaint_missingAllow(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-allow-missing",
		"-state", statePath,
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Check for the warning
	actual := strings.TrimSpace(ui.ErrorWriter.String())
	expected := strings.TrimSpace(`
Warning: No such resource instance

Resource instance test_instance.bar was not found, but this is not an error
because -allow-missing was set.

`)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("wrong output\n%s", diff)
	}
}

func TestUntaint_stateOut(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-state-out", "foo",
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, DefaultStateFilename, strings.TrimSpace(`
test_instance.foo: (tainted)
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
	testStateOutput(t, "foo", strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}

func TestUntaint_module(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
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
				Name: "blah",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey)),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectTainted,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UntaintCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"module.child.test_instance.blah",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("command exited with status code %d; want 0\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, strings.TrimSpace(`
test_instance.foo: (tainted)
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]

module.child:
  test_instance.blah:
    ID = bar
    provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}
