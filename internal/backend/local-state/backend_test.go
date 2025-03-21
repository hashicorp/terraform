// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local_state

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

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

// The `path` attribute should only affect the default workspace's state
// file location and name.
//
// Non-default workspaces' states names and locations are unaffected.
func TestLocal_useOfPathAttribute(t *testing.T) {
	// Setup
	td := testTmpDir(t)

	b := New()

	// Configure local state-storage backend (skip call to PrepareConfig)
	path := "path/to/foobar.tfstate"
	config := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.StringVal(path), // Set
		"workspace_dir": cty.NullVal(cty.String),
	})
	diags := b.Configure(config)
	if diags.HasErrors() {
		t.Fatalf("unexpected error returned from Configure")
	}

	// State file at the `path` location doesn't exist yet
	workspace := backend.DefaultStateName
	stmgr, err := b.StateMgr(workspace)
	if err != nil {
		t.Fatalf("unexpected error returned from StateMgr")
	}
	defaultStatePath := fmt.Sprintf("%s/%s", td, path)
	if _, err := os.Stat(defaultStatePath); !strings.Contains(err.Error(), "no such file or directory") {
		if err != nil {
			t.Fatalf("expected \"no such file or directory\" error when accessing file %q, got: %s", path, err)
		}
		t.Fatalf("expected the state file %q to not exist, but it did", path)
	}

	// Writing to the default workspace's state creates a file
	// at the `path` location.
	// Directories are created to enable the path.
	s := states.NewState()
	s.RootOutputValues = map[string]*states.OutputValue{
		"foobar": {
			Value: cty.StringVal("foobar"),
		},
	}
	err = stmgr.WriteState(s)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	_, err = os.Stat(defaultStatePath)
	if err != nil {
		// The file should exist post-WriteState
		t.Fatalf("unexpected error when getting stats on the state file %q", path)
	}

	// Writing to a non-default workspace's state creates a file
	// that's unaffected by the `path` location
	workspace = "fizzbuzz"
	stmgr, err = b.StateMgr(workspace)
	if err != nil {
		t.Fatalf("unexpected error returned from StateMgr")
	}
	fizzbuzzStatePath := fmt.Sprintf("%s/terraform.tfstate.d/%s/terraform.tfstate", td, workspace)
	err = stmgr.WriteState(s)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	_, err = os.Stat(fizzbuzzStatePath)
	if err != nil {
		t.Fatalf("unexpected error when getting stats on the state file \"terraform.tfstate.d/%s/terraform.tfstate\"", workspace)
	}
}

// Using non-tfstate file extensions in the value of the `path` attribute
// doesn't affect writing to state
func TestLocal_pathAttributeWrongExtension(t *testing.T) {
	// Setup
	td := testTmpDir(t)

	b := New()

	// The path value doesn't have the expected .tfstate file extension
	path := "foobar.docx"
	fullPath := fmt.Sprintf("%s/%s", td, path)
	config := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.StringVal(path), // Set
		"workspace_dir": cty.NullVal(cty.String),
	})
	diags := b.Configure(config)
	if diags.HasErrors() {
		t.Fatalf("unexpected error returned from Configure")
	}

	// Writing to the default workspace's state creates a file
	workspace := backend.DefaultStateName
	stmgr, err := b.StateMgr(workspace)
	if err != nil {
		t.Fatalf("unexpected error returned from StateMgr")
	}
	s := states.NewState()
	s.RootOutputValues = map[string]*states.OutputValue{
		"foobar": {
			Value: cty.StringVal("foobar"),
		},
	}
	err = stmgr.WriteState(s)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	_, err = os.Stat(fullPath)
	if err != nil {
		// The file should exist post-WriteState, despite the odd file extension
		t.Fatalf("unexpected error when getting stats on the state file %q", path)
	}
}

// The `workspace_dir` attribute should only affect where non-default workspaces'
// state files are saved.
//
// The default workspace's name and location are unaffected by this attribute.
func TestLocal_useOfWorkspaceDirAttribute(t *testing.T) {
	// Setup
	td := testTmpDir(t)

	b := New()

	// Configure local state-storage backend (skip call to PrepareConfig)
	workspaceDir := "path/to/workspaces"
	config := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.StringVal(workspaceDir), // set
	})
	diags := b.Configure(config)
	if diags.HasErrors() {
		t.Fatalf("unexpected error returned from Configure")
	}

	// Writing to the default workspace's state creates a file.
	// As path attribute was left null, the default location
	// ./terraform.tfstate is used.
	// Unaffected by the `workspace_dir` location.
	workspace := backend.DefaultStateName
	defaultStatePath := fmt.Sprintf("%s/terraform.tfstate", td)
	stmgr, err := b.StateMgr(workspace)
	if err != nil {
		t.Fatalf("unexpected error returned from StateMgr")
	}
	s := states.NewState()
	s.RootOutputValues = map[string]*states.OutputValue{
		"foobar": {
			Value: cty.StringVal("foobar"),
		},
	}
	err = stmgr.WriteState(s)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	_, err = os.Stat(defaultStatePath)
	if err != nil {
		// The file should exist post-WriteState
		t.Fatal("unexpected error when getting stats on the state file for the default state")
	}

	// Writing to a non-default workspace's state creates a file
	// that's affected by the `workspace_dir` location
	workspace = "fizzbuzz"
	fizzbuzzStatePath := fmt.Sprintf("%s/%s/%s/terraform.tfstate", td, workspaceDir, workspace)
	stmgr, err = b.StateMgr(workspace)
	if err != nil {
		t.Fatalf("unexpected error returned from StateMgr")
	}
	err = stmgr.WriteState(s)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	_, err = os.Stat(fizzbuzzStatePath)
	if err != nil {
		// The file should exist post-WriteState
		t.Fatalf("unexpected error when getting stats on the state file \"%s/%s/terraform.tfstate\"", workspaceDir, workspace)
	}
}

func TestLocal_cannotDeleteDefaultState(t *testing.T) {
	// Setup
	_ = testTmpDir(t)
	dflt := backend.DefaultStateName
	expectedStates := []string{dflt}

	b := New()

	// Only default workspace exists initially.
	states, err := b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected []string{%q}, got %q", dflt, states)
	}

	// Attempt to delete default state - force=false
	err = b.DeleteWorkspace(dflt, false)
	if err == nil {
		t.Fatal("expected error but there was none")
	}
	expectedErr := "cannot delete default state"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got: %q", expectedErr, err)
	}

	// Setting force=true doesn't change outcome
	err = b.DeleteWorkspace(dflt, true)
	if err == nil {
		t.Fatal("expected error but there was none")
	}
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got: %q", expectedErr, err)
	}
}

func TestLocal_addAndRemoveStates(t *testing.T) {
	// Setup
	_ = testTmpDir(t)
	dflt := backend.DefaultStateName
	expectedStates := []string{dflt}

	b := New()

	// Only default workspace exists initially.
	states, err := b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected []string{%q}, got %q", dflt, states)
	}

	// Calling StateMgr with a new workspace/state name creates it.
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

	// Test further by adding a third workspace/state.
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

	// Can reduce workspaces down to only the default workspace
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

	if err := b.DeleteWorkspace(dflt, true); err == nil {
		t.Fatal("expected error deleting default state")
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

// TestLocal_PathsConflictWith_defaultWorkspaceOnly only covers comparison of
// local state backends that have no non-default workspaces.
func TestLocal_PathsConflictWith_defaultWorkspaceOnly(t *testing.T) {

	comparedLocal := Local{
		StatePath:       DefaultStateFilename,
		StateOutPath:    DefaultStateFilename,
		StateBackupPath: DefaultStateFilename + DefaultBackupExtension,
	}

	conflicts := true
	doesNotConflict := false

	cases := map[string]struct {
		comparedTo *Local
		want       bool
	}{
		"a local state backend will conflict when compared to itself": {
			comparedTo: &comparedLocal,
			want:       conflicts,
		},
		"matching values for state path result in a conflict": {
			comparedTo: &Local{
				StatePath:    DefaultStateFilename,  // conflicts
				StateOutPath: "no/conflict.tfstate", // doesn't conflict
			},
			want: conflicts,
		},
		"matching values for state out path do NOT result in a conflict": {
			comparedTo: &Local{
				StatePath:    "no/conflict.tfstate",
				StateOutPath: DefaultStateFilename,
			},
			want: doesNotConflict,
		},
		"a state path override is sufficient to stop conflict": {
			// Conflicting state-out paths do not matter
			comparedTo: &Local{
				StatePath:         DefaultStateFilename,
				StateOutPath:      DefaultStateFilename,
				OverrideStatePath: "no-conflict.tfstate",
			},
			want: doesNotConflict,
		},
		"a state out path override is NOT sufficient to stop conflict": {
			// This is because of conflicting state path values
			comparedTo: &Local{
				StatePath:            DefaultStateFilename,
				StateOutPath:         DefaultStateFilename,
				OverrideStateOutPath: "no-conflict.tfstate",
			},
			want: conflicts,
		},
		"matching backup paths does not get identified as conflict": {
			comparedTo: &Local{
				StatePath:       "no/conflict.tfstate",
				StateOutPath:    "no/conflict.tfstate",
				StateBackupPath: DefaultStateFilename + DefaultBackupExtension, // conflicts
			},
			want: doesNotConflict,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			conflict := comparedLocal.PathsConflictWith(tc.comparedTo)
			if conflict != tc.want {
				t.Fatalf("expected PathsConflictWith to return %v, got: %v", tc.want, conflict)
			}
		})
	}
}
