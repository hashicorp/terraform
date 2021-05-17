package terraform

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/internal/tfdiags"

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
						Filename: "testdata/vars-basic/main.tf",
						Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
						End:      tfdiags.SourcePos{Line: 1, Column: 13, Byte: 12},
					},
				},
				"b": &InputValue{
					Value:      cty.ListValEmpty(cty.String),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "testdata/vars-basic/main.tf",
						Start:    tfdiags.SourcePos{Line: 6, Column: 1, Byte: 55},
						End:      tfdiags.SourcePos{Line: 6, Column: 13, Byte: 67},
					},
				},
				"c": &InputValue{
					Value:      cty.MapValEmpty(cty.String),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "testdata/vars-basic/main.tf",
						Start:    tfdiags.SourcePos{Line: 11, Column: 1, Byte: 113},
						End:      tfdiags.SourcePos{Line: 11, Column: 13, Byte: 125},
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
						Filename: "testdata/vars-basic-bool/main.tf",
						Start:    tfdiags.SourcePos{Line: 4, Column: 1, Byte: 177},
						End:      tfdiags.SourcePos{Line: 4, Column: 13, Byte: 189},
					},
				},
				"b": &InputValue{
					Value:      cty.False,
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "testdata/vars-basic-bool/main.tf",
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

func TestCheckInputVariables(t *testing.T) {
	c := testModule(t, "input-variables")

	t.Run("No variables set", func(t *testing.T) {
		// No variables set
		diags := checkInputVariables(c.Module.Variables, nil)
		if !diags.HasErrors() {
			t.Fatal("check succeeded, but want errors")
		}

		// Required variables set, optional variables unset
		// This is still an error at this layer, since it's the caller's
		// responsibility to have already merged in any default values.
		diags = checkInputVariables(c.Module.Variables, InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCLIArg,
			},
		})
		if !diags.HasErrors() {
			t.Fatal("check succeeded, but want errors")
		}
	})

	t.Run("All variables set", func(t *testing.T) {
		diags := checkInputVariables(c.Module.Variables, InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCLIArg,
			},
			"bar": &InputValue{
				Value:      cty.StringVal("baz"),
				SourceType: ValueFromCLIArg,
			},
			"map": &InputValue{
				Value:      cty.StringVal("baz"), // okay because config has no type constraint
				SourceType: ValueFromCLIArg,
			},
			"object_map": &InputValue{
				Value: cty.MapVal(map[string]cty.Value{
					"uno": cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("baz"),
						"bar": cty.NumberIntVal(2), // type = any
					}),
					"dos": cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("bat"),
						"bar": cty.NumberIntVal(99), // type = any
					}),
				}),
				SourceType: ValueFromCLIArg,
			},
			"object_list": &InputValue{
				Value: cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("baz"),
						"bar": cty.NumberIntVal(2), // type = any
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("bang"),
						"bar": cty.NumberIntVal(42), // type = any
					}),
				}),
				SourceType: ValueFromCLIArg,
			},
		})
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
	})

	t.Run("Invalid Complex Types", func(t *testing.T) {
		diags := checkInputVariables(c.Module.Variables, InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCLIArg,
			},
			"bar": &InputValue{
				Value:      cty.StringVal("baz"),
				SourceType: ValueFromCLIArg,
			},
			"map": &InputValue{
				Value:      cty.StringVal("baz"), // okay because config has no type constraint
				SourceType: ValueFromCLIArg,
			},
			"object_map": &InputValue{
				Value: cty.MapVal(map[string]cty.Value{
					"uno": cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("baz"),
						"bar": cty.NumberIntVal(2), // type = any
					}),
					"dos": cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("bat"),
						"bar": cty.NumberIntVal(99), // type = any
					}),
				}),
				SourceType: ValueFromCLIArg,
			},
			"object_list": &InputValue{
				Value: cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("baz"),
						"bar": cty.NumberIntVal(2), // type = any
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("bang"),
						"bar": cty.StringVal("42"), // type = any, but mismatch with the first list item
					}),
				}),
				SourceType: ValueFromCLIArg,
			},
		})

		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
	})
}
