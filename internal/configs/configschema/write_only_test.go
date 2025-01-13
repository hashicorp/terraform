// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestBlock_WriteOnlyPaths(t *testing.T) {
	schema := &Block{
		Attributes: map[string]*Attribute{
			"not_wo": {
				Type:     cty.String,
				Optional: true,
			},
			"wo": {
				Type:      cty.String,
				WriteOnly: true,
			},
			"nested": {
				NestedType: &Object{
					Attributes: map[string]*Attribute{
						"boop": {
							Type: cty.String,
						},
						"honk": {
							Type:      cty.String,
							WriteOnly: true,
						},
					},
					Nesting: NestingList,
				},
			},
		},

		BlockTypes: map[string]*NestedBlock{
			"list": {
				Nesting: NestingList,
				Block: Block{
					Attributes: map[string]*Attribute{
						"not_wo": {
							Type:     cty.String,
							Optional: true,
						},
						"wo": {
							Type:      cty.String,
							WriteOnly: true,
						},
					},
				},
			},
		},
	}

	testCases := map[string]struct {
		value    cty.Value
		expected []cty.Path
	}{
		"unknown value": {
			cty.UnknownVal(schema.ImpliedType()),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "nested"}, cty.GetAttrStep{Name: "honk"}},
			},
		},
		"null object": {
			cty.NullVal(schema.ImpliedType()),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "nested"}, cty.GetAttrStep{Name: "honk"}},
			},
		},
		"object with unknown attributes and blocks": {
			cty.ObjectVal(map[string]cty.Value{
				"wo":     cty.UnknownVal(cty.String),
				"not_wo": cty.UnknownVal(cty.String),
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"list": cty.UnknownVal(schema.BlockTypes["list"].ImpliedType()),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "nested"}, cty.GetAttrStep{Name: "honk"}},
			},
		},
		"object with block value": {
			cty.ObjectVal(map[string]cty.Value{
				"wo":     cty.NullVal(cty.String),
				"not_wo": cty.UnknownVal(cty.String),
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"wo":     cty.UnknownVal(cty.String),
						"not_wo": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"wo":     cty.NullVal(cty.String),
						"not_wo": cty.NullVal(cty.String),
					}),
				}),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "nested"}, cty.GetAttrStep{Name: "honk"}},
				{cty.GetAttrStep{Name: "list"}, cty.IndexStep{Key: cty.NumberIntVal(0)}, cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "list"}, cty.IndexStep{Key: cty.NumberIntVal(1)}, cty.GetAttrStep{Name: "wo"}},
			},
		},
		"object with known values and nested attribute": {
			cty.ObjectVal(map[string]cty.Value{
				"wo":     cty.StringVal("foo"),
				"not_wo": cty.StringVal("bar"),
				"nested": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.StringVal("foo"),
						"honk": cty.StringVal("bar"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.NullVal(cty.String),
						"honk": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.UnknownVal(cty.String),
						"honk": cty.UnknownVal(cty.String),
					}),
				}),
				"list": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"sensitive":   cty.String,
					"unsensitive": cty.String,
				}))),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "nested"}, cty.GetAttrStep{Name: "honk"}},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			woPaths := schema.WriteOnlyPaths(tc.value, nil)
			if !cty.NewPathSet(tc.expected...).Equal(cty.NewPathSet(woPaths...)) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.expected, woPaths)
			}
		})
	}
}
