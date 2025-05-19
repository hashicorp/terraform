// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// The file should exist post-WriteState
	checkState(t, fizzbuzzStatePath, s.String())
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

	// The file should exist post-WriteState, despite the odd file extension,
	// be readable, and contain the correct state
	checkState(t, fullPath, s.String())
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
	// Assert state
	checkState(t, defaultStatePath, s.String())

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
	// Assert state
	checkState(t, fizzbuzzStatePath, s.String())
}

// When using the local state storage you cannot delete the default workspace's state
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

func TestLocal_StatePaths_defaultWorkspace(t *testing.T) {

	// Default paths are returned for the default workspace
	// when nothing is set via config or overrides
	b := New()
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

	// If `path` is set in the config, this impacts returned paths for the default workspace
	b = New()
	configPath := "new-path.tfstate"
	b.StatePath = configPath    // equivalent of path = "new-path.tfstate" in config
	b.StateOutPath = configPath // equivalent of path = "new-path.tfstate" in config

	path, out, back = b.StatePaths("")

	if path != configPath {
		t.Fatalf("expected %q, got %q", configPath, path)
	}

	if out != configPath {
		t.Fatalf("expected %q, got %q", configPath, out)
	}

	altBackup := configPath + DefaultBackupExtension
	if back != altBackup {
		t.Fatalf("expected %q, got %q", altBackup, back)
	}

	// If overrides are set, they override default values or those from config
	b = New()
	b.StatePath = configPath    // equivalent of path = "new-path.tfstate" in config
	b.StateOutPath = configPath // equivalent of path = "new-path.tfstate" in config
	override := "override.tfstate"
	b.OverrideStatePath = override
	b.OverrideStateOutPath = override
	b.OverrideStateBackupPath = override

	path, out, back = b.StatePaths("")

	if path != override {
		t.Fatalf("expected %q, got %q", override, path)
	}

	if out != override {
		t.Fatalf("expected %q, got %q", override, out)
	}

	if back != override {
		t.Fatalf("expected %q, got %q", override, back)
	}
}

func TestLocal_StatePaths_nonDefaultWorkspace(t *testing.T) {

	// Default paths are returned for a custom workspace
	// when nothing is set via config or overrides
	b := New()
	workspace := "test_env"
	path, out, back := b.StatePaths(workspace)

	expectedPath := filepath.Join(DefaultWorkspaceDir, workspace, DefaultStateFilename)
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

	// This is unaffected by a user setting the path attribute
	b = New()
	b.StatePath = "path-from-config.tfstate" // equivalent of setting path = "path-from-config.tfstate" in config
	b.StateOutPath = "path-from-config.tfstate"

	path, out, back = b.StatePaths(workspace)

	if path != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, path)
	}

	if out != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out)
	}

	if back != expectedBackup {
		t.Fatalf("expected %q, got %q", expectedBackup, back)
	}

	// If a user set working_dir in config it affects returned values
	b = New()
	workingDir := "my/alternative/state/dir"
	b.StateWorkspaceDir = workingDir // equivalent of setting working_dir = "my/alternative/state/dir" in config

	path, out, back = b.StatePaths(workspace)

	expectedPath = filepath.Join(workingDir, workspace, DefaultStateFilename)
	expectedOut = filepath.Join(workingDir, workspace, DefaultStateFilename)
	expectedBackup = filepath.Join(workingDir, workspace, DefaultStateFilename) + DefaultBackupExtension

	if path != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, path)
	}

	if out != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out)
	}

	if back != expectedBackup {
		t.Fatalf("expected %q, got %q", expectedBackup, back)
	}

	// Overrides affect returned values regardless of config
	b = New()
	b.StateWorkspaceDir = workingDir // equivalent of setting working_dir = "my/alternative/state/dir" in config
	override := "override.tfstate"
	b.OverrideStatePath = override
	b.OverrideStateOutPath = override
	b.OverrideStateBackupPath = override

	path, out, back = b.StatePaths(workspace)

	if path != override {
		t.Fatalf("expected %q, got %q", override, path)
	}

	if out != override {
		t.Fatalf("expected %q, got %q", override, out)
	}

	if back != override {
		t.Fatalf("expected %q, got %q", override, back)
	}
}

// TestLocal_PathsConflictWith does not include testing the effects of CLI commands -state, -state-out, and -state-backup
// because PathsConflictWith is only used during state migrations, and the init command does not accept those flags.
// Those flags would cause the local backend struct to have override fields set.
func TestLocal_PathsConflictWith(t *testing.T) {
	// Create a working directory with default and non-default workspace states
	td := testTmpDir(t)
	exampleState := states.NewState()
	exampleState.RootOutputValues = map[string]*states.OutputValue{
		"foobar": {
			Value: cty.StringVal("foobar"),
		},
	}
	foobar := "foobar"
	originalBackend := New()

	// Create a default workspace state file in a non-root directory
	originalBackend.StatePath = "foobar/terraform.tfstate"
	defaultStatePath := filepath.Join(td, originalBackend.StatePath)
	stmgrDefault, _ := originalBackend.StateMgr("")
	err := stmgrDefault.WriteState(exampleState)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	checkState(t, defaultStatePath, exampleState.String())

	// Create a non-default workspace and state file there
	stmgrFoobar, _ := originalBackend.StateMgr(foobar)
	err = stmgrFoobar.WriteState(exampleState)
	if err != nil {
		t.Fatalf("unexpected error returned from WriteState")
	}
	foobarStatePath := filepath.Join(td, DefaultWorkspaceDir, foobar, DefaultStateFilename)
	checkState(t, foobarStatePath, exampleState.String())

	// Scenario where:
	// * original backend has state for a 'foobar' workspace at terraform.tfstate.d/foobar/terraform.tfstate
	// * new local backend is configured via `path` to store 'default' state at terraform.tfstate.d/foobar/terraform.tfstate
	scenario1 := New()
	scenario1.StatePath = foobarStatePath

	if !originalBackend.PathsConflictWith(scenario1) {
		t.Fatal("expected conflict but got none")
	}

	// Scenario where:
	// * original backend has state for the default workspace at ./foobar/terrform.tfstate
	// * local backend is configured to store non-default workspace state in the root dir
	//     this means a foobar workspace would also store state at ./foobar/terrform.tfstate
	scenario2 := New()
	scenario2.StateWorkspaceDir = "."

	if !originalBackend.PathsConflictWith(scenario2) {
		t.Fatal("expected conflict but got none")
	}
}

// a local backend which returns errors for methods to
// verify it's being called.
type testDelegateBackend struct {
	*Local
}

var errTestDelegatePrepareConfig = errors.New("prepare config called")
var errTestDelegateConfigure = errors.New("configure called")
var errTestDelegateState = errors.New("state called")
var errTestDelegateStates = errors.New("states called")
var errTestDelegateDeleteState = errors.New("delete called")

func (b *testDelegateBackend) ConfigSchema() *configschema.Block {
	return nil
}

func (b *testDelegateBackend) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	return cty.NilVal, diags.Append(errTestDelegatePrepareConfig)
}

func (b *testDelegateBackend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	return diags.Append(errTestDelegateConfigure)
}

func (b *testDelegateBackend) StateMgr(name string) (statemgr.Full, error) {
	return nil, errTestDelegateState
}

func (b *testDelegateBackend) Workspaces() ([]string, error) {
	return nil, errTestDelegateStates
}

func (b *testDelegateBackend) DeleteWorkspace(name string, force bool) error {
	return errTestDelegateDeleteState
}

// Verify that all backend.Backend methods are dispatched to the correct Backend when
// the local backend created with a separate state storage backend.
//
// The Local struct type implements both backendrun.OperationsBackend and backend.Backend interfaces.
// If the Local struct is not created with a separate state storage backend then it'll use its own
// backend.Backend method implementations. If a separate state storage backend IS supplied, then
// it should pass those method calls through to the separate backend.Backend.
func TestLocal_callsMethodsOnStateBackend(t *testing.T) {
	// assign a separate backend where we can read the state
	b := NewWithBackend(&testDelegateBackend{})

	if schema := b.ConfigSchema(); schema != nil {
		t.Fatal("expected a nil schema, got:", schema)
	}

	if _, diags := b.PrepareConfig(cty.NilVal); !diags.HasErrors() {
		t.Fatal("expected errTestDelegatePrepareConfig error, got:", diags)
	}

	if diags := b.Configure(cty.NilVal); !diags.HasErrors() {
		t.Fatal("expected errTestDelegateConfigure error, got:", diags)
	}

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
