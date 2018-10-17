package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestBlockImpliedType(t *testing.T) {
	tests := map[string]struct {
		Schema *Block
		Want   cty.Type
	}{
		"nil": {
			nil,
			cty.EmptyObject,
		},
		"empty": {
			&Block{},
			cty.EmptyObject,
		},
		"attributes": {
			&Block{
				Attributes: map[string]*Attribute{
					"optional": {
						Type:     cty.String,
						Optional: true,
					},
					"required": {
						Type:     cty.Number,
						Required: true,
					},
					"computed": {
						Type:     cty.List(cty.Bool),
						Computed: true,
					},
					"optional_computed": {
						Type:     cty.Map(cty.Bool),
						Optional: true,
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"optional":          cty.String,
				"required":          cty.Number,
				"computed":          cty.List(cty.Bool),
				"optional_computed": cty.Map(cty.Bool),
			}),
		},
		"blocks": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"single": &NestedBlock{
						Nesting: NestingSingle,
						Block: Block{
							Attributes: map[string]*Attribute{
								"foo": {
									Type:     cty.DynamicPseudoType,
									Required: true,
								},
							},
						},
					},
					"list": &NestedBlock{
						Nesting: NestingList,
					},
					"set": &NestedBlock{
						Nesting: NestingSet,
					},
					"map": &NestedBlock{
						Nesting: NestingMap,
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"single": cty.Object(map[string]cty.Type{
					"foo": cty.DynamicPseudoType,
				}),
				"list": cty.List(cty.EmptyObject),
				"set":  cty.Set(cty.EmptyObject),
				"map":  cty.Map(cty.EmptyObject),
			}),
		},
		"deep block nesting": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"single": &NestedBlock{
						Nesting: NestingSingle,
						Block: Block{
							BlockTypes: map[string]*NestedBlock{
								"list": &NestedBlock{
									Nesting: NestingList,
									Block: Block{
										BlockTypes: map[string]*NestedBlock{
											"set": &NestedBlock{
												Nesting: NestingSet,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"single": cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"set": cty.Set(cty.EmptyObject),
					})),
				}),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.Schema.ImpliedType()
			if !got.Equals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
