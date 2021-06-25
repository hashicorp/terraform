package configschema

import (
	"fmt"
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

	for _, tc := range []struct {
		given  cty.Value
		expect cty.Value
	}{
		{
			cty.UnknownVal(schema.ImpliedType()),
			cty.UnknownVal(schema.ImpliedType()),
		},
		{
			cty.NullVal(schema.ImpliedType()),
			cty.NullVal(schema.ImpliedType()),
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.UnknownVal(cty.String),
				"unsensitive": cty.UnknownVal(cty.String),
				"list":        cty.UnknownVal(schema.BlockTypes["list"].ImpliedType()),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.UnknownVal(cty.String).Mark(marks.Sensitive),
				"unsensitive": cty.UnknownVal(cty.String),
				"list":        cty.UnknownVal(schema.BlockTypes["list"].ImpliedType()),
			}),
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"sensitive":   cty.NullVal(cty.String),
				"unsensitive": cty.UnknownVal(cty.String),
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
	} {
		t.Run(fmt.Sprintf("%#v", tc.given), func(t *testing.T) {
			got := tc.given.MarkWithPaths(schema.ValueMarks(tc.given, nil))
			if !got.RawEquals(tc.expect) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.expect, got)
			}
		})
	}
}
