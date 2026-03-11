// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package globalref

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestWalkBlock(t *testing.T) {

	primitiveAttribute := &configschema.Attribute{
		Type: cty.String,
	}

	nestedAttributes := map[string]*configschema.Attribute{
		"primitive": primitiveAttribute,
		"0":         primitiveAttribute,
	}

	simpleBlock := configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"primitive": primitiveAttribute,
			"0":         primitiveAttribute,
			"object": {
				Type: cty.Object(map[string]cty.Type{
					"primitive": cty.String,
					"0":         cty.String,
				}),
			},
			"list": {
				Type: cty.List(cty.String),
			},
			"set": {
				Type: cty.Set(cty.String),
			},
			"map": {
				Type: cty.Map(cty.String),
			},
			"nested_single": {
				NestedType: &configschema.Object{
					Attributes: nestedAttributes,
					Nesting:    configschema.NestingSingle,
				},
			},
			"nested_list": {
				NestedType: &configschema.Object{
					Attributes: nestedAttributes,
					Nesting:    configschema.NestingList,
				},
			},
			"nested_set": {
				NestedType: &configschema.Object{
					Attributes: nestedAttributes,
					Nesting:    configschema.NestingSet,
				},
			},
			"nested_map": {
				NestedType: &configschema.Object{
					Attributes: nestedAttributes,
					Nesting:    configschema.NestingMap,
				},
			},
		},
	}

	schema := &configschema.Block{
		Attributes: simpleBlock.Attributes,
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested_single_block": {
				Block:   simpleBlock,
				Nesting: configschema.NestingSingle,
			},
			"nested_list_block": {
				Block:   simpleBlock,
				Nesting: configschema.NestingList,
			},
			"nested_set_block": {
				Block:   simpleBlock,
				Nesting: configschema.NestingSet,
			},
			"nested_map_block": {
				Block:   simpleBlock,
				Nesting: configschema.NestingMap,
			},
		},
	}

	tcs := map[string]struct {
		traversal string
		want      string
	}{
		"empty": {
			traversal: "resource_type.resource_name",
		},

		// normal types

		"primitive": {
			traversal: "resource_type.resource_name.primitive",
			want:      ".primitive",
		},
		"primitive (wrong)": {
			traversal: "resource_type.resource_name.primitive.extra",
			want:      ".primitive",
		},
		"object": {
			traversal: "resource_type.resource_name.object.primitive",
			want:      ".object.primitive",
		},
		"object (missing)": {
			traversal: "resource_type.resource_name.object.missing",
			want:      ".object",
		},
		"object (valid string index)": {
			traversal: "resource_type.resource_name.object[\"primitive\"]",
			want:      ".object.primitive",
		},
		"object (invalid string index)": {
			traversal: "resource_type.resource_name.object[\"missing\"]",
			want:      ".object",
		},
		"object (valid number index)": {
			traversal: "resource_type.resource_name.object[0]",
			want:      ".object.0",
		},
		"object (invalid number index)": {
			traversal: "resource_type.resource_name.object[1]",
			want:      ".object",
		},
		"list": {
			traversal: "resource_type.resource_name.list[0]",
			want:      ".list[0]",
		},
		"list (string index)": {
			traversal: "resource_type.resource_name.list[\"key\"]",
			want:      ".list",
		},
		"list (valid string index)": {
			traversal: "resource_type.resource_name.list[\"0\"]",
			want:      ".list[0]",
		},
		"list (string attribute)": {
			traversal: "resource_type.resource_name.list.primitive",
			want:      ".list",
		},
		"set (integer index)": {
			traversal: "resource_type.resource_name.set[0]",
			want:      ".set",
		},
		"set (string index)": {
			traversal: "resource_type.resource_name.set[\"key\"]",
			want:      ".set",
		},
		"set (string attribute)": {
			traversal: "resource_type.resource_name.set.primitive",
			want:      ".set",
		},
		"map": {
			traversal: "resource_type.resource_name.map[\"key\"]",
			want:      ".map[\"key\"]",
		},
		"map (integer index)": {
			traversal: "resource_type.resource_name.map[0]",
			want:      ".map[\"0\"]",
		},
		"map (string attribute)": {
			traversal: "resource_type.resource_name.map.key",
			want:      ".map[\"key\"]",
		},

		// nested types

		"nested object": {
			traversal: "resource_type.resource_name.nested_single.primitive",
			want:      ".nested_single.primitive",
		},
		"nested object (missing)": {
			traversal: "resource_type.resource_name.nested_single.missing",
			want:      ".nested_single",
		},
		"nested object (valid string index)": {
			traversal: "resource_type.resource_name.nested_single[\"primitive\"]",
			want:      ".nested_single.primitive",
		},
		"nested object (invalid string index)": {
			traversal: "resource_type.resource_name.nested_single[\"missing\"]",
			want:      ".nested_single",
		},
		"nested object (valid number index)": {
			traversal: "resource_type.resource_name.nested_single[0]",
			want:      ".nested_single.0",
		},
		"nested object (invalid number index)": {
			traversal: "resource_type.resource_name.nested_single[1]",
			want:      ".nested_single",
		},
		"nested list": {
			traversal: "resource_type.resource_name.nested_list[0]",
			want:      ".nested_list[0]",
		},
		"nested list (string index)": {
			traversal: "resource_type.resource_name.nested_list[\"key\"]",
			want:      ".nested_list",
		},
		"nested list (valid string index)": {
			traversal: "resource_type.resource_name.nested_list[\"0\"]",
			want:      ".nested_list[0]",
		},
		"nested list (string attribute)": {
			traversal: "resource_type.resource_name.nested_list.primitive",
			want:      ".nested_list",
		},
		"nested set (integer index)": {
			traversal: "resource_type.resource_name.nested_set[0]",
			want:      ".nested_set",
		},
		"nested set (string index)": {
			traversal: "resource_type.resource_name.nested_set[\"key\"]",
			want:      ".nested_set",
		},
		"nested set (string attribute)": {
			traversal: "resource_type.resource_name.nested_set.primitive",
			want:      ".nested_set",
		},
		"nested map": {
			traversal: "resource_type.resource_name.nested_map[\"key\"]",
			want:      ".nested_map[\"key\"]",
		},
		"nested map (integer index)": {
			traversal: "resource_type.resource_name.nested_map[0]",
			want:      ".nested_map[\"0\"]",
		},
		"nested map (string attribute)": {
			traversal: "resource_type.resource_name.nested_map.key",
			want:      ".nested_map[\"key\"]",
		},

		// blocks

		"nested object block": {
			traversal: "resource_type.resource_name.nested_single_block.primitive",
			want:      ".nested_single_block.primitive",
		},
		"nested object block (missing)": {
			traversal: "resource_type.resource_name.nested_single_block.missing",
			want:      ".nested_single_block",
		},
		"nested object block (valid string index)": {
			traversal: "resource_type.resource_name.nested_single_block[\"primitive\"]",
			want:      ".nested_single_block.primitive",
		},
		"nested object block (invalid string index)": {
			traversal: "resource_type.resource_name.nested_single_block[\"missing\"]",
			want:      ".nested_single_block",
		},
		"nested object block (valid number index)": {
			traversal: "resource_type.resource_name.nested_single_block[0]",
			want:      ".nested_single_block.0",
		},
		"nested object block (invalid number index)": {
			traversal: "resource_type.resource_name.nested_single_block[1]",
			want:      ".nested_single_block",
		},
		"nested list block": {
			traversal: "resource_type.resource_name.nested_list_block[0].primitive",
			want:      ".nested_list_block[0].primitive",
		},
		"nested list block (invalid string index)": {
			traversal: "resource_type.resource_name.nested_list_block[\"index\"].primitive",
			want:      ".nested_list_block",
		},
		"nested list block (valid string index)": {
			traversal: "resource_type.resource_name.nested_list_block[\"0\"].primitive",
			want:      ".nested_list_block[0].primitive",
		},
		"nested list block (string attribute)": {
			traversal: "resource_type.resource_name.nested_list_block.primitive",
			want:      ".nested_list_block",
		},
		"nested set block (integer index)": {
			traversal: "resource_type.resource_name.nested_set_block[0].primitive",
			want:      ".nested_set_block",
		},
		"nested set block (string index)": {
			traversal: "resource_type.resource_name.nested_set_block[\"index\"].primitive",
			want:      ".nested_set_block",
		},
		"nested set block (string attribute)": {
			traversal: "resource_type.resource_name.nested_set_block.primitive",
			want:      ".nested_set_block",
		},
		"nested map block": {
			traversal: "resource_type.resource_name.nested_map_block[\"key\"].primitive",
			want:      ".nested_map_block[\"key\"].primitive",
		},
		"nested map block (integer index)": {
			traversal: "resource_type.resource_name.nested_map_block[0].primitive",
			want:      ".nested_map_block[\"0\"].primitive",
		},
		"nested map block (string attribute)": {
			traversal: "resource_type.resource_name.nested_map_block.key.primitive",
			want:      ".nested_map_block[\"key\"].primitive",
		},

		// resource instances

		"resource instance reference (count)": {
			traversal: "resource_type.resource_name[0].primitive",
			want:      ".primitive",
		},
		"resource instance reference (for each)": {
			traversal: "resource_type.resource_name[\"key\"].primitive",
			want:      ".primitive",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ref, diags := addrs.ParseRefStr(tc.traversal)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			ret := walkBlock(schema, ref.Remaining)

			got := tfdiags.TraversalStr(ret)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diags, diff)
			}
		})
	}
}
