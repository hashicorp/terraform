package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestOutput(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	if actual != "bar" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestModuleOutput(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
			{
				Path: []string{"root", "my_module"},
				Outputs: map[string]*terraform.OutputState{
					"blah": {
						Value: "tastatur",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-module", "my_module",
		"blah",
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	if actual != "tastatur" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestModuleOutputs(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
			{
				Path: []string{"root", "my_module"},
				Outputs: map[string]*terraform.OutputState{
					"blah": {
						Value: "tastatur",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-module", "my_module",
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	if actual != "blah = tastatur" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestOutput_nestedListAndMap(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: []interface{}{
							map[string]interface{}{
								"key":  "value",
								"key2": "value2",
							},
							map[string]interface{}{
								"key": "value",
							},
						},
						Type: "list",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "foo = [\n    {\n        key = value,\n        key2 = value2\n    },\n    {\n        key = value\n    }\n]"
	if actual != expected {
		t.Fatalf("bad:\n%#v\n%#v", expected, actual)
	}
}

func TestOutput_json(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "{\n    \"foo\": {\n        \"sensitive\": false,\n        \"type\": \"string\",\n        \"value\": \"bar\"\n    }\n}"
	if actual != expected {
		t.Fatalf("bad:\n%#v\n%#v", expected, actual)
	}
}

func TestOutput_emptyOutputsErr(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path:    []string{"root"},
				Outputs: map[string]*terraform.OutputState{},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_jsonEmptyOutputs(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path:    []string{"root"},
				Outputs: map[string]*terraform.OutputState{},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-json",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "{}"
	if actual != expected {
		t.Fatalf("bad:\n%#v\n%#v", expected, actual)
	}
}

func TestMissingModuleOutput(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-module", "not_existing_module",
		"blah",
	}

	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_badVar(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"bar",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_blank(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
					"name": {
						Value: "john-doe",
						Type:  "string",
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"",
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	expectedOutput := "foo = bar\nname = john-doe\n"
	output := ui.OutputWriter.String()
	if output != expectedOutput {
		t.Fatalf("Expected output: %#v\ngiven: %#v", expectedOutput, output)
	}
}

func TestOutput_manyArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestOutput_noArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestOutput_noState(t *testing.T) {
	originalState := &terraform.State{}
	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_noVars(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path:    []string{"root"},
				Outputs: map[string]*terraform.OutputState{},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"bar",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_stateDefault(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": {
						Value: "bar",
						Type:  "string",
					},
				},
			},
		},
	}

	// Write the state file in a temporary directory with the
	// default filename.
	td := testTempDir(t)
	statePath := filepath.Join(td, DefaultStateFilename)

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = terraform.WriteState(originalState, f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Change to that directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(filepath.Dir(statePath)); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &OutputCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	if actual != "bar" {
		t.Fatalf("bad: %#v", actual)
	}
}
