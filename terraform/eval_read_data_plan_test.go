package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestReadDataCreateEmptyBlocks(t *testing.T) {
	setSchema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"set": {
				Nesting: configschema.NestingSet,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"attr": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	nestedSetSchema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"set": {
				Nesting: configschema.NestingSet,
				Block: configschema.Block{
					BlockTypes: map[string]*configschema.NestedBlock{
						"nested-set": {
							Nesting: configschema.NestingSet,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"attr": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	listSchema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"list": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"attr": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	nestedListSchema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"list": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					BlockTypes: map[string]*configschema.NestedBlock{
						"nested-list": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"attr": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	singleSchema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"single": {
				Nesting: configschema.NestingSingle,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"attr": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	for _, tc := range []struct {
		name   string
		schema *configschema.Block
		val    cty.Value
		expect cty.Value
	}{
		{
			"set-block",
			setSchema,
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("ok"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("ok"),
					}),
				}),
			}),
		},
		{
			"set-block-empty",
			setSchema,
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				),
			}),
		},
		{
			"set-block-null",
			setSchema,
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				),
			}),
		},
		{
			"list-block",
			listSchema,
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("ok"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("ok"),
					}),
				}),
			}),
		},
		{
			"list-block-empty",
			listSchema,
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListValEmpty(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListValEmpty(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				),
			}),
		},
		{
			"list-block-null",
			listSchema,
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.NullVal(cty.List(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListValEmpty(
					cty.Object(map[string]cty.Type{
						"attr": cty.String,
					}),
				),
			}),
		},
		{
			"nested-set-block",
			nestedSetSchema,
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-set": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"attr": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-set": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"attr": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
		},
		{
			"nested-set-block-empty",
			nestedSetSchema,
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-set": cty.SetValEmpty(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-set": cty.SetValEmpty(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						),
					}),
				}),
			}),
		},
		{
			"nested-set-block-null",
			nestedSetSchema,
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-set": cty.NullVal(cty.Set(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						)),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-set": cty.SetValEmpty(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						),
					}),
				}),
			}),
		},
		{
			"nested-list-block-empty",
			nestedListSchema,
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-list": cty.ListValEmpty(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-list": cty.ListValEmpty(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						),
					}),
				}),
			}),
		},
		{
			"nested-list-block-null",
			nestedListSchema,
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-list": cty.NullVal(cty.List(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						)),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"nested-list": cty.ListValEmpty(
							cty.Object(map[string]cty.Type{
								"attr": cty.String,
							}),
						),
					}),
				}),
			}),
		},
		{
			"single-block-null",
			singleSchema,
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			val := createEmptyBlocks(tc.schema, tc.val)
			if !tc.expect.Equals(val).True() {
				t.Fatalf("\nexpected: %#v\ngot     : %#v\n", tc.expect, val)
			}
		})
	}
}
