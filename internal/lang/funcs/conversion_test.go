// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/lang/marks"
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
			// This test case represents evaluating the expression tostring(null)
			// from HCL, since null in HCL is cty.NullVal(cty.DynamicPseudoType).
			// The result in that case should still be null, but a null specifically
			// of type string.
			cty.NullVal(cty.DynamicPseudoType),
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
		{
			cty.UnknownVal(cty.Object(map[string]cty.Type{"foo": cty.String})).Mark(marks.Ephemeral).Mark("boop"),
			cty.Map(cty.String),
			cty.UnknownVal(cty.Map(cty.String)).Mark(marks.Ephemeral).Mark("boop"),
			``,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("world").Mark("beep"),
			}).Mark("boop"),
			cty.Map(cty.String),
			cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("world").Mark("beep"),
			}).Mark("boop"),
			``,
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

func TestEphemeralAsNull(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		// Simple cases
		{
			cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral),
			cty.NullVal(cty.String),
		},
		{
			cty.StringVal("hello"),
			cty.StringVal("hello"),
		},
		{
			// Unknown values stay unknown because an unknown value with
			// an imprecise type constraint is allowed to take on a more
			// precise type in later phases, but known values (even if null)
			// should not. We do know that the final known result definitely
			// won't be ephemeral, though.
			cty.UnknownVal(cty.String).Mark(marks.Ephemeral),
			cty.UnknownVal(cty.String),
		},
		{
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
		},
		{
			// Unknown value refinements should be discarded when unmarking,
			// both because we know our final value is going to be null
			// anyway and because an ephemeral value is not required to
			// have consistent refinements between the plan and apply phases.
			cty.UnknownVal(cty.String).RefineNotNull().Mark(marks.Ephemeral),
			cty.UnknownVal(cty.String),
		},
		{
			// Refinements must be preserved for non-ephemeral values, though.
			cty.UnknownVal(cty.String).RefineNotNull(),
			cty.UnknownVal(cty.String).RefineNotNull(),
		},

		// Should preserve other marks in all cases
		{
			cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral).Mark(marks.Sensitive),
			cty.NullVal(cty.String).Mark(marks.Sensitive),
		},
		{
			cty.StringVal("hello").Mark(marks.Sensitive),
			cty.StringVal("hello").Mark(marks.Sensitive),
		},
		{
			cty.UnknownVal(cty.String).Mark(marks.Ephemeral).Mark(marks.Sensitive),
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
		},
		{
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
		},
		{
			cty.UnknownVal(cty.String).RefineNotNull().Mark(marks.Ephemeral).Mark(marks.Sensitive),
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
		},
		{
			cty.UnknownVal(cty.String).RefineNotNull().Mark(marks.Sensitive),
			cty.UnknownVal(cty.String).RefineNotNull().Mark(marks.Sensitive),
		},

		// Nested ephemeral values
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral),
				cty.StringVal("hello"),
			}),
			cty.ListVal([]cty.Value{
				cty.NullVal(cty.String),
				cty.StringVal("hello"),
			}),
		},
		{
			cty.TupleVal([]cty.Value{
				cty.True,
				cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral),
				cty.StringVal("hello"),
			}),
			cty.TupleVal([]cty.Value{
				cty.True,
				cty.NullVal(cty.String),
				cty.StringVal("hello"),
			}),
		},
		{
			// Sets can't actually preserve individual element marks, so
			// this gets treated as the entire set being ephemeral.
			// (That's true of the input value, despite how it's written here,
			// not just the result value; cty.SetVal does the simplification
			// itself during the construction of the value.)
			cty.SetVal([]cty.Value{
				cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral),
				cty.StringVal("hello"),
			}),
			cty.NullVal(cty.Set(cty.String)),
		},
		{
			cty.MapVal(map[string]cty.Value{
				"addr":  cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral),
				"greet": cty.StringVal("hello"),
			}),
			cty.MapVal(map[string]cty.Value{
				"addr":  cty.NullVal(cty.String),
				"greet": cty.StringVal("hello"),
			}),
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"addr":  cty.StringVal("127.0.0.1:12654").Mark(marks.Ephemeral),
				"greet": cty.StringVal("hello").Mark(marks.Sensitive),
				"happy": cty.True,
				"both":  cty.NumberIntVal(2).WithMarks(cty.NewValueMarks(marks.Sensitive, marks.Ephemeral)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"addr":  cty.NullVal(cty.String),
				"greet": cty.StringVal("hello").Mark(marks.Sensitive),
				"happy": cty.True,
				"both":  cty.NullVal(cty.Number).Mark(marks.Sensitive),
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.Input.GoString(), func(t *testing.T) {
			got, err := EphemeralAsNull(test.Input)
			if err != nil {
				// This function is supposed to be infallible
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
