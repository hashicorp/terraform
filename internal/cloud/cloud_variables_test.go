package cloud

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestParseCloudRunVariables(t *testing.T) {
	t.Run("populates variables from allowed sources", func(t *testing.T) {
		vv := map[string]backend.UnparsedVariableValue{
			"undeclared":            testUnparsedVariableValue{source: terraform.ValueFromCLIArg, value: "0"},
			"declaredFromConfig":    testUnparsedVariableValue{source: terraform.ValueFromConfig, value: "1"},
			"declaredFromNamedFile": testUnparsedVariableValue{source: terraform.ValueFromNamedFile, value: "2"},
			"declaredFromCLIArg":    testUnparsedVariableValue{source: terraform.ValueFromCLIArg, value: "3"},
			"declaredFromEnvVar":    testUnparsedVariableValue{source: terraform.ValueFromEnvVar, value: "4"},
		}

		decls := map[string]*configs.Variable{
			"declaredFromConfig": {
				Name:           "declaredFromConfig",
				Type:           cty.String,
				ConstraintType: cty.String,
				ParsingMode:    configs.VariableParseLiteral,
				DeclRange: hcl.Range{
					Filename: "fake.tf",
					Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 2, Column: 1, Byte: 0},
				},
			},
			"declaredFromNamedFile": {
				Name:           "declaredFromNamedFile",
				Type:           cty.String,
				ConstraintType: cty.String,
				ParsingMode:    configs.VariableParseLiteral,
				DeclRange: hcl.Range{
					Filename: "fake.tf",
					Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 2, Column: 1, Byte: 0},
				},
			},
			"declaredFromCLIArg": {
				Name:           "declaredFromCLIArg",
				Type:           cty.String,
				ConstraintType: cty.String,
				ParsingMode:    configs.VariableParseLiteral,
				DeclRange: hcl.Range{
					Filename: "fake.tf",
					Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 2, Column: 1, Byte: 0},
				},
			},
			"declaredFromEnvVar": {
				Name:           "declaredFromEnvVar",
				Type:           cty.String,
				ConstraintType: cty.String,
				ParsingMode:    configs.VariableParseLiteral,
				DeclRange: hcl.Range{
					Filename: "fake.tf",
					Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 2, Column: 1, Byte: 0},
				},
			},
			"missing": {
				Name:           "missing",
				Type:           cty.String,
				ConstraintType: cty.String,
				Default:        cty.StringVal("2"),
				ParsingMode:    configs.VariableParseLiteral,
				DeclRange: hcl.Range{
					Filename: "fake.tf",
					Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 2, Column: 1, Byte: 0},
				},
			},
		}
		wantVals := make(map[string]string)

		wantVals["declaredFromNamedFile"] = "2"
		wantVals["declaredFromCLIArg"] = "3"
		wantVals["declaredFromEnvVar"] = "4"

		gotVals, diags := ParseCloudRunVariables(vv, decls)
		if diff := cmp.Diff(wantVals, gotVals, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		if got, want := len(diags), 1; got != want {
			t.Fatalf("expected 1 variable error: %v, got %v", diags.Err(), want)
		}

		if got, want := diags[0].Description().Summary, "Value for undeclared variable"; got != want {
			t.Errorf("wrong summary for diagnostic 0\ngot:  %s\nwant: %s", got, want)
		}
	})
}

type testUnparsedVariableValue struct {
	source terraform.ValueSourceType
	value  string
}

func (v testUnparsedVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	return &terraform.InputValue{
		Value:      cty.StringVal(v.value),
		SourceType: v.source,
		SourceRange: tfdiags.SourceRange{
			Filename: "fake.tfvars",
			Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
			End:      tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
		},
	}, nil
}
