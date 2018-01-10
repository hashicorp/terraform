package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestStateRm(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},

					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test we have backups
	backups := testStateBackups(t, filepath.Dir(statePath))
	if len(backups) != 1 {
		t.Fatalf("bad: %#v", backups)
	}
	testStateOutput(t, backups[0], testStateRmOutputOriginal)
}

func TestStateRmNoArgs(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},

					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 1 {
		t.Errorf("wrong exit status %d; want %d", code, 1)
	}

	if msg := ui.ErrorWriter.String(); !strings.Contains(msg, "At least one resource address") {
		t.Errorf("not the error we were looking for:\n%s", msg)
	}

}

func TestStateRm_backupExplicit(t *testing.T) {
	td := tempDir(t)
	defer os.RemoveAll(td)
	backupPath := filepath.Join(td, "backup")

	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},

					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{
		"-backup", backupPath,
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateRmOutputOriginal)
}

func TestStateRm_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestStateRm_needsInit(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-change"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{"foo"}
	if code := c.Run(args); code == 0 {
		t.Fatal("expected error\noutput:", ui.OutputWriter)
	}

	if !strings.Contains(ui.ErrorWriter.String(), "Initialization") {
		t.Fatal("expected initialization error, got:\n", ui.ErrorWriter)
	}
}

func TestStateRm_backendState(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-unchanged"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},

					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	// the local backend state file is "foo"
	statePath := "local-state.tfstate"
	backupPath := "local-state.backup"

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := terraform.WriteState(state, f); err != nil {
		t.Fatal(err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StateRmCommand{
		StateMeta{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			},
		},
	}

	args := []string{
		"-backup", backupPath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test it is correct
	testStateOutput(t, statePath, testStateRmOutput)

	// Test backup
	testStateOutput(t, backupPath, testStateRmOutputOriginal)
}

const testStateRmOutputOriginal = `
test_instance.bar:
  ID = foo
  bar = value
  foo = value
test_instance.foo:
  ID = bar
  bar = value
  foo = value
`

const testStateRmOutput = `
test_instance.bar:
  ID = foo
  bar = value
  foo = value
`
