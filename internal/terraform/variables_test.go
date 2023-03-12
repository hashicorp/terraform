package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/zclconf/go-cty/cty"
)

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

// testInputValuesUnset is a helper for constructing InputValues values for
// situations where all of the root module variables are optional and a
// test case intends to just use those default values and not override them
// at all.
//
// In other words, this constructs an InputValues with one entry per given
// input variable declaration where all of them are declared as unset.
func testInputValuesUnset(decls map[string]*configs.Variable) InputValues {
	if len(decls) == 0 {
		return nil
	}

	ret := make(InputValues, len(decls))
	for name := range decls {
		ret[name] = &InputValue{
			Value:      cty.NilVal,
			SourceType: ValueFromUnknown,
		}
	}
	return ret
}
