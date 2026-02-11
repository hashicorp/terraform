// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
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
	unlock, err := testLockState(t, testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	ui := new(cli.MockUi)
	view, done := testView(t)
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

	output := done(t).Stderr()
	if !strings.Contains(output, "lock") {
		t.Fatal("command output does not look like a lock error:", output)
	}
}

func TestUntaint_backup(t *testing.T) {
	// Get a temp cwd
	tmp := t.TempDir()
	t.Chdir(tmp)

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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
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
	tmp := t.TempDir()
	t.Chdir(tmp)

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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
	}
}

func TestUntaint_defaultState(t *testing.T) {
	// Get a temp cwd
	tmp := t.TempDir()
	t.Chdir(tmp)

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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
	}

	testStateOutput(t, DefaultStateFilename, strings.TrimSpace(`
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
	`))
}

func TestUntaint_defaultWorkspaceState(t *testing.T) {
	// Get a temp cwd
	tmp := t.TempDir()
	t.Chdir(tmp)

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
	view, done := testView(t)
	meta := Meta{Ui: ui, View: view}
	meta.SetWorkspace(testWorkspace)
	c := &UntaintCommand{
		Meta: meta,
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stdout())
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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
	}

	// Check for the warning - warnings are rendered via the view to stdout
	output := done(t).Stdout()
	if !strings.Contains(output, "No such resource instance") {
		t.Fatalf("expected warning summary in output, got:\n%s", output)
	}
	if !strings.Contains(output, "test_instance.bar") {
		t.Fatalf("expected resource address in output, got:\n%s", output)
	}
	if !strings.Contains(output, "-allow-missing") {
		t.Fatalf("expected allow-missing mention in output, got:\n%s", output)
	}
}

func TestUntaint_stateOut(t *testing.T) {
	// Get a temp cwd
	tmp := t.TempDir()
	t.Chdir(tmp)

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
	view, done := testView(t)
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
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
	view, done := testView(t)
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
		t.Fatalf("command exited with status code %d; want 0\n\n%s", code, done(t).Stderr())
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
