package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		Values []cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"a2": cty.ObjectVal(map[string]cty.Value{
							"a3": cty.StringVal("a3"),
							"a4": cty.StringVal("a4"),
						}),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"a2": cty.ObjectVal(map[string]cty.Value{
							"a3": cty.StringVal("a3-changed"),
							"a5": cty.StringVal("a5-new"),
						}),
					}),
				}),
			},
			cty.MapVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a2": cty.MapVal(map[string]cty.Value{
						"a3": cty.StringVal("a3-changed"),
						"a4": cty.StringVal("a4"),
						"a5": cty.StringVal("a5-new"),
					}),
				}),
			}),
			false,
		},
		{
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
			},
			cty.MapVal(map[string]cty.Value{
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
			cty.MapVal(map[string]cty.Value{
				"a": cty.UnknownVal(cty.String),
				"c": cty.StringVal("d"),
			}),
			false,
		},
		{ // handle null map
			[]cty.Value{
				cty.NullVal(cty.Map(cty.String)),
				cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
			},
			cty.MapVal(map[string]cty.Value{
				"c": cty.StringVal("d"),
			}),
			false,
		},
		{ // handle null map
			[]cty.Value{
				cty.NullVal(cty.Map(cty.String)),
				cty.NullVal(cty.Object(map[string]cty.Type{
					"a": cty.List(cty.String),
				})),
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.List(cty.String),
			})),
			false,
		},
		{ // handle null object
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
				cty.NullVal(cty.Object(map[string]cty.Type{
					"a": cty.List(cty.String),
				})),
			},
			cty.MapVal(map[string]cty.Value{
				"c": cty.StringVal("d"),
			}),
			false,
		},
		// { // handle unknowns
		// 	[]cty.Value{
		// 		cty.UnknownVal(cty.Map(cty.String)),
		// 		cty.MapVal(map[string]cty.Value{
		// 			"c": cty.StringVal("d"),
		// 		}),
		// 	},
		// 	cty.UnknownVal(cty.Map(cty.String)),
		// 	false,
		// },
		// { // handle dynamic unknown
		// 	[]cty.Value{
		// 		cty.UnknownVal(cty.DynamicPseudoType),
		// 		cty.MapVal(map[string]cty.Value{
		// 			"c": cty.StringVal("d"),
		// 		}),
		// 	},
		// 	cty.DynamicVal,
		// 	false,
		// },
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
			cty.MapVal(map[string]cty.Value{
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
			cty.MapVal(map[string]cty.Value{
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
			cty.MapVal(map[string]cty.Value{
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
		{ // merge objects of various shapes
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.ListVal([]cty.Value{
						cty.StringVal("b"),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"d": cty.DynamicVal,
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("b"),
				}),
				"d": cty.DynamicVal,
			}),
			false,
		},
		{ // merge maps and objects
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.ListVal([]cty.Value{
						cty.StringVal("b"),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"d": cty.NumberIntVal(2),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("b"),
				}),
				"d": cty.NumberIntVal(2),
			}),
			false,
		},
		{ // attr a type and value is overridden
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.ListVal([]cty.Value{
						cty.StringVal("b"),
					}),
					"b": cty.StringVal("b"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"e": cty.StringVal("f"),
					}),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"e": cty.StringVal("f"),
				}),
				"b": cty.StringVal("b"),
			}),
			false,
		},
		{ // replacing a non-map/object with a map/object
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				cty.MapVal(map[string]cty.Value{
					"a": cty.MapVal(map[string]cty.Value{
						"c": cty.StringVal("d"),
					}),
				}),
			},
			cty.MapVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
			}),
			false,
		},
		{ // replacing a map/object with a non-map/object
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.MapVal(map[string]cty.Value{
						"c": cty.StringVal("d"),
					}),
				}),
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
			},
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			false,
		},
		{ // value of null deletes a field
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				cty.MapVal(map[string]cty.Value{
					"a": cty.NullVal(cty.String),
				}),
			},
			cty.EmptyObjectVal,
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
			got, err := DeepMerge(test.Values...)

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
		{ // convert number keys to strings
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.StringVal("hello"),
				cty.NumberIntVal(2),
				cty.StringVal("goodbye"),
			},
			cty.MapVal(map[string]cty.Value{
				"1": cty.StringVal("hello"),
				"2": cty.StringVal("goodbye"),
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
		{ // null key returns an error
			[]cty.Value{
				cty.NullVal(cty.DynamicPseudoType),
				cty.NumberIntVal(5),
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
				if _, ok := err.(function.PanicError); ok {
					t.Fatalf("unexpected panic: %s", err)
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

func TestSum(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			cty.NumberIntVal(6),
			false,
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
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(10),
				cty.NumberIntVal(-19),
				cty.NumberIntVal(5),
			}),
			cty.NumberIntVal(-4),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberFloatVal(10.2),
				cty.NumberFloatVal(19.4),
				cty.NumberFloatVal(5.7),
			}),
			cty.NumberFloatVal(35.3),
			false,
		},
		{
			cty.ListVal([]cty.Value{
				cty.NumberFloatVal(-10.2),
				cty.NumberFloatVal(-19.4),
				cty.NumberFloatVal(-5.7),
			}),
			cty.NumberFloatVal(-35.3),
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.NullVal(cty.Number)}),
			cty.NilVal,
			true,
		},
		{
			cty.SetVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(10),
				cty.NumberIntVal(-19),
				cty.NumberIntVal(5),
			}),
			cty.NumberIntVal(-4),
			false,
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberIntVal(10),
				cty.NumberIntVal(25),
				cty.NumberIntVal(30),
			}),
			cty.NumberIntVal(65),
			false,
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberFloatVal(2340.8),
				cty.NumberFloatVal(10.2),
				cty.NumberFloatVal(3),
			}),
			cty.NumberFloatVal(2354),
			false,
		},
		{
			cty.SetVal([]cty.Value{
				cty.NumberFloatVal(2),
			}),
			cty.NumberFloatVal(2),
			false,
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
			false,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(12),
				cty.StringVal("a"),
				cty.NumberIntVal(38),
			}),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.NumberIntVal(12),
			cty.NilVal,
			true,
		},
		{
			cty.ListValEmpty(cty.Number),
			cty.NilVal,
			true,
		},
		{
			cty.MapVal(map[string]cty.Value{"hello": cty.True}),
			cty.NilVal,
			true,
		},
		{
			cty.UnknownVal(cty.Number),
			cty.UnknownVal(cty.Number),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("sum(%#v)", test.List), func(t *testing.T) {
			got, err := Sum(test.List)

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
