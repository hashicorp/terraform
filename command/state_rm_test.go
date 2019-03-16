package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/states"
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
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
    provider = provider.test
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
  provider = provider.test
  bar = value
  foo = value

module.child:
  test_instance.foo:
    ID = foo
    provider = provider.test
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.baz", // doesn't exist in the state constructed above
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("expected exit status %d, got: %d", 0, code)
	}

	if msg := ui.OutputWriter.String(); !strings.Contains(msg, "No matching resource instances found") {
		t.Fatalf("unexpected output:\n%s", msg)
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, state)
	backupPath := statePath + ".backup.test"

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
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
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{"foo"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestStateRm_needsInit(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{"foo"}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expected error output, got:\n%s", ui.OutputWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "Initialization") {
		t.Fatalf("expected initialization error, got:\n%s", ui.ErrorWriter.String())
	}
}

func TestStateRm_backendState(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unchanged"), td)
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
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
	c := &StateRmCommand{
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
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateRmOutputOriginal)
}

const testStateRmOutputOriginal = `
test_instance.bar:
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

const testStateRmOutput = `
test_instance.bar:
  ID = foo
  provider = provider.test
  bar = value
  foo = value
`
