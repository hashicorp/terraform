package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestWorkspace_createAndChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	newCmd := &WorkspaceNewCommand{}

	current := newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

	args := []string{"test"}
	ui := new(cli.MockUi)
	newCmd.Meta = Meta{Ui: ui}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current = newCmd.Workspace()
	if current != "test" {
		t.Fatalf("current workspace should be 'test', got %q", current)
	}

	selCmd := &WorkspaceSelectCommand{}
	args = []string{backend.DefaultStateName}
	ui = new(cli.MockUi)
	selCmd.Meta = Meta{Ui: ui}
	if code := selCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current = newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

}

// Create some workspaces and test the list output.
// This also ensures we switch to the correct env after each call
func TestWorkspace_createAndList(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// make sure a vars file doesn't interfere
	err := ioutil.WriteFile(
		DefaultVarsFilename,
		[]byte(`foo = "bar"`),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	envs := []string{"test_a", "test_b", "test_c"}

	// create multiple workspaces
	for _, env := range envs {
		ui := new(cli.MockUi)
		newCmd := &WorkspaceNewCommand{
			Meta: Meta{Ui: ui},
		}
		if code := newCmd.Run([]string{env}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	listCmd := &WorkspaceListCommand{}
	ui := new(cli.MockUi)
	listCmd.Meta = Meta{Ui: ui}

	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "default\n  test_a\n  test_b\n* test_c"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}

// Create some workspaces and test the show output.
func TestWorkspace_createAndShow(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// make sure a vars file doesn't interfere
	err := ioutil.WriteFile(
		DefaultVarsFilename,
		[]byte(`foo = "bar"`),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	// make sure current workspace show outputs "default"
	showCmd := &WorkspaceShowCommand{}
	ui := new(cli.MockUi)
	showCmd.Meta = Meta{Ui: ui}

	if code := showCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "default"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}

	newCmd := &WorkspaceNewCommand{}

	env := []string{"test_a"}

	// create test_a workspace
	ui = new(cli.MockUi)
	newCmd.Meta = Meta{Ui: ui}
	if code := newCmd.Run(env); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	selCmd := &WorkspaceSelectCommand{}
	ui = new(cli.MockUi)
	selCmd.Meta = Meta{Ui: ui}
	if code := selCmd.Run(env); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	showCmd = &WorkspaceShowCommand{}
	ui = new(cli.MockUi)
	showCmd.Meta = Meta{Ui: ui}

	if code := showCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual = strings.TrimSpace(ui.OutputWriter.String())
	expected = "test_a"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}

// Don't allow names that aren't URL safe
func TestWorkspace_createInvalid(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	envs := []string{"test_a*", "test_b/foo", "../../../test_c", "好_d"}

	// create multiple workspaces
	for _, env := range envs {
		ui := new(cli.MockUi)
		newCmd := &WorkspaceNewCommand{
			Meta: Meta{Ui: ui},
		}
		if code := newCmd.Run([]string{env}); code == 0 {
			t.Fatalf("expected failure: \n%s", ui.OutputWriter)
		}
	}

	// list workspaces to make sure none were created
	listCmd := &WorkspaceListCommand{}
	ui := new(cli.MockUi)
	listCmd.Meta = Meta{Ui: ui}

	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "* default"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}

func TestWorkspace_createWithState(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("inmem-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()
	defer inmem.Reset()

	// init the backend
	ui := new(cli.MockUi)
	initCmd := &InitCommand{
		Meta: Meta{Ui: ui},
	}
	if code := initCmd.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	originalState := states.BuildState(func(s *states.SyncState) {
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})

	err := statemgr.NewFilesystem("test.tfstate").WriteState(originalState)
	if err != nil {
		t.Fatal(err)
	}

	workspace := "test_workspace"

	args := []string{"-state", "test.tfstate", workspace}
	ui = new(cli.MockUi)
	newCmd := &WorkspaceNewCommand{
		Meta: Meta{Ui: ui},
	}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	newPath := filepath.Join(local.DefaultWorkspaceDir, "test", DefaultStateFilename)
	envState := state.LocalState{Path: newPath}
	err = envState.RefreshState()
	if err != nil {
		t.Fatal(err)
	}

	b := backend.TestBackendConfig(t, inmem.New(), nil)
	sMgr, err := b.StateMgr(workspace)
	if err != nil {
		t.Fatal(err)
	}

	newState := sMgr.State()

	if got, want := newState.String(), originalState.String(); got != want {
		t.Fatalf("states not equal\ngot: %s\nwant: %s", got, want)
	}
}

func TestWorkspace_delete(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// create the workspace directories
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// create the workspace file
	if err := os.MkdirAll(DefaultDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(DefaultDataDir, local.DefaultWorkspaceFile), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui},
	}

	current := delCmd.Workspace()
	if current != "test" {
		t.Fatal("wrong workspace:", current)
	}

	// we can't delete our current workspace
	args := []string{"test"}
	if code := delCmd.Run(args); code == 0 {
		t.Fatal("expected error deleting current workspace")
	}

	// change back to default
	if err := delCmd.SetWorkspace(backend.DefaultStateName); err != nil {
		t.Fatal(err)
	}

	// try the delete again
	ui = new(cli.MockUi)
	delCmd.Meta.Ui = ui
	if code := delCmd.Run(args); code != 0 {
		t.Fatalf("error deleting workspace: %s", ui.ErrorWriter)
	}

	current = delCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatalf("wrong workspace: %q", current)
	}
}
func TestWorkspace_deleteWithState(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// create the workspace directories
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// create a non-empty state
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	envStatePath := filepath.Join(local.DefaultWorkspaceDir, "test", DefaultStateFilename)
	err := (&state.LocalState{Path: envStatePath}).WriteState(originalState)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui},
	}
	args := []string{"test"}
	if code := delCmd.Run(args); code == 0 {
		t.Fatalf("expected failure without -force.\noutput: %s", ui.OutputWriter)
	}

	ui = new(cli.MockUi)
	delCmd.Meta.Ui = ui

	args = []string{"-force", "test"}
	if code := delCmd.Run(args); code != 0 {
		t.Fatalf("failure: %s", ui.ErrorWriter)
	}

	if _, err := os.Stat(filepath.Join(local.DefaultWorkspaceDir, "test")); !os.IsNotExist(err) {
		t.Fatal("env 'test' still exists!")
	}
}
