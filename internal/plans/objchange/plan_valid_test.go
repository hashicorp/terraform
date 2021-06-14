package objchange

import (
	"testing"

	"github.com/apparentlymart/go-dump/dump"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestAssertPlanValid(t *testing.T) {
	tests := map[string]struct {
		Schema   *configschema.Block
		Prior    cty.Value
		Config   cty.Value
		Planned  cty.Value
		WantErrs []string
	}{
		"all empty": {
			&configschema.Block{},
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			nil,
		},
		"no computed, all match": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			nil,
		},
		"no computed, plan matches, no prior": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.String,
				"b": cty.List(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			nil,
		},
		"no computed, invalid change in plan": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.String,
				"b": cty.List(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("new c value"),
					}),
				}),
			}),
			[]string{
				`.b[0].c: planned value cty.StringVal("new c value") does not match config value cty.StringVal("c value")`,
			},
		},
		"no computed, invalid change in plan sensitive": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:      cty.String,
									Optional:  true,
									Sensitive: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.String,
				"b": cty.List(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("new c value"),
					}),
				}),
			}),
			[]string{
				`.b[0].c: sensitive planned value does not match config value`,
			},
		},
		"no computed, diff suppression in plan": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("new c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"), // plan uses value from prior object
					}),
				}),
			}),
			nil,
		},
		"no computed, all null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
				"b": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.NullVal(cty.String),
					}),
				}),
			}),
			nil,
		},
		"nested map, normal update": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.MapVal(map[string]cty.Value{
					"boop": cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("hello"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.MapVal(map[string]cty.Value{
					"boop": cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("howdy"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.MapVal(map[string]cty.Value{
					"boop": cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("howdy"),
					}),
				}),
			}),
			nil,
		},

		// Nested block collections are never null
		"nested list, null in plan": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"b": cty.List(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"c": cty.String,
				}))),
			}),
			[]string{
				`.b: attribute representing a list of nested blocks must be empty to indicate no blocks, not null`,
			},
		},

		// blocks can be unknown when using dynamic
		"nested list, unknown nested dynamic": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"b": {
									Nesting: configschema.NestingList,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"c": {
												Type:     cty.String,
												Optional: true,
											},
											"computed": {
												Type:     cty.String,
												Computed: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},

			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"computed": cty.NullVal(cty.String),
					"b": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("x"),
					})}),
				})}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"b": cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
						"c":        cty.String,
						"computed": cty.String,
					}))),
				})}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"b": cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
						"c":        cty.String,
						"computed": cty.String,
					}))),
				})}),
			}),
			[]string{},
		},

		"nested set, unknown dynamic cannot be planned": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"computed": {
						Type:     cty.String,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},

			cty.ObjectVal(map[string]cty.Value{
				"computed": cty.NullVal(cty.String),
				"b": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"c": cty.StringVal("x"),
				})}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"computed": cty.NullVal(cty.String),
				"b": cty.UnknownVal(cty.Set(cty.Object(map[string]cty.Type{
					"c": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"computed": cty.StringVal("default"),
				"b": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"c": cty.StringVal("oops"),
				})}),
			}),

			[]string{
				`.b: planned value cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"c":cty.StringVal("oops")})}) for unknown dynamic block`,
			},
		},

		"nested set, null in plan": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"b": cty.Set(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"c": cty.String,
				}))),
			}),
			[]string{
				`.b: attribute representing a set of nested blocks must be empty to indicate no blocks, not null`,
			},
		},
		"nested map, null in plan": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"b": cty.Map(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"c": cty.String,
				}))),
			}),
			[]string{
				`.b: attribute representing a map of nested blocks must be empty to indicate no blocks, not null`,
			},
		},

		// We don't actually do any validation for nested set blocks, and so
		// the remaining cases here are just intending to ensure we don't
		// inadvertently start generating errors incorrectly in future.
		"nested set, no computed, no changes": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			nil,
		},
		"nested set, no computed, invalid change in plan": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("new c value"), // matches neither prior nor config
					}),
				}),
			}),
			nil,
		},
		"nested set, no computed, diff suppressed": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"b": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"c": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("new c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("c value"), // plan uses value from prior object
					}),
				}),
			}),
			nil,
		},

		// Attributes with NestedTypes
		"NestedType attr, no computed, all match": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
			nil,
		},
		"NestedType attr, no computed, plan matches, no prior": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("c value"),
					}),
				}),
			}),
			nil,
		},
		"NestedType, no computed, invalid change in plan": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("c value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("new c value"),
					}),
				}),
			}),
			[]string{
				`.a[0].b: planned value cty.StringVal("new c value") does not match config value cty.StringVal("c value")`,
			},
		},
		"NestedType attr, no computed, invalid change in plan sensitive": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:      cty.String,
									Optional:  true,
									Sensitive: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"a": cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("new b value"),
					}),
				}),
			}),
			[]string{
				`.a[0].b: sensitive planned value does not match config value`,
			},
		},
		"NestedType attr, no computed, diff suppression in plan": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("new b value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"), // plan uses value from prior object
					}),
				}),
			}),
			nil,
		},
		"NestedType attr, no computed, all null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.DynamicPseudoType),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.DynamicPseudoType),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.DynamicPseudoType),
			}),
			nil,
		},
		"NestedType attr, no computed, all zero value": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"b": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				}))),
			}),
			nil,
		},
		"NestedType NestingSet attribute to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				}))),
			}),
			nil,
		},
		"NestedType deep nested optional set attribute to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bleep": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"bloop": {
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSet,
										Attributes: map[string]*configschema.Attribute{
											"blome": {
												Type:     cty.String,
												Optional: true,
											},
										},
									},
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bleep": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bloop": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"blome": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bleep": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bloop": cty.NullVal(cty.Set(
							cty.Object(map[string]cty.Type{
								"blome": cty.String,
							}),
						)),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bleep": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bloop": cty.NullVal(cty.List(
							cty.Object(map[string]cty.Type{
								"blome": cty.String,
							}),
						)),
					}),
				}),
			}),
			nil,
		},
		"NestedType deep nested set": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bleep": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"bloop": {
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSet,
										Attributes: map[string]*configschema.Attribute{
											"blome": {
												Type:     cty.String,
												Optional: true,
											},
										},
									},
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bleep": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bloop": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"blome": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
			// Note: bloop is null in the config
			cty.ObjectVal(map[string]cty.Value{
				"bleep": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bloop": cty.NullVal(cty.Set(
							cty.Object(map[string]cty.Type{
								"blome": cty.String,
							}),
						)),
					}),
				}),
			}),
			// provider sends back the prior value, not matching the config
			cty.ObjectVal(map[string]cty.Value{
				"bleep": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bloop": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"blome": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
			nil, // we cannot validate individual set elements, and trust the provider's response
		},
		"NestedType nested computed list attribute": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Computed: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				}))),
			}),

			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			nil,
		},
		"NestedType nested list attribute to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				}))),
			}),

			// provider returned the old value
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			[]string{`.bloop: planned value cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"blop":cty.StringVal("ok")})}) for a non-computed attribute`},
		},
		"NestedType nested set attribute to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				}))),
			}),
			// provider returned the old value
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("ok"),
					}),
				}),
			}),
			[]string{`.bloop: planned value cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"blop":cty.StringVal("ok")})}) for a non-computed attribute`},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			errs := AssertPlanValid(test.Schema, test.Prior, test.Config, test.Planned)

			wantErrs := make(map[string]struct{})
			gotErrs := make(map[string]struct{})
			for _, err := range errs {
				gotErrs[tfdiags.FormatError(err)] = struct{}{}
			}
			for _, msg := range test.WantErrs {
				wantErrs[msg] = struct{}{}
			}

			t.Logf(
				"\nprior:  %sconfig:  %splanned: %s",
				dump.Value(test.Planned),
				dump.Value(test.Config),
				dump.Value(test.Planned),
			)
			for msg := range wantErrs {
				if _, ok := gotErrs[msg]; !ok {
					t.Errorf("missing expected error: %s", msg)
				}
			}
			for msg := range gotErrs {
				if _, ok := wantErrs[msg]; !ok {
					t.Errorf("unexpected extra error: %s", msg)
				}
			}
		})
	}
}
