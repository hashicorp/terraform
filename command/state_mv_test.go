package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateMvOutput)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateMvOutputOriginal)
}

// don't modify backend state is we supply a -state flag
func TestStateMv_explicitWithBackend(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend"), td)
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	// init our backend
	ui := new(cli.MockUi)
	ic := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
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
	td := tempDir(t)
	defer os.RemoveAll(td)
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	stateOutPath := testStateFile(t, stateDst)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
				addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})

	statePath := testStateFile(t, state)
	stateOutPath := statePath + ".out"

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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

func TestStateMv_withinBackend(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unchanged"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},

					"test_instance.baz": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	// the local backend state file is "foo"
	statePath := "local-state.tfstate"
	backupPath := "local-state.backup"

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := terraform.WriteState(state, f); err != nil {
		t.Fatal(err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
	copy.CopyDir(testFixturePath("backend-unchanged"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},

					"test_instance.baz": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	// the local backend state file is "foo"
	statePath := "local-state.tfstate"

	// real "local" state file
	statePathOut := "real-local.tfstate"

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := terraform.WriteState(state, f); err != nil {
		t.Fatal(err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateMvCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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

const testStateMvOutputOriginal = `
test_instance.baz:
  ID = foo
  provider = provider.test
  bar = value
  foo = value
test_instance.foo:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvOutput = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
test_instance.baz:
  ID = foo
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvCount_stateOut = `
test_instance.bar.0:
  ID = foo
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.1:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvCount_stateOutSrc = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvCount_stateOutOriginal = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.0:
  ID = foo
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.1:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvLargeCount_stateOut = `
test_instance.bar.0:
  ID = foo0
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.1:
  ID = foo1
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.2:
  ID = foo2
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.3:
  ID = foo3
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.4:
  ID = foo4
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.5:
  ID = foo5
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.6:
  ID = foo6
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.7:
  ID = foo7
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.8:
  ID = foo8
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.9:
  ID = foo9
  provider = provider.test
  bar = value
  foo = value
test_instance.bar.10:
  ID = foo10
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvLargeCount_stateOutSrc = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvLargeCount_stateOutOriginal = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.0:
  ID = foo0
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.1:
  ID = foo1
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.2:
  ID = foo2
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.3:
  ID = foo3
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.4:
  ID = foo4
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.5:
  ID = foo5
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.6:
  ID = foo6
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.7:
  ID = foo7
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.8:
  ID = foo8
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.9:
  ID = foo9
  provider = provider.test
  bar = value
  foo = value
test_instance.foo.10:
  ID = foo10
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvNestedModule_stateOut = `
<no state>
module.bar:
  <no state>
module.bar.child1:
  test_instance.foo:
    ID = bar
    provider = provider.test
    bar = value
    foo = value
module.bar.child2:
  test_instance.foo:
    ID = bar
    provider = provider.test
    bar = value
    foo = value
`

const testStateMvNestedModule_stateOutSrc = `
<no state>
`

const testStateMvNestedModule_stateOutOriginal = `
<no state>
module.foo:
  <no state>
module.foo.child1:
  test_instance.foo:
    ID = bar
    provider = provider.test
    bar = value
    foo = value
module.foo.child2:
  test_instance.foo:
    ID = bar
    provider = provider.test
    bar = value
    foo = value
`

const testStateMvOutput_stateOut = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvOutput_stateOutSrc = `
<no state>
`

const testStateMvOutput_stateOutOriginal = `
test_instance.foo:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvExisting_stateSrc = `
<no state>
`

const testStateMvExisting_stateDst = `
test_instance.bar:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
test_instance.qux:
  ID = bar
`

const testStateMvExisting_stateSrcOriginal = `
test_instance.foo:
  ID = bar
  provider = provider.test
  bar = value
  foo = value
`

const testStateMvExisting_stateDstOriginal = `
test_instance.qux:
  ID = bar
  provider = provider.test
`

const testStateMvOriginal_backend = `
test_instance.baz:
  ID = foo
  provider = provider.test
  bar = value
  foo = value
`
