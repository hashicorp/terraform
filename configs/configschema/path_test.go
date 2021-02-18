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
