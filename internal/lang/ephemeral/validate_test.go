// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestNonNullWriteOnlyPaths(t *testing.T) {
	for name, tc := range map[string]struct {
		val    cty.Value
		schema *configschema.Block

		expectedPaths []cty.Path
	}{
		"no write-only attributes": {
			val: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-abc123"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type: cty.String,
					},
				},
			},
		},

		"write-only attribute with null value": {
			val: cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:      cty.String,
						Optional:  true,
						WriteOnly: true,
					},
				},
			},
		},

		"write-only attribute with non-null value": {
			val: cty.ObjectVal(map[string]cty.Value{
				"valid": cty.NullVal(cty.String),
				"id":    cty.StringVal("i-abc123"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"valid": {
						Type:      cty.String,
						Optional:  true,
						WriteOnly: true,
					},
					"id": {
						Type:      cty.String,
						Optional:  true,
						WriteOnly: true,
					},
				},
			},
			expectedPaths: []cty.Path{cty.GetAttrPath("id")},
		},

		"write-only attributes in blocks": {
			val: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"valid-write-only": cty.NullVal(cty.String),
						"valid":            cty.StringVal("valid"),
						"id":               cty.StringVal("i-abc123"),
						"bar": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"valid-write-only": cty.NullVal(cty.String),
								"valid":            cty.StringVal("valid"),
								"id":               cty.StringVal("i-abc123"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"valid-write-only": cty.NullVal(cty.String),
								"valid":            cty.StringVal("valid"),
								"id":               cty.StringVal("i-xyz123"),
							}),
						}),
					}),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"valid-write-only": {
									Type:      cty.String,
									Optional:  true,
									WriteOnly: true,
								},
								"valid": {
									Type:     cty.String,
									Optional: true,
								},
								"id": {
									Type:      cty.String,
									Optional:  true,
									WriteOnly: true,
								},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"bar": {
									Nesting: configschema.NestingList,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"valid-write-only": {
												Type:      cty.String,
												Optional:  true,
												WriteOnly: true,
											},
											"valid": {
												Type:     cty.String,
												Optional: true,
											},
											"id": {
												Type:      cty.String,
												Optional:  true,
												WriteOnly: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedPaths: []cty.Path{
				cty.GetAttrPath("foo").Index(cty.NumberIntVal(0)).GetAttr("id"),
				cty.GetAttrPath("foo").Index(cty.NumberIntVal(0)).GetAttr("bar").Index(cty.NumberIntVal(0)).GetAttr("id"),
				cty.GetAttrPath("foo").Index(cty.NumberIntVal(0)).GetAttr("bar").Index(cty.NumberIntVal(1)).GetAttr("id"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			paths, err := nonNullWriteOnlyPaths(tc.val, tc.schema, nil)
			if err != nil {
				t.Fatal(err)
			}

			if len(paths) != len(tc.expectedPaths) {
				t.Fatalf("expected %d write-only paths, got %d", len(tc.expectedPaths), len(paths))
			}

			for i, path := range paths {
				if !path.Equals(tc.expectedPaths[i]) {
					t.Fatalf("expected path %#v, got %#v", tc.expectedPaths[i], path)
				}
			}
		})
	}
}
