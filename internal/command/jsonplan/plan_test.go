package jsonplan

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestEncodePaths(t *testing.T) {
	tests := map[string]struct {
		Input cty.PathSet
		Want  json.RawMessage
	}{
		"empty set": {
			cty.NewPathSet(),
			json.RawMessage(nil),
		},
		"index path with string and int steps": {
			cty.NewPathSet(cty.IndexStringPath("boop").IndexInt(0)),
			json.RawMessage(`[["boop",0]]`),
		},
		"get attr path with one step": {
			cty.NewPathSet(cty.GetAttrPath("triggers")),
			json.RawMessage(`[["triggers"]]`),
		},
		"multiple paths of different types": {
			cty.NewPathSet(
				cty.GetAttrPath("alpha").GetAttr("beta").GetAttr("gamma"),
				cty.GetAttrPath("triggers").IndexString("name"),
				cty.IndexIntPath(0).IndexInt(1).IndexInt(2).IndexInt(3),
			),
			json.RawMessage(`[["alpha","beta","gamma"],["triggers","name"],[0,1,2,3]]`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := encodePaths(test.Input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !cmp.Equal(got, test.Want) {
				t.Errorf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
			}
		})
	}
}
