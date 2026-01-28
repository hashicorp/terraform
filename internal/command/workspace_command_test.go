// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestWorkspace_allCommands_pluggableStateStore(t *testing.T) {
	// Create a temporary working directory with pluggable state storage in the config
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-new"), td)
	t.Chdir(td)

	mock := testStateStoreMockWithChunkNegotiation(t, 1000)

	// Assumes the mocked provider is hashicorp/test
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	meta := Meta{
		AllowExperimentalFeatures: true,
		Ui:                        ui,
		View:                      view,
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): providers.FactoryFixed(mock),
			},
		},
		ProviderSource: providerSource,
	}

	//// Init
	intCmd := &InitCommand{
		Meta: meta,
	}
	args := []string{"-enable-pluggable-state-storage-experiment"} // Needed to test init changes for PSS project
	code := intCmd.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s\n%s", code, ui.ErrorWriter, ui.OutputWriter)
	}
	// We expect a state to have been created for the default workspace
	if _, ok := mock.MockStates["default"]; !ok {
		t.Fatal("expected the default workspace to exist, but it didn't")
	}

	//// Create Workspace
	newWorkspace := "foobar"
	ui = new(cli.MockUi)
	meta.Ui = ui
	newCmd := &WorkspaceNewCommand{
		Meta: meta,
	}

	current, _ := newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("before creating any custom workspaces, the current workspace should be 'default'")
	}

	args = []string{newWorkspace}
	code = newCmd.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s\n%s", code, ui.ErrorWriter, ui.OutputWriter)
	}
	expectedMsg := fmt.Sprintf("Created and switched to workspace %q!", newWorkspace)
	if !strings.Contains(ui.OutputWriter.String(), expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, ui.OutputWriter)
	}
	// We expect a state to have been created for the new custom workspace
	if _, ok := mock.MockStates[newWorkspace]; !ok {
		t.Fatalf("expected the %s workspace to exist, but it didn't", newWorkspace)
	}
	current, _ = newCmd.Workspace()
	if current != newWorkspace {
		t.Fatalf("current workspace should be %q, got %q", newWorkspace, current)
	}

	//// List Workspaces
	ui = new(cli.MockUi)
	meta.Ui = ui
	listCmd := &WorkspaceListCommand{
		Meta: meta,
	}
	args = []string{}
	code = listCmd.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s\n%s", code, ui.ErrorWriter, ui.OutputWriter)
	}
	if !strings.Contains(ui.OutputWriter.String(), newWorkspace) {
		t.Errorf("unexpected output, expected the new %q workspace to be listed present, but it's missing. Got:\n%s", newWorkspace, ui.OutputWriter)
	}

	//// Select Workspace
	ui = new(cli.MockUi)
	meta.Ui = ui
	selCmd := &WorkspaceSelectCommand{
		Meta: meta,
	}
	selectedWorkspace := backend.DefaultStateName
	args = []string{selectedWorkspace}
	code = selCmd.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s\n%s", code, ui.ErrorWriter, ui.OutputWriter)
	}
	expectedMsg = fmt.Sprintf("Switched to workspace %q.", selectedWorkspace)
	if !strings.Contains(ui.OutputWriter.String(), expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, ui.OutputWriter)
	}

	//// Show Workspace
	ui = new(cli.MockUi)
	meta.Ui = ui
	showCmd := &WorkspaceShowCommand{
		Meta: meta,
	}
	args = []string{}
	code = showCmd.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s\n%s", code, ui.ErrorWriter, ui.OutputWriter)
	}
	expectedMsg = fmt.Sprintf("%s\n", selectedWorkspace)
	if !strings.Contains(ui.OutputWriter.String(), expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, ui.OutputWriter)
	}

	current, _ = newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

	//// Delete Workspace
	ui = new(cli.MockUi)
	meta.Ui = ui
	deleteCmd := &WorkspaceDeleteCommand{
		Meta: meta,
	}
	args = []string{newWorkspace}
	code = deleteCmd.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s\n%s", code, ui.ErrorWriter, ui.OutputWriter)
	}
	expectedMsg = fmt.Sprintf("Deleted workspace %q!\n", newWorkspace)
	if !strings.Contains(ui.OutputWriter.String(), expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, ui.OutputWriter)
	}
}

// Test how the workspace list command behaves when zero workspaces are present.
//
// Historically, the backends built into the Terraform binary would always report that the default workspace exists,
// even when there were no artefacts representing that workspace. All backends were implemented to do this, therefore
// it was impossible for the `workspace list` command to report that no workspaces existed.
//
// After the introduction of pluggable state storage we can't rely on all implementations to include that behaviour.
// Instead, we only report workspaces as existing based on the existence of state files/artefacts. Similarly, we've
// changed how new workspace artefacts are created. Previously the "default" workspace's state file was only created
// after the first apply, and custom workspaces' state files were created as a side-effect of obtaining a state manager
// during `workspace new`. Now the `workspace new` command explicitly writes an empty state file as part of creating a
// new workspace. The "default" workspace is a special case, and now an empty state file is created during init when
// that workspace is selected. These changes together allow Terraform to only report a workspace's existence based on
// the existence of artefacts.
//
// Users will only experience `workspace list` returning no workspaces if they either:
//  1. Have "default" selected and run `workspace list` before running `init`
//     the necessary `workspace new` command to make that workspace.
//  2. Have a custom workspace selected that isn't created yet. This could happen if a user sets `TF_WORKSPACE`
//     (or manually edits .terraform/environment) before they run `workspace new`.
func TestWorkspace_list_noReturnedWorkspaces(t *testing.T) {
	// Create a temporary working directory with pluggable state storage in the config
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged"), td)
	t.Chdir(td)

	mock := testStateStoreMockWithChunkNegotiation(t, 1000)

	// Assumes the mocked provider is hashicorp/test
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	meta := Meta{
		AllowExperimentalFeatures: true,
		Ui:                        ui,
		View:                      view,
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): providers.FactoryFixed(mock),
			},
		},
		ProviderSource: providerSource,
	}

	// What happens if no workspaces are returned from a pluggable state storage implementation?
	// (and there are no error diagnostics)
	mock.GetStatesResponse = &providers.GetStatesResponse{
		States:      []string{},
		Diagnostics: nil,
	}

	listCmd := &WorkspaceListCommand{
		Meta: meta,
	}
	args := []string{}
	if code := listCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	// Users see a warning that the selected workspace doesn't exist yet
	expectedWarningMessages := []string{
		"Warning: Terraform cannot find any existing workspaces.",
		"The \"default\" workspace is selected in your working directory.",
		"init",
	}
	for _, msg := range expectedWarningMessages {
		if !strings.Contains(ui.ErrorWriter.String(), msg) {
			t.Fatalf("expected stderr output to include: %s\ngot: %s",
				msg,
				ui.ErrorWriter,
			)
		}
	}

	// No other output is present
	if ui.OutputWriter.String() != "" {
		t.Fatalf("unexpected stdout: %s",
			ui.OutputWriter,
		)
	}
}

func TestWorkspace_createAndChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	newCmd := &WorkspaceNewCommand{}

	current, _ := newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

	args := []string{"test"}
	ui := new(cli.MockUi)
	view, _ := testView(t)
	newCmd.Meta = Meta{Ui: ui, View: view}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current, _ = newCmd.Workspace()
	if current != "test" {
		t.Fatalf("current workspace should be 'test', got %q", current)
	}

	selCmd := &WorkspaceSelectCommand{}
	args = []string{backend.DefaultStateName}
	ui = new(cli.MockUi)
	selCmd.Meta = Meta{Ui: ui, View: view}
	if code := selCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current, _ = newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}
}

func TestWorkspace_cannotCreateOrSelectEmptyStringWorkspace(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	newCmd := &WorkspaceNewCommand{}

	current, _ := newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

	args := []string{""}
	ui := cli.NewMockUi()
	view, _ := testView(t)
	newCmd.Meta = Meta{Ui: ui, View: view}
	if code := newCmd.Run(args); code != 1 {
		t.Fatalf("expected failure when trying to create the \"\" workspace.\noutput: %s", ui.OutputWriter)
	}

	gotStderr := ui.ErrorWriter.String()
	if want, got := `The workspace name "" is not allowed`, gotStderr; !strings.Contains(got, want) {
		t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, got)
	}

	ui = cli.NewMockUi()
	selectCmd := &WorkspaceSelectCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}
	if code := selectCmd.Run(args); code != 1 {
		t.Fatalf("expected failure when trying to select the the \"\" workspace.\noutput: %s", ui.OutputWriter)
	}

	gotStderr = ui.ErrorWriter.String()
	if want, got := `The workspace name "" is not allowed`, gotStderr; !strings.Contains(got, want) {
		t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, got)
	}
}

// Create some workspaces and test the list output.
// This also ensures we switch to the correct env after each call
func TestWorkspace_createAndList(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// make sure a vars file doesn't interfere
	err := os.WriteFile(
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
		view, _ := testView(t)
		newCmd := &WorkspaceNewCommand{
			Meta: Meta{Ui: ui, View: view},
		}
		if code := newCmd.Run([]string{env}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	listCmd := &WorkspaceListCommand{}
	ui := new(cli.MockUi)
	view, _ := testView(t)
	listCmd.Meta = Meta{Ui: ui, View: view}

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
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// make sure a vars file doesn't interfere
	err := os.WriteFile(
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
	view, _ := testView(t)
	showCmd.Meta = Meta{Ui: ui, View: view}

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
	newCmd.Meta = Meta{Ui: ui, View: view}
	if code := newCmd.Run(env); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	selCmd := &WorkspaceSelectCommand{}
	ui = new(cli.MockUi)
	selCmd.Meta = Meta{Ui: ui, View: view}
	if code := selCmd.Run(env); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	showCmd = &WorkspaceShowCommand{}
	ui = new(cli.MockUi)
	showCmd.Meta = Meta{Ui: ui, View: view}

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
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	envs := []string{"test_a*", "test_b/foo", "../../../test_c", "好_d"}

	// create multiple workspaces
	for _, env := range envs {
		ui := new(cli.MockUi)
		view, _ := testView(t)
		newCmd := &WorkspaceNewCommand{
			Meta: Meta{Ui: ui, View: view},
		}
		if code := newCmd.Run([]string{env}); code == 0 {
			t.Fatalf("expected failure: \n%s", ui.OutputWriter)
		}
	}

	// list workspaces to make sure none were created
	listCmd := &WorkspaceListCommand{}
	ui := new(cli.MockUi)
	view, _ := testView(t)
	listCmd.Meta = Meta{Ui: ui, View: view}

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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("inmem-backend"), td)
	t.Chdir(td)
	defer inmem.Reset()

	// init the backend
	ui := new(cli.MockUi)
	view, _ := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{Ui: ui, View: view},
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
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
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
		Meta: Meta{Ui: ui, View: view},
	}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	newPath := filepath.Join(local.DefaultWorkspaceDir, "test", DefaultStateFilename)
	envState := statemgr.NewFilesystem(newPath)
	err = envState.RefreshState()
	if err != nil {
		t.Fatal(err)
	}

	b := backend.TestBackendConfig(t, inmem.New(), nil)
	sMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		t.Fatal(sDiags)
	}

	newState := sMgr.State()

	if got, want := newState.String(), originalState.String(); got != want {
		t.Fatalf("states not equal\ngot: %s\nwant: %s", got, want)
	}
}

func TestWorkspace_delete(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// create the workspace directories
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// create the workspace file
	if err := os.MkdirAll(DefaultDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(DefaultDataDir, local.DefaultWorkspaceFile), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	view, _ := testView(t)
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui, View: view},
	}

	current, _ := delCmd.Workspace()
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

	current, _ = delCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatalf("wrong workspace: %q", current)
	}
}

// TestWorkspace_deleteInvalid shows that if a workspace with an invalid name
// has been created, Terraform allows users to delete it.
func TestWorkspace_deleteInvalid(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// choose an invalid workspace name
	workspace := "test workspace"
	path := filepath.Join(local.DefaultWorkspaceDir, workspace)

	// create the workspace directories
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	view, _ := testView(t)
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui, View: view},
	}

	// delete the workspace
	if code := delCmd.Run([]string{workspace}); code != 0 {
		t.Fatalf("error deleting workspace: %s", ui.ErrorWriter)
	}

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should have deleted workspace, but %s still exists", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error for workspace path: %s", err)
	}
}

func TestWorkspace_deleteRejectsEmptyString(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// Empty string identifier for workspace
	workspace := ""
	path := filepath.Join(local.DefaultWorkspaceDir, workspace)

	// create the workspace directories
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	view, _ := testView(t)
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui, View: view},
	}

	// delete the workspace
	if code := delCmd.Run([]string{workspace}); code != cli.RunResultHelp {
		t.Fatalf("expected code %d but got %d. Output: %s", cli.RunResultHelp, code, ui.OutputWriter)
	}
	if !strings.Contains(string(ui.ErrorWriter.Bytes()), "got an empty string") {
		t.Fatalf("expected error to include \"got an empty string\" but was missing, got: %s", ui.ErrorWriter)
	}
}

func TestWorkspace_deleteWithState(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// create the workspace directories
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// create a non-empty state
	originalState := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			addrs.AbsResourceInstance{
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_instance",
						Name: "foo",
					},
				},
			},
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte("{}"),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewBuiltInProvider("test"),
			},
		)
	})
	originalStateFile := &statefile.File{
		Serial:  1,
		Lineage: "whatever",
		State:   originalState,
	}

	f, err := os.Create(filepath.Join(local.DefaultWorkspaceDir, "test", "terraform.tfstate"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := statefile.Write(originalStateFile, f); err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	view, _ := testView(t)
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui, View: view},
	}
	args := []string{"test"}
	if code := delCmd.Run(args); code == 0 {
		t.Fatalf("expected failure without -force.\noutput: %s", ui.OutputWriter)
	}
	gotStderr := ui.ErrorWriter.String()
	if want, got := `Workspace "test" is currently tracking the following resource instances`, gotStderr; !strings.Contains(got, want) {
		t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, got)
	}
	if want, got := `- test_instance.foo`, gotStderr; !strings.Contains(got, want) {
		t.Errorf("error message doesn't mention the remaining instance\nwant substring: %s\ngot:\n%s", want, got)
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

func TestWorkspace_cannotDeleteDefaultWorkspace(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// Create an empty default state, i.e. create default workspace.
	originalStateFile := &statefile.File{
		Serial:  1,
		Lineage: "whatever",
		State:   states.NewState(),
	}

	f, err := os.Create(filepath.Join(local.DefaultStateFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := statefile.Write(originalStateFile, f); err != nil {
		t.Fatal(err)
	}

	// Create a non-default workspace
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Select the non-default "test" workspace
	selectCmd := &WorkspaceSelectCommand{}
	args := []string{"test"}
	ui := cli.NewMockUi()
	view, _ := testView(t)
	selectCmd.Meta = Meta{Ui: ui, View: view}
	if code := selectCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	// Assert there is a default and "test" workspace, and "test" is selected
	listCmd := &WorkspaceListCommand{}
	ui = cli.NewMockUi()
	listCmd.Meta = Meta{Ui: ui, View: view}

	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "default\n* test"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}

	// Attempt to delete the default workspace (not forced)
	ui = cli.NewMockUi()
	delCmd := &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui, View: view},
	}
	args = []string{"default"}
	if code := delCmd.Run(args); code != 1 {
		t.Fatalf("expected failure when trying to delete the default workspace.\noutput: %s", ui.OutputWriter)
	}

	// User should be prevented from deleting the default workspace despite:
	// * the state being empty
	// * default not being the selected workspace
	gotStderr := ui.ErrorWriter.String()
	if want, got := `Cannot delete the default workspace`, gotStderr; !strings.Contains(got, want) {
		t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, got)
	}

	// Attempt to force delete the default workspace
	ui = cli.NewMockUi()
	delCmd = &WorkspaceDeleteCommand{
		Meta: Meta{Ui: ui, View: view},
	}
	args = []string{"-force", "default"}
	if code := delCmd.Run(args); code != 1 {
		t.Fatalf("expected failure when trying to delete the default workspace.\noutput: %s", ui.OutputWriter)
	}

	// Outcome should be the same even when forcing
	gotStderr = ui.ErrorWriter.String()
	if want, got := `Cannot delete the default workspace`, gotStderr; !strings.Contains(got, want) {
		t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, got)
	}
}

func TestWorkspace_selectWithOrCreate(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	selectCmd := &WorkspaceSelectCommand{}

	current, _ := selectCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

	args := []string{"-or-create", "test"}
	ui := new(cli.MockUi)
	view, _ := testView(t)
	selectCmd.Meta = Meta{Ui: ui, View: view}
	if code := selectCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current, _ = selectCmd.Workspace()
	if current != "test" {
		t.Fatalf("current workspace should be 'test', got %q", current)
	}
}

// Test that the old `env` subcommands raise a deprecation warning
//
// Test covers:
// - `terraform env new`
// - `terraform env select`
// - `terraform env list`
// - `terraform env delete`
//
// Note: there is no `env` equivalent of `terraform workspace show`.
func TestWorkspace_envCommandDeprecationWarnings(t *testing.T) {
	// We're asserting the warning below is returned whenever a legacy `env` command
	// is executed. Commands are made to be legacy via LegacyName: true
	expectedWarning := `Warning: the "terraform env" family of commands is deprecated`

	// Create a temporary working directory to make workspaces in
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	newCmd := &WorkspaceNewCommand{}
	current, _ := newCmd.Workspace()
	if current != backend.DefaultStateName {
		t.Fatal("current workspace should be 'default'")
	}

	// Assert `terraform env new "foobar"` returns expected deprecation warning
	ui := new(cli.MockUi)
	view, _ := testView(t)
	newCmd = &WorkspaceNewCommand{
		Meta:       Meta{Ui: ui, View: view},
		LegacyName: true,
	}
	newWorkspace := "foobar"
	args := []string{newWorkspace}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	if !strings.Contains(ui.ErrorWriter.String(), expectedWarning) {
		t.Fatalf("expected the command to return a warning, but it was missing.\nwanted: %s\ngot: %s",
			expectedWarning,
			ui.ErrorWriter.String(),
		)
	}

	// Assert `terraform env select "default"` returns expected deprecation warning
	ui = new(cli.MockUi)
	view, _ = testView(t)
	selectCmd := &WorkspaceSelectCommand{
		Meta:       Meta{Ui: ui, View: view},
		LegacyName: true,
	}
	defaultWorkspace := "default"
	args = []string{defaultWorkspace}
	if code := selectCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	if !strings.Contains(ui.ErrorWriter.String(), expectedWarning) {
		t.Fatalf("expected the command to return a warning, but it was missing.\nwanted: %s\ngot: %s",
			expectedWarning,
			ui.ErrorWriter.String(),
		)
	}

	// Assert `terraform env list` returns expected deprecation warning
	ui = new(cli.MockUi)
	view, _ = testView(t)
	listCmd := &WorkspaceListCommand{
		Meta:       Meta{Ui: ui, View: view},
		LegacyName: true,
	}
	args = []string{}
	if code := listCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	if !strings.Contains(ui.ErrorWriter.String(), expectedWarning) {
		t.Fatalf("expected the command to return a warning, but it was missing.\nwanted: %s\ngot: %s",
			expectedWarning,
			ui.ErrorWriter.String(),
		)
	}

	// Assert `terraform env delete` returns expected deprecation warning
	ui = new(cli.MockUi)
	view, _ = testView(t)
	deleteCmd := &WorkspaceDeleteCommand{
		Meta:       Meta{Ui: ui, View: view},
		LegacyName: true,
	}
	args = []string{newWorkspace}
	if code := deleteCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	if !strings.Contains(ui.ErrorWriter.String(), expectedWarning) {
		t.Fatalf("expected the command to return a warning, but it was missing.\nwanted: %s\ngot: %s",
			expectedWarning,
			ui.ErrorWriter.String(),
		)
	}
}

func TestValidWorkspaceName(t *testing.T) {
	cases := map[string]struct {
		input string
		valid bool
	}{
		"foobar": {
			input: "foobar",
			valid: true,
		},
		"valid symbols": {
			input: "-._~@:",
			valid: true,
		},
		"includes space": {
			input: "two words",
			valid: false,
		},
		"empty string": {
			input: "",
			valid: false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			valid := validWorkspaceName(tc.input)
			if valid != tc.valid {
				t.Fatalf("unexpected output when processing input %q. Wanted %v got %v", tc.input, tc.valid, valid)
			}
		})
	}
}
