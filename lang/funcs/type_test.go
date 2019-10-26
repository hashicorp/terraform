package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestType(t *testing.T) {
	tests := []struct {
		Value cty.Value
		Want  string
	}{
		{
			cty.BoolVal(true),
			"bool",
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("foo"),
				cty.StringVal("bar"),
			}),
			"list",
		},
		{
			cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("foo"),
				"bar": cty.StringVal("bar")}),
			"map",
		},
		{
			cty.NilVal,
			"null",
		},
		{
			cty.NumberIntVal(42),
			"number",
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("foo"),
				"bar": cty.StringVal("bar")}),
			"object",
		},
		{
			cty.SetVal([]cty.Value{
				cty.StringVal("foo"),
				cty.StringVal("bar"),
			}),
			"set",
		},
		{
			cty.StringVal("foo"),
			"string",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
				cty.StringVal("bar"),
			}),
			"tuple",
		},
	}

	// prevValue is used to test negative results, to ensure the is* functions
	// do not just always return true
	prevValue := cty.NumberIntVal(42)

	for _, test := range tests {

		t.Run(fmt.Sprintf("type(%#v)", test.Value), func(t *testing.T) {
			got, err := Type(test.Value)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got.AsString() != test.Want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})

		t.Run(fmt.Sprintf("is%s()", test.Want), func(t *testing.T) {
			f := MakeIsTypeFunc(test.Want)

			//testing for positive results
			got, err := f.Call([]cty.Value{test.Value})

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got.False() {
				t.Error("wrong result\ngot:  false\nwant: true")
			}

			// testing for negative results
			got, err = f.Call([]cty.Value{prevValue})

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got.True() {
				t.Error("wrong result\ngot:  true\nwant: false")
			}

			// preparing prevValue for next test iteration
			prevValue = test.Value
		})
	}
}
