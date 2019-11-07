package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
)

func TestOutput(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
	})

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

func TestOutput_nestedListAndMap(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"key":  cty.StringVal("value"),
					"key2": cty.StringVal("value2"),
				}),
				cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("value"),
				}),
			}),
			false,
		)
	})
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
	expected := "foo = [\n  {\n    \"key\" = \"value\"\n    \"key2\" = \"value2\"\n  },\n  {\n    \"key\" = \"value\"\n  },\n]"
	if actual != expected {
		t.Fatalf("wrong output\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestOutput_json(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
	})

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
	expected := "{\n  \"foo\": {\n    \"sensitive\": false,\n    \"type\": \"string\",\n    \"value\": \"bar\"\n  }\n}"
	if actual != expected {
		t.Fatalf("wrong output\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestOutput_emptyOutputsErr(t *testing.T) {
	originalState := states.NewState()
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
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_jsonEmptyOutputs(t *testing.T) {
	originalState := states.NewState()
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
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
	})
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
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
	})
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
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
		s.SetOutputValue(
			addrs.OutputValue{Name: "name"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("john-doe"),
			false,
		)
	})
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
		t.Fatalf("wrong output\ngot:  %#v\nwant: %#v", output, expectedOutput)
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
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestOutput_noState(t *testing.T) {
	originalState := states.NewState()
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
}

func TestOutput_noVars(t *testing.T) {
	originalState := states.NewState()

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
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestOutput_stateDefault(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
	})

	// Write the state file in a temporary directory with the
	// default filename.
	td := testTempDir(t)
	statePath := filepath.Join(td, DefaultStateFilename)

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = writeStateForTesting(originalState, f)
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
