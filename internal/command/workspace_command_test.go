// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestWorkspace_allCommands_pluggableStateStore(t *testing.T) {
	// Create a temporary working directory with pluggable state storage in the config
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-new"), td)
	t.Chdir(td)

	mock := testStateStoreMockWithChunkNegotiation(t, 1000)

	// Mock that a custom workspace already exists.
	preExistingState := "pre-existing"
	mock.MockStates = map[string]interface{}{preExistingState: true}

	// Assumes the mocked provider is hashicorp/test
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

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
		WorkingDir:     workdir.NewDir("."),
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
	// We expect a state to have not been created for the default workspace
	if _, ok := mock.MockStates["default"]; ok {
		t.Fatal("expected the default workspace to not exist, but it did")
	}

	//// Create Workspace
	newWorkspace := "foobar"
	ui = new(cli.MockUi)
	meta.Ui = ui
	newCmd := &WorkspaceNewCommand{
		Meta: meta,
	}

	current, err := newCmd.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	if current != preExistingState {
		t.Fatalf("before creating any custom workspaces, the current workspace should be %q, got: %q", preExistingState, current)
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
	meta.WorkingDir = workdir.NewDir(".")
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
	selectedWorkspace := preExistingState
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
	if current != preExistingState {
		t.Fatalf("current workspace should be %q, got %q", preExistingState, current)
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
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

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
		WorkingDir:     workdir.NewDir("."),
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
	newCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}
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
	selCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}
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
	newCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}
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
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
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
		arguments.DefaultVarsFilename,
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
			Meta: Meta{
				Ui:         ui,
				WorkingDir: workdir.NewDir("."),
			},
		}
		if code := newCmd.Run([]string{env}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	listCmd := &WorkspaceListCommand{}
	ui := new(cli.MockUi)
	listCmd.Meta = Meta{
		Ui:         ui,
		WorkingDir: workdir.NewDir("."),
	}

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
		arguments.DefaultVarsFilename,
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
	showCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}

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
	newCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}

	if code := newCmd.Run(env); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	selCmd := &WorkspaceSelectCommand{}
	ui = new(cli.MockUi)
	selCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}
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
			Meta: Meta{
				Ui:         ui,
				View:       view,
				WorkingDir: workdir.NewDir("."),
			},
		}
		if code := newCmd.Run([]string{env}); code == 0 {
			t.Fatalf("expected failure: \n%s", ui.OutputWriter)
		}
	}

	// list workspaces to make sure none were created
	listCmd := &WorkspaceListCommand{}
	ui := new(cli.MockUi)
	listCmd.Meta = Meta{
		Ui:         ui,
		WorkingDir: workdir.NewDir("."),
	}

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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
	selectCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}
	if code := selectCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	// Assert there is a default and "test" workspace, and "test" is selected
	listCmd := &WorkspaceListCommand{}
	ui = cli.NewMockUi()
	listCmd.Meta = Meta{
		Ui:         ui,
		WorkingDir: workdir.NewDir("."),
	}

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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
	selectCmd.Meta = Meta{
		Ui:         ui,
		View:       view,
		WorkingDir: workdir.NewDir("."),
	}
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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
	listCmd := &WorkspaceListCommand{
		Meta: Meta{
			Ui:         ui,
			WorkingDir: workdir.NewDir("."),
		},
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
		Meta: Meta{
			Ui:         ui,
			View:       view,
			WorkingDir: workdir.NewDir("."),
		},
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

// Test how all workspace subcommands handle unexpected arguments.
func TestWorkspace_extraArgError(t *testing.T) {
	newMeta := func() (Meta, *cli.MockUi) {
		ui := new(cli.MockUi)
		return Meta{
			Ui:         ui,
			WorkingDir: workdir.NewDir("."),
		}, ui
	}

	// Create a temporary working directory that is empty
	td := t.TempDir()
	t.Chdir(td)

	// New
	meta, ui := newMeta()
	newCmd := &WorkspaceNewCommand{
		Meta: meta,
	}
	args := []string{"foobar", "extra-arg"} // The new subcommand only accepts a single argument, so this should error
	if code := newCmd.Run(args); code != cli.RunResultHelp {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	expectedError := "Expected a single argument: NAME.\n\n"
	if ui.ErrorWriter.String() != expectedError {
		t.Fatalf("expected error to include %s but was missing, got: %s", expectedError, ui.ErrorWriter.String())
	}

	// List
	meta, ui = newMeta()
	listCmd := &WorkspaceListCommand{
		Meta: meta,
	}
	args = []string{"extra-arg"} // The list subcommand does not accept any arguments, so this should error
	if code := listCmd.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	expectedError = "Error: Too many command line arguments. Did you mean to use -chdir?\n"
	if !strings.Contains(ui.ErrorWriter.String(), expectedError) {
		t.Fatalf("expected error to include \"%s\" but was missing, got: %s", expectedError, ui.ErrorWriter.String())
	}

	// Show
	meta, ui = newMeta()
	showCmd := &WorkspaceShowCommand{
		Meta: meta,
	}
	args = []string{"extra-arg"} // The show subcommand does not accept any arguments, and doesn't have any logic detecting unexpected args.
	if code := showCmd.Run(args); code != 0 {
		t.Fatalf("expected command to succeed, got: %d\n\n%s", code, ui.ErrorWriter)
	}

	// Select
	meta, ui = newMeta()
	selectCmd := &WorkspaceSelectCommand{
		Meta: meta,
	}
	args = []string{"default", "extra-arg"} // The select subcommand only accepts a single argument, so this should error
	if code := selectCmd.Run(args); code != cli.RunResultHelp {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	expectedError = "Expected a single argument: NAME.\n\n"
	if ui.ErrorWriter.String() != expectedError {
		t.Fatalf("expected error to include %s but was missing, got: %s", expectedError, ui.ErrorWriter.String())
	}

	// Delete
	meta, ui = newMeta()
	deleteCmd := &WorkspaceDeleteCommand{
		Meta: meta,
	}
	args = []string{"default", "extra-arg"} // The delete subcommand only accepts a single argument, so this should error
	if code := deleteCmd.Run(args); code != cli.RunResultHelp {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	expectedError = "Expected a single argument: NAME.\n\n"
	if ui.ErrorWriter.String() != expectedError {
		t.Fatalf("expected error to include %s but was missing, got: %s", expectedError, ui.ErrorWriter.String())
	}
}

// Test human output from commands, with color enabled or disabled
func TestWorkspace_humanOutput(t *testing.T) {
	newMeta := func(colourEnabled bool) (Meta, *cli.MockUi, *views.View, func(t *testing.T) *terminal.TestOutput) {
		ui := new(cli.MockUi)
		view, done := testView(t)
		return Meta{
			Ui:         ui,
			View:       view,
			Color:      colourEnabled,
			WorkingDir: workdir.NewDir("."),
		}, ui, view, done
	}

	// Create a temporary working directory that is empty
	td := t.TempDir()
	t.Chdir(td)

	envsSet1 := []string{"test_a", "test_b", "test_c"}
	envsSet2 := []string{"test_d", "test_e", "test_f"}

	// Assert output from creating a workspace with color enabled
	for _, env := range envsSet1 {
		useColor := true
		meta, ui, _, _ := newMeta(useColor)
		newCmd := &WorkspaceNewCommand{
			Meta: meta,
		}
		if code := newCmd.Run([]string{env}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}

		expectedOutput := fmt.Sprintf("\x1b[0m\x1b[32m\x1b[1mCreated and switched to workspace \"%s\"!\x1b[0m\x1b[32m\n\nYou're now on a new, empty workspace. Workspaces isolate their state,\nso if you run \"terraform plan\" Terraform will not see any existing state\nfor this configuration.\x1b[0m\n", env)
		if ui.OutputWriter.String() != expectedOutput {
			t.Fatalf("want: %s\ngot: %s", expectedOutput, ui.OutputWriter.String())
		}
	}

	// Assert output from creating a workspace with color disabled
	for _, env := range envsSet2 {
		useColor := false
		meta, ui, _, _ := newMeta(useColor)
		newCmd := &WorkspaceNewCommand{
			Meta: meta,
		}
		if code := newCmd.Run([]string{env}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}

		expectedOutput := fmt.Sprintf("Created and switched to workspace \"%s\"!\n\nYou're now on a new, empty workspace. Workspaces isolate their state,\nso if you run \"terraform plan\" Terraform will not see any existing state\nfor this configuration.\n", env)
		if ui.OutputWriter.String() != expectedOutput {
			t.Fatalf("want: %s\ngot: %s", expectedOutput, ui.OutputWriter.String())
		}
	}

	// NOTE: the last-created workspace will be selected: test_f

	// Assert output from listing workspaces with color enabled
	useColor := true
	meta, ui, _, _ := newMeta(useColor)
	listCmd := &WorkspaceListCommand{
		Meta: meta,
	}
	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual := ui.OutputWriter.String()
	expectedOutput := "  default\n  test_a\n  test_b\n  test_c\n  test_d\n  test_e\n* test_f\n\n"
	if actual != expectedOutput {
		t.Fatalf("\nexpected: %q\nactual:  %q", expectedOutput, actual)
	}

	// Assert output from listing workspaces with color disabled
	useColor = false
	meta, ui, _, _ = newMeta(useColor)
	listCmd = &WorkspaceListCommand{
		Meta: meta,
	}
	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "  default\n  test_a\n  test_b\n  test_c\n  test_d\n  test_e\n* test_f\n\n"
	if actual != expectedOutput {
		t.Fatalf("\nexpected: %q\nactual:  %q", expectedOutput, actual)
	}

	// Assert output from showing the current workspace with color enabled
	useColor = true
	meta, ui, _, _ = newMeta(useColor)
	showCmd := &WorkspaceShowCommand{
		Meta: meta,
	}
	if code := showCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "test_f\n"
	if actual != expectedOutput {
		t.Fatalf("\nexpected: %q\nactual:  %q", expectedOutput, actual)
	}

	// Assert output from showing the current workspace with color disabled
	useColor = false
	meta, ui, _, _ = newMeta(useColor)
	showCmd = &WorkspaceShowCommand{
		Meta: meta,
	}
	if code := showCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "test_f\n"
	if actual != expectedOutput {
		t.Fatalf("\nexpected: %q\nactual:  %q", expectedOutput, actual)
	}

	// Assert output from selecting a workspace with color enabled
	useColor = true
	meta, ui, _, _ = newMeta(useColor)
	selectCmd := &WorkspaceSelectCommand{
		Meta: meta,
	}
	args := []string{"test_a"}
	if code := selectCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "\x1b[0m\x1b[32mSwitched to workspace \"test_a\".\x1b[0m\n"
	if actual != expectedOutput {
		t.Fatalf("want: %s\ngot: %s", expectedOutput, actual)
	}

	// Assert output from selecting a workspace with color disabled
	useColor = false
	meta, ui, _, _ = newMeta(useColor)
	selectCmd = &WorkspaceSelectCommand{
		Meta: meta,
	}
	args = []string{"test_b"}
	if code := selectCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "Switched to workspace \"test_b\".\n"
	if actual != expectedOutput {
		t.Fatalf("want: %s\ngot: %s", expectedOutput, actual)
	}

	// Assert output from deleting a workspace with color enabled
	useColor = true
	meta, ui, _, _ = newMeta(useColor)
	deleteCmd := &WorkspaceDeleteCommand{
		Meta: meta,
	}
	args = []string{"test_c"}
	if code := deleteCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "\x1b[0m\x1b[32mDeleted workspace \"test_c\"!\x1b[0m\n"
	if actual != expectedOutput {
		t.Fatalf("want: %s\ngot: %s", expectedOutput, actual)
	}

	// Assert output from deleting a workspace with color disabled
	useColor = false
	meta, ui, _, _ = newMeta(useColor)
	deleteCmd = &WorkspaceDeleteCommand{
		Meta: meta,
	}
	args = []string{"test_d"}
	if code := deleteCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}
	actual = ui.OutputWriter.String()
	expectedOutput = "Deleted workspace \"test_d\"!\n"
	if actual != expectedOutput {
		t.Fatalf("want: %s\ngot: %s", expectedOutput, actual)
	}

	// Assert error output from deleting a non-existent workspace with color enabled
	useColor = true
	meta, ui, _, _ = newMeta(useColor)
	deleteCmd = &WorkspaceDeleteCommand{
		Meta: meta,
	}
	args = []string{"foobar"}
	if code := deleteCmd.Run(args); code != 1 {
		t.Fatalf("expected error but got code %d:\n\n%s\n\n%s", code, ui.OutputWriter, ui.ErrorWriter)
	}
	actual = ui.ErrorWriter.String()
	expectedOutput = "\x1b[31mWorkspace \"foobar\" doesn't exist.\n\nYou can create this workspace with the \"new\" subcommand \nor include the \"-or-create\" flag with the \"select\" subcommand.\x1b[0m\x1b[0m\n"
	if actual != expectedOutput {
		t.Fatalf("want: %s\ngot: %s", expectedOutput, actual)
	}

	// Assert error output from deleting a non-existent workspace with color disabled
	useColor = false
	meta, ui, _, _ = newMeta(useColor)
	deleteCmd = &WorkspaceDeleteCommand{
		Meta: meta,
	}
	args = []string{"foobar"}
	if code := deleteCmd.Run(args); code != 1 {
		t.Fatalf("expected error but got code %d:\n\n%s\n\n%s", code, ui.OutputWriter, ui.ErrorWriter)
	}
	actual = ui.ErrorWriter.String()
	expectedOutput = "Workspace \"foobar\" doesn't exist.\n\nYou can create this workspace with the \"new\" subcommand \nor include the \"-or-create\" flag with the \"select\" subcommand.\n"
	if actual != expectedOutput {
		t.Fatalf("want: %s\ngot: %s", expectedOutput, actual)
	}
}

func TestWorkspace_list_jsonOutput(t *testing.T) {
	// Create a temporary working directory with pluggable state storage in the config
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged"), td)
	t.Chdir(td)

	// Using PSS in this test allows easy mocking of pre-existing workspaces
	mock := testStateStoreMockWithChunkNegotiation(t, 1000)
	mock.GetStatesResponse = &providers.GetStatesResponse{
		States:      []string{"default", "dev", "stage", "prod"},
		Diagnostics: nil,
	}

	ui := new(cli.MockUi)
	view, done := testView(t)
	meta := Meta{
		AllowExperimentalFeatures: true,
		Ui:                        ui,
		View:                      view,
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): providers.FactoryFixed(mock),
			},
		},
		WorkingDir: workdir.NewDir("."),
	}

	// All commands run in this test should receive the -json flag
	args := []string{"-json"}

	// Step 1 - test list output with no diagnostics
	listCmd := &WorkspaceListCommand{
		Meta: meta,
	}
	if code := listCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\nstderr: %s\n\nstdout: %s", code, done(t).Stderr(), done(t).Stdout())
	}
	output := done(t)
	expectedStdOut := `{
  "format_version": "1.0",
  "workspaces": [
    {
      "name": "default",
      "is_current": true
    },
    {
      "name": "dev"
    },
    {
      "name": "stage"
    },
    {
      "name": "prod"
    }
  ],
  "diagnostics": []
}
`
	if output.Stdout() != expectedStdOut {
		diff := cmp.Diff(expectedStdOut, output.Stdout())
		t.Fatalf("want: %s\ngot: %s\n diff: %s",
			expectedStdOut,
			output.Stdout(),
			diff,
		)
	}
	if output.Stderr() != "" {
		t.Fatalf("expected stderr to be empty, but got: %s", output.Stderr())
	}

	// Step 2 - test list output with a warning diagnostics
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Warning from test",
		Detail:   "This is a warning from the mocked state store.",
	})
	mock.GetStatesResponse = &providers.GetStatesResponse{
		States:      []string{"default", "dev", "stage", "prod"},
		Diagnostics: diags,
	}

	view, done = testView(t)
	meta.View = view
	listCmd = &WorkspaceListCommand{
		Meta: meta,
	}
	if code := listCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\nstderr: %s\n\nstdout: %s", code, done(t).Stderr(), done(t).Stdout())
	}
	output = done(t)
	expectedStdOut = `{
  "format_version": "1.0",
  "workspaces": [
    {
      "name": "default",
      "is_current": true
    },
    {
      "name": "dev"
    },
    {
      "name": "stage"
    },
    {
      "name": "prod"
    }
  ],
  "diagnostics": [
    {
      "severity": "warning",
      "summary": "Warning from test",
      "detail": "This is a warning from the mocked state store."
    }
  ]
}
`
	if output.Stdout() != expectedStdOut {
		diff := cmp.Diff(expectedStdOut, output.Stdout())
		t.Fatalf("want: %s\ngot: %s\n diff: %s",
			expectedStdOut,
			output.Stdout(),
			diff,
		)
	}
	if output.Stderr() != "" {
		t.Fatalf("expected stderr to be empty, but got: %s", output.Stderr())
	}

	// Step 3 - test that error diagnostics are shown in isolation (no additional output even if present)
	diags = tfdiags.Diagnostics{} // empty
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Error from test",
		Detail:   "This is a error from the mocked state store.",
	})
	mock.GetStatesResponse = &providers.GetStatesResponse{
		States:      []string{"default", "dev", "stage", "prod"},
		Diagnostics: diags,
	}

	view, done = testView(t)
	meta.View = view
	listCmd = &WorkspaceListCommand{
		Meta: meta,
	}
	if code := listCmd.Run(args); code != 1 {
		t.Fatalf("expected a failure with code 1, but got: %d\n\n%s", code, done(t).All())
	}
	output = done(t)
	expectedStdOut = `{
  "format_version": "1.0",
  "workspaces": [],
  "diagnostics": [
    {
      "severity": "error",
      "summary": "Error from test",
      "detail": "This is a error from the mocked state store."
    }
  ]
}
`
	if output.Stdout() != expectedStdOut {
		t.Fatalf("want: %s\ngot: %s",
			expectedStdOut,
			output.Stdout(),
		)
	}
	if output.Stderr() != "" {
		t.Fatalf("expected stderr to be empty, but got: %s", output.Stderr())
	}
}
