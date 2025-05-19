// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestBlockValueMarks(t *testing.T) {
	schema := &Block{
		Attributes: map[string]*Attribute{
			"unsensitive": {
				Type:     cty.String,
				Optional: true,
			},
			"sensitive": {
				Type:      cty.String,
				Sensitive: true,
			},
			"nested": {
				NestedType: &Object{
					Attributes: map[string]*Attribute{
						"boop": {
							Type: cty.String,
						},
						"honk": {
							Type:      cty.String,
							Sensitive: true,
						},
					},
					Nesting: NestingList,
				},
			},
			"nested_sensitive": {
				NestedType: &Object{
					Attributes: map[string]*Attribute{
						"boop": {
							Type: cty.String,
						},
					},
					Nesting: NestingList,
				},
				Sensitive: true,
				Optional:  true,
			},
		},

		BlockTypes: map[string]*NestedBlock{
			"list": {
				Nesting: NestingList,
				Block: Block{
					Attributes: map[string]*Attribute{
						"unsensitive": {
							Type:     cty.String,
							Optional: true,
						},
						"sensitive": {
							Type:      cty.String,
							Sensitive: true,
						},
					},
				},
			},
		},
	}

	testCases := map[string]struct {
		given  cty.Value
		expect cty.Value
	}{
		"unknown object": {
			cty.UnknownVal(schema.ImpliedType()),
			cty.UnknownVal(schema.ImpliedType()),
		},
		"null object": {
			cty.NullVal(schema.ImpliedType()),
			cty.NullVal(schema.ImpliedType()),
		},
		"object with unknown attributes and blocks": {
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.UnknownVal(cty.String),
				"unsensitive": cty.UnknownVal(cty.String),
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"nested_sensitive": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
				}))),
				"list": cty.UnknownVal(schema.BlockTypes["list"].ImpliedType()),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.UnknownVal(cty.String).Mark(marks.Sensitive),
				"unsensitive": cty.UnknownVal(cty.String),
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"nested_sensitive": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
				}))).Mark(marks.Sensitive),
				"list": cty.UnknownVal(schema.BlockTypes["list"].ImpliedType()),
			}),
		},
		"object with block value": {
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.NullVal(cty.String),
				"unsensitive": cty.UnknownVal(cty.String),
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"nested_sensitive": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"sensitive":   cty.UnknownVal(cty.String),
						"unsensitive": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"sensitive":   cty.NullVal(cty.String),
						"unsensitive": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.NullVal(cty.String).Mark(marks.Sensitive),
				"unsensitive": cty.UnknownVal(cty.String),
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
					"honk": cty.String,
				}))),
				"nested_sensitive": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
				}))).Mark(marks.Sensitive),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"sensitive":   cty.UnknownVal(cty.String).Mark(marks.Sensitive),
						"unsensitive": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"sensitive":   cty.NullVal(cty.String).Mark(marks.Sensitive),
						"unsensitive": cty.NullVal(cty.String),
					}),
				}),
			}),
		},
		"object with known values and nested attribute": {
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.StringVal("foo"),
				"unsensitive": cty.StringVal("bar"),
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
				"nested_sensitive": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
				}))),
				"list": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"sensitive":   cty.String,
					"unsensitive": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.StringVal("foo").Mark(marks.Sensitive),
				"unsensitive": cty.StringVal("bar"),
				"nested": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.StringVal("foo"),
						"honk": cty.StringVal("bar").Mark(marks.Sensitive),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.NullVal(cty.String),
						"honk": cty.NullVal(cty.String).Mark(marks.Sensitive),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.UnknownVal(cty.String),
						"honk": cty.UnknownVal(cty.String).Mark(marks.Sensitive),
					}),
				}),
				"nested_sensitive": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"boop": cty.String,
				}))).Mark(marks.Sensitive),
				"list": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"sensitive":   cty.String,
					"unsensitive": cty.String,
				}))),
			}),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			given, err := schema.CoerceValue(tc.given)
			if err != nil {
				t.Fatal(err)
			}
			expect, err := schema.CoerceValue(tc.expect)
			if err != nil {
				t.Fatal(err)
			}

			sensitivePaths := schema.SensitivePaths(given, nil)
			got := marks.MarkPaths(given, marks.Sensitive, sensitivePaths)
			if !expect.RawEquals(got) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", expect, got)
			}
		})
	}
}
