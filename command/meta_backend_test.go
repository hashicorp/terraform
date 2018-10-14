package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
	backendInit "github.com/hashicorp/terraform/backend/init"
	backendLocal "github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Write some state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
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
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	next.RootModule().SetOutputValue("foo", cty.StringVal("bar"), false)
	s.WriteState(next)
	if err := s.PersistState(); err != nil {
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
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify a backup was made since we're modifying a pre-existing state
	if isEmptyState(statePath + DefaultBackupExtension) {
		t.Fatal("backup state should not be empty")
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	if err := s.PersistState(); err != nil {
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}

	if testStateMgrCurrentLineage(s) != "backend-new-migrate" {
		t.Fatalf("bad: %#v", state)
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(); err != nil {
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if testStateMgrCurrentLineage(s) != "local" {
		t.Fatalf("bad: %#v", state)
	}

	// Write some state
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(); err != nil {
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	if err := s.PersistState(); err != nil {
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
		data, _ := ioutil.ReadFile(DefaultStateFilename)

		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup does exist
	if isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatal("backup state is empty or missing")
	}
}

// Saved backend state matching config
func TestMetaBackend_configuredUnchanged(t *testing.T) {
	defer testChdir(t, testFixturePath("backend-unchanged"))()

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	if err := s.PersistState(); err != nil {
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
		t.Fatalf("unexpected error: %s", err)
	}

	// Get the backend
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check resulting states
	states, err := b.Workspaces()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	sort.Strings(states)
	expected := []string{"default", "env2"}
	if !reflect.DeepEqual(states, expected) {
		t.Fatalf("bad: %#v", states)
	}

	{
		// Check the default state
		s, err := b.StateMgr(backend.DefaultStateName)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
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
		s, err := b.StateMgr("env2")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
		data, _ := ioutil.ReadFile(DefaultStateFilename)
		t.Fatal("state should not exist, but contains:\n", string(data))
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		data, _ := ioutil.ReadFile(DefaultStateFilename + DefaultBackupExtension)
		t.Fatal("backup should not exist, but contains:\n", string(data))
	}

	// Write some state
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	b, diags := m.Backend(&BackendOpts{Init: true})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	if testStateMgrCurrentLineage(s) != "configuredUnset" {
		t.Fatalf("bad: %#v", state)
	}

	// Verify a backup doesn't exist
	if !isEmptyState(DefaultStateFilename + DefaultBackupExtension) {
		t.Fatalf("backup state should be empty")
	}

	// Write some state
	s.WriteState(testState())
	if err := s.PersistState(); err != nil {
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

// A plan that has uses the local backend
func TestMetaBackend_planLocal(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	backendConfig := plans.Backend{
		Type:      "local",
		Config:    plans.DynamicValue("{}"),
		Workspace: "default",
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.BackendForPlan(backendConfig)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	if err := s.PersistState(); err != nil {
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

// A plan with a custom state save path
func TestMetaBackend_planLocalStatePath(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	original := testState()
	mark := markStateForMatching(original, "hello")

	backendConfig := plans.Backend{
		Type:      "local",
		Config:    plans.DynamicValue("{}"),
		Workspace: "default",
	}

	// Create an alternate output path
	statePath := "foo.tfstate"

	// put a initial state there that needs to be backed up
	err := (statemgr.NewFilesystem(statePath)).WriteState(original)
	if err != nil {
		t.Fatal(err)
	}

	// Setup the meta
	m := testMetaBackend(t, nil)
	m.stateOutPath = statePath

	// Get the backend
	b, diags := m.BackendForPlan(backendConfig)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := s.RefreshState(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	state := s.State()
	if state == nil {
		t.Fatal("state is nil")
	}
	assertStateHasMarker(t, state, mark)

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
	mark = markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(); err != nil {
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
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-plan-local-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	backendConfig := plans.Backend{
		Type:      "local",
		Config:    plans.DynamicValue("{}"),
		Workspace: "default",
	}

	// Setup the meta
	m := testMetaBackend(t, nil)

	// Get the backend
	b, diags := m.BackendForPlan(backendConfig)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// Check the state
	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
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
	state = states.NewState()
	mark := markStateForMatching(state, "changing")

	s.WriteState(state)
	if err := s.PersistState(); err != nil {
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

// init a backend using -backend-config options multiple times
func TestMetaBackend_configureWithExtra(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-empty"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	s := testDataStateRead(t, filepath.Join(DefaultDataDir, backendlocal.DefaultStateFilename))
	if s.Backend.Hash != cHash {
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
	s = testDataStateRead(t, filepath.Join(DefaultDataDir, backendlocal.DefaultStateFilename))
	if s.Backend.Hash != cHash {
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
		t.Fatalf("unexpected error: %s", err)
	}

	// Check the state
	s := testDataStateRead(t, filepath.Join(DefaultDataDir, backendlocal.DefaultStateFilename))
	backendHash := s.Backend.Hash

	// init again but remove the path option from the config
	cfg := "terraform {\n  backend \"local\" {}\n}\n"
	if err := ioutil.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
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

	s = testDataStateRead(t, filepath.Join(DefaultDataDir, backendlocal.DefaultStateFilename))

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
		t.Fatalf("unexpected error: %s", err)
	}

	return &m
}
