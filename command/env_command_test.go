package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

	c := &EnvCommand{}

	current, err := currentEnv()
	if err != nil {
		t.Fatal(err)
	}
	if current != backend.DefaultStateName {
		t.Fatal("current env should be 'default'")
	}

	args := []string{"-new", "test"}
	ui := new(cli.MockUi)
	c.Meta = Meta{Ui: ui}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current, err = currentEnv()
	if err != nil {
		t.Fatal(err)
	}
	if current != "test" {
		t.Fatal("current env should be 'test'")
	}

	args = []string{backend.DefaultStateName}
	ui = new(cli.MockUi)
	c.Meta = Meta{Ui: ui}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	current, err = currentEnv()
	if err != nil {
		t.Fatal(err)
	}

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

	c := &EnvCommand{}

	envs := []string{"test_a", "test_b", "test_c"}

	// create multiple envs
	for _, env := range envs {
		args := []string{"-new", env}
		ui := new(cli.MockUi)
		c.Meta = Meta{Ui: ui}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	// now check the listing
	expected := "default\n  test_a\n  test_b\n* test_c"

	ui := new(cli.MockUi)
	c.Meta = Meta{Ui: ui}

	if code := c.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	if actual != expected {
		t.Fatalf("\nexpcted: %q\nactual:  %q", expected, actual)
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

	args := []string{"-new", "test", "-state", "test.tfstate"}
	ui := new(cli.MockUi)
	c := &EnvCommand{
		Meta: Meta{Ui: ui},
	}
	if code := c.Run(args); code != 0 {
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

	current, err := currentEnv()
	if err != nil {
		t.Fatal(err)
	}

	if current != "test" {
		t.Fatal("wrong env:", current)
	}

	ui := new(cli.MockUi)
	c := &EnvCommand{
		Meta: Meta{Ui: ui},
	}
	args := []string{"-delete", "test"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("failure: %s", ui.ErrorWriter)
	}

	current, err = currentEnv()
	if err != nil {
		t.Fatal(err)
	}

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
	c := &EnvCommand{
		Meta: Meta{Ui: ui},
	}
	args := []string{"-delete", "test"}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expected failure without -force.\noutput: %s", ui.OutputWriter)
	}

	ui = new(cli.MockUi)
	c.Meta.Ui = ui

	args = []string{"-delete", "test", "-force"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("failure: %s", ui.ErrorWriter)
	}

	if _, err := os.Stat(filepath.Join(local.DefaultEnvDir, "test")); !os.IsNotExist(err) {
		t.Fatal("env 'test' still exists!")
	}
}

func currentEnv() (string, error) {
	contents, err := ioutil.ReadFile(filepath.Join(DefaultDataDir, local.DefaultEnvFile))
	if os.IsNotExist(err) {
		return backend.DefaultStateName, nil
	}
	if err != nil {
		return "", err
	}

	current := strings.TrimSpace(string(contents))
	if current == "" {
		current = backend.DefaultStateName
	}

	return current, nil
}

func envStatePath() (string, error) {
	currentEnv, err := currentEnv()
	if err != nil {
		return "", err
	}

	if currentEnv == backend.DefaultStateName {
		return DefaultStateFilename, nil
	}

	return filepath.Join(local.DefaultEnvDir, currentEnv, DefaultStateFilename), nil
}

func listEnvs() ([]string, error) {
	entries, err := ioutil.ReadDir(local.DefaultEnvDir)
	// no error if there's no envs configured
	if os.IsNotExist(err) {
		return []string{backend.DefaultStateName}, nil
	}
	if err != nil {
		return nil, err
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, filepath.Base(entry.Name()))
		}
	}

	sort.Strings(envs)

	// always start with "default"
	envs = append([]string{backend.DefaultStateName}, envs...)

	return envs, nil
}
