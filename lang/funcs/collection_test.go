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
			cty.TupleVal([]cty.Value{cty.True}),
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
				cty.StringVal("1"), cty.StringVal("2"),
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
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("coalescelist(%#v)", test.Values), func(t *testing.T) {
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

			if got != test.Want {
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
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("chunklist(%#v, %#v)", test.List, test.Size), func(t *testing.T) {
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
