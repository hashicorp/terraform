package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// Test disabling remote management
func TestRemote_disable(t *testing.T) {
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
	if err := remote.EnsureDirectory(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}
	args := []string{"-disable"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Local state file should be removed
	haveLocal, err := remote.HaveLocalState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if haveLocal {
		t.Fatalf("should be disabled")
	}

	// New state file should be installed
	exists, err := remote.ExistsFile(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !exists {
		t.Fatalf("failed to make state file")
	}

	// Check that the state file was updated
	raw, _ := ioutil.ReadFile(DefaultStateFilename)
	newState, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we updated
	// TODO: Should be 10, but WriteState currently
	// increments incorrectly
	if newState.Serial != 11 {
		t.Fatalf("state file not updated: %#v", newState)
	}
	if newState.Remote != nil {
		t.Fatalf("remote configuration not removed")
	}
}

// Test disabling remote management without pulling
func TestRemote_disable_noPull(t *testing.T) {
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
	if err := remote.EnsureDirectory(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}
	args := []string{"-disable", "-pull=false"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Local state file should be removed
	haveLocal, err := remote.HaveLocalState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if haveLocal {
		t.Fatalf("should be disabled")
	}

	// New state file should be installed
	exists, err := remote.ExistsFile(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !exists {
		t.Fatalf("failed to make state file")
	}

	// Check that the state file was updated
	raw, _ := ioutil.ReadFile(DefaultStateFilename)
	newState, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we DIDNT updated
	// TODO: Should be 5, but WriteState currently increments
	// this which is incorrect.
	if newState.Serial != 7 {
		t.Fatalf("state file updated: %#v", newState)
	}
	if newState.Remote != nil {
		t.Fatalf("remote configuration not removed")
	}
}

// Test disabling remote management when not enabled
func TestRemote_disable_notEnabled(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &RemoteCommand{
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
func TestRemote_disable_otherState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	if err := remote.EnsureDirectory(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
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
	c := &RemoteCommand{
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
func TestRemote_managedAndNonManaged(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	if err := remote.EnsureDirectory(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
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
	c := &RemoteCommand{
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
func TestRemote_initBlank(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &RemoteCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-address", "http://example.com",
		"-access-token=test",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	local, _, err := remote.ReadLocalState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

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
func TestRemote_initBlank_missingRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &RemoteCommand{
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
func TestRemote_updateRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = &terraform.RemoteState{
		Type: "invalid",
	}
	if err := remote.EnsureDirectory(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	ui := new(cli.MockUi)
	c := &RemoteCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-address",
		"http://example.com",
		"-access-token=test",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	local, _, err := remote.ReadLocalState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

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
func TestRemote_enableRemote(t *testing.T) {
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
	c := &RemoteCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-address",
		"http://example.com",
		"-access-token=test",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	local, _, err := remote.ReadLocalState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if local.Remote.Type != "http" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["address"] != "http://example.com" {
		t.Fatalf("Bad: %#v", local.Remote)
	}
	if local.Remote.Config["access_token"] != "test" {
		t.Fatalf("Bad: %#v", local.Remote)
	}

	// Backup file should exist
	exist, err := remote.ExistsFile(DefaultStateFilename + DefaultBackupExtention)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !exist {
		t.Fatalf("backup should exist")
	}

	// State file should not
	exist, err = remote.ExistsFile(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if exist {
		t.Fatalf("state file should not exist")
	}
}
