package jsonplan

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestOmitUnknowns(t *testing.T) {
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
			cty.NilVal,
		},
		{
			cty.ListValEmpty(cty.String),
			cty.EmptyTupleVal,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
		},
		{
			cty.ListVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.NullVal(cty.String)}),
		},
		{
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.NullVal(cty.String)}),
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
		},
		//
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.UnknownVal(cty.String)}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.NullVal(cty.String),
			}),
		},
		{
			cty.MapVal(map[string]cty.Value{
				"hello": cty.True,
				"world": cty.UnknownVal(cty.Bool),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"hello": cty.True,
			}),
		},
		{
			cty.SetVal([]cty.Value{
				cty.StringVal("dev"),
				cty.StringVal("foo"),
				cty.StringVal("stg"),
				cty.UnknownVal(cty.String),
			}),
			cty.TupleVal([]cty.Value{
				cty.StringVal("dev"),
				cty.StringVal("foo"),
				cty.StringVal("stg"),
			}),
		},
		{
			cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.UnknownVal(cty.String),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("known"),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("known"),
				}),
				cty.EmptyObjectVal,
			}),
		},
	}

	for _, test := range tests {
		got := omitUnknowns(test.Input)
		if !reflect.DeepEqual(got, test.Want) {
			t.Errorf(
				"wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v",
				test.Input, got, test.Want,
			)
		}
	}
}

func TestUnknownAsBool(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello"),
			cty.False,
		},
		{
			cty.NullVal(cty.String),
			cty.False,
		},
		{
			cty.UnknownVal(cty.String),
			cty.True,
		},

		{
			cty.NullVal(cty.DynamicPseudoType),
			cty.False,
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{"test": cty.String})),
			cty.False,
		},
		{
			cty.DynamicVal,
			cty.True,
		},

		{
			cty.ListValEmpty(cty.String),
			cty.EmptyTupleVal,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.ListVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.True}),
		},
		{
			cty.SetValEmpty(cty.String),
			cty.EmptyTupleVal,
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.SetVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.SetVal([]cty.Value{cty.UnknownVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.True}),
		},
		{
			cty.EmptyTupleVal,
			cty.EmptyTupleVal,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.TupleVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.TupleVal([]cty.Value{cty.UnknownVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.True}),
		},
		{
			cty.MapValEmpty(cty.String),
			cty.EmptyObjectVal,
		},
		{
			cty.MapVal(map[string]cty.Value{"greeting": cty.StringVal("hello")}),
			cty.EmptyObjectVal,
		},
		{
			cty.MapVal(map[string]cty.Value{"greeting": cty.NullVal(cty.String)}),
			cty.EmptyObjectVal,
		},
		{
			cty.MapVal(map[string]cty.Value{"greeting": cty.UnknownVal(cty.String)}),
			cty.ObjectVal(map[string]cty.Value{"greeting": cty.True}),
		},
		{
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"greeting": cty.StringVal("hello")}),
			cty.EmptyObjectVal,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"greeting": cty.NullVal(cty.String)}),
			cty.EmptyObjectVal,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"greeting": cty.UnknownVal(cty.String)}),
			cty.ObjectVal(map[string]cty.Value{"greeting": cty.True}),
		},
		{
			cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.UnknownVal(cty.String),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("known"),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.EmptyObjectVal,
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.True,
				}),
			}),
		},
		{
			cty.SetVal([]cty.Value{
				cty.MapValEmpty(cty.String),
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("known"),
				}),
				cty.MapVal(map[string]cty.Value{
					"a": cty.UnknownVal(cty.String),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.EmptyObjectVal,
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.True,
				}),
				cty.EmptyObjectVal,
			}),
		},
	}

	for _, test := range tests {
		got := unknownAsBool(test.Input)
		if !reflect.DeepEqual(got, test.Want) {
			t.Errorf(
				"wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v",
				test.Input, got, test.Want,
			)
		}
	}
}

func TestSensitiveAsBool(t *testing.T) {
	sensitive := "sensitive"
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello"),
			cty.False,
		},
		{
			cty.NullVal(cty.String),
			cty.False,
		},
		{
			cty.StringVal("hello").Mark(sensitive),
			cty.True,
		},
		{
			cty.NullVal(cty.String).Mark(sensitive),
			cty.True,
		},

		{
			cty.NullVal(cty.DynamicPseudoType).Mark(sensitive),
			cty.True,
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{"test": cty.String})),
			cty.False,
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{"test": cty.String})).Mark(sensitive),
			cty.True,
		},
		{
			cty.DynamicVal,
			cty.False,
		},
		{
			cty.DynamicVal.Mark(sensitive),
			cty.True,
		},

		{
			cty.ListValEmpty(cty.String),
			cty.EmptyTupleVal,
		},
		{
			cty.ListValEmpty(cty.String).Mark(sensitive),
			cty.True,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("friend").Mark(sensitive),
			}),
			cty.TupleVal([]cty.Value{
				cty.False,
				cty.True,
			}),
		},
		{
			cty.SetValEmpty(cty.String),
			cty.EmptyTupleVal,
		},
		{
			cty.SetValEmpty(cty.String).Mark(sensitive),
			cty.True,
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.False}),
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello").Mark(sensitive)}),
			cty.True,
		},
		{
			cty.EmptyTupleVal.Mark(sensitive),
			cty.True,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.StringVal("friend").Mark(sensitive),
			}),
			cty.TupleVal([]cty.Value{
				cty.False,
				cty.True,
			}),
		},
		{
			cty.MapValEmpty(cty.String),
			cty.EmptyObjectVal,
		},
		{
			cty.MapValEmpty(cty.String).Mark(sensitive),
			cty.True,
		},
		{
			cty.MapVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"animal":   cty.StringVal("horse"),
			}),
			cty.EmptyObjectVal,
		},
		{
			cty.MapVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"animal":   cty.StringVal("horse").Mark(sensitive),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"animal": cty.True,
			}),
		},
		{
			cty.MapVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"animal":   cty.StringVal("horse").Mark(sensitive),
			}).Mark(sensitive),
			cty.True,
		},
		{
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"animal":   cty.StringVal("horse"),
			}),
			cty.EmptyObjectVal,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"animal":   cty.StringVal("horse").Mark(sensitive),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"animal": cty.True,
			}),
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"greeting": cty.StringVal("hello"),
				"animal":   cty.StringVal("horse").Mark(sensitive),
			}).Mark(sensitive),
			cty.True,
		},
		{
			cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.UnknownVal(cty.String),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("known").Mark(sensitive),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.EmptyObjectVal,
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.True,
				}),
			}),
		},
		{
			cty.ListVal([]cty.Value{
				cty.MapValEmpty(cty.String),
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("known").Mark(sensitive),
				}),
				cty.MapVal(map[string]cty.Value{
					"a": cty.UnknownVal(cty.String),
				}),
			}),
			cty.TupleVal([]cty.Value{
				cty.EmptyObjectVal,
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.True,
				}),
				cty.EmptyObjectVal,
			}),
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"list":   cty.UnknownVal(cty.List(cty.String)),
				"set":    cty.UnknownVal(cty.Set(cty.Bool)),
				"tuple":  cty.UnknownVal(cty.Tuple([]cty.Type{cty.String, cty.Number})),
				"map":    cty.UnknownVal(cty.Map(cty.String)),
				"object": cty.UnknownVal(cty.Object(map[string]cty.Type{"a": cty.String})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list":   cty.EmptyTupleVal,
				"set":    cty.EmptyTupleVal,
				"tuple":  cty.EmptyTupleVal,
				"map":    cty.EmptyObjectVal,
				"object": cty.EmptyObjectVal,
			}),
		},
	}

	for _, test := range tests {
		got := sensitiveAsBool(test.Input)
		if !reflect.DeepEqual(got, test.Want) {
			t.Errorf(
				"wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v",
				test.Input, got, test.Want,
			)
		}
	}
}
