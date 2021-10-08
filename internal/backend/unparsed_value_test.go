package backend

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestUnparsedValue(t *testing.T) {
	vv := map[string]UnparsedVariableValue{
		"undeclared0": testUnparsedVariableValue("0"),
		"undeclared1": testUnparsedVariableValue("1"),
		"undeclared2": testUnparsedVariableValue("2"),
		"undeclared3": testUnparsedVariableValue("3"),
		"undeclared4": testUnparsedVariableValue("4"),
		"declared1":   testUnparsedVariableValue("5"),
	}
	decls := map[string]*configs.Variable{
		"declared1": {
			Name:           "declared1",
			Type:           cty.String,
			ConstraintType: cty.String,
			ParsingMode:    configs.VariableParseLiteral,
			DeclRange: hcl.Range{
				Filename: "fake.tf",
				Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 2, Column: 1, Byte: 0},
			},
		},
		"missing1": {
			Name:           "missing1",
			Type:           cty.String,
			ConstraintType: cty.String,
			ParsingMode:    configs.VariableParseLiteral,
			DeclRange: hcl.Range{
				Filename: "fake.tf",
				Start:    hcl.Pos{Line: 3, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 3, Column: 1, Byte: 0},
			},
		},
		"missing2": {
			Name:           "missing1",
			Type:           cty.String,
			ConstraintType: cty.String,
			ParsingMode:    configs.VariableParseLiteral,
			Default:        cty.StringVal("default for missing2"),
			DeclRange: hcl.Range{
				Filename: "fake.tf",
				Start:    hcl.Pos{Line: 4, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 4, Column: 1, Byte: 0},
			},
		},
	}

	const undeclSingular = `Value for undeclared variable`
	const undeclPlural = `Values for undeclared variables`

	t.Run("ParseDeclaredVariableValues", func(t *testing.T) {
		gotVals, diags := ParseDeclaredVariableValues(vv, decls)

		if got, want := len(diags), 0; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}

		wantVals := terraform.InputValues{
			"declared1": {
				Value:      cty.StringVal("5"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:      tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
				},
			},
		}

		if diff := cmp.Diff(wantVals, gotVals, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	t.Run("ParseUndeclaredVariableValues", func(t *testing.T) {
		gotVals, diags := ParseUndeclaredVariableValues(vv, decls)

		if got, want := len(diags), 3; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}

		if got, want := diags[0].Description().Summary, undeclSingular; got != want {
			t.Errorf("wrong summary for diagnostic 0\ngot:  %s\nwant: %s", got, want)
		}

		if got, want := diags[1].Description().Summary, undeclSingular; got != want {
			t.Errorf("wrong summary for diagnostic 1\ngot:  %s\nwant: %s", got, want)
		}

		if got, want := diags[2].Description().Summary, undeclPlural; got != want {
			t.Errorf("wrong summary for diagnostic 2\ngot:  %s\nwant: %s", got, want)
		}

		wantVals := terraform.InputValues{
			"undeclared0": {
				Value:      cty.StringVal("0"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1},
					End:      tfdiags.SourcePos{Line: 1, Column: 1},
				},
			},
			"undeclared1": {
				Value:      cty.StringVal("1"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1},
					End:      tfdiags.SourcePos{Line: 1, Column: 1},
				},
			},
			"undeclared2": {
				Value:      cty.StringVal("2"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1},
					End:      tfdiags.SourcePos{Line: 1, Column: 1},
				},
			},
			"undeclared3": {
				Value:      cty.StringVal("3"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1},
					End:      tfdiags.SourcePos{Line: 1, Column: 1},
				},
			},
			"undeclared4": {
				Value:      cty.StringVal("4"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1},
					End:      tfdiags.SourcePos{Line: 1, Column: 1},
				},
			},
		}
		if diff := cmp.Diff(wantVals, gotVals, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	t.Run("ParseVariableValues", func(t *testing.T) {
		gotVals, diags := ParseVariableValues(vv, decls)
		for _, diag := range diags {
			t.Logf("%s: %s", diag.Description().Summary, diag.Description().Detail)
		}
		if got, want := len(diags), 4; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}

		const missingRequired = `No value for required variable`

		if got, want := diags[0].Description().Summary, undeclSingular; got != want {
			t.Errorf("wrong summary for diagnostic 0\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := diags[1].Description().Summary, undeclSingular; got != want {
			t.Errorf("wrong summary for diagnostic 1\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := diags[2].Description().Summary, undeclPlural; got != want {
			t.Errorf("wrong summary for diagnostic 2\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := diags[2].Description().Detail, "3 other variable(s)"; !strings.Contains(got, want) {
			t.Errorf("wrong detail for diagnostic 2\ngot:  %s\nmust contain: %s", got, want)
		}
		if got, want := diags[3].Description().Summary, missingRequired; got != want {
			t.Errorf("wrong summary for diagnostic 3\ngot:  %s\nwant: %s", got, want)
		}

		wantVals := terraform.InputValues{
			"declared1": {
				Value:      cty.StringVal("5"),
				SourceType: terraform.ValueFromNamedFile,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tfvars",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:      tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
				},
			},
			"missing1": {
				Value:      cty.DynamicVal,
				SourceType: terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tf",
					Start:    tfdiags.SourcePos{Line: 3, Column: 1, Byte: 0},
					End:      tfdiags.SourcePos{Line: 3, Column: 1, Byte: 0},
				},
			},
			"missing2": {
				Value:      cty.StringVal("default for missing2"),
				SourceType: terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRange{
					Filename: "fake.tf",
					Start:    tfdiags.SourcePos{Line: 4, Column: 1, Byte: 0},
					End:      tfdiags.SourcePos{Line: 4, Column: 1, Byte: 0},
				},
			},
		}
		if diff := cmp.Diff(wantVals, gotVals, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
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
