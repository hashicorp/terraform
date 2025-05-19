// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TestStripRemovedStateAttributes(t *testing.T) {
	cases := []struct {
		name     string
		state    map[string]interface{}
		expect   map[string]interface{}
		ty       cty.Type
		modified bool
	}{
		{
			"removed string",
			map[string]interface{}{
				"a": "ok",
				"b": "gone",
			},
			map[string]interface{}{
				"a": "ok",
			},
			cty.Object(map[string]cty.Type{
				"a": cty.String,
			}),
			true,
		},
		{
			"removed null",
			map[string]interface{}{
				"a": "ok",
				"b": nil,
			},
			map[string]interface{}{
				"a": "ok",
			},
			cty.Object(map[string]cty.Type{
				"a": cty.String,
			}),
			true,
		},
		{
			"has large number",
			map[string]interface{}{
				"a": "ok",
				"b": nil,
			},
			map[string]interface{}{
				"a": "ok",
			},
			cty.Object(map[string]cty.Type{
				"a": cty.String,
			}),
			true,
		},
		{
			"removed nested string",
			map[string]interface{}{
				"a": "ok",
				"b": map[string]interface{}{
					"a": "ok",
					"b": "removed",
				},
			},
			map[string]interface{}{
				"a": "ok",
				"b": map[string]interface{}{
					"a": "ok",
				},
			},
			cty.Object(map[string]cty.Type{
				"a": cty.String,
				"b": cty.Object(map[string]cty.Type{
					"a": cty.String,
				}),
			}),
			true,
		},
		{
			"removed nested list",
			map[string]interface{}{
				"a": "ok",
				"b": map[string]interface{}{
					"a": "ok",
					"b": []interface{}{"removed"},
				},
			},
			map[string]interface{}{
				"a": "ok",
				"b": map[string]interface{}{
					"a": "ok",
				},
			},
			cty.Object(map[string]cty.Type{
				"a": cty.String,
				"b": cty.Object(map[string]cty.Type{
					"a": cty.String,
				}),
			}),
			true,
		},
		{
			"removed keys in set of objs",
			map[string]interface{}{
				"a": "ok",
				"b": map[string]interface{}{
					"a": "ok",
					"set": []interface{}{
						map[string]interface{}{
							"x": "ok",
							"y": "removed",
						},
						map[string]interface{}{
							"x": "ok",
							"y": "removed",
						},
					},
				},
			},
			map[string]interface{}{
				"a": "ok",
				"b": map[string]interface{}{
					"a": "ok",
					"set": []interface{}{
						map[string]interface{}{
							"x": "ok",
						},
						map[string]interface{}{
							"x": "ok",
						},
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"a": cty.String,
				"b": cty.Object(map[string]cty.Type{
					"a": cty.String,
					"set": cty.Set(cty.Object(map[string]cty.Type{
						"x": cty.String,
					})),
				}),
			}),
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			modified := removeRemovedAttrs(tc.state, tc.ty)
			if !reflect.DeepEqual(tc.state, tc.expect) {
				t.Fatalf("expected: %#v\n      got: %#v\n", tc.expect, tc.state)
			}
			if modified != tc.modified {
				t.Fatal("incorrect return value")
			}
		})
	}
}

func TestStripRemovedStateAttributesDecoder(t *testing.T) {
	cases := []struct {
		name   string
		state  string
		expect cty.Value
	}{
		{
			"removed string",
			`{"a": "ok","b": "gone"}`,
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("ok"),
			}),
		},
		{
			"removed null",
			`{"a": "ok","b": "gone"}`,
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("ok"),
			}),
		},
		{
			"with large number",
			`{"a": 123456789123456789.123456789,"b": "gone"}`,
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MustParseNumberVal("123456789123456789.123456789"),
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upgraded := stripRemovedStateAttributes([]byte(tc.state), tc.expect.Type())
			got, err := ctyjson.Unmarshal(upgraded, tc.expect.Type())
			if err != nil {
				t.Fatal(err)
			}

			if !tc.expect.RawEquals(got) {
				t.Fatalf("expected: %#v\n      got: %#v\n", tc.expect, got)
			}
		})
	}
}
