package terraform

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"
)

func TestVariables(t *testing.T) {
	tests := map[string]struct {
		Module   string
		Override map[string]cty.Value
		Want     InputValues
	}{
		"config only": {
			"vars-basic",
			nil,
			InputValues{
				"a": &InputValue{
					Value:      cty.StringVal("foo"),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "test-fixtures/vars-basic/main.tf",
						Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
						End:      tfdiags.SourcePos{Line: 1, Column: 13, Byte: 12},
					},
				},
				"b": &InputValue{
					Value:      cty.ListValEmpty(cty.DynamicPseudoType),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "test-fixtures/vars-basic/main.tf",
						Start:    tfdiags.SourcePos{Line: 6, Column: 1, Byte: 58},
						End:      tfdiags.SourcePos{Line: 6, Column: 13, Byte: 70},
					},
				},
				"c": &InputValue{
					Value:      cty.MapValEmpty(cty.DynamicPseudoType),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "test-fixtures/vars-basic/main.tf",
						Start:    tfdiags.SourcePos{Line: 11, Column: 1, Byte: 111},
						End:      tfdiags.SourcePos{Line: 11, Column: 13, Byte: 123},
					},
				},
			},
		},

		"override": {
			"vars-basic",
			map[string]cty.Value{
				"a": cty.StringVal("bar"),
				"b": cty.ListVal([]cty.Value{
					cty.StringVal("foo"),
					cty.StringVal("bar"),
				}),
				"c": cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
			},
			InputValues{
				"a": &InputValue{
					Value:      cty.StringVal("bar"),
					SourceType: ValueFromCaller,
				},
				"b": &InputValue{
					Value: cty.ListVal([]cty.Value{
						cty.StringVal("foo"),
						cty.StringVal("bar"),
					}),
					SourceType: ValueFromCaller,
				},
				"c": &InputValue{
					Value: cty.MapVal(map[string]cty.Value{
						"foo": cty.StringVal("bar"),
					}),
					SourceType: ValueFromCaller,
				},
			},
		},

		"bools: config only": {
			"vars-basic-bool",
			nil,
			InputValues{
				"a": &InputValue{
					Value:      cty.True,
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "test-fixtures/vars-basic-bool/main.tf",
						Start:    tfdiags.SourcePos{Line: 4, Column: 1, Byte: 177},
						End:      tfdiags.SourcePos{Line: 4, Column: 13, Byte: 189},
					},
				},
				"b": &InputValue{
					Value:      cty.False,
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "test-fixtures/vars-basic-bool/main.tf",
						Start:    tfdiags.SourcePos{Line: 8, Column: 1, Byte: 214},
						End:      tfdiags.SourcePos{Line: 8, Column: 13, Byte: 226},
					},
				},
			},
		},

		"bools: override with string": {
			"vars-basic-bool",
			map[string]cty.Value{
				"a": cty.StringVal("foo"),
				"b": cty.StringVal("bar"),
			},
			InputValues{
				"a": &InputValue{
					Value:      cty.StringVal("foo"),
					SourceType: ValueFromCaller,
				},
				"b": &InputValue{
					Value:      cty.StringVal("bar"),
					SourceType: ValueFromCaller,
				},
			},
		},

		"bools: override with bool": {
			"vars-basic-bool",
			map[string]cty.Value{
				"a": cty.False,
				"b": cty.True,
			},
			InputValues{
				"a": &InputValue{
					Value:      cty.False,
					SourceType: ValueFromCaller,
				},
				"b": &InputValue{
					Value:      cty.True,
					SourceType: ValueFromCaller,
				},
			},
		},
	}

	for name, test := range tests {
		// Wrapped in a func so we can get defers to work
		t.Run(name, func(t *testing.T) {
			m := testModule(t, test.Module)
			fromConfig := DefaultVariableValues(m.Module.Variables)
			overrides := InputValuesFromCaller(test.Override)
			got := fromConfig.Override(overrides)

			if !got.Identical(test.Want) {
				t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(test.Want))
			}
			for _, problem := range deep.Equal(got, test.Want) {
				t.Errorf(problem)
			}
		})
	}
}
