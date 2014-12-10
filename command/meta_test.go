package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
)

func TestMetaColorize(t *testing.T) {
	var m *Meta
	var args, args2 []string

	// Test basic, color
	m = new(Meta)
	m.Color = true
	args = []string{"foo", "bar"}
	args2 = []string{"foo", "bar"}
	args = m.process(args, false)
	if !reflect.DeepEqual(args, args2) {
		t.Fatalf("bad: %#v", args)
	}
	if m.Colorize().Disable {
		t.Fatal("should not be disabled")
	}

	// Test basic, no change
	m = new(Meta)
	args = []string{"foo", "bar"}
	args2 = []string{"foo", "bar"}
	args = m.process(args, false)
	if !reflect.DeepEqual(args, args2) {
		t.Fatalf("bad: %#v", args)
	}
	if !m.Colorize().Disable {
		t.Fatal("should be disabled")
	}

	// Test disable #1
	m = new(Meta)
	m.Color = true
	args = []string{"foo", "-no-color", "bar"}
	args2 = []string{"foo", "bar"}
	args = m.process(args, false)
	if !reflect.DeepEqual(args, args2) {
		t.Fatalf("bad: %#v", args)
	}
	if !m.Colorize().Disable {
		t.Fatal("should be disabled")
	}
}

func TestMetaInputMode(t *testing.T) {
	test = false
	defer func() { test = true }()

	m := new(Meta)
	args := []string{}

	fs := m.flagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode() != terraform.InputModeStd {
		t.Fatalf("bad: %#v", m.InputMode())
	}
}

func TestMetaInputMode_disable(t *testing.T) {
	test = false
	defer func() { test = true }()

	m := new(Meta)
	args := []string{"-input=false"}

	fs := m.flagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode() > 0 {
		t.Fatalf("bad: %#v", m.InputMode())
	}
}

func TestMetaInputMode_defaultVars(t *testing.T) {
	test = false
	defer func() { test = true }()

	// Create a temporary directory for our cwd
	d := tempDir(t)
	if err := os.MkdirAll(d, 0755); err != nil {
		t.Fatalf("err: %s", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(d); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	// Create the default vars file
	err = ioutil.WriteFile(
		filepath.Join(d, DefaultVarsFilename),
		[]byte(""),
		0644)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m := new(Meta)
	args := []string{}
	args = m.process(args, true)

	fs := m.flagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode()&terraform.InputModeVar != 0 {
		t.Fatalf("bad: %#v", m.InputMode())
	}
}

func TestMetaInputMode_vars(t *testing.T) {
	test = false
	defer func() { test = true }()

	m := new(Meta)
	args := []string{"-var", "foo=bar"}

	fs := m.flagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode()&terraform.InputModeVar != 0 {
		t.Fatalf("bad: %#v", m.InputMode())
	}
}

func TestMeta_initStatePaths(t *testing.T) {
	m := new(Meta)
	m.initStatePaths()

	if m.statePath != DefaultStateFilename {
		t.Fatalf("bad: %#v", m)
	}
	if m.stateOutPath != DefaultStateFilename {
		t.Fatalf("bad: %#v", m)
	}
	if m.backupPath != DefaultStateFilename+DefaultBackupExtention {
		t.Fatalf("bad: %#v", m)
	}

	m = new(Meta)
	m.statePath = "foo"
	m.initStatePaths()

	if m.stateOutPath != "foo" {
		t.Fatalf("bad: %#v", m)
	}
	if m.backupPath != "foo"+DefaultBackupExtention {
		t.Fatalf("bad: %#v", m)
	}

	m = new(Meta)
	m.stateOutPath = "foo"
	m.initStatePaths()

	if m.statePath != DefaultStateFilename {
		t.Fatalf("bad: %#v", m)
	}
	if m.backupPath != "foo"+DefaultBackupExtention {
		t.Fatalf("bad: %#v", m)
	}
}

func TestMeta_persistLocal(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	m := new(Meta)
	s := terraform.NewState()
	if err := m.persistLocalState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	exists, err := remote.ExistsFile(m.stateOutPath)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !exists {
		t.Fatalf("state should exist")
	}

	// Write again, shoudl backup
	if err := m.persistLocalState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	exists, err = remote.ExistsFile(m.backupPath)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !exists {
		t.Fatalf("backup should exist")
	}
}

func TestMeta_persistRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	err := remote.EnsureDirectory()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	s := terraform.NewState()
	conf, srv := testRemoteState(t, s, 200)
	s.Remote = conf
	defer srv.Close()

	m := new(Meta)
	if err := m.persistRemoteState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	local, _, err := remote.ReadLocalState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if local == nil {
		t.Fatalf("state should exist")
	}

	if err := m.persistRemoteState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	backup := remote.LocalDirectory + "/" + remote.BackupHiddenStateFile
	exists, err := remote.ExistsFile(backup)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !exists {
		t.Fatalf("backup should exist")
	}
}

func TestMeta_loadState_remote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	err := remote.EnsureDirectory()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	s := terraform.NewState()
	s.Serial = 1000
	conf, srv := testRemoteState(t, s, 200)
	s.Remote = conf
	defer srv.Close()

	s.Serial = 500
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	m := new(Meta)
	s1, err := m.loadState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if s1.Serial < 1000 {
		t.Fatalf("Bad: %#v", s1)
	}

	if !m.useRemoteState {
		t.Fatalf("should enable remote")
	}
}

func TestMeta_loadState_statePath(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	m := new(Meta)

	s := terraform.NewState()
	s.Serial = 1000
	if err := m.persistLocalState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	s1, err := m.loadState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if s1.Serial < 1000 {
		t.Fatalf("Bad: %#v", s1)
	}
}

func TestMeta_loadState_conflict(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	err := remote.EnsureDirectory()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	m := new(Meta)

	s := terraform.NewState()
	if err := remote.PersistState(s); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := m.persistLocalState(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	_, err = m.loadState()
	if err == nil {
		t.Fatalf("should error with conflict")
	}
}
