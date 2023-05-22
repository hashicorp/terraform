package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFilter(t *testing.T) {
	testCases := map[string]struct {
		schema          *Block
		filterAttribute FilterT[*Attribute]
		filterBlock     FilterT[*NestedBlock]
		want            *Block
	}{
		"empty": {
			schema:          &Block{},
			filterAttribute: FilterDeprecatedAttribute,
			filterBlock:     FilterDeprecatedBlock,
			want:            &Block{},
		},
		"noop": {
			schema: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Required: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"list": {
						Nesting: NestingList,
						Block: Block{
							Attributes: map[string]*Attribute{
								"string": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			filterAttribute: nil,
			filterBlock:     nil,
			want: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Required: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"list": {
						Nesting: NestingList,
						Block: Block{
							Attributes: map[string]*Attribute{
								"string": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
					},
				},
			},
		},
		"filter_deprecated": {
			schema: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Optional: true,
					},
					"deprecated_string": {
						Type:       cty.String,
						Deprecated: true,
					},
					"nested": {
						NestedType: &Object{
							Attributes: map[string]*Attribute{
								"string": {
									Type: cty.String,
								},
								"deprecated_string": {
									Type:       cty.String,
									Deprecated: true,
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
								"string": {
									Type:     cty.String,
									Optional: true,
								},
							},
							Deprecated: true,
						},
					},
				},
			},
			filterAttribute: FilterDeprecatedAttribute,
			filterBlock:     FilterDeprecatedBlock,
			want: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Optional: true,
					},
					"nested": {
						NestedType: &Object{
							Attributes: map[string]*Attribute{
								"string": {
									Type: cty.String,
								},
							},
							Nesting: NestingList,
						},
					},
				},
			},
		},
		"filter_read_only": {
			schema: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Optional: true,
					},
					"read_only_string": {
						Type:     cty.String,
						Computed: true,
					},
					"nested": {
						NestedType: &Object{
							Attributes: map[string]*Attribute{
								"string": {
									Type:     cty.String,
									Optional: true,
								},
								"read_only_string": {
									Type:     cty.String,
									Computed: true,
								},
								"deeply_nested": {
									NestedType: &Object{
										Attributes: map[string]*Attribute{
											"number": {
												Type:     cty.Number,
												Required: true,
											},
											"read_only_number": {
												Type:     cty.Number,
												Computed: true,
											},
										},
										Nesting: NestingList,
									},
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
								"string": {
									Type:     cty.String,
									Optional: true,
								},
								"read_only_string": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
				},
			},
			filterAttribute: FilterReadOnlyAttribute,
			filterBlock:     nil,
			want: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Optional: true,
					},
					"nested": {
						NestedType: &Object{
							Attributes: map[string]*Attribute{
								"string": {
									Type:     cty.String,
									Optional: true,
								},
								"deeply_nested": {
									NestedType: &Object{
										Attributes: map[string]*Attribute{
											"number": {
												Type:     cty.Number,
												Required: true,
											},
										},
										Nesting: NestingList,
									},
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
								"string": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
		},
		"filter_optional_computed_id": {
			schema: &Block{
				Attributes: map[string]*Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
					"string": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
			},
			filterAttribute: FilterHelperSchemaIdAttribute,
			filterBlock:     nil,
			want: &Block{
				Attributes: map[string]*Attribute{
					"string": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := tc.schema.Filter(tc.filterAttribute, tc.filterBlock)
			if !cmp.Equal(got, tc.want, cmp.Comparer(cty.Type.Equals), cmpopts.EquateEmpty()) {
				t.Fatal(cmp.Diff(got, tc.want, cmp.Comparer(cty.Type.Equals), cmpopts.EquateEmpty()))
			}
		})
	}
}
