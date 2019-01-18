package jsonplan

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestStripUnknowns(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello"),
			cty.StringVal("hello"),
		},
		{
			cty.NumberIntVal(0),
			cty.NumberIntVal(0),
		},
		{
			cty.UnknownVal(cty.String),
			cty.NilVal,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.UnknownVal(cty.String),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
			}),
		},
		{
			cty.MapVal(map[string]cty.Value{
				"hello": cty.True,
				"world": cty.UnknownVal(cty.Bool),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"hello": cty.True,
			}),
		},
		{
			cty.SetVal([]cty.Value{
				cty.StringVal("dev"), cty.StringVal("foo"),
				cty.StringVal("stg"), cty.UnknownVal(cty.String),
			}),
			cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo"), cty.StringVal("stg")}),
		},
	}

	for _, test := range tests {
		got := stripUnknowns(test.Input)
		if !reflect.DeepEqual(got, test.Want) {
			t.Fatalf("wrong result! Got: %#v, Want %#v\n", got, test.Want)
		}
	}
}
