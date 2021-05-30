package configschema

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestStaticValidateTraversal(t *testing.T) {
	attrs := map[string]*Attribute{
		"str":  {Type: cty.String, Optional: true},
		"list": {Type: cty.List(cty.String), Optional: true},
		"dyn":  {Type: cty.DynamicPseudoType, Optional: true},
	}
	schema := &Block{
		Attributes: attrs,
		BlockTypes: map[string]*NestedBlock{
			"single_block": {
				Nesting: NestingSingle,
				Block: Block{
					Attributes: attrs,
				},
			},
			"list_block": {
				Nesting: NestingList,
				Block: Block{
					Attributes: attrs,
				},
			},
			"set_block": {
				Nesting: NestingSet,
				Block: Block{
					Attributes: attrs,
				},
			},
			"map_block": {
				Nesting: NestingMap,
				Block: Block{
					Attributes: attrs,
				},
			},
		},
	}

	tests := []struct {
		Traversal string
		WantError string
	}{
		{
			`obj`,
			``,
		},
		{
			`obj.str`,
			``,
		},
		{
			`obj.str.nonexist`,
			`Unsupported attribute: This value does not have any attributes.`,
		},
		{
			`obj.list`,
			``,
		},
		{
			`obj.list[0]`,
			``,
		},
		{
			`obj.list.nonexist`,
			`Unsupported attribute: This value does not have any attributes.`,
		},
		{
			`obj.dyn`,
			``,
		},
		{
			`obj.dyn.anything_goes`,
			``,
		},
		{
			`obj.dyn[0]`,
			``,
		},
		{
			`obj.nonexist`,
			`Unsupported attribute: This object has no argument, nested block, or exported attribute named "nonexist".`,
		},
		{
			`obj[1]`,
			`Invalid index operation: Only attribute access is allowed here, using the dot operator.`,
		},
		{
			`obj["str"]`, // we require attribute access for the first step to avoid ambiguity with resource instance indices
			`Invalid index operation: Only attribute access is allowed here. Did you mean to access attribute "str" using the dot operator?`,
		},
		{
			`obj.atr`,
			`Unsupported attribute: This object has no argument, nested block, or exported attribute named "atr". Did you mean "str"?`,
		},
		{
			`obj.single_block`,
			``,
		},
		{
			`obj.single_block.str`,
			``,
		},
		{
			`obj.single_block.nonexist`,
			`Unsupported attribute: This object has no argument, nested block, or exported attribute named "nonexist".`,
		},
		{
			`obj.list_block`,
			``,
		},
		{
			`obj.list_block[0]`,
			``,
		},
		{
			`obj.list_block[0].str`,
			``,
		},
		{
			`obj.list_block[0].nonexist`,
			`Unsupported attribute: This object has no argument, nested block, or exported attribute named "nonexist".`,
		},
		{
			`obj.list_block.str`,
			`Invalid operation: Block type "list_block" is represented by a list of objects, so it must be indexed using a numeric key, like .list_block[0].`,
		},
		{
			`obj.set_block`,
			``,
		},
		{
			`obj.set_block[0]`,
			`Cannot index a set value: Block type "set_block" is represented by a set of objects, and set elements do not have addressable keys. To find elements matching specific criteria, use a "for" expression with an "if" clause.`,
		},
		{
			`obj.set_block.str`,
			`Cannot index a set value: Block type "set_block" is represented by a set of objects, and set elements do not have addressable keys. To find elements matching specific criteria, use a "for" expression with an "if" clause.`,
		},
		{
			`obj.map_block`,
			``,
		},
		{
			`obj.map_block.anything`,
			``,
		},
		{
			`obj.map_block["anything"]`,
			``,
		},
		{
			`obj.map_block.anything.str`,
			``,
		},
		{
			`obj.map_block["anything"].str`,
			``,
		},
		{
			`obj.map_block.anything.nonexist`,
			`Unsupported attribute: This object has no argument, nested block, or exported attribute named "nonexist".`,
		},
	}

	for _, test := range tests {
		t.Run(test.Traversal, func(t *testing.T) {
			traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(test.Traversal), "", hcl.Pos{Line: 1, Column: 1})
			for _, diag := range parseDiags {
				t.Error(diag.Error())
			}

			// We trim the "obj." portion from the front since StaticValidateTraversal
			// only works with relative traversals.
			traversal = traversal[1:]

			diags := schema.StaticValidateTraversal(traversal)
			if test.WantError == "" {
				if diags.HasErrors() {
					t.Errorf("unexpected error: %s", diags.Err().Error())
				}
			} else {
				if diags.HasErrors() {
					if got := diags.Err().Error(); got != test.WantError {
						t.Errorf("wrong error\ngot:  %s\nwant: %s", got, test.WantError)
					}
				} else {
					t.Errorf("wrong error\ngot:  <no error>\nwant: %s", test.WantError)
				}
			}
		})
	}
}
