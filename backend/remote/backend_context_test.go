package remote

import (
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"
)

func TestRemoteStoredVariableValue(t *testing.T) {
	tests := map[string]struct {
		Def       *tfe.Variable
		Want      cty.Value
		WantError string
	}{
		"string literal": {
			&tfe.Variable{
				Key:       "test",
				Value:     "foo",
				HCL:       false,
				Sensitive: false,
			},
			cty.StringVal("foo"),
			``,
		},
		"string HCL": {
			&tfe.Variable{
				Key:       "test",
				Value:     `"foo"`,
				HCL:       true,
				Sensitive: false,
			},
			cty.StringVal("foo"),
			``,
		},
		"list HCL": {
			&tfe.Variable{
				Key:       "test",
				Value:     `[]`,
				HCL:       true,
				Sensitive: false,
			},
			cty.EmptyTupleVal,
			``,
		},
		"null HCL": {
			&tfe.Variable{
				Key:       "test",
				Value:     `null`,
				HCL:       true,
				Sensitive: false,
			},
			cty.NullVal(cty.DynamicPseudoType),
			``,
		},
		"literal sensitive": {
			&tfe.Variable{
				Key:       "test",
				HCL:       false,
				Sensitive: true,
			},
			cty.UnknownVal(cty.String),
			``,
		},
		"HCL sensitive": {
			&tfe.Variable{
				Key:       "test",
				HCL:       true,
				Sensitive: true,
			},
			cty.DynamicVal,
			``,
		},
		"HCL computation": {
			// This (stored expressions containing computation) is not a case
			// we intentionally supported, but it became possible for remote
			// operations in Terraform 0.12 (due to Terraform Cloud/Enterprise
			// just writing the HCL verbatim into generated `.tfvars` files).
			// We support it here for consistency, and we continue to support
			// it in both places for backward-compatibility. In practice,
			// there's little reason to do computation in a stored variable
			// value because references are not supported.
			&tfe.Variable{
				Key:       "test",
				Value:     `[for v in ["a"] : v]`,
				HCL:       true,
				Sensitive: false,
			},
			cty.TupleVal([]cty.Value{cty.StringVal("a")}),
			``,
		},
		"HCL syntax error": {
			&tfe.Variable{
				Key:       "test",
				Value:     `[`,
				HCL:       true,
				Sensitive: false,
			},
			cty.DynamicVal,
			`Invalid expression for var.test: The value of variable "test" is marked in the remote workspace as being specified in HCL syntax, but the given value is not valid HCL. Stored variable values must be valid literal expressions and may not contain references to other variables or calls to functions.`,
		},
		"HCL with references": {
			&tfe.Variable{
				Key:       "test",
				Value:     `foo.bar`,
				HCL:       true,
				Sensitive: false,
			},
			cty.DynamicVal,
			`Invalid expression for var.test: The value of variable "test" is marked in the remote workspace as being specified in HCL syntax, but the given value is not valid HCL. Stored variable values must be valid literal expressions and may not contain references to other variables or calls to functions.`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			v := &remoteStoredVariableValue{
				definition: test.Def,
			}
			// This ParseVariableValue implementation ignores the parsing mode,
			// so we'll just always parse literal here. (The parsing mode is
			// selected by the remote server, not by our local configuration.)
			gotIV, diags := v.ParseVariableValue(configs.VariableParseLiteral)
			if test.WantError != "" {
				if !diags.HasErrors() {
					t.Fatalf("missing expected error\ngot:  <no error>\nwant: %s", test.WantError)
				}
				errStr := diags.Err().Error()
				if errStr != test.WantError {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", errStr, test.WantError)
				}
			} else {
				if diags.HasErrors() {
					t.Fatalf("unexpected error\ngot:  %s\nwant: <no error>", diags.Err().Error())
				}
				got := gotIV.Value
				if !test.Want.RawEquals(got) {
					t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
				}
			}
		})
	}
}
