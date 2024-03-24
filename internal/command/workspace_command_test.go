// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestWorkspace_createAndChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	defer testChdir(t, td)()

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

// Create some workspaces and test the list output.
// This also ensures we switch to the correct env after each call
func TestWorkspace_createAndList(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
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

	// Json without env override

	// Make sure an override does not exist
	defer os.Unsetenv("TF_WORKSPACE")
	if _, ok := os.LookupEnv("TF_WORKSPACE"); ok {
		t.Fatalf("TF_WORKSPACE must not be set.")
	}

	args := []string{
		"-json",
	}
	listCmd = &WorkspaceListCommand{}
	ui = new(cli.MockUi)
	listCmd.Meta = Meta{Ui: ui}

	if code := listCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual = strings.TrimSpace(ui.OutputWriter.String())
	expected = "{\"terraform_version\":\"0.13.0-dev\",\"workspaces\":[{\"name\":\"default\",\"selected\":false},{\"name\":\"test_a\",\"selected\":false},{\"name\":\"test_b\",\"selected\":false},{\"name\":\"test_c\",\"selected\":true}],\"is_overridden\":false,\"overridden_note\":\"\"}"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}

	// With workspace override

	// Set the workspace override
	defer os.Unsetenv("TF_WORKSPACE")
	os.Setenv("TF_WORKSPACE", "test_a")

	args = []string{
		"-json",
	}
	listCmd = &WorkspaceListCommand{}
	ui = new(cli.MockUi)
	listCmd.Meta = Meta{Ui: ui}

	if code := listCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual = strings.TrimSpace(ui.OutputWriter.String())
	expected = "{\"terraform_version\":\"0.13.0-dev\",\"workspaces\":[{\"name\":\"default\",\"selected\":false},{\"name\":\"test_a\",\"selected\":true},{\"name\":\"test_b\",\"selected\":false},{\"name\":\"test_c\",\"selected\":false}],\"is_overridden\":true,\"overridden_note\":\"\\n\\nThe active workspace is being overridden using the TF_WORKSPACE environment\\nvariable.\\n\"}"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}

}

// Create some workspaces and test the show output.
func TestWorkspace_createAndShow(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()
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
	td := t.TempDir()
	os.MkdirAll(td, 0755)
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

func TestWorkspace_deleteInvalid(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	defer testChdir(t, td)()

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

func TestWorkspace_deleteWithState(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	defer testChdir(t, td)()

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

func TestWorkspace_selectWithOrCreate(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	defer testChdir(t, td)()

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
