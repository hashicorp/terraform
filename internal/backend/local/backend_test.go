// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/zclconf/go-cty/cty"
)

func TestLocal_impl(t *testing.T) {
	var _ backendrun.OperationsBackend = New()
	var _ backendrun.Local = New()
	var _ backendrun.CLI = New()
}

func TestLocal_backend(t *testing.T) {
	_ = testTmpDir(t)
	b := New()
	backend.TestBackendStates(t, b)
	backend.TestBackendStateLocks(t, b, b)
}

func TestLocal_PrepareConfig(t *testing.T) {
	// Setup
	_ = testTmpDir(t)

	b := New()

	// PATH ATTR
	// Empty string path attribute isn't valid
	config := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.StringVal(""),
		"workspace_dir": cty.NullVal(cty.String),
	})
	_, diags := b.PrepareConfig(config)
	if !diags.HasErrors() {
		t.Fatalf("expected an error from PrepareConfig but got none")
	}
	expectedErr := `The "path" attribute value must not be empty`
	if !strings.Contains(diags.Err().Error(), expectedErr) {
		t.Fatalf("expected an error containing %q, got: %q", expectedErr, diags.Err())
	}

	// PrepareConfig doesn't enforce the path value has .tfstate extension
	config = cty.ObjectVal(map[string]cty.Value{
		"path":          cty.StringVal("path/to/state/my-state.docx"),
		"workspace_dir": cty.NullVal(cty.String),
	})
	_, diags = b.PrepareConfig(config)
	if diags.HasErrors() {
		t.Fatalf("unexpected error returned from PrepareConfig")
	}

	// WORKSPACE_DIR ATTR
	// Empty string workspace_dir attribute isn't valid
	config = cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.StringVal(""),
	})
	_, diags = b.PrepareConfig(config)
	if !diags.HasErrors() {
		t.Fatalf("expected an error from PrepareConfig but got none")
	}
	expectedErr = `The "workspace_dir" attribute value must not be empty`
	if !strings.Contains(diags.Err().Error(), expectedErr) {
		t.Fatalf("expected an error containing %q, got: %q", expectedErr, diags.Err())
	}

	// Existence of directory isn't checked during PrepareConfig
	// (Non-existent directories are created as a side-effect of WriteState)
	config = cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.StringVal("this/does/not/exist"),
	})
	_, diags = b.PrepareConfig(config)
	if diags.HasErrors() {
		t.Fatalf("unexpected error returned from PrepareConfig")
	}
}

func checkState(t *testing.T, path, expected string) {
	t.Helper()
	// Read the state
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := state.State.String()
	expected = strings.TrimSpace(expected)
	if actual != expected {
		t.Fatalf("state does not match! actual:\n%s\n\nexpected:\n%s", actual, expected)
	}
}

func TestLocal_StatePaths(t *testing.T) {
	b := New()

	// Test the defaults
	path, out, back := b.StatePaths("")

	if path != DefaultStateFilename {
		t.Fatalf("expected %q, got %q", DefaultStateFilename, path)
	}

	if out != DefaultStateFilename {
		t.Fatalf("expected %q, got %q", DefaultStateFilename, out)
	}

	dfltBackup := DefaultStateFilename + DefaultBackupExtension
	if back != dfltBackup {
		t.Fatalf("expected %q, got %q", dfltBackup, back)
	}

	// check with env
	testEnv := "test_env"
	path, out, back = b.StatePaths(testEnv)

	expectedPath := filepath.Join(DefaultWorkspaceDir, testEnv, DefaultStateFilename)
	expectedOut := expectedPath
	expectedBackup := expectedPath + DefaultBackupExtension

	if path != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, path)
	}

	if out != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out)
	}

	if back != expectedBackup {
		t.Fatalf("expected %q, got %q", expectedBackup, back)
	}

}

func TestLocal_addAndRemoveStates(t *testing.T) {
	// Setup
	_ = testTmpDir(t)
	dflt := backend.DefaultStateName
	expectedStates := []string{dflt}

	b := New()

	// Only the default workspace exists initially.
	states, err := b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected []string{%q}, got %q", dflt, states)
	}

	// Calling StateMgr with a new workspace name creates that workspace's state file.
	expectedA := "test_A"
	if _, err := b.StateMgr(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = append(expectedStates, expectedA)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	// Creating another workspace appends it to the list of present workspaces.
	expectedB := "test_B"
	if _, err := b.StateMgr(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = append(expectedStates, expectedB)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	// Can delete a given workspace
	if err := b.DeleteWorkspace(expectedA, true); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = []string{dflt, expectedB}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	// Can delete another workspace
	if err := b.DeleteWorkspace(expectedB, true); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = []string{dflt}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	// You cannot delete the default workspace
	if err := b.DeleteWorkspace(dflt, true); err == nil {
		t.Fatal("expected error deleting default state")
	}
}

// a local backend which returns sentinel errors for NamedState methods to
// verify it's being called.
type testDelegateBackend struct {
	*Local

	// return a sentinel error on these calls
	stateErr  bool
	statesErr bool
	deleteErr bool
}

var errTestDelegateState = errors.New("state called")
var errTestDelegateStates = errors.New("states called")
var errTestDelegateDeleteState = errors.New("delete called")

func (b *testDelegateBackend) StateMgr(name string) (statemgr.Full, error) {
	if b.stateErr {
		return nil, errTestDelegateState
	}
	s := statemgr.NewFilesystem("terraform.tfstate")
	return s, nil
}

func (b *testDelegateBackend) Workspaces() ([]string, error) {
	if b.statesErr {
		return nil, errTestDelegateStates
	}
	return []string{"default"}, nil
}

func (b *testDelegateBackend) DeleteWorkspace(name string, force bool) error {
	if b.deleteErr {
		return errTestDelegateDeleteState
	}
	return nil
}

// verify that the MultiState methods are dispatched to the correct Backend.
func TestLocal_multiStateBackend(t *testing.T) {
	// assign a separate backend where we can read the state
	b := NewWithBackend(&testDelegateBackend{
		stateErr:  true,
		statesErr: true,
		deleteErr: true,
	})

	if _, err := b.StateMgr("test"); err != errTestDelegateState {
		t.Fatal("expected errTestDelegateState, got:", err)
	}

	if _, err := b.Workspaces(); err != errTestDelegateStates {
		t.Fatal("expected errTestDelegateStates, got:", err)
	}

	if err := b.DeleteWorkspace("test", true); err != errTestDelegateDeleteState {
		t.Fatal("expected errTestDelegateDeleteState, got:", err)
	}
}

// testTmpDir changes into a tmp dir and change back automatically when the test
// and all its subtests complete.
func testTmpDir(t *testing.T) string {
	tmp := t.TempDir()

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		// ignore errors and try to clean up
		os.Chdir(old)
	})

	return tmp
}
