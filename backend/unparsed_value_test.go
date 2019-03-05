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
		"undeclared0": testUnparsedVariableValue("0"),
		"undeclared1": testUnparsedVariableValue("1"),
		"undeclared2": testUnparsedVariableValue("2"),
		"undeclared3": testUnparsedVariableValue("3"),
		"undeclared4": testUnparsedVariableValue("4"),
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

type testUnparsedVariableValue string

func (v testUnparsedVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
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
