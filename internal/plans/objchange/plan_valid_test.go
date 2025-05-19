// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

		// but don't panic on a null list just in case
		"nested list, null in config": {
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
			cty.ObjectVal(map[string]cty.Value{
				"b": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"c": cty.String,
				})),
			}),
			nil,
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
		"computed within nested objects": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
					// When an object has dynamic attrs, the map may be
					// handled as an object.
					"map_as_obj": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
					"list": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
					"set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
					"single": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.DynamicPseudoType,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"map": cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"map_as_obj": cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.DynamicPseudoType,
				})),
				"list": cty.List(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"set": cty.Set(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"single": cty.Object(map[string]cty.Type{
					"name": cty.String,
				}),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"map_as_obj": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.DynamicPseudoType),
					}),
				}),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"single": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"map_as_obj": cty.ObjectVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("computed"),
					}),
				}),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"single": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
				}),
			}),
			nil,
		},
		"computed nested objects": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type: cty.String,
								},
							},
						},
						Computed: true,
					},
					"list": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type: cty.String,
								},
							},
						},
						Computed: true,
					},
					"set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type: cty.String,
								},
							},
						},
						Optional: true,
						Computed: true,
					},
					"single": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type: cty.DynamicPseudoType,
								},
							},
						},
						Computed: true,
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"map": cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"list": cty.List(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"set": cty.Set(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"single": cty.Object(map[string]cty.Type{
					"name": cty.String,
				}),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.String,
				}))),
				"list": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"name": cty.String,
				}))),
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"single": cty.NullVal(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.UnknownVal(cty.Object(map[string]cty.Type{
						"name": cty.String,
					})),
				}),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("computed"),
					}),
				}),
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"single": cty.UnknownVal(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
			}),
			nil,
		},
		"optional computed within nested objects": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
					// When an object has dynamic attrs, the map may be
					// handled as an object.
					"map_as_obj": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
					"list": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
					"set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
					"single": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"map": cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"map_as_obj": cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.DynamicPseudoType,
				})),
				"list": cty.List(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"set": cty.Set(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
				"single": cty.Object(map[string]cty.Type{
					"name": cty.String,
				}),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"map_as_obj": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.DynamicPseudoType),
					}),
				}),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"single": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("from_config"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"map_as_obj": cty.ObjectVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("computed"),
					}),
				}),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("computed"),
					}),
				}),
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.NullVal(cty.String),
					}),
				}),
				"single": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("from_config"),
				}),
			}),
			nil,
		},
		"cannot replace config nested attr": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.Object(map[string]cty.Type{
				"map": cty.Map(cty.Object(map[string]cty.Type{
					"name": cty.String,
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_provider"),
					}),
				}),
			}),
			[]string{`.map.one.name: planned value cty.StringVal("from_provider") does not match config value cty.StringVal("from_config")`},
		},

		// If a config value ended up in a computed-only attribute it can still
		// be a valid plan. We either got here because the user ignore warnings
		// about ignore_changes on computed attributes, or we failed to
		// validate a config with computed values. Either way, we don't want to
		// indicate an error with the provider.
		"computed only value with config": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("old"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("old"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.UnknownVal(cty.String),
			}),
			nil,
		},

		// When validating collections we start by comparing length, which
		// requires guarding for any unknown values incorrectly returned by the
		// provider.
		"nested collection attrs planned unknown": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"set": {
						Computed: true,
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
									Optional: true,
								},
							},
						},
					},
					"list": {
						Computed: true,
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
									Optional: true,
								},
							},
						},
					},
					"map": {
						Computed: true,
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"name": {
									Type:     cty.String,
									Computed: true,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"list": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"map": cty.MapVal(map[string]cty.Value{
					"key": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"list": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
				"map": cty.MapVal(map[string]cty.Value{
					"key": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("from_config"),
					}),
				}),
			}),
			// provider cannot override the config
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.UnknownVal(cty.Set(
					cty.Object(map[string]cty.Type{
						"name": cty.String,
					}),
				)),
				"list": cty.UnknownVal(cty.Set(
					cty.Object(map[string]cty.Type{
						"name": cty.String,
					}),
				)),
				"map": cty.UnknownVal(cty.Map(
					cty.Object(map[string]cty.Type{
						"name": cty.String,
					}),
				)),
			}),
			[]string{
				`.set: planned unknown for configured value`,
				`.list: planned unknown for configured value`,
				`.map: planned unknown for configured value`,
			},
		},

		"nested set values can contain computed unknown": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"set": {
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"input": {
									Type:     cty.String,
									Optional: true,
								},
								"computed": {
									Type:     cty.String,
									Computed: true,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"input":    cty.StringVal("a"),
						"computed": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"input":    cty.StringVal("b"),
						"computed": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"input":    cty.StringVal("a"),
						"computed": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"input":    cty.StringVal("b"),
						"computed": cty.NullVal(cty.String),
					}),
				}),
			}),
			// Plan can mark the null computed values as unknown
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"input":    cty.StringVal("a"),
						"computed": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"input":    cty.StringVal("b"),
						"computed": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			[]string{},
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
				dump.Value(test.Prior),
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
