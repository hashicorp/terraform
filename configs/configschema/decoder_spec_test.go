package configschema

import (
	"testing"

	"github.com/apparentlymart/go-dump/dump"
	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"
)

func TestBlockDecoderSpec(t *testing.T) {
	tests := map[string]struct {
		Schema    *Block
		TestBody  hcl.Body
		Want      cty.Value
		DiagCount int
	}{
		"empty": {
			&Block{},
			hcl.EmptyBody(),
			cty.EmptyObjectVal,
			0,
		},
		"nil": {
			nil,
			hcl.EmptyBody(),
			cty.EmptyObjectVal,
			0,
		},
		"attributes": {
			&Block{
				Attributes: map[string]*Attribute{
					"optional": {
						Type:     cty.Number,
						Optional: true,
					},
					"required": {
						Type:     cty.String,
						Required: true,
					},
					"computed": {
						Type:     cty.List(cty.Bool),
						Computed: true,
					},
					"optional_computed": {
						Type:     cty.Map(cty.Bool),
						Optional: true,
						Computed: true,
					},
					"optional_computed_overridden": {
						Type:     cty.Bool,
						Optional: true,
						Computed: true,
					},
					"optional_computed_unknown": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Attributes: hcl.Attributes{
					"required": {
						Name: "required",
						Expr: hcltest.MockExprLiteral(cty.NumberIntVal(5)),
					},
					"optional_computed_overridden": {
						Name: "optional_computed_overridden",
						Expr: hcltest.MockExprLiteral(cty.True),
					},
					"optional_computed_unknown": {
						Name: "optional_computed_overridden",
						Expr: hcltest.MockExprLiteral(cty.UnknownVal(cty.String)),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"optional":                     cty.NullVal(cty.Number),
				"required":                     cty.StringVal("5"), // converted from number to string
				"computed":                     cty.NullVal(cty.List(cty.Bool)),
				"optional_computed":            cty.NullVal(cty.Map(cty.Bool)),
				"optional_computed_overridden": cty.True,
				"optional_computed_unknown":    cty.UnknownVal(cty.String),
			}),
			0,
		},
		"dynamically-typed attribute": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.DynamicPseudoType, // any type is permitted
						Required: true,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Attributes: hcl.Attributes{
					"foo": {
						Name: "foo",
						Expr: hcltest.MockExprLiteral(cty.True),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.True,
			}),
			0,
		},
		"dynamically-typed attribute omitted": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.DynamicPseudoType, // any type is permitted
						Optional: true,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.DynamicPseudoType),
			}),
			0,
		},
		"required attribute omitted": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.Bool,
						Required: true,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.Bool),
			}),
			1, // missing required attribute
		},
		"wrong attribute type": {
			&Block{
				Attributes: map[string]*Attribute{
					"optional": {
						Type:     cty.Number,
						Optional: true,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Attributes: hcl.Attributes{
					"optional": {
						Name: "optional",
						Expr: hcltest.MockExprLiteral(cty.True),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"optional": cty.UnknownVal(cty.Number),
			}),
			1, // incorrect type; number required
		},
		"blocks": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"single": {
						Nesting: NestingSingle,
						Block:   Block{},
					},
					"list": {
						Nesting: NestingList,
						Block:   Block{},
					},
					"set": {
						Nesting: NestingSet,
						Block:   Block{},
					},
					"map": {
						Nesting: NestingMap,
						Block:   Block{},
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Blocks: hcl.Blocks{
					&hcl.Block{
						Type: "list",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "single",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "list",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "set",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type:        "map",
						Labels:      []string{"foo"},
						LabelRanges: []hcl.Range{hcl.Range{}},
						Body:        hcl.EmptyBody(),
					},
					&hcl.Block{
						Type:        "map",
						Labels:      []string{"bar"},
						LabelRanges: []hcl.Range{hcl.Range{}},
						Body:        hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "set",
						Body: hcl.EmptyBody(),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.EmptyObjectVal,
				"list": cty.ListVal([]cty.Value{
					cty.EmptyObjectVal,
					cty.EmptyObjectVal,
				}),
				"set": cty.SetVal([]cty.Value{
					cty.EmptyObjectVal,
					cty.EmptyObjectVal,
				}),
				"map": cty.MapVal(map[string]cty.Value{
					"foo": cty.EmptyObjectVal,
					"bar": cty.EmptyObjectVal,
				}),
			}),
			0,
		},
		"blocks with dynamically-typed attributes": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"single": {
						Nesting: NestingSingle,
						Block: Block{
							Attributes: map[string]*Attribute{
								"a": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
								},
							},
						},
					},
					"list": {
						Nesting: NestingList,
						Block: Block{
							Attributes: map[string]*Attribute{
								"a": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
								},
							},
						},
					},
					"map": {
						Nesting: NestingMap,
						Block: Block{
							Attributes: map[string]*Attribute{
								"a": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
								},
							},
						},
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Blocks: hcl.Blocks{
					&hcl.Block{
						Type: "list",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "single",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "list",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type:        "map",
						Labels:      []string{"foo"},
						LabelRanges: []hcl.Range{hcl.Range{}},
						Body:        hcl.EmptyBody(),
					},
					&hcl.Block{
						Type:        "map",
						Labels:      []string{"bar"},
						LabelRanges: []hcl.Range{hcl.Range{}},
						Body:        hcl.EmptyBody(),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"a": cty.NullVal(cty.DynamicPseudoType),
				}),
				"list": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"a": cty.NullVal(cty.DynamicPseudoType),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"a": cty.NullVal(cty.DynamicPseudoType),
					}),
				}),
				"map": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"a": cty.NullVal(cty.DynamicPseudoType),
					}),
					"bar": cty.ObjectVal(map[string]cty.Value{
						"a": cty.NullVal(cty.DynamicPseudoType),
					}),
				}),
			}),
			0,
		},
		"too many list items": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Nesting:  NestingList,
						Block:    Block{},
						MaxItems: 1,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Blocks: hcl.Blocks{
					&hcl.Block{
						Type: "foo",
						Body: hcl.EmptyBody(),
					},
					&hcl.Block{
						Type: "foo",
						Body: hcl.EmptyBody(),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.EmptyObjectVal,
					cty.EmptyObjectVal,
				}),
			}),
			0, // max items cannot be validated during decode
		},
		// dynamic blocks may fulfill MinItems, but there is only one block to
		// decode.
		"required MinItems": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Nesting:  NestingList,
						Block:    Block{},
						MinItems: 2,
					},
				},
			},
			hcltest.MockBody(&hcl.BodyContent{
				Blocks: hcl.Blocks{
					&hcl.Block{
						Type: "foo",
						Body: hcl.EmptyBody(),
					},
				},
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.EmptyObjectVal,
				}),
			}),
			0,
		},
		"extraneous attribute": {
			&Block{},
			hcltest.MockBody(&hcl.BodyContent{
				Attributes: hcl.Attributes{
					"extra": {
						Name: "extra",
						Expr: hcltest.MockExprLiteral(cty.StringVal("hello")),
					},
				},
			}),
			cty.EmptyObjectVal,
			1, // extraneous attribute
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			spec := test.Schema.DecoderSpec()
			got, diags := hcldec.Decode(test.TestBody, spec, nil)
			if len(diags) != test.DiagCount {
				t.Errorf("wrong number of diagnostics %d; want %d", len(diags), test.DiagCount)
				for _, diag := range diags {
					t.Logf("- %s", diag.Error())
				}
			}

			if !got.RawEquals(test.Want) {
				t.Logf("[INFO] implied schema is %s", spew.Sdump(hcldec.ImpliedSchema(spec)))
				t.Errorf("wrong result\ngot:  %s\nwant: %s", dump.Value(got), dump.Value(test.Want))
			}

			// Double-check that we're producing consistent results for DecoderSpec
			// and ImpliedType.
			impliedType := test.Schema.ImpliedType()
			if errs := got.Type().TestConformance(impliedType); len(errs) != 0 {
				t.Errorf("result does not conform to the schema's implied type")
				for _, err := range errs {
					t.Logf("- %s", err.Error())
				}
			}
		})
	}
}
