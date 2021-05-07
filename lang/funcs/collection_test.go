package funcs

import (
	"fmt"
	"math"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

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
		{ // Marked collections return a marked length
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
			}).Mark("secret"),
			cty.NumberIntVal(2).Mark("secret"),
		},
		{ // Marks on values in unmarked collections do not propagate
			cty.ListVal([]cty.Value{
				cty.StringVal("hello").Mark("a"),
				cty.StringVal("world").Mark("b"),
			}),
			cty.NumberIntVal(2),
		},
		{ // Marked strings return a marked length
			cty.StringVal("hello world").Mark("secret"),
			cty.NumberIntVal(11).Mark("secret"),
		},
		{ // Marked tuples return a marked length
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("world"),
			}).Mark("secret"),
			cty.NumberIntVal(2).Mark("secret"),
		},
		{ // Marks on values in unmarked tuples do not propagate
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello").Mark("a"),
				cty.StringVal("world").Mark("b"),
			}),
			cty.NumberIntVal(2),
		},
		{ // Marked objects return a marked length
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
				"b": cty.StringVal("world"),
				"c": cty.StringVal("nice to meet you"),
			}).Mark("secret"),
			cty.NumberIntVal(3).Mark("secret"),
		},
		{ // Marks on object attribute values do not propagate
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello").Mark("a"),
				"b": cty.StringVal("world").Mark("b"),
				"c": cty.StringVal("nice to meet you").Mark("c"),
			}),
			cty.NumberIntVal(3),
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

func TestAllTrue(t *testing.T) {
	tests := []struct {
		Collection cty.Value
		Want       cty.Value
		Err        bool
	}{
		{
			cty.ListValEmpty(cty.Bool),
			cty.True,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.True}),
			cty.True,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.False}),
			cty.False,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.True, cty.False}),
			cty.False,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.False, cty.True}),
			cty.False,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.True, cty.NullVal(cty.Bool)}),
			cty.False,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.Bool)}),
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.Bool),
				cty.UnknownVal(cty.Bool),
			}),
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			cty.UnknownVal(cty.List(cty.Bool)),
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			cty.NullVal(cty.List(cty.Bool)),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("alltrue(%#v)", test.Collection), func(t *testing.T) {
			got, err := AllTrue(test.Collection)

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

func TestAnyTrue(t *testing.T) {
	tests := []struct {
		Collection cty.Value
		Want       cty.Value
		Err        bool
	}{
		{
			cty.ListValEmpty(cty.Bool),
			cty.False,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.True}),
			cty.True,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.False}),
			cty.False,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.True, cty.False}),
			cty.True,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.False, cty.True}),
			cty.True,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.NullVal(cty.Bool), cty.True}),
			cty.True,
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.Bool)}),
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.Bool),
				cty.False,
			}),
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.Bool),
				cty.True,
			}),
			cty.True,
			false,
		},
		{
			cty.UnknownVal(cty.List(cty.Bool)),
			cty.UnknownVal(cty.Bool),
			false,
		},
		{
			cty.NullVal(cty.List(cty.Bool)),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("anytrue(%#v)", test.Collection), func(t *testing.T) {
			got, err := AnyTrue(test.Collection)

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
	mapOfMaps := cty.MapVal(map[string]cty.Value{
		"foo": cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal("bar"),
		}),
		"baz": cty.MapVal(map[string]cty.Value{
			"b": cty.StringVal("bat"),
		}),
	})
	mapOfTuples := cty.MapVal(map[string]cty.Value{
		"foo": cty.TupleVal([]cty.Value{cty.StringVal("bar")}),
		"baz": cty.TupleVal([]cty.Value{cty.StringVal("bat")}),
	})
	objectOfMaps := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal("bar"),
		}),
		"baz": cty.MapVal(map[string]cty.Value{
			"b": cty.StringVal("bat"),
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
		{
			[]cty.Value{
				mapOfMaps,
				cty.StringVal("foo"),
			},
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("bar"),
			}),
			false,
		},
		{
			[]cty.Value{
				objectOfMaps,
				cty.StringVal("foo"),
			},
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("bar"),
			}),
			false,
		},
		{
			[]cty.Value{
				mapOfTuples,
				cty.StringVal("foo"),
			},
			cty.TupleVal([]cty.Value{cty.StringVal("bar")}),
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
		{ // Supplied default with valid (int) key
			[]cty.Value{
				simpleMap,
				cty.StringVal("foobar"),
				cty.NumberIntVal(-1),
			},
			cty.StringVal("-1"),
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
		{ // Supplied default with type mismatch: expects a map return
			[]cty.Value{
				mapOfMaps,
				cty.StringVal("foo"),
				cty.StringVal(""),
			},
			cty.NilVal,
			true,
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
				mapWithUnknowns,
				cty.StringVal("foo"),
			},
			cty.StringVal("bar"),
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
		{ // successful marked collection lookup returns marked value
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep"),
				}).Mark("a"),
				cty.StringVal("boop"),
				cty.StringVal("nope"),
			},
			cty.StringVal("beep").Mark("a"),
			false,
		},
		{ // apply collection marks to unknown return vaue
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep"),
					"frob": cty.UnknownVal(cty.String),
				}).Mark("a"),
				cty.StringVal("frob"),
				cty.StringVal("nope"),
			},
			cty.UnknownVal(cty.String).Mark("a"),
			false,
		},
		{ // propagate collection marks to default when returning
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep"),
				}).Mark("a"),
				cty.StringVal("frob"),
				cty.StringVal("nope").Mark("b"),
			},
			cty.StringVal("nope").WithMarks(cty.NewValueMarks("a", "b")),
			false,
		},
		{ // on unmarked collection, return only marks from found value
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep").Mark("a"),
					"frob": cty.StringVal("honk").Mark("b"),
				}),
				cty.StringVal("frob"),
				cty.StringVal("nope").Mark("c"),
			},
			cty.StringVal("honk").Mark("b"),
			false,
		},
		{ // on unmarked collection, return default exactly on missing
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep").Mark("a"),
					"frob": cty.StringVal("honk").Mark("b"),
				}),
				cty.StringVal("squish"),
				cty.StringVal("nope").Mark("c"),
			},
			cty.StringVal("nope").Mark("c"),
			false,
		},
		{ // retain marks on default if converted
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep").Mark("a"),
					"frob": cty.StringVal("honk").Mark("b"),
				}),
				cty.StringVal("squish"),
				cty.NumberIntVal(5).Mark("c"),
			},
			cty.StringVal("5").Mark("c"),
			false,
		},
		{ // propagate marks from key
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep"),
					"frob": cty.StringVal("honk"),
				}),
				cty.StringVal("boop").Mark("a"),
				cty.StringVal("nope"),
			},
			cty.StringVal("beep").Mark("a"),
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

func TestOne(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
			}),
			cty.NumberIntVal(1),
			"",
		},
		{
			cty.ListValEmpty(cty.Number),
			cty.NullVal(cty.Number),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.Number),
			}),
			cty.UnknownVal(cty.Number),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.Number),
				cty.UnknownVal(cty.Number),
			}),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.UnknownVal(cty.List(cty.String)),
			cty.UnknownVal(cty.String),
			"",
		},
		{
			cty.NullVal(cty.List(cty.String)),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
			}).Mark("boop"),
			cty.NumberIntVal(1).Mark("boop"),
			"",
		},
		{
			cty.ListValEmpty(cty.Bool).Mark("boop"),
			cty.NullVal(cty.Bool).Mark("boop"),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1).Mark("boop"),
			}),
			cty.NumberIntVal(1).Mark("boop"),
			"",
		},

		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
			}),
			cty.NumberIntVal(1),
			"",
		},
		{
			cty.SetValEmpty(cty.Number),
			cty.NullVal(cty.Number),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.SetVal([]cty.Value{
				cty.UnknownVal(cty.Number),
			}),
			cty.UnknownVal(cty.Number),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.UnknownVal(cty.Number),
				cty.UnknownVal(cty.Number),
			}),
			// The above would be valid if those two unknown values were
			// equal known values, so this returns unknown rather than failing.
			cty.UnknownVal(cty.Number),
			"",
		},
		{
			cty.UnknownVal(cty.Set(cty.String)),
			cty.UnknownVal(cty.String),
			"",
		},
		{
			cty.NullVal(cty.Set(cty.String)),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
			}).Mark("boop"),
			cty.NumberIntVal(1).Mark("boop"),
			"",
		},
		{
			cty.SetValEmpty(cty.Bool).Mark("boop"),
			cty.NullVal(cty.Bool).Mark("boop"),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(1).Mark("boop"),
			}),
			cty.NumberIntVal(1).Mark("boop"),
			"",
		},

		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
			}),
			cty.NumberIntVal(1),
			"",
		},
		{
			cty.EmptyTupleVal,
			cty.NullVal(cty.DynamicPseudoType),
			"",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.UnknownVal(cty.Number),
			}),
			cty.UnknownVal(cty.Number),
			"",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.UnknownVal(cty.Number),
				cty.UnknownVal(cty.Number),
			}),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.UnknownVal(cty.EmptyTuple),
			// Could actually return null here, but don't for consistency with unknown lists
			cty.UnknownVal(cty.DynamicPseudoType),
			"",
		},
		{
			cty.UnknownVal(cty.Tuple([]cty.Type{cty.Bool})),
			cty.UnknownVal(cty.Bool),
			"",
		},
		{
			cty.UnknownVal(cty.Tuple([]cty.Type{cty.Bool, cty.Number})),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.NullVal(cty.EmptyTuple),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.NullVal(cty.Tuple([]cty.Type{cty.Bool})),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.NullVal(cty.Tuple([]cty.Type{cty.Bool, cty.Number})),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
			}).Mark("boop"),
			cty.NumberIntVal(1).Mark("boop"),
			"",
		},
		{
			cty.EmptyTupleVal.Mark("boop"),
			cty.NullVal(cty.DynamicPseudoType).Mark("boop"),
			"",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1).Mark("boop"),
			}),
			cty.NumberIntVal(1).Mark("boop"),
			"",
		},

		{
			cty.DynamicVal,
			cty.DynamicVal,
			"",
		},
		{
			cty.NullVal(cty.DynamicPseudoType),
			cty.NilVal,
			"argument must not be null",
		},
		{
			cty.MapValEmpty(cty.String),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.EmptyObjectVal,
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.True,
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
		{
			cty.UnknownVal(cty.Bool),
			cty.NilVal,
			"must be a list, set, or tuple value with either zero or one elements",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("one(%#v)", test.List), func(t *testing.T) {
			got, err := One(test.List)

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				} else if got, want := err.Error(), test.Err; got != want {
					t.Fatalf("wrong error\n got: %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !test.Want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestSum(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NumberIntVal(6),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1476),
				cty.NumberIntVal(2093),
				cty.NumberIntVal(2092495),
				cty.NumberIntVal(64589234),
				cty.NumberIntVal(234),
			}),
			cty.NumberIntVal(66685532),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.UnknownVal(cty.String),
			"argument must be list, set, or tuple of number values",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(10),
				cty.NumberIntVal(-19),
				cty.NumberIntVal(5),
			}),
			cty.NumberIntVal(-4),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberFloatVal(10.2),
				cty.NumberFloatVal(19.4),
				cty.NumberFloatVal(5.7),
			}),
			cty.NumberFloatVal(35.3),
			"",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberFloatVal(-10.2),
				cty.NumberFloatVal(-19.4),
				cty.NumberFloatVal(-5.7),
			}),
			cty.NumberFloatVal(-35.3),
			"",
		},
		{
			cty.ListVal([]cty.Value{cty.NullVal(cty.Number)}),
			cty.NilVal,
			"argument must be list, set, or tuple of number values",
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(5),
				cty.NullVal(cty.Number),
			}),
			cty.NilVal,
			"argument must be list, set, or tuple of number values",
		},
		{
			cty.SetVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.UnknownVal(cty.String),
			"argument must be list, set, or tuple of number values",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(10),
				cty.NumberIntVal(-19),
				cty.NumberIntVal(5),
			}),
			cty.NumberIntVal(-4),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(10),
				cty.NumberIntVal(25),
				cty.NumberIntVal(30),
			}),
			cty.NumberIntVal(65),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberFloatVal(2340.8),
				cty.NumberFloatVal(10.2),
				cty.NumberFloatVal(3),
			}),
			cty.NumberFloatVal(2354),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberFloatVal(2),
			}),
			cty.NumberFloatVal(2),
			"",
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberFloatVal(-2),
				cty.NumberFloatVal(-50),
				cty.NumberFloatVal(-20),
				cty.NumberFloatVal(-123),
				cty.NumberFloatVal(-4),
			}),
			cty.NumberFloatVal(-199),
			"",
		},
		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(12),
				cty.StringVal("a"),
				cty.NumberIntVal(38),
			}),
			cty.UnknownVal(cty.String),
			"argument must be list, set, or tuple of number values",
		},
		{
			cty.NumberIntVal(12),
			cty.NilVal,
			"cannot sum noniterable",
		},
		{
			cty.ListValEmpty(cty.Number),
			cty.NilVal,
			"cannot sum an empty list",
		},
		{
			cty.MapVal(map[string]cty.Value{"hello": cty.True}),
			cty.NilVal,
			"argument must be list, set, or tuple. Received map of bool",
		},
		{
			cty.UnknownVal(cty.Number),
			cty.UnknownVal(cty.Number),
			"",
		},
		{
			cty.UnknownVal(cty.List(cty.Number)),
			cty.UnknownVal(cty.Number),
			"",
		},
		{ // known list containing unknown values
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.Number)}),
			cty.UnknownVal(cty.Number),
			"",
		},
		{ // numbers too large to represent as float64
			cty.ListVal([]cty.Value{
				cty.MustParseNumberVal("1e+500"),
				cty.MustParseNumberVal("1e+500"),
			}),
			cty.MustParseNumberVal("2e+500"),
			"",
		},
		{ // edge case we have a special error handler for
			cty.ListVal([]cty.Value{
				cty.NumberFloatVal(math.Inf(1)),
				cty.NumberFloatVal(math.Inf(-1)),
			}),
			cty.NilVal,
			"can't compute sum of opposing infinities",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("sum(%#v)", test.List), func(t *testing.T) {
			got, err := Sum(test.List)

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				} else if got, want := err.Error(), test.Err; got != want {
					t.Fatalf("wrong error\n got: %s\nwant: %s", got, want)
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
			cty.MapValEmpty(cty.List(cty.String)),
			false,
		},
		{ // bad map - value not a list
			cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("a"),
			}),
			cty.NilVal,
			true,
		},
		{ // marks (deep or shallow) on any elements will propegate to the entire return value
			cty.MapVal(map[string]cty.Value{
				"key1": cty.ListVal([]cty.Value{
					cty.StringVal("a").Mark("beep"), // mark on the inner list element
					cty.StringVal("b"),
				}),
				"key2": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}).Mark("boop"), // mark on the map element
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
					cty.StringVal("key3")}),
			}).WithMarks(cty.NewValueMarks("beep", "boop")),
			false,
		},
		{ // Marks on the input value will be applied to the return value
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
			}).Mark("beep"), // mark on the entire input value
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
			}).Mark("beep"),
			false,
		},
		{ // Marks on the entire input value AND inner elements (deep or shallow) ALL apply to the return
			cty.MapVal(map[string]cty.Value{
				"key1": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}).Mark("beep"), // mark on the map element
				"key2": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
				"key3": cty.ListVal([]cty.Value{
					cty.StringVal("c").Mark("boop"), // mark on the inner list element
				}),
			}).Mark("bloop"), // mark on the entire input value
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
			}).WithMarks(cty.NewValueMarks("beep", "boop", "bloop")),
			false,
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
