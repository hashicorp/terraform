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
