package command

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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

func TestMeta_addModuleDepthFlag(t *testing.T) {
	old := os.Getenv(ModuleDepthEnvVar)
	defer os.Setenv(ModuleDepthEnvVar, old)

	cases := map[string]struct {
		EnvVar   string
		Args     []string
		Expected int
	}{
		"env var sets value when no flag present": {
			EnvVar:   "4",
			Args:     []string{},
			Expected: 4,
		},
		"flag overrides envvar": {
			EnvVar:   "4",
			Args:     []string{"-module-depth=-1"},
			Expected: -1,
		},
		"negative envvar works": {
			EnvVar:   "-1",
			Args:     []string{},
			Expected: -1,
		},
		"invalid envvar is ignored": {
			EnvVar:   "-#",
			Args:     []string{},
			Expected: 0,
		},
		"empty envvar is okay too": {
			EnvVar:   "",
			Args:     []string{},
			Expected: 0,
		},
	}

	for tn, tc := range cases {
		m := new(Meta)
		var moduleDepth int
		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		os.Setenv(ModuleDepthEnvVar, tc.EnvVar)
		m.addModuleDepthFlag(flags, &moduleDepth)
		err := flags.Parse(tc.Args)
		if err != nil {
			t.Fatalf("%s: err: %#v", tn, err)
		}
		if moduleDepth != tc.Expected {
			t.Fatalf("%s: expected: %#v, got: %#v", tn, tc.Expected, moduleDepth)
		}
	}
}
