// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package objchange

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

func TestProposedNew(t *testing.T) {
	tests := map[string]struct {
		Schema *configschema.Block
		Prior  cty.Value
		Config cty.Value
		Want   cty.Value
	}{
		"empty": {
			&configschema.Block{},
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
		},
		"no prior": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"bar": {
						Type:     cty.String,
						Computed: true,
					},
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"baz": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"boz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"biz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.DynamicPseudoType),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				})),
				"bar": cty.NullVal(cty.String),
				"baz": cty.ObjectVal(map[string]cty.Value{
					"boz": cty.StringVal("world"),

					// An unknown in the config represents a situation where
					// an argument is explicitly set to an expression result
					// that is derived from an unknown value. This is distinct
					// from leaving it null, which allows the provider itself
					// to decide the value during PlanResourceChange.
					"biz": cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),

				// unset computed attributes are null in the proposal; provider
				// usually changes them to "unknown" during PlanResourceChange,
				// to indicate that the value will be decided during apply.
				"bar": cty.NullVal(cty.String),
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				})),

				"baz": cty.ObjectVal(map[string]cty.Value{
					"boz": cty.StringVal("world"),
					"biz": cty.UnknownVal(cty.String), // explicit unknown preserved from config
				}),
			}),
		},
		"null block remains null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"baz": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"boz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.NullVal(cty.DynamicPseudoType),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				})),
				"baz": cty.NullVal(cty.Object(map[string]cty.Type{
					"boz": cty.String,
				})),
			}),
			// The bloop attribue and baz block does not exist in the config,
			// and therefore shouldn't be planned.
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop": cty.String,
				})),
				"baz": cty.NullVal(cty.Object(map[string]cty.Type{
					"boz": cty.String,
				})),
			}),
		},
		"no prior with set": {
			// This one is here because our handling of sets is more complex
			// than others (due to the fuzzy correlation heuristic) and
			// historically that caused us some panic-related grief.
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"baz": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"boz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
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
						Computed: true,
						Optional: true,
					},
				},
			},
			cty.NullVal(cty.DynamicPseudoType),
			cty.ObjectVal(map[string]cty.Value{
				"baz": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"boz": cty.StringVal("world"),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blub"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"baz": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"boz": cty.StringVal("world"),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blub"),
					}),
				}),
			}),
		},
		"prior attributes": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"bar": {
						Type:     cty.String,
						Computed: true,
					},
					"baz": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
					"boz": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
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
				"foo": cty.StringVal("bonjour"),
				"bar": cty.StringVal("petit dejeuner"),
				"baz": cty.StringVal("grande dejeuner"),
				"boz": cty.StringVal("a la monde"),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop": cty.StringVal("glub"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
				"bar": cty.NullVal(cty.String),
				"baz": cty.NullVal(cty.String),
				"boz": cty.StringVal("world"),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop": cty.StringVal("bleep"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("petit dejeuner"),
				"baz": cty.StringVal("grande dejeuner"),
				"boz": cty.StringVal("world"),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop": cty.StringVal("bleep"),
				}),
			}),
		},
		"prior nested single": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"baz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
								"bleep": {
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
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("beep"),
					"baz": cty.StringVal("boop"),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop":  cty.StringVal("glub"),
					"bleep": cty.NullVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("bap"),
					"baz": cty.NullVal(cty.String),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop":  cty.StringVal("glub"),
					"bleep": cty.StringVal("beep"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("bap"),
					"baz": cty.StringVal("boop"),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop":  cty.StringVal("glub"),
					"bleep": cty.StringVal("beep"),
				}),
			}),
		},
		"prior nested single to null": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"baz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
								"bleep": {
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
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("beep"),
					"baz": cty.StringVal("boop"),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop":  cty.StringVal("glub"),
					"bleep": cty.NullVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.String,
				})),
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop":  cty.String,
					"bleep": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.String,
				})),
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop":  cty.String,
					"bleep": cty.String,
				})),
			}),
		},

		"prior optional computed nested single to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
								"bleep": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Optional: true,
						Computed: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"blop":  cty.StringVal("glub"),
					"bleep": cty.NullVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop":  cty.String,
					"bleep": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Object(map[string]cty.Type{
					"blop":  cty.String,
					"bleep": cty.String,
				})),
			}),
		},

		"prior nested list": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"baz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
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
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.StringVal("boop"),
					}),
				}),
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("bar"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("baz"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("bar"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("baz"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.StringVal("boop"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("bar"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("baz"),
					}),
				}),
			}),
		},
		"prior nested list with dynamic": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"baz": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.DynamicPseudoType,
									Required: true,
								},
								"blub": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.StringVal("boop"),
					}),
				}),
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("bar"),
						"blub": cty.StringVal("glub"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("baz"),
						"blub": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("bar"),
						"blub": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.StringVal("boop"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("bar"),
						"blub": cty.NullVal(cty.String),
					}),
				}),
			}),
		},
		"prior nested map": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"baz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
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
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.StringVal("boop"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.StringVal("boot"),
					}),
				}),
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("glub"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blub"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.NullVal(cty.String),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bosh"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("glub"),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blub"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.StringVal("boop"),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bosh"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("glub"),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blub"),
					}),
				}),
			}),
		},

		"prior optional computed nested map elem to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Optional: true,
								},
								"bleep": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glub"),
						"bleep": cty.StringVal("computed"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("blub"),
						"bleep": cty.StringVal("computed"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.NullVal(cty.Object(map[string]cty.Type{
						"blop":  cty.String,
						"bleep": cty.String,
					})),
					"c": cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("blub"),
						"bleep": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.NullVal(cty.Object(map[string]cty.Type{
						"blop":  cty.String,
						"bleep": cty.String,
					})),
					"c": cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("blub"),
						"bleep": cty.NullVal(cty.String),
					}),
				}),
			}),
		},

		"prior optional computed nested map to null": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Optional: true,
								},
								"bleep": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
						Optional: true,
						Computed: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glub"),
						"bleep": cty.StringVal("computed"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("blub"),
						"bleep": cty.StringVal("computed"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Map(
					cty.Object(map[string]cty.Type{
						"blop":  cty.String,
						"bleep": cty.String,
					}),
				)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bloop": cty.NullVal(cty.Map(
					cty.Object(map[string]cty.Type{
						"blop":  cty.String,
						"bleep": cty.String,
					}),
				)),
			}),
		},

		"prior nested map with dynamic": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"baz": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.DynamicPseudoType,
									Required: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.StringVal("boop"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.ListVal([]cty.Value{cty.StringVal("boot")}),
					}),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("glub"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.NumberIntVal(13),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.NullVal(cty.String),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bosh"),
						"baz": cty.NullVal(cty.List(cty.String)),
					}),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blep"),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.NumberIntVal(13),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.StringVal("boop"),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bosh"),
						"baz": cty.NullVal(cty.List(cty.String)),
					}),
				}),
				"bloop": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.StringVal("blep"),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"blop": cty.NumberIntVal(13),
					}),
				}),
			}),
		},
		"prior nested set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									// This non-computed attribute will serve
									// as our matching key for propagating
									// "baz" from elements in the prior value.
									Type:     cty.String,
									Optional: true,
								},
								"baz": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"bloop": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"blop": {
									Type:     cty.String,
									Required: true,
								},
								"bleep": {
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
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.StringVal("boop"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.StringVal("boot"),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glubglub"),
						"bleep": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glubglub"),
						"bleep": cty.StringVal("beep"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bosh"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glubglub"),
						"bleep": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glub"),
						"bleep": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.StringVal("boop"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bosh"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glubglub"),
						"bleep": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop":  cty.StringVal("glub"),
						"bleep": cty.NullVal(cty.String),
					}),
				}),
			}),
		},

		"set with partial optional computed change": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"multi": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"opt": {
									Type:     cty.String,
									Optional: true,
								},
								"cmp": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"cmp": cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"cmp": cty.StringVal("OK"),
					}),
				}),
			}),

			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"cmp": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("replaced"),
						"cmp": cty.NullVal(cty.String),
					}),
				}),
			}),
			// "one" can be correlated because it is a non-computed value in
			// the configuration.
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"cmp": cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("replaced"),
						"cmp": cty.NullVal(cty.String),
					}),
				}),
			}),
		},

		"set without partial optional computed change": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"multi": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"opt": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"req": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"req": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"req": cty.StringVal("two"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.NullVal(cty.String),
						"req": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.NullVal(cty.String),
						"req": cty.StringVal("two"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"req": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"req": cty.StringVal("two"),
					}),
				}),
			}),
		},

		"sets differing only by unknown": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"multi": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"optional": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
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
			cty.NullVal(cty.DynamicPseudoType),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"optional": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"optional": cty.UnknownVal(cty.String),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					// These remain distinct because unknown values never
					// compare equal. They may be consolidated together once
					// the values become known, though.
					cty.ObjectVal(map[string]cty.Value{
						"optional": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"optional": cty.UnknownVal(cty.String),
					}),
				}),
				"bloop": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"blop": cty.UnknownVal(cty.String),
					}),
				}),
			}),
		},
		"nested list in set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"bar": {
									Nesting: configschema.NestingList,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"baz": {
												Type: cty.String,
											},
											"qux": {
												Type:     cty.String,
												Computed: true,
												Optional: true,
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
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("beep"),
								"qux": cty.StringVal("boop"),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("beep"),
								"qux": cty.NullVal(cty.String),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("beep"),
								"qux": cty.StringVal("boop"),
							}),
						}),
					}),
				}),
			}),
		},
		"empty nested list in set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"bar": {
									Nesting: configschema.NestingList,
									Block:   configschema.Block{},
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ListValEmpty((&configschema.Block{}).ImpliedType()),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ListValEmpty((&configschema.Block{}).ImpliedType()),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ListValEmpty((&configschema.Block{}).ImpliedType()),
					}),
				}),
			}),
		},
		"nested list with dynamic in set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"bar": {
									Nesting: configschema.NestingList,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"baz": {
												Type: cty.DynamicPseudoType,
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
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.TupleVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("true"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.ListVal([]cty.Value{cty.StringVal("true")}),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.TupleVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("true"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.ListVal([]cty.Value{cty.StringVal("true")}),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.TupleVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("true"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"baz": cty.ListVal([]cty.Value{cty.StringVal("true")}),
							}),
						}),
					}),
				}),
			}),
		},
		"nested map with dynamic in set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"bar": {
									Nesting: configschema.NestingMap,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"baz": {
												Type:     cty.DynamicPseudoType,
												Optional: true,
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
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ObjectVal(map[string]cty.Value{
							"bing": cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("true"),
							}),
							"bang": cty.ObjectVal(map[string]cty.Value{
								"baz": cty.ListVal([]cty.Value{cty.StringVal("true")}),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ObjectVal(map[string]cty.Value{
							"bing": cty.ObjectVal(map[string]cty.Value{
								"baz": cty.ListVal([]cty.Value{cty.StringVal("true")}),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.ObjectVal(map[string]cty.Value{
							"bing": cty.ObjectVal(map[string]cty.Value{
								"baz": cty.ListVal([]cty.Value{cty.StringVal("true")}),
							}),
						}),
					}),
				}),
			}),
		},
		"empty nested map in set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"bar": {
									Nesting: configschema.NestingMap,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"baz": {
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.MapValEmpty(cty.Object(map[string]cty.Type{
							"baz": cty.String,
						})),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.MapVal(map[string]cty.Value{
							"bing": cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("true"),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.MapVal(map[string]cty.Value{
							"bing": cty.ObjectVal(map[string]cty.Value{
								"baz": cty.StringVal("true"),
							}),
						}),
					}),
				}),
			}),
		},
		// This example has a mixture of optional, computed and required in a deeply-nested NestedType attribute
		"deeply NestedType": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									NestedType: &configschema.Object{
										Nesting:    configschema.NestingSingle,
										Attributes: testAttributes,
									},
									Required: true,
								},
								"baz": {
									NestedType: &configschema.Object{
										Nesting:    configschema.NestingSingle,
										Attributes: testAttributes,
									},
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			// prior
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.NullVal(cty.DynamicPseudoType),
					"baz": cty.ObjectVal(map[string]cty.Value{
						"optional":          cty.NullVal(cty.String),
						"computed":          cty.StringVal("hello"),
						"optional_computed": cty.StringVal("prior"),
						"required":          cty.StringVal("present"),
					}),
				}),
			}),
			// config
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.UnknownVal(cty.Object(map[string]cty.Type{ // explicit unknown from the config
						"optional":          cty.String,
						"computed":          cty.String,
						"optional_computed": cty.String,
						"required":          cty.String,
					})),
					"baz": cty.ObjectVal(map[string]cty.Value{
						"optional":          cty.NullVal(cty.String),
						"computed":          cty.NullVal(cty.String),
						"optional_computed": cty.StringVal("hello"),
						"required":          cty.StringVal("present"),
					}),
				}),
			}),
			// want
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.UnknownVal(cty.Object(map[string]cty.Type{ // explicit unknown preserved from the config
						"optional":          cty.String,
						"computed":          cty.String,
						"optional_computed": cty.String,
						"required":          cty.String,
					})),
					"baz": cty.ObjectVal(map[string]cty.Value{
						"optional":          cty.NullVal(cty.String),  // config is null
						"computed":          cty.StringVal("hello"),   // computed values come from prior
						"optional_computed": cty.StringVal("hello"),   // config takes precedent over prior in opt+computed
						"required":          cty.StringVal("present"), // value from config
					}),
				}),
			}),
		},
		"deeply nested set": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									NestedType: &configschema.Object{
										Nesting:    configschema.NestingSet,
										Attributes: testAttributes,
									},
									Required: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			// prior values
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"optional":          cty.StringVal("prior"),
								"computed":          cty.StringVal("prior"),
								"optional_computed": cty.StringVal("prior"),
								"required":          cty.StringVal("prior"),
							}),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"optional":          cty.StringVal("other_prior"),
								"computed":          cty.StringVal("other_prior"),
								"optional_computed": cty.StringVal("other_prior"),
								"required":          cty.StringVal("other_prior"),
							}),
						}),
					}),
				}),
			}),
			// config differs from prior
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"optional":          cty.StringVal("configured"),
							"computed":          cty.NullVal(cty.String), // computed attrs are null in config
							"optional_computed": cty.StringVal("configured"),
							"required":          cty.StringVal("configured"),
						})}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"optional":          cty.NullVal(cty.String), // explicit null in config
							"computed":          cty.NullVal(cty.String), // computed attrs are null in config
							"optional_computed": cty.StringVal("other_configured"),
							"required":          cty.StringVal("other_configured"),
						})}),
					}),
				}),
			}),
			// want:
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"optional":          cty.StringVal("configured"),
							"computed":          cty.NullVal(cty.String),
							"optional_computed": cty.StringVal("configured"),
							"required":          cty.StringVal("configured"),
						})}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"optional":          cty.NullVal(cty.String), // explicit null in config is preserved
							"computed":          cty.NullVal(cty.String),
							"optional_computed": cty.StringVal("other_configured"),
							"required":          cty.StringVal("other_configured"),
						})}),
					}),
				}),
			}),
		},
		"expected null NestedTypes": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"single": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String},
							},
						},
						Optional: true,
					},
					"list": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String},
							},
						},
						Optional: true,
					},
					"set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String},
							},
						},
						Optional: true,
					},
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String},
							},
						},
						Optional: true,
					},
					"nested_map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"inner": {
									NestedType: &configschema.Object{
										Nesting:    configschema.NestingSingle,
										Attributes: testAttributes,
									},
								},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{"bar": cty.StringVal("baz")}),
				"list":   cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"bar": cty.StringVal("baz")})}),
				"map": cty.MapVal(map[string]cty.Value{
					"map_entry": cty.ObjectVal(map[string]cty.Value{"bar": cty.StringVal("baz")}),
				}),
				"set": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"bar": cty.StringVal("baz")})}),
				"nested_map": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"inner": cty.ObjectVal(map[string]cty.Value{
							"optional":          cty.StringVal("foo"),
							"computed":          cty.StringVal("foo"),
							"optional_computed": cty.StringVal("foo"),
							"required":          cty.StringVal("foo"),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.NullVal(cty.Object(map[string]cty.Type{"bar": cty.String})),
				"list":   cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"bar": cty.String}))),
				"map":    cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{"bar": cty.String}))),
				"set":    cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{"bar": cty.String}))),
				"nested_map": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"inner": cty.Object(map[string]cty.Type{
						"optional":          cty.String,
						"computed":          cty.String,
						"optional_computed": cty.String,
						"required":          cty.String,
					}),
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.NullVal(cty.Object(map[string]cty.Type{"bar": cty.String})),
				"list":   cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"bar": cty.String}))),
				"map":    cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{"bar": cty.String}))),
				"set":    cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{"bar": cty.String}))),
				"nested_map": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"inner": cty.Object(map[string]cty.Type{
						"optional":          cty.String,
						"computed":          cty.String,
						"optional_computed": cty.String,
						"required":          cty.String,
					}),
				}))),
			}),
		},
		"expected empty NestedTypes": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String},
							},
						},
						Optional: true,
					},
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String},
							},
						},
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.Object(map[string]cty.Type{"bar": cty.String})),
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{"bar": cty.String})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.Object(map[string]cty.Type{"bar": cty.String})),
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{"bar": cty.String})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.Object(map[string]cty.Type{"bar": cty.String})),
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{"bar": cty.String})),
			}),
		},
		"optional types set replacement": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"bar": {
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
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("old"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("new"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("new"),
					}),
				}),
			}),
		},
		"prior null nested objects": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"single": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"list": {
									NestedType: &configschema.Object{
										Nesting: configschema.NestingList,
										Attributes: map[string]*configschema.Attribute{
											"foo": {
												Type: cty.String,
											},
										},
									},
									Optional: true,
								},
							},
						},
						Optional: true,
					},
					"map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"map": {
									NestedType: &configschema.Object{
										Nesting: configschema.NestingList,
										Attributes: map[string]*configschema.Attribute{
											"foo": {
												Type: cty.String,
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
			cty.NullVal(cty.Object(map[string]cty.Type{
				"single": cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"foo": cty.String,
					})),
				}),
				"map": cty.Map(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"foo": cty.String,
					})),
				})),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("a"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("b"),
						}),
					}),
				}),
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("a"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("b"),
							}),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("a"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("b"),
						}),
					}),
				}),
				"map": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("a"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("b"),
							}),
						}),
					}),
				}),
			}),
		},

		// Data sources are planned with an unknown value.
		// Note that this plan fails AssertPlanValid, because for managed
		// resources an instance would never be completely unknown.
		"unknown prior nested objects": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"list": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"list": {
									NestedType: &configschema.Object{
										Nesting: configschema.NestingList,
										Attributes: map[string]*configschema.Attribute{
											"foo": {
												Type: cty.String,
											},
										},
									},
									Computed: true,
								},
							},
						},
						Computed: true,
					},
				},
			},
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"list": cty.List(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"foo": cty.String,
					})),
				})),
			})),
			cty.NullVal(cty.Object(map[string]cty.Type{
				"list": cty.List(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"foo": cty.String,
					})),
				})),
			})),
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"list": cty.List(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"foo": cty.String,
					})),
				})),
			})),
		},

		// A nested object with computed attributes, which is contained in an
		// optional+computed container. The nested computed values should be
		// represented in the proposed new object.
		"config within optional+computed": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"list_obj": {
						Optional: true,
						Computed: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"obj": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"optional": {Type: cty.String, Optional: true},
											"computed": {Type: cty.String, Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"list_obj": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("prior"),
							"computed": cty.StringVal("prior computed"),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list_obj": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("prior"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list_obj": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("prior"),
							"computed": cty.StringVal("prior computed"),
						}),
					}),
				}),
			}),
		},

		// A nested object with computed attributes, which is contained in an
		// optional+computed container. The prior nested object contains values
		// which could not be computed, therefor the proposed new value must be
		// the null value from the configuration.
		"computed within optional+computed": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"list_obj": {
						Optional: true,
						Computed: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"obj": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"optional": {Type: cty.String, Optional: true},
											"computed": {Type: cty.String, Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"list_obj": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("prior"),
							"computed": cty.StringVal("prior computed"),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list_obj": cty.NullVal(cty.List(
					cty.Object(map[string]cty.Type{
						"obj": cty.Object(map[string]cty.Type{
							"optional": cty.String,
							"computed": cty.String,
						}),
					}),
				)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"list_obj": cty.NullVal(cty.List(
					cty.Object(map[string]cty.Type{
						"obj": cty.Object(map[string]cty.Type{
							"optional": cty.String,
							"computed": cty.String,
						}),
					}),
				)),
			}),
		},

		// A nested object with computed attributes, which is contained in an
		// optional+computed set. The nested computed values should be
		// represented in the proposed new object, and correlated with state
		// via the non-computed attributes.
		"config add within optional+computed set": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"set_obj": {
						Optional: true,
						Computed: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"obj": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"optional": {Type: cty.String, Optional: true},
											"computed": {Type: cty.String, Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.StringVal("first computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.StringVal("second computed"),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("third"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.StringVal("first computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.StringVal("second computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("third"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
		},

		// A nested object with computed attributes, which is contained in a
		// set. The nested computed values should be represented in the
		// proposed new object, and correlated with state via the non-computed
		// attributes.
		"config add within set block": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"set_obj": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"obj": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"optional": {Type: cty.String, Optional: true},
											"computed": {Type: cty.String, Optional: true, Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.StringVal("first computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.StringVal("second from config"),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.StringVal("second from config"),
						}),
					}),
					// new "third" value added
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("third"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.StringVal("first computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.StringVal("second from config"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("third"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
		},

		// A nested object with computed attributes, which is contained in a
		// set. The nested computed values should be represented in the
		// proposed new object, and correlated with state via the non-computed
		// attributes.
		"config change within set block": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"set_obj": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"obj": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"optional": {Type: cty.String, Optional: true},
											"computed": {Type: cty.String, Optional: true, Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.StringVal("first computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("second"),
							"computed": cty.StringVal("second computed"),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("changed"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"set_obj": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("first"),
							"computed": cty.StringVal("first computed"),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"optional": cty.StringVal("changed"),
							"computed": cty.NullVal(cty.String),
						}),
					}),
				}),
			}),
		},

		"set attr with partial optional computed change": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"multi": {
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"opt": {
									Type:     cty.String,
									Optional: true,
								},
								"oc": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("replaced"),
						"oc":  cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("replaced"),
						"oc":  cty.NullVal(cty.String),
					}),
				}),
			}),
		},

		"set attr without optional computed change": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"multi": {
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"opt": {
									Type:     cty.String,
									Optional: true,
								},
								"oc": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
					}),
				}),
			}),
		},

		"set attr with all optional computed": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"multi": {
						Optional: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"opt": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"oc": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
					}),
				}),
			}),
			// Each of these values can be correlated by the existence of the
			// optional config attribute. Because "one" and "two" are set in
			// the config, they must exist in the state regardless of
			// optional&computed.
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
					}),
				}),
			}),
		},

		"set block with all optional computed and nested object types": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"multi": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"opt": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"oc": {
									Type:     cty.String,
									Optional: true,
									Computed: true,
								},
								"attr": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSet,
										Attributes: map[string]*configschema.Attribute{
											"opt": {
												Type:     cty.String,
												Optional: true,
												Computed: true,
											},
											"oc": {
												Type:     cty.String,
												Optional: true,
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
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
						"attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"opt": cty.StringVal("one"),
							"oc":  cty.StringVal("OK"),
						})}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
						"attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"opt": cty.StringVal("two"),
							"oc":  cty.StringVal("OK"),
						})}),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.NullVal(cty.String),
						"attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"opt": cty.StringVal("one"),
							"oc":  cty.StringVal("OK"),
						})}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
						"attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"opt": cty.StringVal("two"),
							"oc":  cty.NullVal(cty.String),
						})}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("three"),
						"oc":  cty.NullVal(cty.String),
						"attr": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
							"opt": cty.String,
							"oc":  cty.String,
						}))),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"multi": cty.SetVal([]cty.Value{
					// We can correlate this with prior from the outer object
					// attributes, and the equal nested set.
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("one"),
						"oc":  cty.StringVal("OK"),
						"attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"opt": cty.StringVal("one"),
							"oc":  cty.StringVal("OK"),
						})}),
					}),
					// This value is overridden by config, because we can't
					// correlate optional+computed config values within nested
					// sets.
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("two"),
						"oc":  cty.StringVal("OK"),
						"attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
							"opt": cty.StringVal("two"),
							"oc":  cty.NullVal(cty.String),
						})}),
					}),
					// This value was taken only from config
					cty.ObjectVal(map[string]cty.Value{
						"opt": cty.StringVal("three"),
						"oc":  cty.NullVal(cty.String),
						"attr": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
							"opt": cty.String,
							"oc":  cty.String,
						}))),
					}),
				}),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := ProposedNew(test.Schema, test.Prior, test.Config)
			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

var testAttributes = map[string]*configschema.Attribute{
	"optional": {
		Type:     cty.String,
		Optional: true,
	},
	"computed": {
		Type:     cty.String,
		Computed: true,
	},
	"optional_computed": {
		Type:     cty.String,
		Computed: true,
		Optional: true,
	},
	"required": {
		Type:     cty.String,
		Required: true,
	},
}
