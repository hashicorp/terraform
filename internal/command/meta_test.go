package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestMetaColorize(t *testing.T) {
	var m *Meta
	var args, args2 []string

	// Test basic, color
	m = new(Meta)
	m.Color = true
	args = []string{"foo", "bar"}
	args2 = []string{"foo", "bar"}
	args = m.process(args)
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
	args = m.process(args)
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
	args = m.process(args)
	if !reflect.DeepEqual(args, args2) {
		t.Fatalf("bad: %#v", args)
	}
	if !m.Colorize().Disable {
		t.Fatal("should be disabled")
	}

	// Test disable #2
	// Verify multiple -no-color options are removed from args slice.
	// E.g. an additional -no-color arg could be added by TF_CLI_ARGS.
	m = new(Meta)
	m.Color = true
	args = []string{"foo", "-no-color", "bar", "-no-color"}
	args2 = []string{"foo", "bar"}
	args = m.process(args)
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

	fs := m.extendedFlagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode() != terraform.InputModeStd {
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

	fs := m.extendedFlagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	off := terraform.InputMode(0)
	on := terraform.InputModeStd
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

	fs := m.extendedFlagSet("foo")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("err: %s", err)
	}

	if m.InputMode() > 0 {
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

	env, err := m.Workspace()
	if err != nil {
		t.Fatal(err)
	}

	if env != backend.DefaultStateName {
		t.Fatalf("expected env %q, got env %q", backend.DefaultStateName, env)
	}

	testEnv := "test_env"
	if err := m.SetWorkspace(testEnv); err != nil {
		t.Fatal("error setting env:", err)
	}

	env, _ = m.Workspace()
	if env != testEnv {
		t.Fatalf("expected env %q, got env %q", testEnv, env)
	}

	if err := m.SetWorkspace(backend.DefaultStateName); err != nil {
		t.Fatal("error setting env:", err)
	}

	env, _ = m.Workspace()
	if env != backend.DefaultStateName {
		t.Fatalf("expected env %q, got env %q", backend.DefaultStateName, env)
	}
}

func TestMeta_Workspace_override(t *testing.T) {
	defer func(value string) {
		os.Setenv(WorkspaceNameEnvVar, value)
	}(os.Getenv(WorkspaceNameEnvVar))

	m := new(Meta)

	testCases := map[string]struct {
		workspace string
		err       error
	}{
		"": {
			"default",
			nil,
		},
		"development": {
			"development",
			nil,
		},
		"invalid name": {
			"",
			errInvalidWorkspaceNameEnvVar,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			os.Setenv(WorkspaceNameEnvVar, name)
			workspace, err := m.Workspace()
			if workspace != tc.workspace {
				t.Errorf("Unexpected workspace\n got: %s\nwant: %s\n", workspace, tc.workspace)
			}
			if err != tc.err {
				t.Errorf("Unexpected error\n got: %s\nwant: %s\n", err, tc.err)
			}
		})
	}
}

func TestMeta_Workspace_invalidSelected(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// this is an invalid workspace name
	workspace := "test workspace"

	// create the workspace directories
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, workspace), 0755); err != nil {
		t.Fatal(err)
	}

	// create the workspace file to select it
	if err := os.MkdirAll(DefaultDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(DefaultDataDir, local.DefaultWorkspaceFile), []byte(workspace), 0644); err != nil {
		t.Fatal(err)
	}

	m := new(Meta)

	ws, err := m.Workspace()
	if ws != workspace {
		t.Errorf("Unexpected workspace\n got: %s\nwant: %s\n", ws, workspace)
	}
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
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

	// At one point it was the responsibility of this process function to
	// insert fake additional -var-file options into the command line
	// if the automatic tfvars files were present. This is no longer the
	// responsibility of process (it happens in collectVariableValues instead)
	// but we're still testing with these files in place to verify that
	// they _aren't_ being interpreted by process, since that could otherwise
	// cause them to be added more than once and mess up the precedence order.
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

	tests := []struct {
		GivenArgs    []string
		FilteredArgs []string
		ExtraCheck   func(*testing.T, *Meta)
	}{
		{
			[]string{},
			[]string{},
			func(t *testing.T, m *Meta) {
				if got, want := m.color, true; got != want {
					t.Errorf("wrong m.color value %#v; want %#v", got, want)
				}
				if got, want := m.Color, true; got != want {
					t.Errorf("wrong m.Color value %#v; want %#v", got, want)
				}
			},
		},
		{
			[]string{"-no-color"},
			[]string{},
			func(t *testing.T, m *Meta) {
				if got, want := m.color, false; got != want {
					t.Errorf("wrong m.color value %#v; want %#v", got, want)
				}
				if got, want := m.Color, false; got != want {
					t.Errorf("wrong m.Color value %#v; want %#v", got, want)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s", test.GivenArgs), func(t *testing.T) {
			m := new(Meta)
			m.Color = true // this is the default also for normal use, overridden by -no-color
			args := test.GivenArgs
			args = m.process(args)

			if !cmp.Equal(test.FilteredArgs, args) {
				t.Errorf("wrong filtered arguments\n%s", cmp.Diff(test.FilteredArgs, args))
			}

			if test.ExtraCheck != nil {
				test.ExtraCheck(t, m)
			}
		})
	}
}
