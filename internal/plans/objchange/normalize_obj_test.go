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

func TestNormalizeObjectFromLegacySDK(t *testing.T) {
	tests := map[string]struct {
		Schema *configschema.Block
		Input  cty.Value
		Want   cty.Value
	}{
		"empty": {
			&configschema.Block{},
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
		},
		"attributes only": {
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"a": {Type: cty.String, Required: true},
					"b": {Type: cty.String, Optional: true},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.StringVal("b value"),
			}),
		},
		"null block single": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.Object(map[string]cty.Type{
					"b": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.Object(map[string]cty.Type{
					"b": cty.String,
				})),
			}),
		},
		"unknown block single": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"c": {Nesting: configschema.NestingSingle},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.UnknownVal(cty.Object(map[string]cty.Type{
					"b": cty.String,
					"c": cty.EmptyObject,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.UnknownVal(cty.String),
					"c": cty.EmptyObjectVal,
				}),
			}),
		},
		"null block list": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"c": {Nesting: configschema.NestingSingle},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
					"c": cty.EmptyObject,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"b": cty.String,
					"c": cty.EmptyObject,
				})),
			}),
		},
		"unknown block list": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
					"b": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.UnknownVal(cty.String),
					}),
				}),
			}),
		},
		"null block set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"b": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"b": cty.String,
				})),
			}),
		},
		"unknown block set": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.UnknownVal(cty.Set(cty.Object(map[string]cty.Type{
					"b": cty.String,
				}))),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.UnknownVal(cty.String),
					}),
				}),
			}),
		},
		"map block passes through": {
			// Legacy SDK doesn't use NestingMap, so we don't do any transforms
			// related to it but we still need to verify that map blocks pass
			// through unscathed.
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("b value"),
					}),
				}),
			}),
		},
		"block list with dynamic type": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.DynamicPseudoType, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.True,
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.True,
					}),
				}),
			}),
		},
		"block map with dynamic type": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"a": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"b": {Type: cty.DynamicPseudoType, Optional: true},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					"another": cty.ObjectVal(map[string]cty.Value{
						"b": cty.True,
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					"another": cty.ObjectVal(map[string]cty.Value{
						"b": cty.True,
					}),
				}),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := NormalizeObjectFromLegacySDK(test.Input, test.Schema)
			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
