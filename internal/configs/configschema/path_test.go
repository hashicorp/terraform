// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestAttributeByPath(t *testing.T) {
	schema := &Block{
		Attributes: map[string]*Attribute{
			"a1": {Description: "a1"},
			"a2": {Description: "a2"},
			"a3": {
				Description: "a3",
				NestedType: &Object{
					Nesting: NestingList,
					Attributes: map[string]*Attribute{
						"nt1": {Description: "nt1"},
						"nt2": {
							Description: "nt2",
							NestedType: &Object{
								Nesting: NestingSingle,
								Attributes: map[string]*Attribute{
									"deeply_nested": {Description: "deeply_nested"},
								},
							},
						},
					},
				},
			},
		},
		BlockTypes: map[string]*NestedBlock{
			"b1": {
				Nesting: NestingList,
				Block: Block{
					Attributes: map[string]*Attribute{
						"a3": {Description: "a3"},
						"a4": {Description: "a4"},
					},
					BlockTypes: map[string]*NestedBlock{
						"b2": {
							Nesting: NestingMap,
							Block: Block{
								Attributes: map[string]*Attribute{
									"a5": {Description: "a5"},
									"a6": {Description: "a6"},
								},
							},
						},
					},
				},
			},
			"b3": {
				Nesting: NestingMap,
				Block: Block{
					Attributes: map[string]*Attribute{
						"a7": {Description: "a7"},
						"a8": {Description: "a8"},
					},
					BlockTypes: map[string]*NestedBlock{
						"b4": {
							Nesting: NestingSet,
							Block: Block{
								Attributes: map[string]*Attribute{
									"a9":  {Description: "a9"},
									"a10": {Description: "a10"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range []struct {
		path            cty.Path
		attrDescription string
		exists          bool
	}{
		{
			cty.GetAttrPath("a2"),
			"a2",
			true,
		},
		{
			cty.GetAttrPath("a3").IndexInt(1).GetAttr("nt2"),
			"nt2",
			true,
		},
		{
			cty.GetAttrPath("a3").IndexInt(1).GetAttr("b2").IndexString("foo").GetAttr("no"),
			"missing",
			false,
		},
		{
			cty.GetAttrPath("b1"),
			"block",
			false,
		},
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("a3"),
			"a3",
			true,
		},
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("b2").IndexString("foo").GetAttr("a7"),
			"missing",
			false,
		},
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("b2").IndexString("foo").GetAttr("a6"),
			"a6",
			true,
		},
		{
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("b2").IndexString("foo").GetAttr("a7"),
			"missing_block",
			false,
		},
		{
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("a7"),
			"a7",
			true,
		},
		{
			// Index steps don't apply to the schema, so the set Index value doesn't matter.
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("b4").Index(cty.EmptyObjectVal).GetAttr("a9"),
			"a9",
			true,
		},
	} {
		t.Run(tc.attrDescription, func(t *testing.T) {
			attr := schema.AttributeByPath(tc.path)
			if !tc.exists && attr == nil {
				return
			}

			if attr == nil {
				t.Fatalf("missing attribute from path %#v\n", tc.path)
			}

			if attr.Description != tc.attrDescription {
				t.Fatalf("expected Attribute for %q, got %#v\n", tc.attrDescription, attr)
			}
		})
	}
}

func TestObject_AttributeByPath(t *testing.T) {
	obj := &Object{
		Nesting: NestingList,
		Attributes: map[string]*Attribute{
			"a1": {Description: "a1"},
			"a2": {
				Description: "a2",
				NestedType: &Object{
					Nesting: NestingSingle,
					Attributes: map[string]*Attribute{
						"n1": {Description: "n1"},
						"n2": {
							Description: "n2",
							NestedType: &Object{
								Attributes: map[string]*Attribute{
									"dn1": {Description: "dn1"},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		path            cty.Path
		attrDescription string
		exists          bool
	}{
		{
			cty.GetAttrPath("a2"),
			"a2",
			true,
		},
		{
			cty.GetAttrPath("a3"),
			"missing",
			false,
		},
		{
			cty.GetAttrPath("a2").IndexString("foo").GetAttr("n1"),
			"n1",
			true,
		},
		{
			cty.GetAttrPath("a2").IndexString("foo").GetAttr("n2").IndexInt(11).GetAttr("dn1"),
			"dn1",
			true,
		},
		{
			cty.GetAttrPath("a2").IndexString("foo").GetAttr("n2").IndexInt(11).GetAttr("dn1").IndexString("hello").GetAttr("nope"),
			"missing_nested",
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.attrDescription, func(t *testing.T) {
			attr := obj.AttributeByPath(tc.path)
			if !tc.exists && attr == nil {
				return
			}

			if !tc.exists && attr != nil {
				t.Fatalf("found Attribute, expected nil from path %#v\n", tc.path)
			}

			if attr == nil {
				t.Fatalf("missing attribute from path %#v\n", tc.path)
			}

			if attr.Description != tc.attrDescription {
				t.Fatalf("expected Attribute for %q, got %#v\n", tc.attrDescription, attr)
			}
		})
	}

}
