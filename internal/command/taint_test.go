// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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
	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
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

	output := done(t).All()
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-backup", "-",
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
	}

	if _, err := os.Stat(DefaultStateFilename + ".backup"); err == nil {
		t.Fatal("backup path should not exist")
	}

	testStateOutput(t, DefaultStateFilename, testTaintStr)
}

func TestTaint_badState(t *testing.T) {
	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-state", "i-should-not-exist-ever",
		"foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
	}

	testStateOutput(t, DefaultStateFilename, testTaintStr)
}

func TestTaint_constVariable(t *testing.T) {
	t.Run("missing value", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		view, done := testView(t)
		c := &TaintCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
				WorkingDir:       wd,
			},
		}

		args := []string{"module.child.test_instance.test"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error, got 0")
		}

		errStr := done(t).All()
		if !strings.Contains(errStr, "No value for required variable") {
			t.Fatalf("expected missing variable error, got: %s", errStr)
		}
	})

	t.Run("value via cli", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		view, done := testView(t)
		c := &TaintCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
				WorkingDir:       wd,
			},
		}

		args := []string{"-var", "module_name=child", "module.child.test_instance.test"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, done(t).All())
		}

		actual := strings.TrimSpace(testStateRead(t, "terraform.tfstate").String())
		expected := strings.TrimSpace(`<no state>
module.child:
  test_instance.test: (tainted)
    ID = 
    provider = provider["registry.terraform.io/hashicorp/test"]`)
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

		view, done := testView(t)
		c := &TaintCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
				WorkingDir:       wd,
			},
		}

		args := []string{"module.child.test_instance.test"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, done(t).All())
		}

		actual := strings.TrimSpace(testStateRead(t, "terraform.tfstate").String())
		expected := strings.TrimSpace(`<no state>
module.child:
  test_instance.test: (tainted)
    ID = 
    provider = provider["registry.terraform.io/hashicorp/test"]`)
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Fatalf("unexpected state output\n%s", diff)
		}
	})
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

	view, done := testView(t)
	meta := Meta{View: view}
	meta.SetWorkspace(testWorkspace)
	c := &TaintCommand{
		Meta: meta,
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.bar",
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-no-color",
		"-allow-missing",
		"-state", statePath,
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
	}

	// Check for the warning
	actual := strings.TrimSpace(done(t).All())
	expected := strings.TrimSpace(`
Warning: No such resource instance

Resource instance test_instance.bar was not found, but this is not an error
because -allow-missing was set.

`)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("wrong output\n%s", diff)
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-state-out", "foo",
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			View: view,
		},
	}

	args := []string{
		"-state", statePath,
		"module.child.test_instance.blah",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, done(t).All())
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

	view, done := testView(t)
	c := &TaintCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{"test_instance.foo"}
	if code := c.Run(args); code != 1 {
		output := done(t)
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
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
