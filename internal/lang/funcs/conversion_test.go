package funcs

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/lang/marks"
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
			cty.StringVal("a").Mark("boop"),
			cty.String,
			cty.StringVal("a").Mark("boop"),
			``,
		},
		{
			cty.NullVal(cty.String).Mark("boop"),
			cty.String,
			cty.NullVal(cty.String).Mark("boop"),
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
			cty.StringVal("a").Mark("boop"),
			cty.Bool,
			cty.DynamicVal,
			`cannot convert "a" to bool; only the strings "true" or "false" are allowed`,
		},
		{
			cty.StringVal("a").Mark(marks.Sensitive),
			cty.Bool,
			cty.DynamicVal,
			`cannot convert this sensitive string to bool`,
		},
		{
			cty.StringVal("a"),
			cty.Number,
			cty.DynamicVal,
			`cannot convert "a" to number; given string must be a decimal representation of a number`,
		},
		{
			cty.StringVal("a").Mark("boop"),
			cty.Number,
			cty.DynamicVal,
			`cannot convert "a" to number; given string must be a decimal representation of a number`,
		},
		{
			cty.StringVal("a").Mark(marks.Sensitive),
			cty.Number,
			cty.DynamicVal,
			`cannot convert this sensitive string to number`,
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
			cty.ObjectVal(map[string]cty.Value{"foo": cty.StringVal("hello"), "bar": cty.StringVal("world").Mark("boop")}),
			cty.Map(cty.String),
			cty.MapVal(map[string]cty.Value{"foo": cty.StringVal("hello"), "bar": cty.StringVal("world").Mark("boop")}),
			``,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"foo": cty.StringVal("hello"), "bar": cty.StringVal("world")}).Mark("boop"),
			cty.Map(cty.String),
			cty.MapVal(map[string]cty.Value{"foo": cty.StringVal("hello"), "bar": cty.StringVal("world")}).Mark("boop"),
			``,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world").Mark("boop")}),
			cty.List(cty.String),
			cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world").Mark("boop")}),
			``,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world")}).Mark("boop"),
			cty.List(cty.String),
			cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world")}).Mark("boop"),
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

func TestType(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  string
	}{
		// Primititves
		{
			cty.StringVal("a"),
			"string",
		},
		{
			cty.NumberIntVal(42),
			"number",
		},
		{
			cty.BoolVal(true),
			"bool",
		},
		// Collections
		{
			cty.EmptyObjectVal,
			`object({})`,
		},
		{
			cty.EmptyTupleVal,
			`tuple([])`,
		},
		{
			cty.ListValEmpty(cty.String),
			`list(string)`,
		},
		{
			cty.MapValEmpty(cty.String),
			`map(string)`,
		},
		{
			cty.SetValEmpty(cty.String),
			`set(string)`,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a")}),
			`list(string)`,
		},
		{
			cty.ListVal([]cty.Value{cty.ListVal([]cty.Value{cty.NumberIntVal(42)})}),
			`list(list(number))`,
		},
		{
			cty.ListVal([]cty.Value{cty.MapValEmpty(cty.String)}),
			`list(map(string))`,
		},
		{
			cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			})}),
			"list(\n    object({\n        foo: string,\n    }),\n)",
		},
		// Unknowns and Nulls
		{
			cty.UnknownVal(cty.String),
			"string",
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			"object({\n    foo: string,\n})",
		},
		{ // irrelevant marks do nothing
			cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar").Mark("ignore me"),
			})}),
			"list(\n    object({\n        foo: string,\n    }),\n)",
		},
	}
	for _, test := range tests {
		got, err := Type([]cty.Value{test.Input})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		// The value is marked to help with formatting
		got, _ = got.Unmark()

		if got.AsString() != test.Want {
			t.Errorf("wrong result:\n%s", cmp.Diff(got.AsString(), test.Want))
		}
	}
}
