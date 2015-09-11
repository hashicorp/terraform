package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// Test disabling remote management
func TestRemoteConfig_disable(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	s := terraform.NewState()
	s.Serial = 10
	conf, srv := testRemoteState(t, s, 200)
	defer srv.Close()

	// Persist local remote state
	s = terraform.NewState()
	s.Serial = 5
	s.Remote = conf

	// Write the state
	statePath := filepath.Join(tmp, DefaultDataDir, DefaultStateFilename)
	state := &state.LocalState{Path: statePath}
	if err := state.WriteState(s); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := state.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}
	args := []string{"-disable"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Local state file should be removed and the local cache should exist
	testRemoteLocal(t, true)
	testRemoteLocalCache(t, false)

	// Check that the state file was updated
	raw, _ := ioutil.ReadFile(DefaultStateFilename)
	newState, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we updated
	if newState.Remote != nil {
		t.Fatalf("remote configuration not removed")
	}
}

// Test disabling remote management without pulling
func TestRemoteConfig_disable_noPull(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	s := terraform.NewState()
	s.Serial = 10
	conf, srv := testRemoteState(t, s, 200)
	defer srv.Close()

	// Persist local remote state
	s = terraform.NewState()
	s.Serial = 5
	s.Remote = conf

	// Write the state
	statePath := filepath.Join(tmp, DefaultDataDir, DefaultStateFilename)
	state := &state.LocalState{Path: statePath}
	if err := state.WriteState(s); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := state.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}
	args := []string{"-disable", "-pull=false"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Local state file should be removed and the local cache should exist
	testRemoteLocal(t, true)
	testRemoteLocalCache(t, false)

	// Check that the state file was updated
	raw, _ := ioutil.ReadFile(DefaultStateFilename)
	newState, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if newState.Remote != nil {
		t.Fatalf("remote configuration not removed")
	}
}

// Test disabling remote management when not enabled
func TestRemoteConfig_disable_notEnabled(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{"-disable"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// Test disabling remote management with a state file in the way
func TestRemoteConfig_disable_otherState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5

	// Write the state
	statePath := filepath.Join(tmp, DefaultDataDir, DefaultStateFilename)
	state := &state.LocalState{Path: statePath}
	if err := state.WriteState(s); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := state.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Also put a file at the default path
	fh, err := os.Create(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = terraform.WriteState(s, fh)
	fh.Close()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{"-disable"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// Test the case where both managed and non managed state present
func TestRemoteConfig_managedAndNonManaged(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5

	// Write the state
	statePath := filepath.Join(tmp, DefaultDataDir, DefaultStateFilename)
	state := &state.LocalState{Path: statePath}
	if err := state.WriteState(s); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := state.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Also put a file at the default path
	fh, err := os.Create(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = terraform.WriteState(s, fh)
	fh.Close()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// Test initializing blank state
func TestRemoteConfig_initBlank(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-backend-config", "address=http://example.com",
		"-backend-config", "access_token=test",
		"-pull=false",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	remotePath := filepath.Join(DefaultDataDir, DefaultStateFilename)
	ls := &state.LocalState{Path: remotePath}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	local := ls.State()
	if local.Remote.Type != "http" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["address"] != "http://example.com" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["access_token"] != "test" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
}

// Test initializing without remote settings
func TestRemoteConfig_initBlank_missingRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// Test updating remote config
func TestRemoteConfig_updateRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = &terraform.RemoteState{
		Type: "invalid",
	}

	// Write the state
	statePath := filepath.Join(tmp, DefaultDataDir, DefaultStateFilename)
	ls := &state.LocalState{Path: statePath}
	if err := ls.WriteState(s); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := ls.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-backend-config", "address=http://example.com",
		"-backend-config", "access_token=test",
		"-pull=false",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	remotePath := filepath.Join(DefaultDataDir, DefaultStateFilename)
	ls = &state.LocalState{Path: remotePath}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}
	local := ls.State()

	if local.Remote.Type != "http" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["address"] != "http://example.com" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["access_token"] != "test" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
}

// Test enabling remote state
func TestRemoteConfig_enableRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create a non-remote enabled state
	s := terraform.NewState()
	s.Serial = 5

	// Add the state at the default path
	fh, err := os.Create(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = terraform.WriteState(s, fh)
	fh.Close()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteConfigCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-backend-config", "address=http://example.com",
		"-backend-config", "access_token=test",
		"-pull=false",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	remotePath := filepath.Join(DefaultDataDir, DefaultStateFilename)
	ls := &state.LocalState{Path: remotePath}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}
	local := ls.State()

	if local.Remote.Type != "http" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["address"] != "http://example.com" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["access_token"] != "test" {
		t.Fatalf("Bad: %#v", local.Remote)
	}

	// Backup file should exist, state file should not
	testRemoteLocal(t, false)
	testRemoteLocalBackup(t, true)
}

func testRemoteLocal(t *testing.T, exists bool) {
	_, err := os.Stat(DefaultStateFilename)
	if os.IsNotExist(err) && !exists {
		return
	}
	if err == nil && exists {
		return
	}

	t.Fatalf("bad: %#v", err)
}

func testRemoteLocalBackup(t *testing.T, exists bool) {
	_, err := os.Stat(DefaultStateFilename + DefaultBackupExtension)
	if os.IsNotExist(err) && !exists {
		return
	}
	if err == nil && exists {
		return
	}
	if err == nil && !exists {
		t.Fatal("expected local backup to exist")
	}

	t.Fatalf("bad: %#v", err)
}

func testRemoteLocalCache(t *testing.T, exists bool) {
	_, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename))
	if os.IsNotExist(err) && !exists {
		return
	}
	if err == nil && exists {
		return
	}
	if err == nil && !exists {
		t.Fatal("expected local cache to exist")
	}

	t.Fatalf("bad: %#v", err)
}
