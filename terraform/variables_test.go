package terraform

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestVariables(t *testing.T) {
	cases := map[string]struct {
		Module   string
		Override map[string]cty.Value
		Expected map[string]cty.Value
	}{
		"config only": {
			"vars-basic",
			nil,
			map[string]cty.Value{
				"a": cty.StringVal("foo"),
				"b": cty.ListValEmpty(cty.String),
				"c": cty.MapValEmpty(cty.String),
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
		},

		"bools: config only": {
			"vars-basic-bool",
			nil,
			map[string]cty.Value{
				"a": cty.StringVal("1"),
				"b": cty.StringVal("0"),
			},
		},

		"bools: override with string": {
			"vars-basic-bool",
			map[string]cty.Value{
				"a": cty.StringVal("foo"),
				"b": cty.StringVal("bar"),
			},
			map[string]cty.Value{
				"a": cty.StringVal("foo"),
				"b": cty.StringVal("bar"),
			},
		},

		"bools: override with bool": {
			"vars-basic-bool",
			map[string]cty.Value{
				"a": cty.False,
				"b": cty.True,
			},
			map[string]cty.Value{
				"a": cty.StringVal("0"),
				"b": cty.StringVal("1"),
			},
		},
	}

	for name, tc := range cases {
		// Wrapped in a func so we can get defers to work
		t.Run(name, func(t *testing.T) {
			m := testModule(t, tc.Module)
			fromConfig := DefaultVariableValues(m.Module.Variables)
			overrides := InputValuesFromCaller(tc.Override)
			actual := fromConfig.Override(overrides)

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("%s\n\nexpected: %#v\n\ngot: %#v", name, tc.Expected, actual)
			}
		})
	}
}
