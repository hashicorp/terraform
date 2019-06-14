package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestElement(t *testing.T) {
	tests := []struct {
		List  cty.Value
		Index cty.Value
		Want  cty.Value
	}{
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			cty.NumberIntVal(0),
			cty.StringVal("hello"),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			cty.NumberIntVal(1),
			cty.StringVal("hello"),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.NumberIntVal(0),
			cty.StringVal("hello"),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.NumberIntVal(1),
			cty.StringVal("bonjour"),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.NumberIntVal(2),
			cty.StringVal("hello"),
		},

		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			cty.NumberIntVal(0),
			cty.StringVal("hello"),
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			cty.NumberIntVal(1),
			cty.StringVal("hello"),
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.NumberIntVal(0),
			cty.StringVal("hello"),
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.NumberIntVal(1),
			cty.StringVal("bonjour"),
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.NumberIntVal(2),
			cty.StringVal("hello"),
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("bonjour"),
			}),
			cty.UnknownVal(cty.Number),
			cty.DynamicVal,
		},
		{
			cty.UnknownVal(cty.Tuple([]cty.Type{cty.String, cty.Bool})),
			cty.NumberIntVal(1),
			cty.UnknownVal(cty.Bool),
		},
		{
			cty.UnknownVal(cty.Tuple([]cty.Type{cty.String, cty.String})),
			cty.UnknownVal(cty.Number),
			cty.DynamicVal,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Element(%#v, %#v)", test.List, test.Index), func(t *testing.T) {
			got, err := Element(test.List, test.Index)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}

}

func TestLength(t *testing.T) {
	tests := []struct {
		Value cty.Value
		Want  cty.Value
	}{
		{
			cty.ListValEmpty(cty.Number),
			cty.NumberIntVal(0),
		},
		{
			cty.ListVal([]cty.Value{cty.True}),
			cty.NumberIntVal(1),
		},
		{
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.Bool)}),
			cty.NumberIntVal(1),
		},
		{
			cty.SetValEmpty(cty.Number),
			cty.NumberIntVal(0),
		},
		{
			cty.SetVal([]cty.Value{cty.True}),
			cty.NumberIntVal(1),
		},
		{
			cty.MapValEmpty(cty.Bool),
			cty.NumberIntVal(0),
		},
		{
			cty.MapVal(map[string]cty.Value{"hello": cty.True}),
			cty.NumberIntVal(1),
		},
		{
			cty.EmptyTupleVal,
			cty.NumberIntVal(0),
		},
		{
			cty.UnknownVal(cty.EmptyTuple),
			cty.NumberIntVal(0),
		},
		{
			cty.TupleVal([]cty.Value{cty.True}),
			cty.NumberIntVal(1),
		},
		{
			cty.EmptyObjectVal,
			cty.NumberIntVal(0),
		},
		{
			cty.UnknownVal(cty.EmptyObject),
			cty.NumberIntVal(0),
		},
		{
			cty.ObjectVal(map[string]cty.Value{"true": cty.True}),
			cty.NumberIntVal(1),
		},
		{
			cty.UnknownVal(cty.List(cty.Bool)),
			cty.UnknownVal(cty.Number),
		},
		{
			cty.DynamicVal,
			cty.UnknownVal(cty.Number),
		},
		{
			cty.StringVal("hello"),
			cty.NumberIntVal(5),
		},
		{
			cty.StringVal(""),
			cty.NumberIntVal(0),
		},
		{
			cty.StringVal("1"),
			cty.NumberIntVal(1),
		},
		{
			cty.StringVal("Живой Журнал"),
			cty.NumberIntVal(12),
		},
		{
			// note that the dieresis here is intentionally a combining
			// ligature.
			cty.StringVal("noël"),
			cty.NumberIntVal(4),
		},
		{
			// The Es in this string has three combining acute accents.
			// This tests something that NFC-normalization cannot collapse
			// into a single precombined codepoint, since otherwise we might
			// be cheating and relying on the single-codepoint forms.
			cty.StringVal("wé́́é́́é́́!"),
			cty.NumberIntVal(5),
		},
		{
			// Go's normalization forms don't handle this ligature, so we
			// will produce the wrong result but this is now a compatibility
			// constraint and so we'll test it.
			cty.StringVal("baﬄe"),
			cty.NumberIntVal(4),
		},
		{
			cty.StringVal("😸😾"),
			cty.NumberIntVal(2),
		},
		{
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.Number),
		},
		{
			cty.DynamicVal,
			cty.UnknownVal(cty.Number),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Length(%#v)", test.Value), func(t *testing.T) {
			got, err := Length(test.Value)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{cty.StringVal("first"), cty.StringVal("second"), cty.StringVal("third")},
			cty.StringVal("first"),
			false,
		},
		{
			[]cty.Value{cty.StringVal(""), cty.StringVal("second"), cty.StringVal("third")},
			cty.StringVal("second"),
			false,
		},
		{
			[]cty.Value{cty.StringVal(""), cty.StringVal("")},
			cty.NilVal,
			true,
		},
		{
			[]cty.Value{cty.True},
			cty.True,
			false,
		},
		{
			[]cty.Value{cty.NullVal(cty.Bool), cty.True},
			cty.True,
			false,
		},
		{
			[]cty.Value{cty.NullVal(cty.Bool), cty.False},
			cty.False,
			false,
		},
		{
			[]cty.Value{cty.NullVal(cty.Bool), cty.False, cty.StringVal("hello")},
			cty.StringVal("false"),
			false,
		},
		{
			[]cty.Value{cty.True, cty.UnknownVal(cty.Bool)},
			cty.True,
			false,
		},
		{
			[]cty.Value{cty.UnknownVal(cty.Bool), cty.True},
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			[]cty.Value{cty.UnknownVal(cty.Bool), cty.StringVal("hello")},
			cty.UnknownVal(cty.String),
			false,
		},
		{
			[]cty.Value{cty.DynamicVal, cty.True},
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			[]cty.Value{cty.DynamicVal},
			cty.DynamicVal,
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Coalesce(%#v...)", test.Values), func(t *testing.T) {
			got, err := Coalesce(test.Values...)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestCoalesceList(t *testing.T) {
	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("first"), cty.StringVal("second"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("first"), cty.StringVal("second"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.ListValEmpty(cty.String),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("third"), cty.StringVal("fourth"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.ListValEmpty(cty.Number),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
			false,
		},
		{ // lists with mixed types
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("first"), cty.StringVal("second"),
				}),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("first"), cty.StringVal("second"),
			}),
			false,
		},
		{ // lists with mixed types
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("first"), cty.StringVal("second"),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1), cty.NumberIntVal(2),
			}),
			false,
		},
		{ // list with unknown values
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("first"), cty.StringVal("second"),
				}),
				cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("first"), cty.StringVal("second"),
			}),
			false,
		},
		{ // list with unknown values
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.String),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.MapValEmpty(cty.DynamicPseudoType),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.NilVal,
			true,
		},
		{ // unknown list
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
				cty.UnknownVal(cty.List(cty.String)),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("third"), cty.StringVal("fourth"),
			}),
			false,
		},
		{ // unknown list
			[]cty.Value{
				cty.ListValEmpty(cty.String),
				cty.UnknownVal(cty.List(cty.String)),
			},
			cty.DynamicVal,
			false,
		},
		{ // unknown list
			[]cty.Value{
				cty.UnknownVal(cty.List(cty.String)),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.DynamicVal,
			false,
		},
		{ // unknown tuple
			[]cty.Value{
				cty.UnknownVal(cty.Tuple([]cty.Type{cty.String})),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.DynamicVal,
			false,
		},
		{ // empty tuple
			[]cty.Value{
				cty.EmptyTupleVal,
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("third"), cty.StringVal("fourth"),
			}),
			false,
		},
		{ // tuple value
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("a"),
					cty.NumberIntVal(2),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.NumberIntVal(2),
			}),
			false,
		},
		{ // reject set value
			[]cty.Value{
				cty.SetVal([]cty.Value{
					cty.StringVal("a"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("third"), cty.StringVal("fourth"),
				}),
			},
			cty.NilVal,
			true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-coalescelist(%#v)", i, test.Values), func(t *testing.T) {
			got, err := CoalesceList(test.Values...)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestCompact(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("test"),
				cty.StringVal(""),
				cty.StringVal("test"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("test"),
				cty.StringVal("test"),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal(""),
				cty.StringVal(""),
				cty.StringVal(""),
			}),
			cty.ListValEmpty(cty.String),
			false,
		},
		{
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("test"),
				cty.StringVal("test"),
				cty.StringVal(""),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("test"),
				cty.StringVal("test"),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("test"),
				cty.UnknownVal(cty.String),
				cty.StringVal(""),
			}),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{ // errors on list of lists
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("test"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal(""),
				}),
			}),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("compact(%#v)", test.List), func(t *testing.T) {
			got, err := Compact(test.List)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestContains(t *testing.T) {
	listOfStrings := cty.ListVal([]cty.Value{
		cty.StringVal("the"),
		cty.StringVal("quick"),
		cty.StringVal("brown"),
		cty.StringVal("fox"),
	})
	listOfInts := cty.ListVal([]cty.Value{
		cty.NumberIntVal(1),
		cty.NumberIntVal(2),
		cty.NumberIntVal(3),
		cty.NumberIntVal(4),
	})
	listWithUnknown := cty.ListVal([]cty.Value{
		cty.StringVal("the"),
		cty.StringVal("quick"),
		cty.StringVal("brown"),
		cty.UnknownVal(cty.String),
	})

	tests := []struct {
		List  cty.Value
		Value cty.Value
		Want  cty.Value
		Err   bool
	}{
		{
			listOfStrings,
			cty.StringVal("the"),
			cty.BoolVal(true),
			false,
		},
		{
			listWithUnknown,
			cty.StringVal("the"),
			cty.BoolVal(true),
			false,
		},
		{
			listOfStrings,
			cty.StringVal("penguin"),
			cty.BoolVal(false),
			false,
		},
		{
			listOfInts,
			cty.NumberIntVal(1),
			cty.BoolVal(true),
			false,
		},
		{
			listOfInts,
			cty.NumberIntVal(42),
			cty.BoolVal(false),
			false,
		},
		{ // And now we mix and match
			listOfInts,
			cty.StringVal("1"),
			cty.BoolVal(false),
			false,
		},
		{ // Check a list with an unknown value
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.String),
				cty.StringVal("quick"),
				cty.StringVal("brown"),
				cty.StringVal("fox"),
			}),
			cty.StringVal("quick"),
			cty.BoolVal(true),
			false,
		},
		{ // set val
			cty.SetVal([]cty.Value{
				cty.StringVal("quick"),
				cty.StringVal("brown"),
				cty.StringVal("fox"),
			}),
			cty.StringVal("quick"),
			cty.BoolVal(true),
			false,
		},
		{ // tuple val
			cty.TupleVal([]cty.Value{
				cty.StringVal("quick"),
				cty.StringVal("brown"),
				cty.NumberIntVal(3),
			}),
			cty.NumberIntVal(3),
			cty.BoolVal(true),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("contains(%#v, %#v)", test.List, test.Value), func(t *testing.T) {
			got, err := Contains(test.List, test.Value)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestIndex(t *testing.T) {
	tests := []struct {
		List  cty.Value
		Value cty.Value
		Want  cty.Value
		Err   bool
	}{
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.StringVal("a"),
			cty.NumberIntVal(0),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.UnknownVal(cty.String),
			}),
			cty.StringVal("a"),
			cty.NumberIntVal(0),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.StringVal("b"),
			cty.NumberIntVal(1),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.StringVal("z"),
			cty.NilVal,
			true,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("1"),
				cty.StringVal("2"),
				cty.StringVal("3"),
			}),
			cty.NumberIntVal(1),
			cty.NumberIntVal(0),
			true,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NumberIntVal(2),
			cty.NumberIntVal(1),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NumberIntVal(4),
			cty.NilVal,
			true,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.StringVal("1"),
			cty.NumberIntVal(0),
			true,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NumberIntVal(1),
			cty.NumberIntVal(0),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("index(%#v, %#v)", test.List, test.Value), func(t *testing.T) {
			got, err := Index(test.List, test.Value)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestDistinct(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("a"),
				cty.StringVal("b"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
			}),
			false,
		},
		{
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("a"),
				cty.UnknownVal(cty.String),
			}),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
				cty.StringVal("d"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
				cty.StringVal("d"),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
			}),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(3),
					cty.NumberIntVal(4),
				}),
			}),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(3),
					cty.NumberIntVal(4),
				}),
			}),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("distinct(%#v)", test.List), func(t *testing.T) {
			got, err := Distinct(test.List)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestChunklist(t *testing.T) {
	tests := []struct {
		List cty.Value
		Size cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.NumberIntVal(1),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("b"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("c"),
				}),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.NumberIntVal(-1),
			cty.NilVal,
			true,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.NumberIntVal(0),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
			}),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.UnknownVal(cty.String),
			}),
			cty.NumberIntVal(1),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("b"),
				}),
				cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
				}),
			}),
			false,
		},
		{
			cty.UnknownVal(cty.List(cty.String)),
			cty.NumberIntVal(1),
			cty.UnknownVal(cty.List(cty.List(cty.String))),
			false,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-chunklist(%#v, %#v)", i, test.List, test.Size), func(t *testing.T) {
			got, err := Chunklist(test.List, test.Size)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestFlatten(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("c"),
					cty.StringVal("d"),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
				cty.StringVal("d"),
			}),
			false,
		},
		// handle single elements as arguments
		{
			cty.TupleVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.StringVal("c"),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}), false,
		},
		// handle single elements and mixed primitive types as arguments
		{
			cty.TupleVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.StringVal("c"),
				cty.TupleVal([]cty.Value{
					cty.StringVal("x"),
					cty.NumberIntVal(1),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
				cty.StringVal("x"),
				cty.NumberIntVal(1),
			}),
			false,
		},
		// Primitive unknowns should still be flattened to a tuple
		{
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
					cty.StringVal("d"),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.UnknownVal(cty.String),
				cty.StringVal("d"),
			}), false,
		},
		// An unknown series should return an unknown dynamic value
		{
			cty.TupleVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.TupleVal([]cty.Value{
					cty.UnknownVal(cty.List(cty.String)),
					cty.StringVal("d"),
				}),
			}),
			cty.UnknownVal(cty.DynamicPseudoType), false,
		},
		{
			cty.ListValEmpty(cty.String),
			cty.EmptyTupleVal,
			false,
		},
		{
			cty.SetVal([]cty.Value{
				cty.SetVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.SetVal([]cty.Value{
					cty.StringVal("c"),
					cty.StringVal("d"),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
				cty.StringVal("d"),
			}),
			false,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.SetVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("c"),
					cty.StringVal("d"),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
				cty.StringVal("d"),
			}),
			false,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-flatten(%#v)", i, test.List), func(t *testing.T) {
			got, err := Flatten(test.List)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestKeys(t *testing.T) {
	tests := []struct {
		Map  cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.MapVal(map[string]cty.Value{
				"hello":   cty.NumberIntVal(1),
				"goodbye": cty.NumberIntVal(42),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("goodbye"),
				cty.StringVal("hello"),
			}),
			false,
		},
		{ // same as above, but an object type
			cty.ObjectVal(map[string]cty.Value{
				"hello":   cty.NumberIntVal(1),
				"goodbye": cty.StringVal("adieu"),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("goodbye"),
				cty.StringVal("hello"),
			}),
			false,
		},
		{ // for an unknown object we can still return the keys, since they are part of the type
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"hello":   cty.Number,
				"goodbye": cty.String,
			})),
			cty.TupleVal([]cty.Value{
				cty.StringVal("goodbye"),
				cty.StringVal("hello"),
			}),
			false,
		},
		{ // an empty object has no keys
			cty.EmptyObjectVal,
			cty.EmptyTupleVal,
			false,
		},
		{ // an empty map has no keys, but the result should still be properly typed
			cty.MapValEmpty(cty.Number),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // Unknown map has unknown keys
			cty.UnknownVal(cty.Map(cty.String)),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{ // Not a map at all, so invalid
			cty.StringVal("foo"),
			cty.NilVal,
			true,
		},
		{ // Can't get keys from a null object
			cty.NullVal(cty.Object(map[string]cty.Type{
				"hello":   cty.Number,
				"goodbye": cty.String,
			})),
			cty.NilVal,
			true,
		},
		{ // Can't get keys from a null map
			cty.NullVal(cty.Map(cty.Number)),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("keys(%#v)", test.Map), func(t *testing.T) {
			got, err := Keys(test.Map)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestList(t *testing.T) {
	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{
				cty.NilVal,
			},
			cty.NilVal,
			true,
		},
		{
			[]cty.Value{
				cty.StringVal("Hello"),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.StringVal("Hello"),
				cty.StringVal("World"),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
				cty.StringVal("World"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.StringVal("Hello"),
				cty.NumberIntVal(42),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
				cty.StringVal("42"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.StringVal("Hello"),
				cty.UnknownVal(cty.String),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
				cty.UnknownVal(cty.String),
			}),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("list(%#v)", test.Values), func(t *testing.T) {
			got, err := List(test.Values...)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestLookup(t *testing.T) {
	simpleMap := cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})
	intsMap := cty.MapVal(map[string]cty.Value{
		"foo": cty.NumberIntVal(42),
	})
	mapOfLists := cty.MapVal(map[string]cty.Value{
		"foo": cty.ListVal([]cty.Value{
			cty.StringVal("bar"),
			cty.StringVal("baz"),
		}),
	})
	mapWithUnknowns := cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
		"baz": cty.UnknownVal(cty.String),
	})
	mapWithObjects := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
		"baz": cty.NumberIntVal(42),
	})

	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{
				simpleMap,
				cty.StringVal("foo"),
			},
			cty.StringVal("bar"),
			false,
		},
		{
			[]cty.Value{
				mapWithObjects,
				cty.StringVal("foo"),
			},
			cty.StringVal("bar"),
			false,
		},
		{
			[]cty.Value{
				intsMap,
				cty.StringVal("foo"),
			},
			cty.NumberIntVal(42),
			false,
		},
		{ // Invalid key
			[]cty.Value{
				simpleMap,
				cty.StringVal("bar"),
			},
			cty.NilVal,
			true,
		},
		{ // Invalid key
			[]cty.Value{
				mapWithObjects,
				cty.StringVal("bar"),
			},
			cty.NilVal,
			true,
		},
		{ // Supplied default with valid key
			[]cty.Value{
				simpleMap,
				cty.StringVal("foo"),
				cty.StringVal(""),
			},
			cty.StringVal("bar"),
			false,
		},
		{ // Supplied default with valid (int) key
			[]cty.Value{
				simpleMap,
				cty.StringVal("foo"),
				cty.NumberIntVal(-1),
			},
			cty.StringVal("bar"),
			false,
		},
		{ // Supplied default with valid key
			[]cty.Value{
				mapWithObjects,
				cty.StringVal("foobar"),
				cty.StringVal(""),
			},
			cty.StringVal(""),
			false,
		},
		{ // Supplied default with invalid key
			[]cty.Value{
				simpleMap,
				cty.StringVal("baz"),
				cty.StringVal(""),
			},
			cty.StringVal(""),
			false,
		},
		{ // Supplied non-empty default with invalid key
			[]cty.Value{
				simpleMap,
				cty.StringVal("bar"),
				cty.StringVal("xyz"),
			},
			cty.StringVal("xyz"),
			false,
		},
		{ // too many args
			[]cty.Value{
				simpleMap,
				cty.StringVal("foo"),
				cty.StringVal("bar"),
				cty.StringVal("baz"),
			},
			cty.NilVal,
			true,
		},
		{ // cannot search a map of lists
			[]cty.Value{
				mapOfLists,
				cty.StringVal("baz"),
			},
			cty.NilVal,
			true,
		},
		{
			[]cty.Value{
				mapWithUnknowns,
				cty.StringVal("baz"),
			},
			cty.UnknownVal(cty.String),
			false,
		},
		{
			[]cty.Value{
				simpleMap,
				cty.UnknownVal(cty.String),
			},
			cty.UnknownVal(cty.String),
			false,
		},
		{
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("a"),
					"bar": cty.StringVal("b"),
				}),
				cty.UnknownVal(cty.String),
			},
			cty.DynamicVal, // if the key is unknown then we don't know which object attribute and thus can't know the type
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("lookup(%#v)", test.Values), func(t *testing.T) {
			got, err := Lookup(test.Values...)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestMap(t *testing.T) {
	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
			},
			cty.MapVal(map[string]cty.Value{
				"hello": cty.StringVal("world"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.StringVal("hello"),
				cty.UnknownVal(cty.String),
			},
			cty.UnknownVal(cty.Map(cty.String)),
			false,
		},
		{
			[]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
				cty.StringVal("what's"),
				cty.StringVal("up"),
			},
			cty.MapVal(map[string]cty.Value{
				"hello":  cty.StringVal("world"),
				"what's": cty.StringVal("up"),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.StringVal("hello"),
				cty.NumberIntVal(1),
				cty.StringVal("goodbye"),
				cty.NumberIntVal(42),
			},
			cty.MapVal(map[string]cty.Value{
				"hello":   cty.NumberIntVal(1),
				"goodbye": cty.NumberIntVal(42),
			}),
			false,
		},
		{ // convert numbers to strings
			[]cty.Value{
				cty.StringVal("hello"),
				cty.NumberIntVal(1),
				cty.StringVal("goodbye"),
				cty.StringVal("42"),
			},
			cty.MapVal(map[string]cty.Value{
				"hello":   cty.StringVal("1"),
				"goodbye": cty.StringVal("42"),
			}),
			false,
		},
		{ // map of lists is okay
			[]cty.Value{
				cty.StringVal("hello"),
				cty.ListVal([]cty.Value{
					cty.StringVal("world"),
				}),
				cty.StringVal("what's"),
				cty.ListVal([]cty.Value{
					cty.StringVal("up"),
				}),
			},
			cty.MapVal(map[string]cty.Value{
				"hello":  cty.ListVal([]cty.Value{cty.StringVal("world")}),
				"what's": cty.ListVal([]cty.Value{cty.StringVal("up")}),
			}),
			false,
		},
		{ // map of maps is okay
			[]cty.Value{
				cty.StringVal("hello"),
				cty.MapVal(map[string]cty.Value{
					"there": cty.StringVal("world"),
				}),
				cty.StringVal("what's"),
				cty.MapVal(map[string]cty.Value{
					"really": cty.StringVal("up"),
				}),
			},
			cty.MapVal(map[string]cty.Value{
				"hello": cty.MapVal(map[string]cty.Value{
					"there": cty.StringVal("world"),
				}),
				"what's": cty.MapVal(map[string]cty.Value{
					"really": cty.StringVal("up"),
				}),
			}),
			false,
		},
		{ // single argument returns an error
			[]cty.Value{
				cty.StringVal("hello"),
			},
			cty.NilVal,
			true,
		},
		{ // duplicate keys returns an error
			[]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
				cty.StringVal("hello"),
				cty.StringVal("universe"),
			},
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("map(%#v)", test.Values), func(t *testing.T) {
			got, err := Map(test.Values...)
			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestMatchkeys(t *testing.T) {
	tests := []struct {
		Keys      cty.Value
		Values    cty.Value
		Searchset cty.Value
		Want      cty.Value
		Err       bool
	}{
		{ // normal usage
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
				cty.StringVal("ref2"),
				cty.StringVal("ref3"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			false,
		},
		{ // normal usage 2, check the order
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
				cty.StringVal("ref2"),
				cty.StringVal("ref3"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref2"),
				cty.StringVal("ref1"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
			}),
			false,
		},
		{ // no matches
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
				cty.StringVal("ref2"),
				cty.StringVal("ref3"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref4"),
			}),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // no matches 2
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
				cty.StringVal("ref2"),
				cty.StringVal("ref3"),
			}),
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // zero case
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			cty.ListVal([]cty.Value{cty.StringVal("nope")}),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // complex values
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("a"),
				}),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("a"),
				}),
			}),
			false,
		},
		{ // unknowns
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.UnknownVal(cty.String),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
				cty.StringVal("ref2"),
				cty.UnknownVal(cty.String),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("ref1"),
			}),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{ // different types that can be unified
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // complex values: values is a different type from keys and searchset
			cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("baz"),
				}),
				cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("beep"),
				}),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("c"),
			}),
			cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("beep"),
				}),
			}),
			false,
		},
		// errors
		{ // different types
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
				}),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.NilVal,
			true,
		},
		{ // lists of different length
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
			}),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("matchkeys(%#v, %#v, %#v)", test.Keys, test.Values, test.Searchset), func(t *testing.T) {
			got, err := Matchkeys(test.Keys, test.Values, test.Searchset)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestMerge(t *testing.T) {
	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
				"c": cty.StringVal("d"),
			}),
			false,
		},
		{ // handle unknowns
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.UnknownVal(cty.String),
				}),
				cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
			},
			cty.DynamicVal,
			false,
		},
		{ // merge with conflicts is ok, last in wins
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
					"c": cty.StringVal("d"),
				}),
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("x"),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("x"),
				"c": cty.StringVal("d"),
			}),
			false,
		},
		{ // only accept maps
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
					"c": cty.StringVal("d"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("x"),
				}),
			},
			cty.NilVal,
			true,
		},

		{ // argument error, for a null type
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				cty.NullVal(cty.String),
			},
			cty.NilVal,
			true,
		},
		{ // merge maps of maps
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.MapVal(map[string]cty.Value{
						"b": cty.StringVal("c"),
					}),
				}),
				cty.MapVal(map[string]cty.Value{
					"d": cty.MapVal(map[string]cty.Value{
						"e": cty.StringVal("f"),
					}),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"b": cty.StringVal("c"),
				}),
				"d": cty.MapVal(map[string]cty.Value{
					"e": cty.StringVal("f"),
				}),
			}),
			false,
		},
		{ // map of lists
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.ListVal([]cty.Value{
						cty.StringVal("b"),
						cty.StringVal("c"),
					}),
				}),
				cty.MapVal(map[string]cty.Value{
					"d": cty.ListVal([]cty.Value{
						cty.StringVal("e"),
						cty.StringVal("f"),
					}),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
				"d": cty.ListVal([]cty.Value{
					cty.StringVal("e"),
					cty.StringVal("f"),
				}),
			}),
			false,
		},
		{ // merge map of various kinds
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.ListVal([]cty.Value{
						cty.StringVal("b"),
						cty.StringVal("c"),
					}),
				}),
				cty.MapVal(map[string]cty.Value{
					"d": cty.MapVal(map[string]cty.Value{
						"e": cty.StringVal("f"),
					}),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
				"d": cty.MapVal(map[string]cty.Value{
					"e": cty.StringVal("f"),
				}),
			}),
			false,
		},
		{ // argument error: non map type
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.ListVal([]cty.Value{
						cty.StringVal("b"),
						cty.StringVal("c"),
					}),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("d"),
					cty.StringVal("e"),
				}),
			},
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("merge(%#v)", test.Values), func(t *testing.T) {
			got, err := Merge(test.Values...)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestReverse(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			"",
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a")}),
			cty.ListVal([]cty.Value{cty.StringVal("a")}),
			"",
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			cty.ListVal([]cty.Value{cty.StringVal("b"), cty.StringVal("a")}),
			"",
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b"), cty.StringVal("c")}),
			cty.ListVal([]cty.Value{cty.StringVal("c"), cty.StringVal("b"), cty.StringVal("a")}),
			"",
		},
		{
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.String), cty.StringVal("b"), cty.StringVal("c")}),
			cty.ListVal([]cty.Value{cty.StringVal("c"), cty.StringVal("b"), cty.UnknownVal(cty.String)}),
			"",
		},
		{
			cty.EmptyTupleVal,
			cty.EmptyTupleVal,
			"",
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("a")}),
			cty.TupleVal([]cty.Value{cty.StringVal("a")}),
			"",
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.True}),
			cty.TupleVal([]cty.Value{cty.True, cty.StringVal("a")}),
			"",
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.True, cty.Zero}),
			cty.TupleVal([]cty.Value{cty.Zero, cty.True, cty.StringVal("a")}),
			"",
		},
		{
			cty.SetValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			"",
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("a")}),
			cty.ListVal([]cty.Value{cty.StringVal("a")}),
			"",
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			cty.ListVal([]cty.Value{cty.StringVal("b"), cty.StringVal("a")}), // set-of-string iterates in lexicographical order
			"",
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("b"), cty.StringVal("a"), cty.StringVal("c")}),
			cty.ListVal([]cty.Value{cty.StringVal("c"), cty.StringVal("b"), cty.StringVal("a")}), // set-of-string iterates in lexicographical order
			"",
		},
		{
			cty.StringVal("no"),
			cty.NilVal,
			"can only reverse list or tuple values, not string",
		},
		{
			cty.True,
			cty.NilVal,
			"can only reverse list or tuple values, not bool",
		},
		{
			cty.MapValEmpty(cty.String),
			cty.NilVal,
			"can only reverse list or tuple values, not map of string",
		},
		{
			cty.NullVal(cty.List(cty.String)),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.UnknownVal(cty.List(cty.String)),
			cty.UnknownVal(cty.List(cty.String)),
			"",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("reverse(%#v)", test.List), func(t *testing.T) {
			got, err := Reverse(test.List)

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

func TestSetProduct(t *testing.T) {
	tests := []struct {
		Sets []cty.Value
		Want cty.Value
		Err  string
	}{
		{
			nil,
			cty.DynamicVal,
			"at least two arguments are required",
		},
		{
			[]cty.Value{
				cty.SetValEmpty(cty.String),
			},
			cty.DynamicVal,
			"at least two arguments are required",
		},
		{
			[]cty.Value{
				cty.SetValEmpty(cty.String),
				cty.StringVal("hello"),
			},
			cty.DynamicVal,
			"a set or a list is required", // this is an ArgError, so is presented against the second argument in particular
		},
		{
			[]cty.Value{
				cty.SetValEmpty(cty.String),
				cty.SetValEmpty(cty.String),
			},
			cty.SetValEmpty(cty.Tuple([]cty.Type{cty.String, cty.String})),
			"",
		},
		{
			[]cty.Value{
				cty.SetVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.SetVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.SetVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.SetVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.ListVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			},
			cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			},
			cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("bar")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.True}),
			},
			cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("true")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("true")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("true")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.EmptyTupleVal,
			},
			cty.ListValEmpty(cty.Tuple([]cty.Type{cty.String, cty.DynamicPseudoType})),
			"",
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.EmptyObjectVal}),
			},
			cty.DynamicVal,
			"all elements must be of the same type", // this is an ArgError for the second argument
		},
		{
			[]cty.Value{
				cty.SetVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.SetVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
				cty.SetVal([]cty.Value{cty.StringVal("baz")}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("foo"), cty.StringVal("baz")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("foo"), cty.StringVal("baz")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("foo"), cty.StringVal("baz")}),
				cty.TupleVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("bar"), cty.StringVal("baz")}),
				cty.TupleVal([]cty.Value{cty.StringVal("stg"), cty.StringVal("bar"), cty.StringVal("baz")}),
				cty.TupleVal([]cty.Value{cty.StringVal("prd"), cty.StringVal("bar"), cty.StringVal("baz")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.SetVal([]cty.Value{cty.StringVal("dev"), cty.StringVal("stg"), cty.StringVal("prd")}),
				cty.SetValEmpty(cty.String),
			},
			cty.SetValEmpty(cty.Tuple([]cty.Type{cty.String, cty.String})),
			"",
		},
		{
			[]cty.Value{
				cty.SetVal([]cty.Value{cty.StringVal("foo")}),
				cty.SetVal([]cty.Value{cty.StringVal("bar")}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("foo")}),
				cty.TupleVal([]cty.Value{cty.StringVal("bar")}),
			},
			cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.StringVal("bar")}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.SetVal([]cty.Value{cty.StringVal("foo")}),
				cty.SetVal([]cty.Value{cty.DynamicVal}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.DynamicVal}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.SetVal([]cty.Value{cty.StringVal("foo")}),
				cty.SetVal([]cty.Value{cty.True, cty.DynamicVal}),
			},
			cty.SetVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.True}),
				cty.TupleVal([]cty.Value{cty.StringVal("foo"), cty.UnknownVal(cty.Bool)}),
			}),
			"",
		},
		{
			[]cty.Value{
				cty.UnknownVal(cty.Set(cty.String)),
				cty.SetVal([]cty.Value{cty.True, cty.False}),
			},
			cty.UnknownVal(cty.Set(cty.Tuple([]cty.Type{cty.String, cty.Bool}))),
			"",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("setproduct(%#v)", test.Sets), func(t *testing.T) {
			got, err := SetProduct(test.Sets...)

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

func TestSlice(t *testing.T) {
	listOfStrings := cty.ListVal([]cty.Value{
		cty.StringVal("a"),
		cty.StringVal("b"),
	})
	listOfInts := cty.ListVal([]cty.Value{
		cty.NumberIntVal(1),
		cty.NumberIntVal(2),
	})
	listWithUnknowns := cty.ListVal([]cty.Value{
		cty.StringVal("a"),
		cty.UnknownVal(cty.String),
	})
	tuple := cty.TupleVal([]cty.Value{
		cty.StringVal("a"),
		cty.NumberIntVal(1),
		cty.UnknownVal(cty.List(cty.String)),
	})
	tests := []struct {
		List       cty.Value
		StartIndex cty.Value
		EndIndex   cty.Value
		Want       cty.Value
		Err        bool
	}{
		{ // normal usage
			listOfStrings,
			cty.NumberIntVal(1),
			cty.NumberIntVal(2),
			cty.ListVal([]cty.Value{
				cty.StringVal("b"),
			}),
			false,
		},
		{ // slice only an unknown value
			listWithUnknowns,
			cty.NumberIntVal(1),
			cty.NumberIntVal(2),
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
			false,
		},
		{ // slice multiple values, which contain an unknown
			listWithUnknowns,
			cty.NumberIntVal(0),
			cty.NumberIntVal(2),
			listWithUnknowns,
			false,
		},
		{ // an unknown list should be slicable, returning an unknown list
			cty.UnknownVal(cty.List(cty.String)),
			cty.NumberIntVal(0),
			cty.NumberIntVal(2),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{ // normal usage
			listOfInts,
			cty.NumberIntVal(1),
			cty.NumberIntVal(2),
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(2),
			}),
			false,
		},
		{ // empty result
			listOfStrings,
			cty.NumberIntVal(1),
			cty.NumberIntVal(1),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // index out of bounds
			listOfStrings,
			cty.NumberIntVal(1),
			cty.NumberIntVal(4),
			cty.NilVal,
			true,
		},
		{ // StartIndex index > EndIndex
			listOfStrings,
			cty.NumberIntVal(2),
			cty.NumberIntVal(1),
			cty.NilVal,
			true,
		},
		{ // negative StartIndex
			listOfStrings,
			cty.NumberIntVal(-1),
			cty.NumberIntVal(0),
			cty.NilVal,
			true,
		},
		{ // sets are not slice-able
			cty.SetVal([]cty.Value{
				cty.StringVal("x"),
				cty.StringVal("y"),
			}),
			cty.NumberIntVal(0),
			cty.NumberIntVal(0),
			cty.NilVal,
			true,
		},
		{ // tuple slice
			tuple,
			cty.NumberIntVal(1),
			cty.NumberIntVal(3),
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.UnknownVal(cty.List(cty.String)),
			}),
			false,
		},
		{ // unknown tuple slice
			cty.UnknownVal(tuple.Type()),
			cty.NumberIntVal(1),
			cty.NumberIntVal(3),
			cty.UnknownVal(cty.Tuple([]cty.Type{
				cty.Number,
				cty.List(cty.String),
			})),
			false,
		},
		{ // empty list slice
			listOfStrings,
			cty.NumberIntVal(2),
			cty.NumberIntVal(2),
			cty.ListValEmpty(cty.String),
			false,
		},
		{ // empty tuple slice
			tuple,
			cty.NumberIntVal(3),
			cty.NumberIntVal(3),
			cty.EmptyTupleVal,
			false,
		},
		{ // list with unknown start offset
			listOfStrings,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(2),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{ // list with unknown start offset but end out of bounds
			listOfStrings,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(200),
			cty.UnknownVal(cty.List(cty.String)),
			true,
		},
		{ // list with unknown start offset but end < 0
			listOfStrings,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(-4),
			cty.UnknownVal(cty.List(cty.String)),
			true,
		},
		{ // list with unknown end offset
			listOfStrings,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(0),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
		{ // list with unknown end offset but start out of bounds
			listOfStrings,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(200),
			cty.UnknownVal(cty.List(cty.String)),
			true,
		},
		{ // list with unknown end offset but start < 0
			listOfStrings,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(-3),
			cty.UnknownVal(cty.List(cty.String)),
			true,
		},
		{ // tuple slice with unknown start offset
			tuple,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(3),
			cty.DynamicVal,
			false,
		},
		{ // tuple slice with unknown start offset but end out of bounds
			tuple,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(200),
			cty.DynamicVal,
			true,
		},
		{ // tuple slice with unknown start offset but end < 0
			tuple,
			cty.UnknownVal(cty.Number),
			cty.NumberIntVal(-20),
			cty.DynamicVal,
			true,
		},
		{ // tuple slice with unknown end offset
			tuple,
			cty.NumberIntVal(0),
			cty.UnknownVal(cty.Number),
			cty.DynamicVal,
			false,
		},
		{ // tuple slice with unknown end offset but start < 0
			tuple,
			cty.NumberIntVal(-2),
			cty.UnknownVal(cty.Number),
			cty.DynamicVal,
			true,
		},
		{ // tuple slice with unknown end offset but start out of bounds
			tuple,
			cty.NumberIntVal(200),
			cty.UnknownVal(cty.Number),
			cty.DynamicVal,
			true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-slice(%#v, %#v, %#v)", i, test.List, test.StartIndex, test.EndIndex), func(t *testing.T) {
			got, err := Slice(test.List, test.StartIndex, test.EndIndex)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestTranspose(t *testing.T) {
	tests := []struct {
		Values cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.MapVal(map[string]cty.Value{
				"key1": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
				"key2": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
				"key3": cty.ListVal([]cty.Value{
					cty.StringVal("c"),
				}),
				"key4": cty.ListValEmpty(cty.String),
			}),
			cty.MapVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("key1"),
					cty.StringVal("key2"),
				}),
				"b": cty.ListVal([]cty.Value{
					cty.StringVal("key1"),
					cty.StringVal("key2"),
				}),
				"c": cty.ListVal([]cty.Value{
					cty.StringVal("key2"),
					cty.StringVal("key3"),
				}),
			}),
			false,
		},
		{ // map - unknown value
			cty.MapVal(map[string]cty.Value{
				"key1": cty.UnknownVal(cty.List(cty.String)),
			}),
			cty.UnknownVal(cty.Map(cty.List(cty.String))),
			false,
		},
		{ // bad map - empty value
			cty.MapVal(map[string]cty.Value{
				"key1": cty.ListValEmpty(cty.String),
			}),
			cty.NilVal,
			true,
		},
		{ // bad map - value not a list
			cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("a"),
			}),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("transpose(%#v)", test.Values), func(t *testing.T) {
			got, err := Transpose(test.Values)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestValues(t *testing.T) {
	tests := []struct {
		Values cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.MapVal(map[string]cty.Value{
				"hello":  cty.StringVal("world"),
				"what's": cty.StringVal("up"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("world"),
				cty.StringVal("up"),
			}),
			false,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"what's": cty.StringVal("up"),
				"hello":  cty.StringVal("world"),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("world"),
				cty.StringVal("up"),
			}),
			false,
		},
		{ // empty object
			cty.EmptyObjectVal,
			cty.EmptyTupleVal,
			false,
		},
		{
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"what's": cty.String,
				"hello":  cty.Bool,
			})),
			cty.UnknownVal(cty.Tuple([]cty.Type{
				cty.Bool,
				cty.String,
			})),
			false,
		},
		{ // note ordering: keys are sorted first
			cty.MapVal(map[string]cty.Value{
				"hello":   cty.NumberIntVal(1),
				"goodbye": cty.NumberIntVal(42),
			}),
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(42),
				cty.NumberIntVal(1),
			}),
			false,
		},
		{ // map of lists
			cty.MapVal(map[string]cty.Value{
				"hello":  cty.ListVal([]cty.Value{cty.StringVal("world")}),
				"what's": cty.ListVal([]cty.Value{cty.StringVal("up")}),
			}),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("world")}),
				cty.ListVal([]cty.Value{cty.StringVal("up")}),
			}),
			false,
		},
		{ // map with unknowns
			cty.MapVal(map[string]cty.Value{
				"hello":  cty.ListVal([]cty.Value{cty.StringVal("world")}),
				"what's": cty.UnknownVal(cty.List(cty.String)),
			}),
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{cty.StringVal("world")}),
				cty.UnknownVal(cty.List(cty.String)),
			}),
			false,
		},
		{ // empty m
			cty.MapValEmpty(cty.DynamicPseudoType),
			cty.ListValEmpty(cty.DynamicPseudoType),
			false,
		},
		{ // unknown m
			cty.UnknownVal(cty.Map(cty.String)),
			cty.UnknownVal(cty.List(cty.String)),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("values(%#v)", test.Values), func(t *testing.T) {
			got, err := Values(test.Values)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
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

func TestZipmap(t *testing.T) {
	list1 := cty.ListVal([]cty.Value{
		cty.StringVal("hello"),
		cty.StringVal("world"),
	})
	list2 := cty.ListVal([]cty.Value{
		cty.StringVal("bar"),
		cty.StringVal("baz"),
	})
	list3 := cty.ListVal([]cty.Value{
		cty.StringVal("hello"),
		cty.StringVal("there"),
		cty.StringVal("world"),
	})
	list4 := cty.ListVal([]cty.Value{
		cty.NumberIntVal(1),
		cty.NumberIntVal(42),
	})
	list5 := cty.ListVal([]cty.Value{
		cty.ListVal([]cty.Value{
			cty.StringVal("bar"),
		}),
		cty.ListVal([]cty.Value{
			cty.StringVal("baz"),
		}),
	})
	tests := []struct {
		Keys   cty.Value
		Values cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			list1,
			list2,
			cty.MapVal(map[string]cty.Value{
				"hello": cty.StringVal("bar"),
				"world": cty.StringVal("baz"),
			}),
			false,
		},
		{
			list1,
			list4,
			cty.MapVal(map[string]cty.Value{
				"hello": cty.NumberIntVal(1),
				"world": cty.NumberIntVal(42),
			}),
			false,
		},
		{ // length mismatch
			list1,
			list3,
			cty.NilVal,
			true,
		},
		{ // map of lists
			list1,
			list5,
			cty.MapVal(map[string]cty.Value{
				"hello": cty.ListVal([]cty.Value{cty.StringVal("bar")}),
				"world": cty.ListVal([]cty.Value{cty.StringVal("baz")}),
			}),
			false,
		},
		{ // tuple values produce object
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("bar"),
				cty.UnknownVal(cty.Bool),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"hello": cty.StringVal("bar"),
				"world": cty.UnknownVal(cty.Bool),
			}),
			false,
		},
		{ // empty tuple produces empty object
			cty.ListValEmpty(cty.String),
			cty.EmptyTupleVal,
			cty.EmptyObjectVal,
			false,
		},
		{ // tuple with any unknown keys produces DynamicVal
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.UnknownVal(cty.String),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("bar"),
				cty.True,
			}),
			cty.DynamicVal,
			false,
		},
		{ // tuple with all keys unknown produces DynamicVal
			cty.UnknownVal(cty.List(cty.String)),
			cty.TupleVal([]cty.Value{
				cty.StringVal("bar"),
				cty.True,
			}),
			cty.DynamicVal,
			false,
		},
		{ // list with all keys unknown produces correctly-typed unknown map
			cty.UnknownVal(cty.List(cty.String)),
			cty.ListVal([]cty.Value{
				cty.StringVal("bar"),
				cty.StringVal("baz"),
			}),
			cty.UnknownVal(cty.Map(cty.String)),
			false,
		},
		{ // unknown tuple as values produces correctly-typed unknown object
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
			}),
			cty.UnknownVal(cty.Tuple([]cty.Type{
				cty.String,
				cty.Bool,
			})),
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"hello": cty.String,
				"world": cty.Bool,
			})),
			false,
		},
		{ // unknown list as values produces correctly-typed unknown map
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
			}),
			cty.UnknownVal(cty.List(cty.String)),
			cty.UnknownVal(cty.Map(cty.String)),
			false,
		},
		{ // empty input returns an empty map
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			cty.MapValEmpty(cty.String),
			false,
		},
		{ // keys cannot be a list of lists
			list5,
			list1,
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("zipmap(%#v, %#v)", test.Keys, test.Values), func(t *testing.T) {
			got, err := Zipmap(test.Keys, test.Values)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\n\nkeys:   %#v\nvalues: %#v\ngot:    %#v\nwant:   %#v", test.Keys, test.Values, got, test.Want)
			}
		})
	}
}
