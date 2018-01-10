package diffs

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestUnknownAsNull(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello"),
			cty.StringVal("hello"),
		},
		{
			cty.NullVal(cty.String),
			cty.NullVal(cty.String),
		},
		{
			cty.UnknownVal(cty.String),
			cty.NullVal(cty.String),
		},

		{
			cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.UnknownVal(cty.String)}),
			cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.NullVal(cty.String)}),
		},
		{
			cty.UnknownVal(cty.List(cty.String)),
			cty.NullVal(cty.List(cty.String)),
		},
		{
			cty.NullVal(cty.List(cty.String)),
			cty.NullVal(cty.List(cty.String)),
		},

		{
			cty.SetVal([]cty.Value{cty.StringVal("hello"), cty.UnknownVal(cty.String)}),
			cty.SetVal([]cty.Value{cty.StringVal("hello"), cty.NullVal(cty.String)}),
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello"), cty.UnknownVal(cty.String), cty.UnknownVal(cty.String)}),
			cty.SetVal([]cty.Value{cty.StringVal("hello"), cty.NullVal(cty.String)}), // the two unknowns collide once converted to null
		},
		{
			cty.UnknownVal(cty.Set(cty.String)),
			cty.NullVal(cty.Set(cty.String)),
		},
		{
			cty.NullVal(cty.Set(cty.String)),
			cty.NullVal(cty.Set(cty.String)),
		},

		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.UnknownVal(cty.Bool)}),
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.NullVal(cty.Bool)}),
		},
		{
			cty.UnknownVal(cty.Tuple([]cty.Type{cty.String})),
			cty.NullVal(cty.Tuple([]cty.Type{cty.String})),
		},
		{
			cty.NullVal(cty.Tuple([]cty.Type{cty.String})),
			cty.NullVal(cty.Tuple([]cty.Type{cty.String})),
		},

		{
			cty.MapVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"greetee":  cty.UnknownVal(cty.String),
			}),
			cty.MapVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"greetee":  cty.NullVal(cty.String),
			}),
		},
		{
			cty.UnknownVal(cty.Map(cty.String)),
			cty.NullVal(cty.Map(cty.String)),
		},
		{
			cty.NullVal(cty.Map(cty.String)),
			cty.NullVal(cty.Map(cty.String)),
		},

		{
			cty.ObjectVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"greetee":  cty.UnknownVal(cty.Bool),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"greetee":  cty.NullVal(cty.Bool),
			}),
		},
		{
			cty.UnknownVal(cty.Object(map[string]cty.Type{"foo": cty.String})),
			cty.NullVal(cty.Object(map[string]cty.Type{"foo": cty.String})),
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{"foo": cty.String})),
			cty.NullVal(cty.Object(map[string]cty.Type{"foo": cty.String})),
		},
	}

	for _, test := range tests {
		t.Run(test.Input.GoString(), func(t *testing.T) {
			got := UnknownAsNull(test.Input)
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, got, test.Want)
			}
		})
	}
}
