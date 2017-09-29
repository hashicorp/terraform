package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestConfigValueFromHCL2(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  interface{}
	}{
		{
			cty.True,
			true,
		},
		{
			cty.False,
			false,
		},
		{
			cty.NumberIntVal(12),
			int(12),
		},
		{
			cty.NumberFloatVal(12.5),
			float64(12.5),
		},
		{
			cty.StringVal("hello world"),
			"hello world",
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("Ermintrude"),
				"age":  cty.NumberIntVal(19),
				"address": cty.ObjectVal(map[string]cty.Value{
					"street": cty.ListVal([]cty.Value{cty.StringVal("421 Shoreham Loop")}),
					"city":   cty.StringVal("Fridgewater"),
					"state":  cty.StringVal("MA"),
					"zip":    cty.StringVal("91037"),
				}),
			}),
			map[string]interface{}{
				"name": "Ermintrude",
				"age":  int(19),
				"address": map[string]interface{}{
					"street": []interface{}{"421 Shoreham Loop"},
					"city":   "Fridgewater",
					"state":  "MA",
					"zip":    "91037",
				},
			},
		},
		{
			cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bar": cty.StringVal("baz"),
			}),
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
				cty.True,
			}),
			[]interface{}{
				"foo",
				true,
			},
		},
		{
			cty.NullVal(cty.String),
			nil,
		},
		{
			cty.UnknownVal(cty.String),
			UnknownVariableValue,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Input), func(t *testing.T) {
			got := configValueFromHCL2(test.Input)
			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, got, test.Want)
			}
		})
	}
}

func TestHCL2ValueFromConfigValue(t *testing.T) {
	tests := []struct {
		Input interface{}
		Want  cty.Value
	}{
		{
			nil,
			cty.NullVal(cty.DynamicPseudoType),
		},
		{
			UnknownVariableValue,
			cty.DynamicVal,
		},
		{
			true,
			cty.True,
		},
		{
			false,
			cty.False,
		},
		{
			int(12),
			cty.NumberIntVal(12),
		},
		{
			int(0),
			cty.Zero,
		},
		{
			float64(12.5),
			cty.NumberFloatVal(12.5),
		},
		{
			"hello world",
			cty.StringVal("hello world"),
		},
		{
			"O\u0308",               // decomposed letter + diacritic
			cty.StringVal("\u00D6"), // NFC-normalized on entry into cty
		},
		{
			[]interface{}{},
			cty.EmptyTupleVal,
		},
		{
			[]interface{}(nil),
			cty.EmptyTupleVal,
		},
		{
			[]interface{}{"hello", "world"},
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world")}),
		},
		{
			map[string]interface{}{},
			cty.EmptyObjectVal,
		},
		{
			map[string]interface{}(nil),
			cty.EmptyObjectVal,
		},
		{
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bar": cty.StringVal("baz"),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Input), func(t *testing.T) {
			got := hcl2ValueFromConfigValue(test.Input)
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, got, test.Want)
			}
		})
	}
}
