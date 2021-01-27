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
	if actual != `"bar"` {
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
	expected := strings.TrimSpace(`
foo = tolist([
  tomap({
    "key" = "value"
    "key2" = "value2"
  }),
  tomap({
    "key" = "value"
  }),
])
`)
	if actual != expected {
		t.Fatalf("wrong output\ngot:  %s\nwant: %s", actual, expected)
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

func TestOutput_raw(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "str"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false,
		)
		s.SetOutputValue(
			addrs.OutputValue{Name: "multistr"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar\nbaz"),
			false,
		)
		s.SetOutputValue(
			addrs.OutputValue{Name: "num"}.Absolute(addrs.RootModuleInstance),
			cty.NumberIntVal(2),
			false,
		)
		s.SetOutputValue(
			addrs.OutputValue{Name: "bool"}.Absolute(addrs.RootModuleInstance),
			cty.True,
			false,
		)
		s.SetOutputValue(
			addrs.OutputValue{Name: "obj"}.Absolute(addrs.RootModuleInstance),
			cty.EmptyObjectVal,
			false,
		)
		s.SetOutputValue(
			addrs.OutputValue{Name: "null"}.Absolute(addrs.RootModuleInstance),
			cty.NullVal(cty.String),
			false,
		)
	})

	statePath := testStateFile(t, originalState)

	tests := map[string]struct {
		WantOutput string
		WantErr    bool
	}{
		"str":      {WantOutput: "bar"},
		"multistr": {WantOutput: "bar\nbaz"},
		"num":      {WantOutput: "2"},
		"bool":     {WantOutput: "true"},
		"obj":      {WantErr: true},
		"null":     {WantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var printed string
			ui := cli.NewMockUi()
			c := &OutputCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
				},
				rawPrint: func(s string) {
					printed = s
				},
			}
			args := []string{
				"-state", statePath,
				"-raw",
				name,
			}
			code := c.Run(args)

			if code != 0 {
				if !test.WantErr {
					t.Errorf("unexpected failure\n%s", ui.ErrorWriter.String())
				}
				return
			}

			if test.WantErr {
				t.Fatalf("succeeded, but want error")
			}

			if got, want := printed, test.WantOutput; got != want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}

func TestOutput_emptyOutputs(t *testing.T) {
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
	if got, want := ui.ErrorWriter.String(), "Warning: No outputs found"; !strings.Contains(got, want) {
		t.Fatalf("bad output: expected to contain %q, got:\n%s", want, got)
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

	expectedOutput := "foo = \"bar\"\nname = \"john-doe\"\n"
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
	if actual != `"bar"` {
		t.Fatalf("bad: %#v", actual)
	}
}
