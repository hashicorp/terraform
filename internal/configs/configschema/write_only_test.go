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
				Optional:  true,
				WriteOnly: true,
			},
			"nested": {
				Optional: true,
				NestedType: &Object{
					Attributes: map[string]*Attribute{
						"boop": {
							Type:     cty.String,
							Optional: true,
						},
						"honk": {
							Type:      cty.String,
							Optional:  true,
							WriteOnly: true,
						},
					},
					Nesting: NestingList,
				},
			},
			"single": {
				Optional: true,
				NestedType: &Object{
					Nesting: NestingSingle,
					Attributes: map[string]*Attribute{
						"not_wo": {
							Optional: true,
							Type:     cty.String,
						},
						"wo": {
							Type:      cty.String,
							Optional:  true,
							WriteOnly: true,
						},
						"nested_single": {
							Optional: true,
							NestedType: &Object{
								Nesting: NestingSingle,
								Attributes: map[string]*Attribute{
									"not_wo": {
										Optional: true,
										Type:     cty.String,
									},
									"wo": {
										Type:      cty.String,
										Optional:  true,
										WriteOnly: true,
									},
								},
							},
						},
						"single_wo": {
							Optional:  true,
							WriteOnly: true,
							NestedType: &Object{
								Nesting: NestingSingle,
								Attributes: map[string]*Attribute{
									"not_wo": {
										Optional: true,
										Type:     cty.String,
									},
								},
							},
						},
					},
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
							Optional:  true,
							WriteOnly: true,
						},
					},
				},
			},
			"single_block": {
				Nesting: NestingSingle,
				Block: Block{
					Attributes: map[string]*Attribute{
						"not_wo": {
							Type:     cty.String,
							Optional: true,
						},
						"wo": {
							Type:      cty.String,
							Optional:  true,
							WriteOnly: true,
						},
					},
				},
			}},
	}

	testCases := map[string]struct {
		value    cty.Value
		expected []cty.Path
	}{
		"unknown value": {
			cty.UnknownVal(schema.ImpliedType()),
			[]cty.Path{},
		},
		"null object": {
			cty.NullVal(schema.ImpliedType()),
			[]cty.Path{},
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
			},
		},
		"object with block value": {
			cty.ObjectVal(map[string]cty.Value{
				"wo":     cty.NullVal(cty.String),
				"not_wo": cty.UnknownVal(cty.String),
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
					"not_wo": cty.String,
					"wo":     cty.String,
				}))),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "nested"}, cty.IndexStep{Key: cty.NumberIntVal(0)}, cty.GetAttrStep{Name: "honk"}},
				{cty.GetAttrStep{Name: "nested"}, cty.IndexStep{Key: cty.NumberIntVal(1)}, cty.GetAttrStep{Name: "honk"}},
				{cty.GetAttrStep{Name: "nested"}, cty.IndexStep{Key: cty.NumberIntVal(2)}, cty.GetAttrStep{Name: "honk"}},
			},
		},
		"object with single nested block and attribute": {
			cty.ObjectVal(map[string]cty.Value{
				"wo":     cty.StringVal("foo"),
				"not_wo": cty.StringVal("bar"),
				"single": cty.ObjectVal(map[string]cty.Value{
					"not_wo": cty.StringVal("foo"),
					"wo":     cty.StringVal("bar"),
				}),
				"single_block": cty.ObjectVal(map[string]cty.Value{
					"not_wo": cty.StringVal("test"),
					"wo":     cty.StringVal("secret"),
				}),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "single"}, cty.GetAttrStep{Name: "wo"}},
				cty.GetAttrPath("single").GetAttr(("single_wo")),
				{cty.GetAttrStep{Name: "single_block"}, cty.GetAttrStep{Name: "wo"}},
			},
		},
		"object with doubly single-nested attribute": {
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"not_wo": cty.StringVal("foo"),
					"wo":     cty.NullVal(cty.String),
					"nested_single": cty.ObjectVal(map[string]cty.Value{
						"not_wo": cty.StringVal("foo"),
						"wo":     cty.NullVal(cty.String),
					}),
				}),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
				{cty.GetAttrStep{Name: "single"}, cty.GetAttrStep{Name: "wo"}},
				cty.GetAttrPath("single").GetAttr(("single_wo")),
				{cty.GetAttrStep{Name: "single"}, cty.GetAttrStep{Name: "nested_single"}, cty.GetAttrStep{Name: "wo"}},
			},
		},
		"single nested write-only attr": {
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"single_wo": cty.ObjectVal(map[string]cty.Value{
						"not_wo": cty.StringVal("foo").Mark("test"),
					}),
				}),
			}),
			[]cty.Path{
				cty.GetAttrPath("wo"),
				cty.GetAttrPath("single").GetAttr(("wo")),
				cty.GetAttrPath("single").GetAttr(("single_wo")),
			},
		},
		"single nested null write-only attr": {
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.NullVal(cty.Object(map[string]cty.Type{
					"not_wo": cty.String,
					"wo":     cty.String,
					"nested_single": cty.Object(map[string]cty.Type{
						"not_wo": cty.String,
						"wo":     cty.String,
					}),
					"single_wo": cty.Object(map[string]cty.Type{
						"not_wo": cty.String,
					}),
				})),
			}),
			[]cty.Path{
				{cty.GetAttrStep{Name: "wo"}},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := schema.InternalValidate(); err != nil {
				t.Fatal(err)
			}
			val, err := schema.CoerceValue(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			woPaths := schema.WriteOnlyPaths(val, nil)
			if !cty.NewPathSet(tc.expected...).Equal(cty.NewPathSet(woPaths...)) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.expected, woPaths)
			}
		})
	}
}
