package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/backend"
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
	args, err := m.process(args, false)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
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
	args, err = m.process(args, false)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
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
	args, err = m.process(args, false)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
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

	if m.InputMode() != terraform.InputModeStd|terraform.InputModeVarUnset {
		t.Fatalf("bad: %#v", m.InputMode())
	}
}

func TestMetaInputMode_envVar(t *testing.T) {
	test = false
	defer func() { test = true }()
	old := os.Getenv(InputModeEnvVar)
	defer os.Setenv(InputModeEnvVar, old)

	m := new(Meta)
	args := []string{}

	fs := m.flagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	off := terraform.InputMode(0)
	on := terraform.InputModeStd | terraform.InputModeVarUnset
	cases := []struct {
		EnvVar   string
		Expected terraform.InputMode
	}{
		{"false", off},
		{"0", off},
		{"true", on},
		{"1", on},
	}

	for _, tc := range cases {
		os.Setenv(InputModeEnvVar, tc.EnvVar)
		if m.InputMode() != tc.Expected {
			t.Fatalf("expected InputMode: %#v, got: %#v", tc.Expected, m.InputMode())
		}
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
	os.MkdirAll(d, 0755)
	defer os.RemoveAll(d)
	defer testChdir(t, d)()

	// Create the default vars file
	err := ioutil.WriteFile(
		filepath.Join(d, DefaultVarsFilename),
		[]byte(""),
		0644)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m := new(Meta)
	args := []string{}
	args, err = m.process(args, false)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	fs := m.flagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode()&terraform.InputModeVar == 0 {
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

	if m.InputMode()&terraform.InputModeVar == 0 {
		t.Fatalf("bad: %#v", m.InputMode())
	}

	if m.InputMode()&terraform.InputModeVarUnset == 0 {
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
	if m.backupPath != DefaultStateFilename+DefaultBackupExtension {
		t.Fatalf("bad: %#v", m)
	}

	m = new(Meta)
	m.statePath = "foo"
	m.initStatePaths()

	if m.stateOutPath != "foo" {
		t.Fatalf("bad: %#v", m)
	}
	if m.backupPath != "foo"+DefaultBackupExtension {
		t.Fatalf("bad: %#v", m)
	}

	m = new(Meta)
	m.stateOutPath = "foo"
	m.initStatePaths()

	if m.statePath != DefaultStateFilename {
		t.Fatalf("bad: %#v", m)
	}
	if m.backupPath != "foo"+DefaultBackupExtension {
		t.Fatalf("bad: %#v", m)
	}
}

func TestMeta_Env(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	m := new(Meta)

	env := m.Workspace()

	if env != backend.DefaultStateName {
		t.Fatalf("expected env %q, got env %q", backend.DefaultStateName, env)
	}

	testEnv := "test_env"
	if err := m.SetWorkspace(testEnv); err != nil {
		t.Fatal("error setting env:", err)
	}

	env = m.Workspace()
	if env != testEnv {
		t.Fatalf("expected env %q, got env %q", testEnv, env)
	}

	if err := m.SetWorkspace(backend.DefaultStateName); err != nil {
		t.Fatal("error setting env:", err)
	}

	env = m.Workspace()
	if env != backend.DefaultStateName {
		t.Fatalf("expected env %q, got env %q", backend.DefaultStateName, env)
	}
}

func TestMeta_process(t *testing.T) {
	test = false
	defer func() { test = true }()

	// Create a temporary directory for our cwd
	d := tempDir(t)
	os.MkdirAll(d, 0755)
	defer os.RemoveAll(d)
	defer testChdir(t, d)()

	// Create two vars files
	defaultVarsfile := "terraform.tfvars"
	err := ioutil.WriteFile(
		filepath.Join(d, defaultVarsfile),
		[]byte(""),
		0644)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	fileFirstAlphabetical := "a-file.auto.tfvars"
	err = ioutil.WriteFile(
		filepath.Join(d, fileFirstAlphabetical),
		[]byte(""),
		0644)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	fileLastAlphabetical := "z-file.auto.tfvars"
	err = ioutil.WriteFile(
		filepath.Join(d, fileLastAlphabetical),
		[]byte(""),
		0644)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	// Regular tfvars files will not be autoloaded
	fileIgnored := "ignored.tfvars"
	err = ioutil.WriteFile(
		filepath.Join(d, fileIgnored),
		[]byte(""),
		0644)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m := new(Meta)
	args := []string{}
	args, err = m.process(args, true)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %v", args)
	}

	if args[0] != "-var-file-default" {
		t.Fatalf("expected %q, got %q", "-var-file-default", args[0])
	}
	if args[1] != defaultVarsfile {
		t.Fatalf("expected %q, got %q", defaultVarsfile, args[1])
	}
	if args[2] != "-var-file-default" {
		t.Fatalf("expected %q, got %q", "-var-file-default", args[2])
	}
	if args[3] != fileFirstAlphabetical {
		t.Fatalf("expected %q, got %q", fileFirstAlphabetical, args[3])
	}
	if args[4] != "-var-file-default" {
		t.Fatalf("expected %q, got %q", "-var-file-default", args[4])
	}
	if args[5] != fileLastAlphabetical {
		t.Fatalf("expected %q, got %q", fileLastAlphabetical, args[5])
	}
}
