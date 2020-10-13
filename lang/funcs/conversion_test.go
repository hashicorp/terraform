package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestTo(t *testing.T) {
	tests := []struct {
		Value    cty.Value
		TargetTy cty.Type
		Want     cty.Value
		Err      string
	}{
		{
			cty.StringVal("a"),
			cty.String,
			cty.StringVal("a"),
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.String,
			cty.UnknownVal(cty.String),
			``,
		},
		{
			cty.NullVal(cty.String),
			cty.String,
			cty.NullVal(cty.String),
			``,
		},
		{
			cty.True,
			cty.String,
			cty.StringVal("true"),
			``,
		},
		{
			cty.StringVal("a"),
			cty.Bool,
			cty.DynamicVal,
			`cannot convert "a" to bool; only the strings "true" or "false" are allowed`,
		},
		{
			cty.StringVal("a"),
			cty.Number,
			cty.DynamicVal,
			`cannot convert "a" to number; given string must be a decimal representation of a number`,
		},
		{
			cty.NullVal(cty.String),
			cty.Number,
			cty.NullVal(cty.Number),
			``,
		},
		{
			cty.UnknownVal(cty.Bool),
			cty.String,
			cty.UnknownVal(cty.String),
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.Bool,
			cty.UnknownVal(cty.Bool), // conversion is optimistic
			``,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.True}),
			cty.List(cty.String),
			cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("true")}),
			``,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.True}),
			cty.Set(cty.String),
			cty.SetVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("true")}),
			``,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"foo": cty.StringVal("hello"), "bar": cty.True}),
			cty.Map(cty.String),
			cty.MapVal(map[string]cty.Value{"foo": cty.StringVal("hello"), "bar": cty.StringVal("true")}),
			``,
		},
		{
			cty.EmptyTupleVal,
			cty.String,
			cty.DynamicVal,
			`cannot convert tuple to string`,
		},
		{
			cty.UnknownVal(cty.EmptyTuple),
			cty.String,
			cty.DynamicVal,
			`cannot convert tuple to string`,
		},
		{
			cty.EmptyObjectVal,
			cty.Object(map[string]cty.Type{"foo": cty.String}),
			cty.DynamicVal,
			`incompatible object type for conversion: attribute "foo" is required`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("to %s(%#v)", test.TargetTy.FriendlyNameForConstraint(), test.Value), func(t *testing.T) {
			f := MakeToFunc(test.TargetTy)
			got, err := f.Call([]cty.Value{test.Value})

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.Err; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
