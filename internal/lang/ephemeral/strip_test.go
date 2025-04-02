// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
)

func TestStripWriteOnlyAttributes(t *testing.T) {
	tcs := map[string]struct {
		val    cty.Value
		schema *configschema.Block
		want   cty.Value
	}{
		"primitive": {
			val: cty.ObjectVal(map[string]cty.Value{
				"value": cty.StringVal("value"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:      cty.String,
						WriteOnly: true,
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"value": cty.NullVal(cty.String),
			}),
		},
		"complex": {
			val: cty.ObjectVal(map[string]cty.Value{
				"value": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("value"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"value": {
									Type: cty.String,
								},
							},
							Nesting: configschema.NestingSingle,
						},
						WriteOnly: true,
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"value": cty.NullVal(cty.Object(map[string]cty.Type{
					"value": cty.String,
				})),
			}),
		},
		"nested in object": {
			val: cty.ObjectVal(map[string]cty.Value{
				"value": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("value"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"value": {
									Type:      cty.String,
									WriteOnly: true,
								},
							},
							Nesting: configschema.NestingSingle,
						},
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"value": cty.ObjectVal(map[string]cty.Value{
					"value": cty.NullVal(cty.String),
				}),
			}),
		},
		"nested in list": {
			val: cty.ObjectVal(map[string]cty.Value{
				"value": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("value"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("value"),
					}),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"value": {
									Type:      cty.String,
									WriteOnly: true,
								},
							},
							Nesting: configschema.NestingList,
						},
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"value": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
				}),
			}),
		},
		"nested in map": {
			val: cty.ObjectVal(map[string]cty.Value{
				"value": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("value"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("value"),
					}),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"value": {
									Type:      cty.String,
									WriteOnly: true,
								},
							},
							Nesting: configschema.NestingMap,
						},
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"value": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
				}),
			}),
		},
		"preserves marks": {
			val: cty.ObjectVal(map[string]cty.Value{
				"value": cty.StringVal("value"),
			}).Mark(marks.Sensitive),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:      cty.String,
						WriteOnly: true,
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"value": cty.NullVal(cty.String),
			}).Mark(marks.Sensitive),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got := StripWriteOnlyAttributes(tc.val, tc.schema)
			if diff := cmp.Diff(got, tc.want, ctydebug.CmpOptions); len(diff) > 0 {
				t.Errorf("got diff:\n%s", diff)
			}
		})
	}
}
