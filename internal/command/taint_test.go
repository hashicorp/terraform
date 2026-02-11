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

func TestTaint(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
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
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &TaintCommand{
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

	testStateOutput(t, statePath, testTaintStr)
}

func TestTaint_lockedState(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
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
	statePath := testStateFile(t, state)

	unlock, err := testLockState(t, testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &TaintCommand{
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

func TestTaint_backup(t *testing.T) {
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
				Status:    states.ObjectReady,
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
	c := &TaintCommand{
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

	testStateOutput(t, DefaultStateFilename+".backup", testTaintDefaultStr)
	testStateOutput(t, DefaultStateFilename, testTaintStr)
}

func TestTaint_backupDisable(t *testing.T) {
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
				Status:    states.ObjectReady,
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
	c := &TaintCommand{
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

	testStateOutput(t, DefaultStateFilename, testTaintStr)
}

func TestTaint_badState(t *testing.T) {
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &TaintCommand{
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

func TestTaint_defaultState(t *testing.T) {
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
				Status:    states.ObjectReady,
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
	c := &TaintCommand{
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

	testStateOutput(t, DefaultStateFilename, testTaintStr)
}

func TestTaint_defaultWorkspaceState(t *testing.T) {
	// Get a temp cwd
	tmp := t.TempDir()
	t.Chdir(tmp)

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
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
	testWorkspace := "development"
	path := testStateFileWorkspaceDefault(t, testWorkspace, state)

	ui := new(cli.MockUi)
	view, done := testView(t)
	meta := Meta{Ui: ui, View: view}
	meta.SetWorkspace(testWorkspace)
	c := &TaintCommand{
		Meta: meta,
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
	}

	testStateOutput(t, path, testTaintStr)
}

func TestTaint_missing(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
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
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &TaintCommand{
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

func TestTaint_missingAllow(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
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
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &TaintCommand{
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

func TestTaint_stateOut(t *testing.T) {
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
				Status:    states.ObjectReady,
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
	c := &TaintCommand{
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

	testStateOutput(t, DefaultStateFilename, testTaintDefaultStr)
	testStateOutput(t, "foo", testTaintStr)
}

func TestTaint_module(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
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
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "blah",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey)),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"blah"}`),
				Status:    states.ObjectReady,
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
	c := &TaintCommand{
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
		t.Fatalf("bad: %d\n\n%s", code, done(t).Stderr())
	}

	testStateOutput(t, statePath, testTaintModuleStr)
}

func TestTaint_checkRequiredVersion(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("command-check-required-version"), td)
	t.Chdir(td)

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
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	path := testStateFile(t, state)

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"test_instance.foo"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, done(t).Stderr(), done(t).Stdout())
	}

	// State is unchanged
	testStateOutput(t, path, testTaintDefaultStr)

	// Required version diags are correct
	errStr := done(t).Stderr()
	if !strings.Contains(errStr, `required_version = "~> 0.9.0"`) {
		t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
	}
	if strings.Contains(errStr, `required_version = ">= 0.13.0"`) {
		t.Fatalf("output should not point to met version constraint, but is:\n\n%s", errStr)
	}
}

const testTaintStr = `
test_instance.foo: (tainted)
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
`

const testTaintDefaultStr = `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]
`

const testTaintModuleStr = `
test_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/test"]

module.child:
  test_instance.blah: (tainted)
    ID = blah
    provider = provider["registry.terraform.io/hashicorp/test"]
`
