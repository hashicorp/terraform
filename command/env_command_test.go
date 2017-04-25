package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestEnv_createAndChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	newCmd := &EnvNewCommand{}

	current := newCmd.Env()
	if current != backend.DefaultStateName {
		t.Fatal("current env should be 'default'")
	}

	args := []string{"test"}
	ui := new(cli.MockUi)
	newCmd.Meta = Meta{Ui: ui}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current = newCmd.Env()
	if current != "test" {
		t.Fatalf("current env should be 'test', got %q", current)
	}

	selCmd := &EnvSelectCommand{}
	args = []string{backend.DefaultStateName}
	ui = new(cli.MockUi)
	selCmd.Meta = Meta{Ui: ui}
	if code := selCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current = newCmd.Env()
	if current != backend.DefaultStateName {
		t.Fatal("current env should be 'default'")
	}

}

// Create some environments and test the list output.
// This also ensures we switch to the correct env after each call
func TestEnv_createAndList(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// make sure a vars file doesn't interfere
	err := ioutil.WriteFile(
		DefaultVarsFilename,
		[]byte(`foo = "bar"`),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	newCmd := &EnvNewCommand{}

	envs := []string{"test_a", "test_b", "test_c"}

	// create multiple envs
	for _, env := range envs {
		ui := new(cli.MockUi)
		newCmd.Meta = Meta{Ui: ui}
		if code := newCmd.Run([]string{env}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	listCmd := &EnvListCommand{}
	ui := new(cli.MockUi)
	listCmd.Meta = Meta{Ui: ui}

	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "default\n  test_a\n  test_b\n* test_c"

	if actual != expected {
		t.Fatalf("\nexpcted: %q\nactual:  %q", expected, actual)
	}
}

// Don't allow names that aren't URL safe
func TestEnv_createInvalid(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	newCmd := &EnvNewCommand{}

	envs := []string{"test_a*", "test_b/foo", "../../../test_c", "好_d"}

	// create multiple envs
	for _, env := range envs {
		ui := new(cli.MockUi)
		newCmd.Meta = Meta{Ui: ui}
		if code := newCmd.Run([]string{env}); code == 0 {
			t.Fatalf("expected failure: \n%s", ui.OutputWriter)
		}
	}

	// list envs to make sure none were created
	listCmd := &EnvListCommand{}
	ui := new(cli.MockUi)
	listCmd.Meta = Meta{Ui: ui}

	if code := listCmd.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "* default"

	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}

func TestEnv_createWithState(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// create a non-empty state
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	err := (&state.LocalState{Path: "test.tfstate"}).WriteState(originalState)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{"-state", "test.tfstate", "test"}
	ui := new(cli.MockUi)
	newCmd := &EnvNewCommand{
		Meta: Meta{Ui: ui},
	}
	if code := newCmd.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	newPath := filepath.Join(local.DefaultEnvDir, "test", DefaultStateFilename)
	envState := state.LocalState{Path: newPath}
	err = envState.RefreshState()
	if err != nil {
		t.Fatal(err)
	}

	newState := envState.State()
	if !originalState.Equal(newState) {
		t.Fatalf("states not equal\norig: %s\nnew: %s", originalState, newState)
	}
}

func TestEnv_delete(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// create the env directories
	if err := os.MkdirAll(filepath.Join(local.DefaultEnvDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// create the environment file
	if err := os.MkdirAll(DefaultDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(DefaultDataDir, local.DefaultEnvFile), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	delCmd := &EnvDeleteCommand{
		Meta: Meta{Ui: ui},
	}

	current := delCmd.Env()
	if current != "test" {
		t.Fatal("wrong env:", current)
	}

	// we can't delete out current environment
	args := []string{"test"}
	if code := delCmd.Run(args); code == 0 {
		t.Fatal("expected error deleting current env")
	}

	// change back to default
	if err := delCmd.SetEnv(backend.DefaultStateName); err != nil {
		t.Fatal(err)
	}

	// try the delete again
	ui = new(cli.MockUi)
	delCmd.Meta.Ui = ui
	if code := delCmd.Run(args); code != 0 {
		t.Fatalf("error deleting env: %s", ui.ErrorWriter)
	}

	current = delCmd.Env()
	if current != backend.DefaultStateName {
		t.Fatalf("wrong env: %q", current)
	}
}
func TestEnv_deleteWithState(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// create the env directories
	if err := os.MkdirAll(filepath.Join(local.DefaultEnvDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// create a non-empty state
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	envStatePath := filepath.Join(local.DefaultEnvDir, "test", DefaultStateFilename)
	err := (&state.LocalState{Path: envStatePath}).WriteState(originalState)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	delCmd := &EnvDeleteCommand{
		Meta: Meta{Ui: ui},
	}
	args := []string{"test"}
	if code := delCmd.Run(args); code == 0 {
		t.Fatalf("expected failure without -force.\noutput: %s", ui.OutputWriter)
	}

	ui = new(cli.MockUi)
	delCmd.Meta.Ui = ui

	args = []string{"-force", "test"}
	if code := delCmd.Run(args); code != 0 {
		t.Fatalf("failure: %s", ui.ErrorWriter)
	}

	if _, err := os.Stat(filepath.Join(local.DefaultEnvDir, "test")); !os.IsNotExist(err) {
		t.Fatal("env 'test' still exists!")
	}
}
