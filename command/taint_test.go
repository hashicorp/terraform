package command

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestTaint(t *testing.T) {
	state := &terraform.State{
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
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testTaintStr)
}

func TestTaint_badwildcard(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo.1": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo1",
						},
					},
					"test_instance.foo.2": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo2",
						},
					},
					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance*",
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testTaintStrBadWildcard)
}

func TestTaint_wildcard1(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo.1": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo1",
						},
					},
					"test_instance.foo.2": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo2",
						},
					},
					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.*",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testTaintStrWildcard1)
}

func TestTaint_wildcard2(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo.1": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo1",
						},
					},
					"test_instance.foo.2": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo2",
						},
					},
					"test_instance.bar": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo.*",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testTaintStrWildcard2)
}

func TestTaint_wildcard3(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo.1": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo1",
						},
					},
					"test_instance.foo.2": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "foo2",
						},
					},
					"test_instance.bar.1": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar1",
						},
					},
				},
			},
		},
	}
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.*.1",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testTaintStrWildcard3)
}

func TestTaint_backup(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := &terraform.State{
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
	path := testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, path+".backup", testTaintDefaultStr)
	testStateOutput(t, path, testTaintStr)
}

func TestTaint_backupDisable(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := &terraform.State{
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
	path := testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-backup", "-",
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(path + ".backup"); err == nil {
		t.Fatal("backup path should not exist")
	}

	testStateOutput(t, path, testTaintStr)
}

func TestTaint_badState(t *testing.T) {
	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", "i-should-not-exist-ever",
		"foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestTaint_defaultState(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := &terraform.State{
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
	path := testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, path, testTaintStr)
}

func TestTaint_missing(t *testing.T) {
	state := &terraform.State{
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
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.bar",
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.OutputWriter.String())
	}
}

func TestTaint_missingAllow(t *testing.T) {
	state := &terraform.State{
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
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-allow-missing",
		"-state", statePath,
		"test_instance.bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestTaint_stateOut(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write the temp state
	state := &terraform.State{
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
	path := testStateFileDefault(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state-out", "foo",
		"test_instance.foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, path, testTaintDefaultStr)
	testStateOutput(t, "foo", testTaintStr)
}

func TestTaint_module(t *testing.T) {
	state := &terraform.State{
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
			&terraform.ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.blah": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "blah",
						},
					},
				},
			},
		},
	}
	statePath := testStateFile(t, state)

	ui := new(cli.MockUi)
	c := &TaintCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-module=child",
		"-state", statePath,
		"test_instance.blah",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testStateOutput(t, statePath, testTaintModuleStr)
}

const testTaintStr = `
test_instance.foo: (tainted)
  ID = bar
`

const testTaintStrBadWildcard = `
test_instance.bar:
  ID = bar
test_instance.foo.1:
  ID = foo1
test_instance.foo.2:
  ID = foo2
`

const testTaintStrWildcard1 = `
test_instance.bar: (tainted)
  ID = bar
test_instance.foo.1: (tainted)
  ID = foo1
test_instance.foo.2: (tainted)
  ID = foo2
`

const testTaintStrWildcard2 = `
test_instance.bar:
  ID = bar
test_instance.foo.1: (tainted)
  ID = foo1
test_instance.foo.2: (tainted)
  ID = foo2
`

const testTaintStrWildcard3 = `
test_instance.bar.1: (tainted)
  ID = bar1
test_instance.foo.1: (tainted)
  ID = foo1
test_instance.foo.2:
  ID = foo2
`

const testTaintDefaultStr = `
test_instance.foo:
  ID = bar
`

const testTaintModuleStr = `
test_instance.foo:
  ID = bar

module.child:
  test_instance.blah: (tainted)
    ID = blah
`
