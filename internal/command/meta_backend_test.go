// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/cli"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/copy"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"

	"github.com/zclconf/go-cty/cty"

	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/backend/local"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/pluggable"
	backendInmem "github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
)

// Test empty directory with no config/state creates a local state.
func TestMetaBackend_emptyDir(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// Get the backend
	m := testMetaBackend(t, nil)
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Write some state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	s.WriteState(testState())
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify it exists where we expect it to
	if isEmptyState(DefaultStateFilename) {
		t.Fatalf("no state was written")
	}

	// Verify no backup since it was empty to start
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state should be empty")
	}

	// Verify no backend state was made
	if !isEmptyState(filepath.Join(m.DataDir(), DefaultStateFilename)) {
		t.Fatal("backend state should be empty")
	}
}

// check for no state. Either the file doesn't exist, or is empty
func isEmptyState(path string) bool {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return true
	}

	if fi.Size() == 0 {
		return true
	}

	return false
}

// Test a directory with a legacy state and no config continues to
// use the legacy state.
func TestMetaBackend_emptyWithDefaultState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// Write the legacy state
	statePath := DefaultStateFilename
	{
		f, err := os.Create(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		err = writeStateForTesting(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Get the backend
	m := testMetaBackend(t, nil)
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual := s.State().String(); actual != testState().String() {
		t.Fatalf("bad: %s", actual)
	}

	// Verify it exists where we expect it to
	if _, err := os.Stat(DefaultStateFilename); err != nil {
		t.Fatalf("err: %s", err)
	}

	stateName := filepath.Join(m.DataDir(), DefaultStateFilename)
	if !isEmptyState(stateName) {
		t.Fatal("expected no state at", stateName)
	}

	// Write some state
	next := testState()
	next.SetOutputValue(
		addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
		cty.StringVal("bar"), false,
	)
	s.WriteState(next)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify a backup was made since we're modifying a pre-existing state
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state should not be empty")
	}
}

// Test an empty directory with an explicit state path (outside the dir)
func TestMetaBackend_emptyWithExplicitState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// Create another directory to store our state
	stateDir := t.TempDir()
	os.MkdirAll(stateDir, 0755)

	// Write the legacy state
	statePath := filepath.Join(stateDir, "foo")
	{
		f, err := os.Create(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		err = writeStateForTesting(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.statePath = statePath

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual := s.State().String(); actual != testState().String() {
		t.Fatalf("bad: %s", actual)
	}

	// Verify neither defaults exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	stateName := filepath.Join(m.DataDir(), DefaultStateFilename)
	if !isEmptyState(stateName) {
		t.Fatal("expected no state at", stateName)
	}

	// Write some state
	next := testState()
	markStateForMatching(next, "bar") // just any change so it shows as different than before
	s.WriteState(next)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify a backup was made since we're modifying a pre-existing state
	if isEmptyState(statePath + DefaultBackupExtension) {
		t.Fatal("backup state should not be empty")
	}
}

// Verify that interpolations result in an error
func TestMetaBackend_configureBackendInterpolation(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new-interp"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	_, err := m.Backend(&BackendOpts{Init: true})
	if err == nil {
		t.Fatal("should error")
	}
	wantErr := "Variables not allowed"
	if !strings.Contains(err.Err().Error(), wantErr) {
		t.Fatalf("error should include %q, got: %s", wantErr, err.Err())
	}
}

// Newly configured backend
func TestMetaBackend_configureNewBackend(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify the default paths don't exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Newly configured backend with prior local state and no remote state
func TestMetaBackend_configureNewBackendWithState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new-migrate"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// This combination should not require the extra -migrate-state flag, since
	// there is no existing backend config
	m.migrateState = false

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	state, err := statemgr.RefreshAndRead(s)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if state == nil {
		t.Fatal("state is nil")
	}

	if got, want := testStateMgrCurrentLineage(s), "backend-new-migrate"; got != want {
		t.Fatalf("lineage changed during migration\nnow: %s\nwas: %s", got, want)
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	if err := statemgr.WriteAndPersist(s, state, nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := os.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with matching local and remote state doesn't prompt
// for copy.
func TestMetaBackend_configureNewBackendWithoutCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new-migrate"), td)
	t.Chdir(td)

	if err := copy.CopyFile(DefaultStateFilename, "local-state.tfstate"); err != nil {
		t.Fatal(err)
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.input = false

	// init the backend
	_, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Verify the state is where we expect
	f, err := os.Open("local-state.tfstate")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	actual, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Lineage != "backend-new-migrate" {
		t.Fatalf("incorrect state lineage: %q", actual.Lineage)
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := os.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with prior local state and no remote state,
// but opting to not migrate.
func TestMetaBackend_configureNewBackendWithStateNoMigrate(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new-migrate"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if state := s.State(); state != nil {
		t.Fatal("state is not nil")
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := os.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with prior local state and remote state
func TestMetaBackend_configureNewBackendWithStateExisting(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new-migrate-existing"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)
	// suppress input
	m.forceInitCopy = true

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if got, want := testStateMgrCurrentLineage(s), "local"; got != want {
		t.Fatalf("wrong lineage %q; want %q", got, want)
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := os.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with prior local state and remote state
func TestMetaBackend_configureNewBackendWithStateExistingNoMigrate(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-new-migrate-existing"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if testStateMgrCurrentLineage(s) != "remote" {
		t.Fatalf("bad: %#v", state)
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")
	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := os.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Saved backend state matching config
func TestMetaBackend_configuredBackendUnchanged(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unchanged"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("nil state")
	}
	if testStateMgrCurrentLineage(s) != "configuredUnchanged" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default paths don't exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Changing a configured backend
func TestMetaBackend_changeConfiguredBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Verify the default paths don't exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state-2.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify no local state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Reconfiguring with an already configured backend.
// This should ignore the existing backend config, and configure the new
// backend is if this is the first time.
func TestMetaBackend_reconfigureBackendChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-single-to-single"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-single", backendLocal.TestNewLocalSingle)
	defer backendInit.Set("local-single", nil)

	// Setup the meta
	m := testMetaBackend(t, nil)

	// this should not ask for input
	m.input = false

	// cli flag -reconfigure
	m.reconfigure = true

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	newState := s.State()
	if newState != nil || !newState.Empty() {
		t.Fatal("state should be nil/empty after forced reconfiguration")
	}

	// verify that the old state is still there
	s = statemgr.NewFilesystem("local-state.tfstate")
	if err := s.RefreshState(); err != nil {
		t.Fatal(err)
	}
	oldState := s.State()
	if oldState == nil || oldState.Empty() {
		t.Fatal("original state should be untouched")
	}
}

// Initializing a backend which supports workspaces and does *not* have
// the currently selected workspace should prompt the user with a list of
// workspaces to choose from to select a valid one, if more than one workspace
// is available.
func TestMetaBackend_initBackendSelectedWorkspaceDoesNotExist(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-selected-workspace-doesnt-exist-multi"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)

	defer testInputMap(t, map[string]string{
		"select-workspace": "2",
	})()

	// Get the backend
	_, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	expected := "foo"
	actual, err := m.Workspace()
	if err != nil {
		t.Fatal(err)
	}

	if actual != expected {
		t.Fatalf("expected selected workspace to be %q, but was %q", expected, actual)
	}
}

// Initializing a backend which supports workspaces and does *not* have the
// currently selected workspace - and which only has a single workspace - should
// automatically select that single workspace.
func TestMetaBackend_initBackendSelectedWorkspaceDoesNotExistAutoSelect(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-selected-workspace-doesnt-exist-single"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)

	// this should not ask for input
	m.input = false

	// Assert test precondition: The current selected workspace is "bar"
	previousName, err := m.Workspace()
	if err != nil {
		t.Fatal(err)
	}

	if previousName != "bar" {
		t.Fatalf("expected test fixture to start with 'bar' as the current selected workspace")
	}

	// Get the backend
	_, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	expected := "default"
	actual, err := m.Workspace()
	if err != nil {
		t.Fatal(err)
	}

	if actual != expected {
		t.Fatalf("expected selected workspace to be %q, but was %q", expected, actual)
	}
}

// Initializing a backend which supports workspaces and does *not* have
// the currently selected workspace with input=false should fail.
func TestMetaBackend_initBackendSelectedWorkspaceDoesNotExistInputFalse(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-selected-workspace-doesnt-exist-multi"), td)
	t.Chdir(td)

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.input = false

	// Get the backend
	_, diags := m.Backend(&BackendOpts{Init: true})

	// Should fail immediately
	if got, want := diags.ErrWithWarnings().Error(), `Currently selected workspace "bar" does not exist`; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
	}
}

// Changing a configured backend, copying state
func TestMetaBackend_configuredBackendChangeCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if testStateMgrCurrentLineage(s) != "backend-change" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify no local state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Changing a configured backend that supports only single states to another
// backend that only supports single states.
func TestMetaBackend_configuredBackendChangeCopy_singleState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-single-to-single"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-single", backendLocal.TestNewLocalSingle)
	defer backendInit.Set("local-single", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if testStateMgrCurrentLineage(s) != "backend-change" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify no local state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Changing a configured backend that supports multi-state to a
// backend that only supports single states. The multi-state only has
// a default state.
func TestMetaBackend_configuredBackendChangeCopy_multiToSingleDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-multi-default-to-single"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-single", backendLocal.TestNewLocalSingle)
	defer backendInit.Set("local-single", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if testStateMgrCurrentLineage(s) != "backend-change" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify no local state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Changing a configured backend that supports multi-state to a
// backend that only supports single states.
func TestMetaBackend_configuredBackendChangeCopy_multiToSingle(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-multi-to-single"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-single", backendLocal.TestNewLocalSingle)
	defer backendInit.Set("local-single", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-single": "yes",
		"backend-migrate-copy-to-empty":        "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if testStateMgrCurrentLineage(s) != "backend-change" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify no local state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify existing workspaces exist
	envPath := filepath.Join(backendLocal.DefaultWorkspaceDir, "env2", backendLocal.DefaultStateFilename)
	if _, err := os.Stat(envPath); err != nil {
		t.Fatal("env should exist")
	}

	// Verify we are now in the default env, or we may not be able to access the new backend
	env, err := m.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	if env != backend.DefaultStateName {
		t.Fatal("using non-default env with single-env backend")
	}
}

// Changing a configured backend that supports multi-state to a
// backend that only supports single states.
func TestMetaBackend_configuredBackendChangeCopy_multiToSingleCurrentEnv(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-multi-to-single"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-single", backendLocal.TestNewLocalSingle)
	defer backendInit.Set("local-single", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-single": "yes",
		"backend-migrate-copy-to-empty":        "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Change env
	if err := m.SetWorkspace("env2"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if testStateMgrCurrentLineage(s) != "backend-change-env2" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify no local state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify existing workspaces exist
	envPath := filepath.Join(backendLocal.DefaultWorkspaceDir, "env2", backendLocal.DefaultStateFilename)
	if _, err := os.Stat(envPath); err != nil {
		t.Fatal("env should exist")
	}
}

// Changing a configured backend that supports multi-state to a
// backend that also supports multi-state.
func TestMetaBackend_configuredBackendChangeCopy_multiToMulti(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-multi-to-multi"), td)
	t.Chdir(td)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-multistate": "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check resulting states
	workspaces, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", wDiags.Err())
	}
	if wDiags.HasWarnings() {
		t.Logf("warning returned : %s", wDiags.ErrWithWarnings())
	}

	sort.Strings(workspaces)
	expected := []string{"default", "env2"}
	if !reflect.DeepEqual(workspaces, expected) {
		t.Fatalf("bad: %#v", workspaces)
	}

	{
		// Check the default state
		s, sDiags := b.StateMgr(backend.DefaultStateName)
		if sDiags.HasErrors() {
			t.Fatalf("unexpected error: %s", sDiags.Err())
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if testStateMgrCurrentLineage(s) != "backend-change" {
			t.Fatalf("bad: %#v", state)
		}
	}

	{
		// Check the other state
		s, sDiags := b.StateMgr("env2")
		if sDiags.HasErrors() {
			t.Fatalf("unexpected error: %s", sDiags.Err())
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if testStateMgrCurrentLineage(s) != "backend-change-env2" {
			t.Fatalf("bad: %#v", state)
		}
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	{
		// Verify existing workspaces exist
		envPath := filepath.Join(backendLocal.DefaultWorkspaceDir, "env2", backendLocal.DefaultStateFilename)
		if _, err := os.Stat(envPath); err != nil {
			t.Fatalf("%s should exist, but does not", envPath)
		}
	}

	{
		// Verify new workspaces exist
		envPath := filepath.Join("envdir-new", "env2", backendLocal.DefaultStateFilename)
		if _, err := os.Stat(envPath); err != nil {
			t.Fatalf("%s should exist, but does not", envPath)
		}
	}
}

// Changing a configured backend that supports multi-state to a
// backend that also supports multi-state, but doesn't allow a
// default state while the default state is non-empty.
func TestMetaBackend_configuredBackendChangeCopy_multiToNoDefaultWithDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-multi-to-no-default-with-default"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-no-default", backendLocal.TestNewLocalNoDefault)
	defer backendInit.Set("local-no-default", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-multistate": "yes",
		"new-state-name": "env1",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check resulting states
	workspaces, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", wDiags.Err())
	}
	if wDiags.HasWarnings() {
		t.Logf("warning returned : %s", wDiags.ErrWithWarnings())
	}

	sort.Strings(workspaces)
	expected := []string{"env1", "env2"}
	if !reflect.DeepEqual(workspaces, expected) {
		t.Fatalf("bad: %#v", workspaces)
	}

	{
		// Check the renamed default state
		s, sDiags := b.StateMgr("env1")
		if sDiags.HasErrors() {
			t.Fatalf("unexpected error: %s", sDiags.Err())
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if testStateMgrCurrentLineage(s) != "backend-change-env1" {
			t.Fatalf("bad: %#v", state)
		}
	}

	{
		// Verify existing workspaces exist
		envPath := filepath.Join(backendLocal.DefaultWorkspaceDir, "env2", backendLocal.DefaultStateFilename)
		if _, err := os.Stat(envPath); err != nil {
			t.Fatal("env should exist")
		}
	}

	{
		// Verify new workspaces exist
		envPath := filepath.Join("envdir-new", "env2", backendLocal.DefaultStateFilename)
		if _, err := os.Stat(envPath); err != nil {
			t.Fatal("env should exist")
		}
	}
}

// Changing a configured backend that supports multi-state to a
// backend that also supports multi-state, but doesn't allow a
// default state while the default state is empty.
func TestMetaBackend_configuredBackendChangeCopy_multiToNoDefaultWithoutDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-change-multi-to-no-default-without-default"), td)
	t.Chdir(td)

	// Register the single-state backend
	backendInit.Set("local-no-default", backendLocal.TestNewLocalNoDefault)
	defer backendInit.Set("local-no-default", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-multistate": "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check resulting states
	workspaces, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", wDiags.Err())
	}
	if wDiags.HasWarnings() {
		t.Logf("warning returned : %s", wDiags.ErrWithWarnings())
	}

	sort.Strings(workspaces)
	expected := []string{"env2"} // default is skipped because it is absent in the source backend
	if !reflect.DeepEqual(workspaces, expected) {
		t.Fatalf("wrong workspaces\ngot:  %#v\nwant: %#v", workspaces, expected)
	}

	{
		// Check the named state
		s, sDiags := b.StateMgr("env2")
		if sDiags.HasErrors() {
			t.Fatalf("unexpected error: %s", sDiags.Err())
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if testStateMgrCurrentLineage(s) != "backend-change-env2" {
			t.Fatalf("bad: %#v", state)
		}
	}

	{
		// Verify existing workspaces exist
		envPath := filepath.Join(backendLocal.DefaultWorkspaceDir, "env2", backendLocal.DefaultStateFilename)
		if _, err := os.Stat(envPath); err != nil {
			t.Fatalf("%s should exist, but does not", envPath)
		}
	}

	{
		// Verify new workspaces exist
		envPath := filepath.Join("envdir-new", "env2", backendLocal.DefaultStateFilename)
		if _, err := os.Stat(envPath); err != nil {
			t.Fatalf("%s should exist, but does not", envPath)
		}
	}
}

// Unsetting a saved backend
func TestMetaBackend_configuredBackendUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unset"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := os.ReadFile(DefaultStateFilename)
		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		data, _ := os.ReadFile(DefaultStateFilename + DefaultBackupExtension)
		t.Fatal("backup should not exist, but contains:\n", string(data))
	}

	// Write some state
	s.WriteState(testState())
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify it exists where we expect it to
	if isEmptyState(DefaultStateFilename) {
		t.Fatal(DefaultStateFilename, "is empty")
	}

	// Verify no backup since it was empty to start
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		data, _ := os.ReadFile(DefaultStateFilename + DefaultBackupExtension)
		t.Fatal("backup state should be empty, but contains:\n", string(data))
	}
}

// Unsetting a saved backend and copying the remote state
func TestMetaBackend_configuredBackendUnsetCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unset"), td)
	t.Chdir(td)

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if got, want := testStateMgrCurrentLineage(s), "configuredUnset"; got != want {
		t.Fatalf("wrong state lineage %q; want %q", got, want)
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatalf("backup state should be empty")
	}

	// Write some state
	s.WriteState(testState())
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify it exists where we expect it to
	if _, err := os.Stat(DefaultStateFilename); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify a backup since it wasn't empty to start
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// A plan that has uses the local backend and local state storage
func TestMetaBackend_planLocal(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-plan-local"), td)
	t.Chdir(td)

	backendConfigBlock := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.NullVal(cty.String),
	})
	backendConfigRaw, err := plans.NewDynamicValue(backendConfigBlock, backendConfigBlock.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := &plans.Plan{
		Backend: &plans.Backend{
			Type:      "local",
			Config:    backendConfigRaw,
			Workspace: "default",
		},
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.BackendForLocalPlan(plan)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatalf("state should be nil: %#v", state)
	}

	// The default state file should not exist yet
	if !isEmptyState(DefaultStateFilename) {
		t.Fatal("expected empty state")
	}

	// A backup file shouldn't exist yet either.
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("expected empty backup")
	}

	// Verify we have no configured backend
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify no local backup
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatalf("backup state should be empty")
	}
}

// A plan that has uses the local backend and pluggable state storage
func TestMetaBackend_planLocal_stateStore(t *testing.T) {
	// Create a temporary working directory
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged"), td)
	t.Chdir(td)

	stateStoreConfigBlock := cty.ObjectVal(map[string]cty.Value{
		"value": cty.StringVal("foobar"),
	})
	stateStoreConfigRaw, err := plans.NewDynamicValue(stateStoreConfigBlock, stateStoreConfigBlock.Type())
	if err != nil {
		t.Fatal(err)
	}
	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")

	plan := &plans.Plan{
		StateStore: &plans.StateStore{
			Type:      "test_store",
			Config:    stateStoreConfigRaw,
			Workspace: backend.DefaultStateName,
			Provider: &plans.Provider{
				Version: version.Must(version.NewVersion("1.2.3")), // Matches lock file in the test fixtures
				Source:  &providerAddr,
				Config:  nil,
			},
		},
	}

	// Setup the meta, including a mock provider set up to mock PSS
	m := testMetaBackend(t, nil)
	mock := testStateStoreMockWithChunkNegotiation(t, 1000)
	m.testingOverrides = &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): providers.FactoryFixed(mock),
		},
	}

	// Get the backend
	b, diags := m.BackendForLocalPlan(plan)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatalf("state should be nil: %#v", state)
	}

	// Write some state
	state = states.NewState()
	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

// A plan with a custom state save path
func TestMetaBackend_planLocalStatePath(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-plan-local"), td)
	t.Chdir(td)

	original := testState()
	markStateForMatching(original, "hello")

	backendConfigBlock := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.NullVal(cty.String),
	})
	backendConfigRaw, err := plans.NewDynamicValue(backendConfigBlock, backendConfigBlock.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := &plans.Plan{
		Backend: &plans.Backend{
			Type:      "local",
			Config:    backendConfigRaw,
			Workspace: "default",
		},
	}

	// Create an alternate output path
	statePath := "foo.tfstate"

	// put an initial state there that needs to be backed up
	err = (statemgr.NewFilesystem(statePath)).WriteState(original)
	if err != nil {
		t.Fatal(err)
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.stateOutPath = statePath

	// Get the backend
	b, diags := m.BackendForLocalPlan(plan)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("default workspace state is not nil, but should be because we've not put anything there")
	}

	// Verify the default path doesn't exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatalf("err: %s", err)
	}

	// Verify a backup doesn't exists
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify we have a backup
	if isEmptyState(statePath + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// A plan that has no backend config, matching local state
func TestMetaBackend_planLocalMatch(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-plan-local-match"), td)
	t.Chdir(td)

	backendConfigBlock := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.NullVal(cty.String),
	})
	backendConfigRaw, err := plans.NewDynamicValue(backendConfigBlock, backendConfigBlock.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := &plans.Plan{
		Backend: &plans.Backend{
			Type:      "local",
			Config:    backendConfigRaw,
			Workspace: "default",
		},
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.BackendForLocalPlan(plan)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", sDiags.Err())
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("should is nil")
	}
	if testStateMgrCurrentLineage(s) != "hello" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default path
	if isEmptyState(DefaultStateFilename) {
		t.Fatal("state is empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(nil); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := statefile.Read(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		assertStateHasMarker(t, actual.State, mark)
	}

	// Verify local backup
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// A plan that contains a workspace that isn't the currently selected workspace
func TestMetaBackend_planLocal_mismatchedWorkspace(t *testing.T) {
	t.Run("local backend", func(t *testing.T) {
		td := t.TempDir()
		t.Chdir(td)

		backendConfigBlock := cty.ObjectVal(map[string]cty.Value{
			"path":          cty.NullVal(cty.String),
			"workspace_dir": cty.NullVal(cty.String),
		})
		backendConfigRaw, err := plans.NewDynamicValue(backendConfigBlock, backendConfigBlock.Type())
		if err != nil {
			t.Fatal(err)
		}
		planWorkspace := "default"
		plan := &plans.Plan{
			Backend: &plans.Backend{
				Type:      "local",
				Config:    backendConfigRaw,
				Workspace: planWorkspace,
			},
		}

		// Setup the meta
		m := testMetaBackend(t, nil)
		otherWorkspace := "foobar"
		err = m.SetWorkspace(otherWorkspace)
		if err != nil {
			t.Fatalf("error in test setup: %s", err)
		}

		// Get the backend
		_, diags := m.BackendForLocalPlan(plan)
		if !diags.HasErrors() {
			t.Fatalf("expected an error but got none: %s", diags.ErrWithWarnings())
		}
		expectedMsgs := []string{
			fmt.Sprintf("The plan file describes changes to the %q workspace, but the %q workspace is currently in use",
				planWorkspace,
				otherWorkspace,
			),
			fmt.Sprintf("terraform workspace select %s", planWorkspace),
		}
		for _, msg := range expectedMsgs {
			if !strings.Contains(diags.Err().Error(), msg) {
				t.Fatalf("expected error to include %q, but got:\n%s",
					msg,
					diags.Err())
			}
		}
	})

	t.Run("cloud backend", func(t *testing.T) {
		td := t.TempDir()
		t.Chdir(td)

		planWorkspace := "prod"
		cloudConfigBlock := cty.ObjectVal(map[string]cty.Value{
			"organization": cty.StringVal("hashicorp"),
			"workspaces": cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal(planWorkspace),
			}),
		})
		cloudConfigRaw, err := plans.NewDynamicValue(cloudConfigBlock, cloudConfigBlock.Type())
		if err != nil {
			t.Fatal(err)
		}
		plan := &plans.Plan{
			Backend: &plans.Backend{
				Type:      "cloud",
				Config:    cloudConfigRaw,
				Workspace: planWorkspace,
			},
		}

		// Setup the meta
		m := testMetaBackend(t, nil)
		otherWorkspace := "foobar"
		err = m.SetWorkspace(otherWorkspace)
		if err != nil {
			t.Fatalf("error in test setup: %s", err)
		}

		// Get the backend
		_, diags := m.BackendForLocalPlan(plan)
		if !diags.HasErrors() {
			t.Fatalf("expected an error but got none: %s", diags.ErrWithWarnings())
		}
		expectedMsgs := []string{
			fmt.Sprintf("The plan file describes changes to the %q workspace, but the %q workspace is currently in use",
				planWorkspace,
				otherWorkspace,
			),
			fmt.Sprintf(`If you'd like to continue to use the plan file, make sure the cloud block in your configuration contains the workspace name %q`, planWorkspace),
		}
		for _, msg := range expectedMsgs {
			if !strings.Contains(diags.Err().Error(), msg) {
				t.Fatalf("expected error to include `%s`, but got:\n%s",
					msg,
					diags.Err())
			}
		}
	})
}

// init a backend using -backend-config options multiple times
func TestMetaBackend_configureBackendWithExtra(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-empty"), td)
	t.Chdir(td)

	extras := map[string]cty.Value{"path": cty.StringVal("hello")}
	m := testMetaBackend(t, nil)
	opts := &BackendOpts{
		ConfigOverride: configs.SynthBody("synth", extras),
		Init:           true,
	}

	_, cHash, err := m.backendConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	// init the backend
	_, diags := m.Backend(&BackendOpts{
		ConfigOverride: configs.SynthBody("synth", extras),
		Init:           true,
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s := testDataStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))
	if s.Backend.Hash != uint64(cHash) {
		t.Fatal("mismatched state and config backend hashes")
	}

	// init the backend again with the same options
	m = testMetaBackend(t, nil)
	_, err = m.Backend(&BackendOpts{
		ConfigOverride: configs.SynthBody("synth", extras),
		Init:           true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Check the state
	s = testDataStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))
	if s.Backend.Hash != uint64(cHash) {
		t.Fatal("mismatched state and config backend hashes")
	}
}

// when configuring a default local state, don't delete local state
func TestMetaBackend_localDoesNotDeleteLocal(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-empty"), td)
	t.Chdir(td)

	// // create our local state
	orig := states.NewState()
	orig.SetOutputValue(
		addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
		cty.StringVal("bar"), false,
	)
	testStateFileDefault(t, orig)

	m := testMetaBackend(t, nil)
	m.forceInitCopy = true
	// init the backend
	_, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// check that we can read the state
	s := testStateRead(t, DefaultStateFilename)
	if s.Empty() {
		t.Fatal("our state was deleted")
	}
}

// move options from config to -backend-config
func TestMetaBackend_backendConfigToExtra(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	// init the backend
	m := testMetaBackend(t, nil)
	_, err := m.Backend(&BackendOpts{
		Init: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Check the state
	s := testDataStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))
	backendHash := s.Backend.Hash

	// init again but remove the path option from the config
	cfg := "terraform {\n  backend \"local\" {}\n}\n"
	if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	// init the backend again with the  options
	extras := map[string]cty.Value{"path": cty.StringVal("hello")}
	m = testMetaBackend(t, nil)
	m.forceInitCopy = true
	_, diags := m.Backend(&BackendOpts{
		ConfigOverride: configs.SynthBody("synth", extras),
		Init:           true,
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	s = testDataStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))

	if s.Backend.Hash == backendHash {
		t.Fatal("state.Backend.Hash was not updated")
	}
}

// no config; return inmem backend stored in state
func TestBackendFromState(t *testing.T) {
	wd := tempWorkingDirFixture(t, "backend-from-state")
	t.Chdir(wd.RootModuleDir())

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.WorkingDir = wd
	// terraform caches a small "state" file that stores the backend config.
	// This test must override m.dataDir so it loads the "terraform.tfstate" file in the
	// test directory as the backend config cache. This fixture is really a
	// fixture for the data dir rather than the module dir, so we'll override
	// them to match just for this test.
	wd.OverrideDataDir(".")

	stateBackend, diags := m.backendFromState(context.Background())
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if _, ok := stateBackend.(*backendInmem.Backend); !ok {
		t.Fatal("did not get expected inmem backend")
	}
}

func Test_determineInitReason(t *testing.T) {
	cases := map[string]struct {
		cloudMode     cloud.ConfigChangeMode
		backendState  workdir.BackendStateFile
		backendConfig configs.Backend

		wantErr string
	}{
		// All scenarios involving Cloud backend
		"change in cloud config": {
			cloudMode: cloud.ConfigChangeInPlace,
			backendState: workdir.BackendStateFile{
				Backend: &workdir.BackendConfigState{
					Type: "cloud",
					// Other fields unnecessary
				},
			},
			backendConfig: configs.Backend{
				Type: "cloud",
				// Other fields unnecessary
			},
			wantErr: `HCP Terraform configuration block has changed`,
		},
		"migrate backend to cloud": {
			cloudMode: cloud.ConfigMigrationIn,
			backendState: workdir.BackendStateFile{
				Backend: &workdir.BackendConfigState{
					Type: "foobar",
					// Other fields unnecessary
				},
			},
			backendConfig: configs.Backend{
				Type: "cloud",
				// Other fields unnecessary
			},
			wantErr: `Changed from backend "foobar" to HCP Terraform`,
		},
		"migrate cloud to backend": {
			cloudMode: cloud.ConfigMigrationOut,
			backendState: workdir.BackendStateFile{
				Backend: &workdir.BackendConfigState{
					Type: "cloud",
					// Other fields unnecessary
				},
			},
			backendConfig: configs.Backend{
				Type: "foobar",
				// Other fields unnecessary
			},
			wantErr: `Changed from HCP Terraform to backend "foobar"`,
		},

		// Changes within the backend config block
		"backend type changed": {
			cloudMode: cloud.ConfigChangeIrrelevant,
			backendState: workdir.BackendStateFile{
				Backend: &workdir.BackendConfigState{
					Type: "foobar1",
					// Other fields unnecessary
				},
			},
			backendConfig: configs.Backend{
				Type: "foobar2",
				// Other fields unnecessary
			},
			wantErr: `Backend type changed from "foobar1" to "foobar2`,
		},
		"backend config changed": {
			// Note that we don't need to include differing config to trigger this
			// scenario, as we're hitting the default case. If the types match, then
			// only the config is left to differ.
			// See the comment above determineInitReason for more info.
			cloudMode: cloud.ConfigChangeIrrelevant,
			backendState: workdir.BackendStateFile{
				Backend: &workdir.BackendConfigState{
					Type: "foobar",
					// Other fields unnecessary
				},
			},
			backendConfig: configs.Backend{
				Type: "foobar",
				// Other fields unnecessary
			},
			wantErr: `Backend configuration block has changed`,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			m := Meta{}
			diags := m.determineInitReason(tc.backendState.Backend.Type, tc.backendConfig.Type, tc.cloudMode)
			if !strings.Contains(diags.Err().Error(), tc.wantErr) {
				t.Fatalf("expected error diagnostic detail to include \"%s\" but it's missing: %s", tc.wantErr, diags.Err())
			}
		})
	}
}

// Verify that using variables results in an error
func TestMetaBackend_configureStateStoreVariableUse(t *testing.T) {
	wantErr := "Variables not allowed"

	locks := depsfile.NewLocks()
	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")
	constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
	if err != nil {
		t.Fatalf("test setup failed when making constraint: %s", err)
	}
	locks.SetProvider(
		providerAddr,
		versions.MustParseVersion("9.9.9"),
		constraint,
		[]providerreqs.Hash{""},
	)

	cases := map[string]struct {
		fixture string
		wantErr string
	}{
		"no variables in nested provider block": {
			fixture: "state-store-new-vars-in-provider",
			wantErr: wantErr,
		},
		"no variables in the state_store block": {
			fixture: "state-store-new-vars-in-store",
			wantErr: wantErr,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Create a temporary working directory that is empty
			td := t.TempDir()
			testCopyDir(t, testFixturePath(tc.fixture), td)
			t.Chdir(td)

			mock := testStateStoreMock(t)

			// Setup the meta
			m := testMetaBackend(t, nil)
			m.testingOverrides = metaOverridesForProvider(mock)
			m.AllowExperimentalFeatures = true

			// Get the state store's config
			mod, loadDiags := m.loadSingleModule(td)
			if loadDiags.HasErrors() {
				t.Fatalf("unexpected error when loading test config: %s", loadDiags.Err())
			}

			// Get the operations backend
			_, err := m.Backend(&BackendOpts{
				Init:                 true,
				StateStoreConfig:     mod.StateStore,
				ProviderRequirements: mod.ProviderRequirements,
				Locks:                locks,
			})
			if err == nil {
				t.Fatal("should error")
			}
			if !strings.Contains(err.Err().Error(), tc.wantErr) {
				t.Fatalf("error should include %q, got: %s", tc.wantErr, err.Err())
			}
		})
	}
}

func TestSavedBackend(t *testing.T) {
	// Create a temporary working directory
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-unset"), td) // Backend state file describes local backend, config lacks backend config
	t.Chdir(td)

	// Make a state manager for the backend state file,
	// read state from file
	m := testMetaBackend(t, nil)
	statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
	sMgr := &clistate.LocalState{Path: statePath}
	err := sMgr.RefreshState()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Code under test
	b, diags := m.savedBackend(sMgr)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// The test fixtures used in this test include a backend state file describing
	// a local backend with the non-default path value below (local-state.tfstate)
	localB, ok := b.(*local.Local)
	if !ok {
		t.Fatalf("expected the returned backend to be a local backend, matching the test fixtures.")
	}
	if localB.StatePath != "local-state.tfstate" {
		t.Fatalf("expected the local backend to be configured using the backend state file, but got unexpected configuration values.")
	}
}

func TestSavedStateStore(t *testing.T) {
	t.Run("the returned state store is configured with the backend state and not the current config", func(t *testing.T) {
		// Create a temporary working directory
		chunkSize := 42
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td) // Fixtures with config that differs from backend state file
		t.Chdir(td)

		mock := testStateStoreMock(t)
		mock.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
			// Assert that the state store is configured using backend state file values from the fixtures
			config := req.Config.AsValueMap()
			if v, ok := config["region"]; ok && (v.Equals(cty.NullVal(cty.String)) != cty.True) {
				// The backend state file has a null value for region, so if we're here we've somehow got a non-null value
				t.Fatalf("expected the provider to be configured with values from the backend state file (where region is unset/null), not the config. Got value: %#v", v)
			}
			return providers.ConfigureProviderResponse{}
		}
		mock.ConfigureStateStoreFn = func(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
			// Assert that the state store is configured using backend state file values from the fixtures
			config := req.Config.AsValueMap()
			if config["value"].AsString() != "old-value" {
				t.Fatalf("expected the state store to be configured with values from the backend state file (the string \"old-value\"), not the config. Got: %#v", config)
			}
			return providers.ConfigureStateStoreResponse{
				Capabilities: providers.StateStoreServerCapabilities{
					ChunkSize: int64(chunkSize),
				},
			}
		}
		mock.SetStateStoreChunkSizeFn = func(storeType string, size int) {
			if storeType != "test_store" || size != chunkSize {
				t.Fatalf("expected SetStateStoreChunkSize to be passed store type %q and chunk size %v, but got %q and %v",
					"test_store",
					chunkSize,
					storeType,
					size,
				)
			}
		}

		// Make a state manager for accessing the backend state file,
		// and read the backend state from file
		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)
		statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		err := sMgr.RefreshState()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// Code under test
		b, diags := m.savedStateStore(sMgr)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		if _, ok := b.(*pluggable.Pluggable); !ok {
			t.Fatalf(
				"expected savedStateStore to return a backend.Backend interface with concrete type %s, but got something else: %#v",
				"*pluggable.Pluggable",
				b,
			)
		}

		if !mock.SetStateStoreChunkSizeCalled {
			t.Fatal("expected configuring the pluggable state store to include a call to SetStateStoreChunkSize on the provider")
		}
	})

	t.Run("error - when there's no state stores in provider", func(t *testing.T) {
		// Create a temporary working directory
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td) // Fixtures with config that differs from backend state file
		t.Chdir(td)

		mock := testStateStoreMock(t)
		delete(mock.GetProviderSchemaResponse.StateStores, "test_store") // Remove the only state store impl.

		// Make a state manager for accessing the backend state file,
		// and read the backend state from file
		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)

		statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		err := sMgr.RefreshState()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		_, diags := m.savedStateStore(sMgr)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "Provider does not support pluggable state storage"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
	})

	t.Run("error - when there's no matching state store in provider Terraform suggests different identifier", func(t *testing.T) {
		// Create a temporary working directory
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td) // Fixtures with config that differs from backend state file
		t.Chdir(td)

		mock := testStateStoreMock(t)
		testStore := mock.GetProviderSchemaResponse.StateStores["test_store"]
		delete(mock.GetProviderSchemaResponse.StateStores, "test_store")
		// Make the provider contain a "test_bore" impl., while the config specifies a "test_store" impl.
		mock.GetProviderSchemaResponse.StateStores["test_bore"] = testStore

		// Make a state manager for accessing the backend state file,
		// and read the backend state from file
		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)

		statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		err := sMgr.RefreshState()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		_, diags := m.savedStateStore(sMgr)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "State store not implemented by the provider"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
		expectedMsg := `Did you mean "test_bore"?`
		if !strings.Contains(diags.Err().Error(), expectedMsg) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedMsg,
				diags.Err(),
			)
		}
	})
}

func TestMetaBackend_GetStateStoreProviderFactory(t *testing.T) {
	// See internal/command/e2etest/meta_backend_test.go for test case
	// where a provider factory is found using a local provider cache

	t.Run("returns an error if a matching factory can't be found", func(t *testing.T) {
		// Set up locks
		locks := depsfile.NewLocks()
		providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/simple")
		constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
		if err != nil {
			t.Fatalf("test setup failed when making constraint: %s", err)
		}
		locks.SetProvider(
			providerAddr,
			versions.MustParseVersion("9.9.9"),
			constraint,
			[]providerreqs.Hash{""},
		)

		config := &configs.StateStore{
			ProviderAddr: tfaddr.MustParseProviderSource("registry.terraform.io/hashicorp/simple"),
			Provider: &configs.Provider{
				Name: "foobar",
			},
			Type: "store",
		}

		// Setup the meta and test providerFactoriesDuringInit
		m := testMetaBackend(t, nil)
		_, diags := m.StateStoreProviderFactoryFromConfig(config, locks)
		if !diags.HasErrors() {
			t.Fatalf("expected error but got none")
		}
		expectedErr := "Provider unavailable"
		expectedDetail := "Terraform experienced an error when trying to use provider foobar (\"registry.terraform.io/hashicorp/simple\") to initialize the \"store\" state store"
		if diags[0].Description().Summary != expectedErr {
			t.Fatalf("expected error summary to include %q but got: %s",
				expectedErr,
				diags[0].Description().Summary,
			)
		}
		if !strings.Contains(diags[0].Description().Detail, expectedDetail) {
			t.Fatalf("expected error detail to include %q but got: %s",
				expectedErr,
				diags[0].Description().Detail,
			)
		}
	})

	t.Run("returns an error if provider addr data is missing", func(t *testing.T) {
		// Only minimal locks needed
		locks := depsfile.NewLocks()

		config := &configs.StateStore{
			ProviderAddr: tfaddr.Provider{}, // Empty
		}

		// Setup the meta and test providerFactoriesDuringInit
		m := testMetaBackend(t, nil)
		_, diags := m.StateStoreProviderFactoryFromConfig(config, locks)
		if !diags.HasErrors() {
			t.Fatal("expected and error but got none")
		}
		expectedErr := "Unknown provider used for state storage"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected error to include %q but got: %s",
				expectedErr,
				diags.Err().Error(),
			)
		}
	})
}

// Test the stateStoreInitFromConfig method, which relies on calling code to have already parsed the state_store block
// from the config and for config overrides to already be reflected in the first config argument.
func TestMetaBackend_stateStoreInitFromConfig(t *testing.T) {
	expectedRegionAttr := "foobar"
	expectedValueAttr := "foobar"
	config := &configs.StateStore{
		Type:   "test_store",
		Config: configBodyForTest(t, fmt.Sprintf(`value = "%s"`, expectedValueAttr)),
		Provider: &configs.Provider{
			Config: configBodyForTest(t, fmt.Sprintf(`region = "%s"`, expectedRegionAttr)),
		},
		ProviderAddr: addrs.NewDefaultProvider("test"),
	}

	t.Run("the returned state store is configured with the provided config and expected chunk size", func(t *testing.T) {
		// Prepare provider factories for use
		chunkSize := 42
		mock := testStateStoreMock(t)
		mock.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
			// Assert that the state store is configured using backend state file values from the fixtures
			config := req.Config.AsValueMap()
			if config["region"].AsString() != expectedRegionAttr {
				t.Fatalf("expected the provider attr to be configured with %q, got %q", expectedRegionAttr, config["region"].AsString())
			}
			return providers.ConfigureProviderResponse{}
		}
		mock.ConfigureStateStoreFn = func(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
			// Assert that the state store is configured using backend state file values from the fixtures
			config := req.Config.AsValueMap()
			if config["value"].AsString() != expectedValueAttr {
				t.Fatalf("expected the state store attr to be configured with %q, got %q", expectedValueAttr, config["value"].AsString())
			}
			return providers.ConfigureStateStoreResponse{
				Capabilities: providers.StateStoreServerCapabilities{
					ChunkSize: int64(chunkSize),
				},
			}
		}
		mock.SetStateStoreChunkSizeFn = func(storeType string, size int) {
			if storeType != "test_store" || size != chunkSize {
				t.Fatalf("expected SetStateStoreChunkSize to be passed store type %q and chunk size %v, but got %q and %v",
					"test_store",
					chunkSize,
					storeType,
					size,
				)
			}
		}

		providerAddr := tfaddr.MustParseProviderSource("hashicorp/test")
		constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
		if err != nil {
			t.Fatalf("test setup failed when making constraint: %s", err)
		}
		locks := depsfile.NewLocks()
		locks.SetProvider(
			providerAddr,
			versions.MustParseVersion("1.2.3"),
			constraint,
			[]providerreqs.Hash{""},
		)

		// Prepare the meta
		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)

		// Code under test
		b, _, _, diags := m.stateStoreInitFromConfig(config, locks)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
		if _, ok := b.(*pluggable.Pluggable); !ok {
			t.Fatalf(
				"expected stateStoreInitFromConfig to return a backend.Backend interface with concrete type %s, but got something else: %#v",
				"*pluggable.Pluggable",
				b,
			)
		}

		if !mock.SetStateStoreChunkSizeCalled {
			t.Fatal("expected configuring the pluggable state store to include a call to SetStateStoreChunkSize on the provider")
		}
	})

	t.Run("error - when there's no state stores in provider", func(t *testing.T) {
		// Prepare the meta
		m := testMetaBackend(t, nil)
		mock := testStateStoreMock(t)
		delete(mock.GetProviderSchemaResponse.StateStores, "test_store") // Remove the only state store impl.
		m.testingOverrides = metaOverridesForProvider(mock)

		locks := depsfile.NewLocks()
		providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")
		constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
		if err != nil {
			t.Fatalf("test setup failed when making constraint: %s", err)
		}
		locks.SetProvider(
			providerAddr,
			versions.MustParseVersion("9.9.9"),
			constraint,
			[]providerreqs.Hash{""},
		)

		_, _, _, diags := m.stateStoreInitFromConfig(config, locks)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "Provider does not support pluggable state storage"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
	})

	t.Run("error - when there's no matching state store in provider Terraform suggests different identifier", func(t *testing.T) {
		// Prepare the meta
		m := testMetaBackend(t, nil)
		mock := testStateStoreMock(t)
		testStore := mock.GetProviderSchemaResponse.StateStores["test_store"]
		delete(mock.GetProviderSchemaResponse.StateStores, "test_store")
		// Make the provider contain a "test_bore" impl., while the config specifies a "test_store" impl.
		mock.GetProviderSchemaResponse.StateStores["test_bore"] = testStore
		m.testingOverrides = metaOverridesForProvider(mock)

		locks := depsfile.NewLocks()
		providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")
		constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
		if err != nil {
			t.Fatalf("test setup failed when making constraint: %s", err)
		}
		locks.SetProvider(
			providerAddr,
			versions.MustParseVersion("1.2.3"),
			constraint,
			[]providerreqs.Hash{""},
		)

		_, _, _, diags := m.stateStoreInitFromConfig(config, locks)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "State store not implemented by the provider"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
		expectedMsg := `Did you mean "test_bore"?`
		if !strings.Contains(diags.Err().Error(), expectedMsg) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedMsg,
				diags.Err(),
			)
		}
	})
}

func TestMetaBackend_stateStoreConfig(t *testing.T) {
	// Reused in tests
	config := &configs.StateStore{
		Type:   "test_store",
		Config: configBodyForTest(t, fmt.Sprintf(`value = "%s"`, "foobar")),
		Provider: &configs.Provider{
			Config: configBodyForTest(t, fmt.Sprintf(`region = "%s"`, "foobar")),
		},
		ProviderAddr: addrs.NewDefaultProvider("test"),
	}

	locks := depsfile.NewLocks()
	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")
	constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
	if err != nil {
		t.Fatalf("test setup failed when making constraint: %s", err)
	}
	locks.SetProvider(
		providerAddr,
		versions.MustParseVersion("9.9.9"),
		constraint,
		[]providerreqs.Hash{""},
	)

	t.Run("override config can change values of custom attributes in the state_store block", func(t *testing.T) {
		overrideValue := "overridden"
		configOverride := configs.SynthBody("synth", map[string]cty.Value{"value": cty.StringVal(overrideValue)})
		opts := &BackendOpts{
			StateStoreConfig:     config,
			ProviderRequirements: &configs.RequiredProviders{},
			ConfigOverride:       configOverride,
			Init:                 true,
			Locks:                locks,
		}

		mock := testStateStoreMock(t)

		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)
		finalConfig, _, diags := m.stateStoreConfig(opts)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
		attrs, attrDiags := finalConfig.Config.JustAttributes()
		if attrDiags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
		gotAttr, attrDiags := attrs["value"].Expr.Value(nil)
		if attrDiags.HasErrors() {
			t.Fatalf("unexpected errors: %s", attrDiags.Error())
		}
		if gotAttr.AsString() != overrideValue {
			t.Fatalf("expected the `value` attr in the state_store block to be overridden with value %q, but got %q",
				overrideValue,
				attrs["value"],
			)
		}
	})

	t.Run("error - no config present", func(t *testing.T) {
		opts := &BackendOpts{
			StateStoreConfig: nil, // unset
			Init:             true,
			Locks:            locks,
		}

		mock := testStateStoreMock(t)

		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)
		_, _, diags := m.stateStoreConfig(opts)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "Missing state store configuration"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
	})

	t.Run("error - when there's no state stores in provider", func(t *testing.T) {
		mock := testStateStoreMock(t)
		delete(mock.GetProviderSchemaResponse.StateStores, "test_store") // Remove the only state store impl.

		opts := &BackendOpts{
			StateStoreConfig:     config,
			ProviderRequirements: &configs.RequiredProviders{},
			Init:                 true,
			Locks:                locks,
		}

		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)
		_, _, diags := m.stateStoreConfig(opts)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "Provider does not support pluggable state storage"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
	})

	t.Run("error - when there's no matching state store in provider Terraform suggests different identifier", func(t *testing.T) {
		mock := testStateStoreMock(t)
		testStore := mock.GetProviderSchemaResponse.StateStores["test_store"]
		delete(mock.GetProviderSchemaResponse.StateStores, "test_store")
		// Make the provider contain a "test_bore" impl., while the config specifies a "test_store" impl.
		mock.GetProviderSchemaResponse.StateStores["test_bore"] = testStore

		opts := &BackendOpts{
			StateStoreConfig:     config,
			ProviderRequirements: &configs.RequiredProviders{},
			Init:                 true,
			Locks:                locks,
		}

		m := testMetaBackend(t, nil)
		m.testingOverrides = metaOverridesForProvider(mock)

		_, _, diags := m.stateStoreConfig(opts)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectedErr := "State store not implemented by the provider"
		if !strings.Contains(diags.Err().Error(), expectedErr) {
			t.Fatalf("expected the returned error to include %q, got: %s",
				expectedErr,
				diags.Err(),
			)
		}
		expectedSuggestion := `Did you mean "test_bore"?`
		if !strings.Contains(diags.Err().Error(), expectedSuggestion) {
			t.Fatalf("expected the returned error to include a suggestion for fixing a typo %q, got: %s",
				expectedSuggestion,
				diags.Err(),
			)
		}
	})
}

func Test_getStateStorageProviderVersion(t *testing.T) {
	// Locks only contain hashicorp/test provider
	locks := depsfile.NewLocks()
	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")
	constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
	if err != nil {
		t.Fatalf("test setup failed when making constraint: %s", err)
	}
	setVersion := versions.MustParseVersion("9.9.9")
	locks.SetProvider(
		providerAddr,
		setVersion,
		constraint,
		[]providerreqs.Hash{""},
	)

	t.Run("returns the version of the provider represented in the locks", func(t *testing.T) {
		c := &configs.StateStore{
			Provider:     &configs.Provider{},
			ProviderAddr: tfaddr.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "test"),
		}
		v, diags := getStateStorageProviderVersion(c, locks)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		expectedVersion, err := providerreqs.GoVersionFromVersion(setVersion)
		if err != nil {
			t.Fatalf("test setup failed when making expected version: %s", err)
		}
		if !v.Equal(expectedVersion) {
			t.Fatalf("expected version to be %#v, got %#v", expectedVersion, v)
		}
	})

	t.Run("returns a nil version when using a builtin provider", func(t *testing.T) {
		c := &configs.StateStore{
			Provider:     &configs.Provider{},
			ProviderAddr: tfaddr.NewProvider(addrs.BuiltInProviderHost, addrs.BuiltInProviderNamespace, "test"),
		}
		v, diags := getStateStorageProviderVersion(c, locks)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		var expectedVersion *version.Version = nil
		if !v.Equal(expectedVersion) {
			t.Fatalf("expected version to be %#v, got %#v", expectedVersion, v)
		}
	})

	t.Run("returns a nil version when using a re-attached provider", func(t *testing.T) {
		t.Setenv("TF_REATTACH_PROVIDERS", `{
			"test": {
				"Protocol": "grpc",
				"ProtocolVersion": 6,
				"Pid": 12345,
				"Test": true,
				"Addr": {
					"Network": "unix",
					"String":"/var/folders/xx/abcde12345/T/plugin12345"
				}
			}
		}`)
		c := &configs.StateStore{
			Provider:     &configs.Provider{},
			ProviderAddr: tfaddr.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "test"),
		}
		v, diags := getStateStorageProviderVersion(c, locks)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		var expectedVersion *version.Version = nil
		if !v.Equal(expectedVersion) {
			t.Fatalf("expected version to be %#v, got %#v", expectedVersion, v)
		}
	})

	t.Run("returns an error diagnostic when version info cannot be obtained from locks", func(t *testing.T) {
		c := &configs.StateStore{
			Type: "missing-provider_foobar",
			Provider: &configs.Provider{
				Name: "missing-provider",
			},
			ProviderAddr: tfaddr.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "missing-provider"),
		}
		_, diags := getStateStorageProviderVersion(c, locks)
		if !diags.HasErrors() {
			t.Fatal("expected errors but got none")
		}
		expectMsg := "not present in the lockfile"
		if !strings.Contains(diags.Err().Error(), expectMsg) {
			t.Fatalf("expected error to include %q but got: %s",
				expectMsg,
				diags.Err(),
			)
		}
	})
}

func TestMetaBackend_prepareBackend(t *testing.T) {
	t.Run("it returns a cloud backend from cloud backend config", func(t *testing.T) {
		// Create a temporary working directory with cloud configuration in
		td := t.TempDir()
		testCopyDir(t, testFixturePath("cloud-config"), td)
		t.Chdir(td)

		m := testMetaBackend(t, nil)

		// We cannot initialize a cloud backend so we instead check
		// the init error is referencing HCP Terraform
		_, bDiags := m.backend(td, arguments.ViewHuman)
		if !bDiags.HasErrors() {
			t.Fatal("expected error but got none")
		}
		wantErr := "HCP Terraform or Terraform Enterprise initialization required: please run \"terraform init\""
		if !strings.Contains(bDiags.Err().Error(), wantErr) {
			t.Fatalf("expected error to contain %q, but got: %q",
				wantErr,
				bDiags.Err())
		}
	})

	t.Run("it returns a backend from backend config", func(t *testing.T) {
		// Create a temporary working directory with backend configuration in
		td := t.TempDir()
		testCopyDir(t, testFixturePath("backend-unchanged"), td)
		t.Chdir(td)

		m := testMetaBackend(t, nil)

		b, bDiags := m.backend(td, arguments.ViewHuman)
		if bDiags.HasErrors() {
			t.Fatal("unexpected error: ", bDiags.Err())
		}

		if _, ok := b.(*local.Local); !ok {
			t.Fatal("expected returned operations backend to be a Local backend")
		}
		// Check the type of backend inside the Local via schema
		// In this case a `local` backend should have been returned by default.
		//
		// Look for the path attribute.
		schema := b.ConfigSchema()
		if _, ok := schema.Attributes["path"]; !ok {
			t.Fatalf("expected the operations backend to report the schema of a local backend, but got something unexpected: %#v", schema)
		}
	})

	t.Run("it returns a local backend when there is empty configuration", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("empty"), td)
		t.Chdir(td)

		m := testMetaBackend(t, nil)
		b, bDiags := m.backend(td, arguments.ViewHuman)
		if bDiags.HasErrors() {
			t.Fatal("unexpected error: ", bDiags.Err())
		}

		if _, ok := b.(*local.Local); !ok {
			t.Fatal("expected returned operations backend to be a Local backend")
		}
		// Check the type of backend inside the Local via schema
		// In this case a `local` backend should have been returned by default.
		//
		// Look for the path attribute.
		schema := b.ConfigSchema()
		if _, ok := schema.Attributes["path"]; !ok {
			t.Fatalf("expected the operations backend to report the schema of a local backend, but got something unexpected: %#v", schema)
		}
	})

	t.Run("it returns a state_store from state_store config", func(t *testing.T) {
		// Create a temporary working directory with backend configuration in
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged"), td)
		t.Chdir(td)

		m := testMetaBackend(t, nil)
		m.AllowExperimentalFeatures = true
		mock := testStateStoreMockWithChunkNegotiation(t, 12345) // chunk size needs to be set, value is arbitrary
		m.testingOverrides = &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): providers.FactoryFixed(mock),
			},
		}

		// Prepare appropriate locks; config uses a hashicorp/test provider @ v1.2.3
		locks := depsfile.NewLocks()
		providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test")
		constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
		if err != nil {
			t.Fatalf("test setup failed when making constraint: %s", err)
		}
		locks.SetProvider(
			providerAddr,
			versions.MustParseVersion("1.2.3"),
			constraint,
			[]providerreqs.Hash{""},
		)

		b, bDiags := m.backend(td, arguments.ViewHuman)
		if bDiags.HasErrors() {
			t.Fatalf("unexpected error: %s", bDiags.Err())
		}

		if _, ok := b.(*local.Local); !ok {
			t.Fatal("expected returned operations backend to be a Local backend")
		}
		// Check the state_store inside the Local via schema
		// Look for the mock state_store's attribute called `value`.
		schema := b.ConfigSchema()
		if _, ok := schema.Attributes["value"]; !ok {
			t.Fatalf("expected the operations backend to report the schema of the state_store, but got something unexpected: %#v", schema)
		}
	})
}

func testMetaBackend(t *testing.T, args []string) *Meta {
	var m Meta
	m.Ui = new(cli.MockUi)
	view, _ := testView(t)
	m.View = view
	m.process(args)
	f := m.extendedFlagSet("test")
	if err := f.Parse(args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// metaBackend tests are verifying migrate actions
	m.migrateState = true

	return &m
}

// testStateStoreMock returns a mock provider that has a state store implementation
// The provider uses the name "test" and the store inside is "test_store".
func testStateStoreMock(t *testing.T) *testing_provider.MockProvider {
	t.Helper()
	return &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"region": {Type: cty.String, Optional: true},
					},
				},
			},
			DataSources:       map[string]providers.Schema{},
			ResourceTypes:     map[string]providers.Schema{},
			ListResourceTypes: map[string]providers.Schema{},
			StateStores: map[string]providers.Schema{
				"test_store": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
		ConfigureStateStoreFn: func(cssr providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
			return providers.ConfigureStateStoreResponse{
				Capabilities: providers.StateStoreServerCapabilities{
					ChunkSize: cssr.Capabilities.ChunkSize,
				},
			}
		},
	}
}

// testStateStoreMockWithChunkNegotiation is just like testStateStoreMock but the returned mock is set up so it'll be configured
// without this error: `Failed to negotiate acceptable chunk size`
//
// This is meant to be a convenience method when a test is definitely not testing anything related to state store configuration.
func testStateStoreMockWithChunkNegotiation(t *testing.T, chunkSize int64) *testing_provider.MockProvider {
	t.Helper()
	mock := testStateStoreMock(t)
	mock.ConfigureStateStoreResponse = &providers.ConfigureStateStoreResponse{
		Capabilities: providers.StateStoreServerCapabilities{
			ChunkSize: chunkSize,
		},
	}
	return mock
}

func configBodyForTest(t *testing.T, config string) hcl.Body {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(config), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("failure creating hcl.Body during test setup: %s", diags.Error())
	}
	return f.Body
}
