package schema

import (
	"bytes"
	"testing"
)

func TestSerializeForHash(t *testing.T) {
	type testCase struct {
		Schema   interface{}
		Value    interface{}
		Expected string
	}

	tests := []testCase{
		{
			Schema: &Schema{
				Type: TypeInt,
			},
			Value:    0,
			Expected: "0;",
		},

		{
			Schema: &Schema{
				Type: TypeInt,
			},
			Value:    200,
			Expected: "200;",
		},

		{
			Schema: &Schema{
				Type: TypeBool,
			},
			Value:    true,
			Expected: "1;",
		},

		{
			Schema: &Schema{
				Type: TypeBool,
			},
			Value:    false,
			Expected: "0;",
		},

		{
			Schema: &Schema{
				Type: TypeFloat,
			},
			Value:    1.0,
			Expected: "1;",
		},

		{
			Schema: &Schema{
				Type: TypeFloat,
			},
			Value:    1.54,
			Expected: "1.54;",
		},

		{
			Schema: &Schema{
				Type: TypeFloat,
			},
			Value:    0.1,
			Expected: "0.1;",
		},

		{
			Schema: &Schema{
				Type: TypeString,
			},
			Value:    "hello",
			Expected: "hello;",
		},

		{
			Schema: &Schema{
				Type: TypeString,
			},
			Value:    "1",
			Expected: "1;",
		},

		{
			Schema: &Schema{
				Type: TypeList,
				Elem: &Schema{
					Type: TypeString,
				},
			},
			Value:    []interface{}{},
			Expected: "();",
		},

		{
			Schema: &Schema{
				Type: TypeList,
				Elem: &Schema{
					Type: TypeString,
				},
			},
			Value:    []interface{}{"hello", "world"},
			Expected: "(hello;world;);",
		},

		{
			Schema: &Schema{
				Type: TypeList,
				Elem: &Resource{
					Schema: map[string]*Schema{
						"fo": {
							Type:     TypeString,
							Required: true,
						},
						"fum": {
							Type:     TypeString,
							Required: true,
						},
					},
				},
			},
			Value: []interface{}{
				map[string]interface{}{
					"fo": "bar",
				},
				map[string]interface{}{
					"fo":  "baz",
					"fum": "boz",
				},
			},
			Expected: "(<fo:bar;fum:;>;<fo:baz;fum:boz;>;);",
		},

		{
			Schema: &Schema{
				Type: TypeSet,
				Elem: &Schema{
					Type: TypeString,
				},
			},
			Value: NewSet(func(i interface{}) int { return len(i.(string)) }, []interface{}{
				"hello",
				"woo",
			}),
			Expected: "{woo;hello;};",
		},

		{
			Schema: &Schema{
				Type: TypeMap,
				Elem: &Schema{
					Type: TypeString,
				},
			},
			Value: map[string]interface{}{
				"foo": "bar",
				"baz": "foo",
			},
			Expected: "[baz:foo;foo:bar;];",
		},

		{
			Schema: &Resource{
				Schema: map[string]*Schema{
					"name": {
						Type:     TypeString,
						Required: true,
					},
					"size": {
						Type:     TypeInt,
						Optional: true,
					},
					"green": {
						Type:     TypeBool,
						Optional: true,
						Computed: true,
					},
					"upside_down": {
						Type:     TypeBool,
						Computed: true,
					},
				},
			},
			Value: map[string]interface{}{
				"name":  "my-fun-database",
				"size":  12,
				"green": true,
			},
			Expected: "green:1;name:my-fun-database;size:12;",
		},

		// test TypeMap nested in Schema: GH-7091
		{
			Schema: &Resource{
				Schema: map[string]*Schema{
					"outer": {
						Type:     TypeSet,
						Required: true,
						Elem: &Schema{
							Type:     TypeMap,
							Optional: true,
						},
					},
				},
			},
			Value: map[string]interface{}{
				"outer": NewSet(func(i interface{}) int { return 42 }, []interface{}{
					map[string]interface{}{
						"foo": "bar",
						"baz": "foo",
					},
				}),
			},
			Expected: "outer:{[baz:foo;foo:bar;];};",
		},
	}

	for _, test := range tests {
		var gotBuf bytes.Buffer
		schema := test.Schema

		switch s := schema.(type) {
		case *Schema:
			SerializeValueForHash(&gotBuf, test.Value, s)
		case *Resource:
			SerializeResourceForHash(&gotBuf, test.Value, s)
		}

		got := gotBuf.String()
		if got != test.Expected {
			t.Errorf("hash(%#v) got %#v, but want %#v", test.Value, got, test.Expected)
		}
	}
}
