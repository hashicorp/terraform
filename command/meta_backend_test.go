package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	backendInit "github.com/hashicorp/terraform/backend/init"
	backendLocal "github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// Test empty directory with no config/state creates a local state.
func TestMetaBackend_emptyDir(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Get the backend
	m := testMetaBackend(t, nil)
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Write some state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
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
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Write the legacy state
	statePath := DefaultStateFilename
	{
		f, err := os.Create(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		err = terraform.WriteState(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Get the backend
	m := testMetaBackend(t, nil)
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
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
	next.Modules[0].Outputs["foo"] = &terraform.OutputState{Value: "bar"}
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify a backup was made since we're modifying a pre-existing state
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state should not be empty")
	}
}

// Test an empty directory with an explicit state path (outside the dir)
func TestMetaBackend_emptyWithExplicitState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create another directory to store our state
	stateDir := tempDir(t)
	os.MkdirAll(stateDir, 0755)
	defer os.RemoveAll(stateDir)

	// Write the legacy state
	statePath := filepath.Join(stateDir, "foo")
	{
		f, err := os.Create(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		err = terraform.WriteState(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.statePath = statePath

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
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
	next.Modules[0].Outputs["foo"] = &terraform.OutputState{Value: "bar"}
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify a backup was made since we're modifying a pre-existing state
	if isEmptyState(statePath + DefaultBackupExtension) {
		t.Fatal("backup state should not be empty")
	}
}

// Empty directory with legacy remote state
func TestMetaBackend_emptyLegacyRemote(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create some legacy remote state
	legacyState := testState()
	_, srv := testRemoteState(t, legacyState, 200)
	defer srv.Close()
	statePath := testStateFileRemote(t, legacyState)

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if actual := state.String(); actual != legacyState.String() {
		t.Fatalf("bad: %s", actual)
	}

	// Verify we didn't setup the backend state
	if !state.Backend.Empty() {
		t.Fatal("shouldn't configure backend")
	}

	// Verify the default paths don't exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
	if _, err := os.Stat(statePath + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// Verify that interpolations result in an error
func TestMetaBackend_configureInterpolation(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-interp"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	_, err := m.Backend(&BackendOpts{Init: true})
	if err == nil {
		t.Fatal("should error")
	}
}

// Newly configured backend
func TestMetaBackend_configureNew(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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
func TestMetaBackend_configureNewWithState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-migrate"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}

	if state.Lineage != "backend-new-migrate" {
		t.Fatalf("bad: %#v", state)
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with matching local and remote state doesn't prompt
// for copy.
func TestMetaBackend_configureNewWithoutCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-migrate"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	if err := copy.CopyFile(DefaultStateFilename, "local-state.tfstate"); err != nil {
		t.Fatal(err)
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.input = false

	// init the backend
	_, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	f, err := os.Open("local-state.tfstate")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	actual, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Lineage != "backend-new-migrate" {
		t.Fatalf("incorrect state lineage: %q", actual.Lineage)
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with prior local state and no remote state,
// but opting to not migrate.
func TestMetaBackend_configureNewWithStateNoMigrate(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-migrate"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if state := s.State(); state != nil {
		t.Fatal("state is not nil")
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with prior local state and remote state
func TestMetaBackend_configureNewWithStateExisting(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-migrate-existing"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Setup the meta
	m := testMetaBackend(t, nil)
	// suppress input
	m.forceInitCopy = true

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "local" {
		t.Fatalf("bad: %#v", state)
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with prior local state and remote state
func TestMetaBackend_configureNewWithStateExistingNoMigrate(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-migrate-existing"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "remote" {
		t.Fatalf("bad: %#v", state)
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Newly configured backend with lgacy
func TestMetaBackend_configureNewLegacy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("backup should be empty, but contains:\n", string(data))
	}
}

// Newly configured backend with legacy
func TestMetaBackend_configureNewLegacyCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-new-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// suppress input
	m.forceInitCopy = true

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("nil state")
	}
	if state.Lineage != "backend-new-legacy" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify we have no configured legacy in the state itself
	{
		if !state.Remote.Empty() {
			t.Fatalf("legacy has remote state: %#v", state.Remote)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state matching config
func TestMetaBackend_configuredUnchanged(t *testing.T) {
	defer testChdir(t, testFixturePath("backend-unchanged"))()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("nil state")
	}
	if state.Lineage != "configuredUnchanged" {
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
func TestMetaBackend_configuredChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
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
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state-2.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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
func TestMetaBackend_reconfigureChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-single-to-single"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	newState := s.State()
	if newState != nil || !newState.Empty() {
		t.Fatal("state should be nil/empty after forced reconfiguration")
	}

	// verify that the old state is still there
	s = (&state.LocalState{Path: "local-state.tfstate"})
	if err := s.RefreshState(); err != nil {
		t.Fatal(err)
	}
	oldState := s.State()
	if oldState == nil || oldState.Empty() {
		t.Fatal("original state should be untouched")
	}
}

// Changing a configured backend, copying state
func TestMetaBackend_configuredChangeCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if state.Lineage != "backend-change" {
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
func TestMetaBackend_configuredChangeCopy_singleState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-single-to-single"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if state.Lineage != "backend-change" {
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
func TestMetaBackend_configuredChangeCopy_multiToSingleDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-multi-default-to-single"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if state.Lineage != "backend-change" {
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
func TestMetaBackend_configuredChangeCopy_multiToSingle(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-multi-to-single"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if state.Lineage != "backend-change" {
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
	if env := m.Workspace(); env != backend.DefaultStateName {
		t.Fatal("using non-default env with single-env backend")
	}
}

// Changing a configured backend that supports multi-state to a
// backend that only supports single states.
func TestMetaBackend_configuredChangeCopy_multiToSingleCurrentEnv(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-multi-to-single"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
		t.Fatalf("bad: %s", err)
	}

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if state.Lineage != "backend-change-env2" {
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
func TestMetaBackend_configuredChangeCopy_multiToMulti(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-multi-to-multi"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-multistate": "yes",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check resulting states
	states, err := b.States()
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	sort.Strings(states)
	expected := []string{"default", "env2"}
	if !reflect.DeepEqual(states, expected) {
		t.Fatalf("bad: %#v", states)
	}

	{
		// Check the default state
		s, err := b.State(backend.DefaultStateName)
		if err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("bad: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if state.Lineage != "backend-change" {
			t.Fatalf("bad: %#v", state)
		}
	}

	{
		// Check the other state
		s, err := b.State("env2")
		if err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("bad: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if state.Lineage != "backend-change-env2" {
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
// default state while the default state is non-empty.
func TestMetaBackend_configuredChangeCopy_multiToNoDefaultWithDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-multi-to-no-default-with-default"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check resulting states
	states, err := b.States()
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	sort.Strings(states)
	expected := []string{"env1", "env2"}
	if !reflect.DeepEqual(states, expected) {
		t.Fatalf("bad: %#v", states)
	}

	{
		// Check the renamed default state
		s, err := b.State("env1")
		if err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("bad: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if state.Lineage != "backend-change-env1" {
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
func TestMetaBackend_configuredChangeCopy_multiToNoDefaultWithoutDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change-multi-to-no-default-without-default"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Register the single-state backend
	backendInit.Set("local-no-default", backendLocal.TestNewLocalNoDefault)
	defer backendInit.Set("local-no-default", nil)

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-multistate": "yes",
		"select-workspace":                         "1",
	})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check resulting states
	states, err := b.States()
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	sort.Strings(states)
	expected := []string{"env2"}
	if !reflect.DeepEqual(states, expected) {
		t.Fatalf("bad: %#v", states)
	}

	{
		// Check the named state
		s, err := b.State("env2")
		if err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := s.RefreshState(); err != nil {
			t.Fatalf("bad: %s", err)
		}
		state := s.State()
		if state == nil {
			t.Fatal("state should not be nil")
		}
		if state.Lineage != "backend-change-env2" {
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

// Unsetting a saved backend
func TestMetaBackend_configuredUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unset"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Verify the default paths don't exist
	if !isEmptyState(DefaultStateFilename) {
		data, _ := ioutil.ReadFile(DefaultStateFilename)
		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		data, _ := ioutil.ReadFile(DefaultStateFilename + DefaultBackupExtension)
		t.Fatal("backup should not exist, but contains:\n", string(data))
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if !actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify it exists where we expect it to
	if isEmptyState(DefaultStateFilename) {
		t.Fatal(DefaultStateFilename, "is empty")
	}

	// Verify no backup since it was empty to start
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		data, _ := ioutil.ReadFile(DefaultStateFilename + DefaultBackupExtension)
		t.Fatal("backup state should be empty, but contains:\n", string(data))
	}
}

// Unsetting a saved backend and copying the remote state
func TestMetaBackend_configuredUnsetCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unset"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "configuredUnset" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatalf("backup state should be empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if !actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
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

// Saved backend state matching config, with legacy
func TestMetaBackend_configuredUnchangedLegacy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unchanged-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "configured" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default paths don't exist
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatalf("err: %s", err)
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state matching config, with legacy
func TestMetaBackend_configuredUnchangedLegacyCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unchanged-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.forceInitCopy = true

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "backend-unchanged-with-legacy" {
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

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state, new config, legacy remote state
func TestMetaBackend_configuredChangedLegacy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-changed-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no", "no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
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

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state-2.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state, new config, legacy remote state
func TestMetaBackend_configuredChangedLegacyCopyBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-changed-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "configured" {
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

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state-2.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state, new config, legacy remote state
func TestMetaBackend_configuredChangedLegacyCopyLegacy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-changed-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no", "yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "legacy" {
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

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state-2.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state, new config, legacy remote state
func TestMetaBackend_configuredChangedLegacyCopyBoth(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-changed-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "yes", "yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "legacy" {
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

	// Verify we have no configured legacy
	{
		path := filepath.Join(m.DataDir(), DefaultStateFilename)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state-2.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
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

// Saved backend state, unset config, legacy remote state
func TestMetaBackend_configuredUnsetWithLegacyNoCopy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unset-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no", "no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatal("state should be nil")
	}

	// Verify the default paths dont exist since we had no state
	if !isEmptyState(DefaultStateFilename) {
		t.Fatal("state should be empty")
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup should be empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if !actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}
}

// Saved backend state, unset config, legacy remote state
func TestMetaBackend_configuredUnsetWithLegacyCopyBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unset-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "no"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "backend" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default paths exist
	if isEmptyState(DefaultStateFilename) {
		t.Fatalf("default state was empty")
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backupstate should be empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if !actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify a local backup
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// Saved backend state, unset config, legacy remote state
func TestMetaBackend_configuredUnsetWithLegacyCopyLegacy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unset-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no", "yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "legacy" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default paths exist
	if isEmptyState(DefaultStateFilename) {
		t.Fatalf("default state was empty")
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backupstate should be empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if !actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify a local backup
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// Saved backend state, unset config, legacy remote state
func TestMetaBackend_configuredUnsetWithLegacyCopyBoth(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unset-with-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes", "yes", "yes", "yes"})()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Init: true})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "legacy" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default paths exist
	if isEmptyState(DefaultStateFilename) {
		t.Fatal("state is empty")
	}

	// Verify a backup exists
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !actual.Remote.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
		if !actual.Backend.Empty() {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify a local backup
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// A plan that has no backend config
func TestMetaBackend_planLocal(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create the plan
	plan := &terraform.Plan{
		Module: testModule(t, "backend-plan-local"),
		State:  nil,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Plan: plan})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state != nil {
		t.Fatalf("state should be nil: %#v", state)
	}

	// Verify the default path doens't exist
	if !isEmptyState(DefaultStateFilename) {
		t.Fatal("expected empty state")
	}

	// Verify a backup doesn't exists
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("expected empty backup")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify no local backup
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatalf("backup state should be empty")
	}
}

// A plan with a custom state save path
func TestMetaBackend_planLocalStatePath(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create our state
	original := testState()
	original.Lineage = "hello"

	// Create the plan
	plan := &terraform.Plan{
		Module: testModule(t, "backend-plan-local"),
		State:  original,
	}

	// Create an alternate output path
	statePath := "foo.tfstate"

	// put a initial state there that needs to be backed up
	err := (&state.LocalState{Path: statePath}).WriteState(original)
	if err != nil {
		t.Fatal(err)
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.stateOutPath = statePath

	// Get the backend
	b, err := m.Backend(&BackendOpts{Plan: plan})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.Lineage != "hello" {
		t.Fatalf("bad: %#v", state)
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
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify we have a backup
	if isEmptyState(statePath + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// A plan that has no backend config, matching local state
func TestMetaBackend_planLocalMatch(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create the plan
	plan := &terraform.Plan{
		Module: testModule(t, "backend-plan-local-match"),
		State:  testStateRead(t, DefaultStateFilename),
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Plan: plan})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("should is nil")
	}
	if state.Lineage != "hello" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default path
	if isEmptyState(DefaultStateFilename) {
		t.Fatal("state is empty")
	}

	// Verify a backup exists
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify local backup
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is empty")
	}
}

// A plan that has no backend config, mismatched lineage
func TestMetaBackend_planLocalMismatchLineage(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local-mismatch-lineage"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Save the original
	original := testStateRead(t, DefaultStateFilename)

	// Change the lineage
	planState := testStateRead(t, DefaultStateFilename)
	planState.Lineage = "bad"

	// Create the plan
	plan := &terraform.Plan{
		Module: testModule(t, "backend-plan-local-mismatch-lineage"),
		State:  planState,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	_, err := m.Backend(&BackendOpts{Plan: plan})
	if err == nil {
		t.Fatal("should have error")
	}
	if !strings.Contains(err.Error(), "lineage") {
		t.Fatalf("bad: %s", err)
	}

	// Verify our local state didn't change
	actual := testStateRead(t, DefaultStateFilename)
	if !actual.Equal(original) {
		t.Fatalf("bad: %#v", actual)
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
}

// A plan that has no backend config, newer local
func TestMetaBackend_planLocalNewer(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local-newer"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Save the original
	original := testStateRead(t, DefaultStateFilename)

	// Change the serial
	planState := testStateRead(t, DefaultStateFilename)
	planState.Serial = 7
	planState.RootModule().Dependencies = []string{"foo"}

	// Create the plan
	plan := &terraform.Plan{
		Module: testModule(t, "backend-plan-local-newer"),
		State:  planState,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	_, err := m.Backend(&BackendOpts{Plan: plan})
	if err == nil {
		t.Fatal("should have error")
	}
	if !strings.Contains(err.Error(), "older") {
		t.Fatalf("bad: %s", err)
	}

	// Verify our local state didn't change
	actual := testStateRead(t, DefaultStateFilename)
	if !actual.Equal(original) {
		t.Fatalf("bad: %#v", actual)
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
}

// A plan that has a backend in an empty dir
func TestMetaBackend_planBackendEmptyDir(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-backend-empty"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Get the state for the plan by getting the real state and
	// adding the backend config to it.
	original := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-backend-empty-config"),
		"local-state.tfstate"))
	backendState := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-backend-empty-config"),
		DefaultDataDir, DefaultStateFilename))
	planState := original.DeepCopy()

	// Create the plan
	plan := &terraform.Plan{
		Module:  testModule(t, "backend-plan-backend-empty-config"),
		State:   planState,
		Backend: backendState.Backend,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Plan: plan})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("should is nil")
	}
	if state.Lineage != "hello" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default path doesn't exist
	if !isEmptyState(DefaultStateFilename) {
		t.Fatal("state is not empty")
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup is not empty")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify no default path
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// A plan that has a backend with matching state
func TestMetaBackend_planBackendMatch(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-backend-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Get the state for the plan by getting the real state and
	// adding the backend config to it.
	original := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-backend-empty-config"),
		"local-state.tfstate"))
	backendState := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-backend-empty-config"),
		DefaultDataDir, DefaultStateFilename))
	planState := original.DeepCopy()

	// Create the plan
	plan := &terraform.Plan{
		Module:  testModule(t, "backend-plan-backend-empty-config"),
		State:   planState,
		Backend: backendState.Backend,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Plan: plan})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("should is nil")
	}
	if state.Lineage != "hello" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default path exists
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify no default path
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// A plan that has a backend with mismatching lineage
func TestMetaBackend_planBackendMismatchLineage(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-backend-mismatch"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Get the state for the plan by getting the real state and
	// adding the backend config to it.
	original := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-backend-empty-config"),
		"local-state.tfstate"))
	backendState := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-backend-empty-config"),
		DefaultDataDir, DefaultStateFilename))
	planState := original.DeepCopy()

	// Get the real original
	original = testStateRead(t, "local-state.tfstate")

	// Create the plan
	plan := &terraform.Plan{
		Module:  testModule(t, "backend-plan-backend-empty-config"),
		State:   planState,
		Backend: backendState.Backend,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	_, err := m.Backend(&BackendOpts{Plan: plan})
	if err == nil {
		t.Fatal("should have error")
	}
	if !strings.Contains(err.Error(), "lineage") {
		t.Fatalf("bad: %s", err)
	}

	// Verify our local state didn't change
	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(original) {
		t.Fatalf("bad: %#v", actual)
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Verify we have no default state
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}
}

// A plan that has a legacy remote state
func TestMetaBackend_planLegacy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-legacy"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Get the state for the plan by getting the real state and
	// adding the backend config to it.
	original := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-legacy-data"), "local-state.tfstate"))
	dataState := testStateRead(t, filepath.Join(
		testFixturePath("backend-plan-legacy-data"), "state.tfstate"))
	planState := original.DeepCopy()
	planState.Remote = dataState.Remote

	// Create the plan
	plan := &terraform.Plan{
		Module: testModule(t, "backend-plan-legacy-data"),
		State:  planState,
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, err := m.Backend(&BackendOpts{Plan: plan})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check the state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("should is nil")
	}
	if state.Lineage != "hello" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify the default path
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify a backup doesn't exist
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify we have no configured backend/legacy
	path := filepath.Join(m.DataDir(), DefaultStateFilename)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("should not have backend configured")
	}

	// Write some state
	state = terraform.NewState()
	state.Lineage = "changing"
	s.WriteState(state)
	if err := s.PersistState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify the state is where we expect
	{
		f, err := os.Open("local-state.tfstate")
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		actual, err := terraform.ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual.Lineage != state.Lineage {
			t.Fatalf("bad: %#v", actual)
		}
	}

	// Verify no default path
	if _, err := os.Stat(DefaultStateFilename); err == nil {
		t.Fatal("file should not exist")
	}

	// Verify no local backup
	if _, err := os.Stat(DefaultStateFilename + DefaultBackupExtension); err == nil {
		t.Fatal("file should not exist")
	}
}

// init a backend using -backend-config options multiple times
func TestMetaBackend_configureWithExtra(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-empty"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	extras := map[string]interface{}{"path": "hello"}
	m := testMetaBackend(t, nil)
	opts := &BackendOpts{
		ConfigExtra: extras,
		Init:        true,
	}

	backendCfg, err := m.backendConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	// init the backend
	_, err = m.Backend(&BackendOpts{
		ConfigExtra: extras,
		Init:        true,
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s := testStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))
	if s.Backend.Hash != backendCfg.Hash {
		t.Fatal("mismatched state and config backend hashes")
	}
	if s.Backend.Rehash() == s.Backend.Hash {
		t.Fatal("saved hash should not match actual hash")
	}
	if s.Backend.Rehash() != backendCfg.Rehash() {
		t.Fatal("mismatched state and config re-hashes")
	}

	// init the backend again with the same options
	m = testMetaBackend(t, nil)
	_, err = m.Backend(&BackendOpts{
		ConfigExtra: extras,
		Init:        true,
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s = testStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))
	if s.Backend.Hash != backendCfg.Hash {
		t.Fatal("mismatched state and config backend hashes")
	}
}

// when confniguring a default local state, don't delete local state
func TestMetaBackend_localDoesNotDeleteLocal(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-empty"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// create our local state
	orig := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
		},
	}

	err := (&state.LocalState{Path: DefaultStateFilename}).WriteState(orig)
	if err != nil {
		t.Fatal(err)
	}

	m := testMetaBackend(t, nil)
	m.forceInitCopy = true
	// init the backend
	_, err = m.Backend(&BackendOpts{
		Init: true,
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// check that we can read the state
	s := testStateRead(t, DefaultStateFilename)
	if s.Empty() {
		t.Fatal("our state was deleted")
	}
}

// move options from config to -backend-config
func TestMetaBackend_configToExtra(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// init the backend
	m := testMetaBackend(t, nil)
	_, err := m.Backend(&BackendOpts{
		Init: true,
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Check the state
	s := testStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))
	backendHash := s.Backend.Hash

	// init again but remove the path option from the config
	cfg := "terraform {\n  backend \"local\" {}\n}\n"
	if err := ioutil.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	// init the backend again with the  options
	extras := map[string]interface{}{"path": "hello"}
	m = testMetaBackend(t, nil)
	m.forceInitCopy = true
	_, err = m.Backend(&BackendOpts{
		ConfigExtra: extras,
		Init:        true,
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	s = testStateRead(t, filepath.Join(DefaultDataDir, backendLocal.DefaultStateFilename))

	if s.Backend.Hash == backendHash {
		t.Fatal("state.Backend.Hash was not updated")
	}
}

func testMetaBackend(t *testing.T, args []string) *Meta {
	var m Meta
	m.Ui = new(cli.MockUi)
	m.process(args, true)
	f := m.flagSet("test")
	if err := f.Parse(args); err != nil {
		t.Fatalf("bad: %s", err)
	}

	return &m
}
