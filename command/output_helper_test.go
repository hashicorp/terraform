package command

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

// testStateConfig provides a mock state for testing.
func testStateConfig() *terraform.State {
	return &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"foo": &terraform.OutputState{
						Value: "bar",
					},
				},
			},
			&terraform.ModuleState{
				Path: []string{"root", "my_module"},
				Outputs: map[string]*terraform.OutputState{
					"blah": &terraform.OutputState{
						Value: "tastatur",
					},
				},
			},
		},
	}
}

// testModuleStateConfig provides a mock ModuleState for testing.
func testModuleStateConfig() *terraform.ModuleState {
	return &terraform.ModuleState{
		Path: []string{"root", "my_module"},
		Outputs: map[string]*terraform.OutputState{
			"foo": &terraform.OutputState{
				Value: "bar",
			},
			"baz": &terraform.OutputState{
				Value: "qux",
			},
			"listoutput": &terraform.OutputState{
				Value: []interface{}{"one", "two"},
			},
			"mapoutput": &terraform.OutputState{
				Value: map[string]interface{}{
					"key": "value",
				},
			},
			"emptylist": &terraform.OutputState{
				Value: []interface{}{},
			},
			"emptymap": &terraform.OutputState{
				Value: map[string]interface{}{},
			},
			"emptystring": &terraform.OutputState{
				Value: "",
			},
		},
	}
}

// testOutputSchemaConfig provides a mock []*config.Output for testing.
func testOutputSchemaConfig() []*config.Output {
	return []*config.Output{
		&config.Output{
			Name:      "foo",
			Sensitive: false,
		},
		&config.Output{
			Name:      "baz",
			Sensitive: true,
		},
		&config.Output{
			Name:      "listoutput",
			Sensitive: false,
		},
		&config.Output{
			Name:      "mapoutput",
			Sensitive: false,
		},
		&config.Output{
			Name:      "emptylist",
			Sensitive: false,
		},
		&config.Output{
			Name:      "emptymap",
			Sensitive: false,
		},
		&config.Output{
			Name:      "emptystring",
			Sensitive: false,
		},
	}
}

const testOutputAsStringExpected = `baz = <sensitive>
emptylist = []
emptymap = {}
emptystring = 
foo = bar
listoutput = [
  one
  two
]
mapoutput = {
  key = value
}`

func TestOutputHelper_parseOutputNameIndex(t *testing.T) {
	name, index, err := parseOutputNameIndex([]string{"foo", "2"})

	if err != nil {
		t.Fatalf("bad: %s", err.Error())
	}

	if name != "foo" {
		t.Fatalf("expected name to be foo, got %s", name)
	}

	if index != "2" {
		t.Fatalf("expected index to be 2, got %s", index)
	}
}

func TestOutputHelper_parseOutputNameIndex_noArgs(t *testing.T) {
	name, index, err := parseOutputNameIndex([]string{})

	if err != nil {
		t.Fatalf("bad: %s", err.Error())
	}

	if name != "" {
		t.Fatalf("expected name to be foo, got %s", name)
	}

	if index != "" {
		t.Fatalf("expected index to be 2, got %s", index)
	}
}

func TestOutputHelper_parseOutputNameIndex_tooManyArgs(t *testing.T) {
	name, index, err := parseOutputNameIndex([]string{"foo", "2", "bar"})

	if err == nil {
		t.Fatalf("bad: %s, %s", name, index)
	}

	expected := `This command expects exactly one argument with the name
of an output variable or no arguments to show all outputs.
`
	if err.Error() != expected {
		t.Fatalf("Expected error to be %s, got %s", expected, err.Error())
	}
}

func TestOutputHelper_moduleFromState(t *testing.T) {
	originalState := testStateConfig()
	mod, err := moduleFromState(originalState, "my_module")

	if err != nil {
		t.Fatalf("bad: %s", err.Error())
	}

	expected := []string{"root", "my_module"}

	if reflect.DeepEqual(mod.Path, expected) != true {
		t.Fatalf("Expected module path to be %v, got %v", expected, mod.Path)
	}
}

func TestOutputHelper_moduleFromState_badModule(t *testing.T) {
	originalState := testStateConfig()
	mod, err := moduleFromState(originalState, "wrong_module")

	if err == nil {
		t.Fatalf("expected error, got %v", mod)
	}

	expected := "The module root.wrong_module could not be found. There is nothing to output."

	if err.Error() != expected {
		t.Fatalf("Expected error to be %s, got %s", expected, err.Error())
	}
}

func TestOutputHelper_moduleFromState_emptyState(t *testing.T) {
	originalState := testStateConfig()
	originalState.Modules[0].Outputs = map[string]*terraform.OutputState{}
	mod, err := moduleFromState(originalState, "")

	if err == nil {
		t.Fatalf("expected error, got %v", mod)
	}

	expected := `The state file has no outputs defined. Define an output
in your configuration with the ` + "`output`" + ` directive and re-run
` + "`terraform apply`" + ` for it to become available.`

	if err.Error() != expected {
		t.Fatalf("Expected error to be %s, got %s", expected, err.Error())
	}
}

func TestOutputHelper_singleOutputAsString(t *testing.T) {
	mod := testModuleStateConfig()

	out, err := singleOutputAsString(mod, "foo", "0")
	if err != nil {
		t.Fatalf("bad: %s", err.Error())
	}

	if out != "bar" {
		t.Fatalf("expected out to be bar, got %s", out)
	}
}

func TestOutputHelper_singleOutputAsString_notFound(t *testing.T) {
	mod := testModuleStateConfig()

	out, err := singleOutputAsString(mod, "nonexistent", "0")
	if err == nil {
		t.Fatalf("expected error, got %v", out)
	}

	expected := `The output variable requested could not be found in the state.
If you recently added this to your configuration, be
sure to run ` + "`terraform apply`," + ` since the state won't be updated
with new output variables until that command is run.`

	if err.Error() != expected {
		t.Fatalf("Expected error to be %s, got %s", expected, err.Error())
	}
}

func TestOutputHelper_singleOutputAsString_list(t *testing.T) {
	mod := testModuleStateConfig()

	out, err := singleOutputAsString(mod, "listoutput", "0")
	if err != nil {
		t.Fatalf("bad: %s", err.Error())
	}

	if out != "one" {
		t.Fatalf("expected out to be one, got %s", out)
	}
}

func TestOutputHelper_singleOutputAsString_listAllEntries(t *testing.T) {
	mod := testModuleStateConfig()

	out, err := singleOutputAsString(mod, "listoutput", "")
	if err != nil {
		t.Fatalf("bad: %s", err.Error())
	}

	if out != "one\ntwo" {
		t.Fatalf("expected out to be one\\ntwo, got %s", out)
	}
}

func TestOutputHelper_singleOutputAsString_listBadIndex(t *testing.T) {
	mod := testModuleStateConfig()

	out, err := singleOutputAsString(mod, "listoutput", "nope")
	if err == nil {
		t.Fatalf("expected error, got %v", out)
	}

	expected := `The index "nope" requested is not valid for the list output
"listoutput" - indices must be numeric, and in the range 0-1`

	if err.Error() != expected {
		t.Fatalf("Expected error to be %s, got %s", expected, err.Error())
	}
}

func TestOutputHelper_singleOutputAsString_listOutOfRange(t *testing.T) {
	mod := testModuleStateConfig()

	out, err := singleOutputAsString(mod, "listoutput", "100")
	if err == nil {
		t.Fatalf("expected error, got %v", out)
	}

	expected := `The index 100 requested is not valid for the list output
"listoutput" - indices must be in the range 0-1`

	if err.Error() != expected {
		t.Fatalf("Expected error to be %s, got %s", expected, err.Error())
	}
}

func TestOutputHelper_allOutputsAsString(t *testing.T) {
	mod := testModuleStateConfig()
	schema := testOutputSchemaConfig()

	text := allOutputsAsString(mod, schema, false)

	expected := testOutputAsStringExpected
	actual := text
	if expected != actual {
		t.Fatalf("Expected output: %q\ngiven: \n%q", expected, actual)
	}
}
