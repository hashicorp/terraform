package backend

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestParseVariableValuesUndeclared(t *testing.T) {
	vv := map[string]UnparsedVariableValue{
		// these two shouldn't generate a diag message
		"undeclared0": testUnparsedFileVariableValue("0"),
		"undeclared1": testUnparsedEnvVariableValue("1"),
		// these two should generate 2 diag messages
		"undeclared2": testUnparsedAutoFileVariableValue("2"),
		"undeclared3": testUnparsedConfigVariableValue("3"),
		// add more undeclared values that generate a diag to test suppression
		"undeclared4": testUnparsedConfigVariableValue("4"),
		"undeclared5": testUnparsedConfigVariableValue("5"),
		"undeclared6": testUnparsedConfigVariableValue("6"),
		"undeclared7": testUnparsedConfigVariableValue("7"),
	}
	decls := map[string]*configs.Variable{}

	_, diags := ParseVariableValues(vv, decls)
	for _, diag := range diags {
		t.Logf("%s: %s", diag.Description().Summary, diag.Description().Detail)
	}
	if got, want := len(diags), 4; got != want {
		t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
	}

	const undeclSingular = `Value for undeclared variable`
	const undeclPlural = `Values for undeclared variables`

	if got, want := diags[0].Description().Summary, undeclSingular; got != want {
		t.Errorf("wrong summary for diagnostic 0\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := diags[1].Description().Summary, undeclSingular; got != want {
		t.Errorf("wrong summary for diagnostic 1\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := diags[2].Description().Summary, undeclSingular; got != want {
		t.Errorf("wrong summary for diagnostic 2\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := diags[3].Description().Summary, undeclPlural; got != want {
		t.Errorf("wrong summary for diagnostic 3\ngot:  %s\nwant: %s", got, want)
	}
}

type testUnparsedFileVariableValue string

func (v testUnparsedFileVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	return &terraform.InputValue{
		Value:      cty.StringVal(string(v)),
		SourceType: terraform.ValueFromNamedFile,
		SourceRange: tfdiags.SourceRange{
			Filename: "fake.tfvars",
			Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
			End:      tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
		},
	}, nil
}

type testUnparsedEnvVariableValue string

func (v testUnparsedEnvVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	return &terraform.InputValue{
		Value:      cty.StringVal(string(v)),
		SourceType: terraform.ValueFromEnvVar,
	}, nil
}

type testUnparsedAutoFileVariableValue string

func (v testUnparsedAutoFileVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	return &terraform.InputValue{
		Value:      cty.StringVal(string(v)),
		SourceType: terraform.ValueFromAutoFile,
	}, nil
}

type testUnparsedConfigVariableValue string

func (v testUnparsedConfigVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	return &terraform.InputValue{
		Value:      cty.StringVal(string(v)),
		SourceType: terraform.ValueFromConfig,
	}, nil
}
