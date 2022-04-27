package typeexpr

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestTypeString(t *testing.T) {
	tests := []struct {
		Type cty.Type
		Want string
	}{
		{
			cty.DynamicPseudoType,
			"any",
		},
		{
			cty.String,
			"string",
		},
		{
			cty.Number,
			"number",
		},
		{
			cty.Bool,
			"bool",
		},
		{
			cty.List(cty.Number),
			"list(number)",
		},
		{
			cty.Set(cty.Bool),
			"set(bool)",
		},
		{
			cty.Map(cty.String),
			"map(string)",
		},
		{
			cty.EmptyObject,
			"object({})",
		},
		{
			cty.Object(map[string]cty.Type{"foo": cty.Bool}),
			"object({foo=bool})",
		},
		{
			cty.Object(map[string]cty.Type{"foo": cty.Bool, "bar": cty.String}),
			"object({bar=string,foo=bool})",
		},
		{
			cty.EmptyTuple,
			"tuple([])",
		},
		{
			cty.Tuple([]cty.Type{cty.Bool}),
			"tuple([bool])",
		},
		{
			cty.Tuple([]cty.Type{cty.Bool, cty.String}),
			"tuple([bool,string])",
		},
		{
			cty.List(cty.DynamicPseudoType),
			"list(any)",
		},
		{
			cty.Tuple([]cty.Type{cty.DynamicPseudoType}),
			"tuple([any])",
		},
		{
			cty.Object(map[string]cty.Type{"foo": cty.DynamicPseudoType}),
			"object({foo=any})",
		},
		{
			// We don't expect to find attributes that aren't valid identifiers
			// because we only promise to support types that this package
			// would've created, but we allow this situation during rendering
			// just because it's convenient for applications trying to produce
			// error messages about mismatched types. Note that the quoted
			// attribute name is not actually accepted by our Type and
			// TypeConstraint functions, so this is one situation where the
			// TypeString result cannot be re-parsed by those functions.
			cty.Object(map[string]cty.Type{"foo bar baz": cty.String}),
			`object({"foo bar baz"=string})`,
		},
	}

	for _, test := range tests {
		t.Run(test.Type.GoString(), func(t *testing.T) {
			got := TypeString(test.Type)
			if got != test.Want {
				t.Errorf("wrong result\ntype: %#v\ngot:  %s\nwant: %s", test.Type, got, test.Want)
			}
		})
	}
}
