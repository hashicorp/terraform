// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestCoerceValue(t *testing.T) {
	tests := map[string]struct {
		Schema    *Block
		Input     cty.Value
		WantValue cty.Value
		WantErr   string
	}{
		"empty schema and value": {
			&Block{},
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			``,
		},
		"attribute present": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.True,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("true"),
			}),
			``,
		},
		"single block present": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingSingle,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.EmptyObjectVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.EmptyObjectVal,
			}),
			``,
		},
		"single block wrong type": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingSingle,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.True,
			}),
			cty.DynamicVal,
			`.foo: an object is required`,
		},
		"list block with one item": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingList,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			``,
		},
		"set block with one item": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingSet,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}), // can implicitly convert to set
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			``,
		},
		"map block with one item": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingMap,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{"foo": cty.EmptyObjectVal}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{"foo": cty.EmptyObjectVal}),
			}),
			``,
		},
		"list block with one item having an attribute": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block: Block{
							Attributes: map[string]*Attribute{
								"bar": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Nesting: NestingList,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("hello"),
				})}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("hello"),
				})}),
			}),
			``,
		},
		"list block with one item having a missing attribute": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block: Block{
							Attributes: map[string]*Attribute{
								"bar": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Nesting: NestingList,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			cty.DynamicVal,
			`.foo[0]: attribute "bar" is required`,
		},
		"list block with one item having an extraneous attribute": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingList,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("hello"),
				})}),
			}),
			cty.DynamicVal,
			`.foo[0]: unexpected attribute "bar"`,
		},
		"missing optional attribute": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			``,
		},
		"missing optional single block": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingSingle,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.EmptyObject),
			}),
			``,
		},
		"missing optional list block": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingList,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListValEmpty(cty.EmptyObject),
			}),
			``,
		},
		"missing optional set block": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingSet,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetValEmpty(cty.EmptyObject),
			}),
			``,
		},
		"missing optional map block": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:   Block{},
						Nesting: NestingMap,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapValEmpty(cty.EmptyObject),
			}),
			``,
		},
		"missing required attribute": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.DynamicVal,
			`attribute "foo" is required`,
		},
		"missing required single block": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:    Block{},
						Nesting:  NestingSingle,
						MinItems: 1,
						MaxItems: 1,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.EmptyObject),
			}),
			``,
		},
		"unknown nested list": {
			&Block{
				Attributes: map[string]*Attribute{
					"attr": {
						Type:     cty.String,
						Required: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:    Block{},
						Nesting:  NestingList,
						MinItems: 2,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.EmptyObject),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.List(cty.EmptyObject)),
			}),
			"",
		},
		"unknowns in nested list": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block: Block{
							Attributes: map[string]*Attribute{
								"attr": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Nesting:  NestingList,
						MinItems: 2,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			"",
		},
		"unknown nested set": {
			&Block{
				Attributes: map[string]*Attribute{
					"attr": {
						Type:     cty.String,
						Required: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:    Block{},
						Nesting:  NestingSet,
						MinItems: 1,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.EmptyObject),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.Set(cty.EmptyObject)),
			}),
			"",
		},
		"unknown nested map": {
			&Block{
				Attributes: map[string]*Attribute{
					"attr": {
						Type:     cty.String,
						Required: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Block:    Block{},
						Nesting:  NestingMap,
						MinItems: 1,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.Map(cty.String)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.Map(cty.EmptyObject)),
			}),
			"",
		},
		"extraneous attribute": {
			&Block{},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			cty.DynamicVal,
			`unexpected attribute "foo"`,
		},
		"wrong attribute type": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.Number,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.False,
			}),
			cty.DynamicVal,
			`.foo: number required`,
		},
		"unset computed value": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			``,
		},
		"omitted attribute requirements": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type: cty.String,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.String),
			}),
			`attribute "foo" has none of required, optional, or computed set`,
		},
		"dynamic value attributes": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Nesting: NestingMap,
						Block: Block{
							Attributes: map[string]*Attribute{
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NumberIntVal(8),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.NullVal(cty.DynamicPseudoType),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NumberIntVal(8),
					}),
				}),
			}),
			``,
		},
		"dynamic attributes in map": {
			// Convert a block represented as a map to an object if a
			// DynamicPseudoType causes the element types to mismatch.
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Nesting: NestingMap,
						Block: Block{
							Attributes: map[string]*Attribute{
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.NullVal(cty.DynamicPseudoType),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NullVal(cty.DynamicPseudoType),
					}),
				}),
			}),
			``,
		},
		"nested types": {
			// handle NestedTypes
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						NestedType: &Object{
							Nesting: NestingList,
							Attributes: map[string]*Attribute{
								"bar": {
									Type:     cty.String,
									Required: true,
								},
								"baz": {
									Type:     cty.Map(cty.String),
									Optional: true,
								},
							},
						},
						Optional: true,
					},
					"fob": {
						NestedType: &Object{
							Nesting: NestingSet,
							Attributes: map[string]*Attribute{
								"bar": {
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
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.NullVal(cty.Map(cty.String)),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NullVal(cty.Map(cty.String)),
					}),
				}),
				"fob": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"bar": cty.String,
				}))),
			}),
			``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotValue, gotErrObj := test.Schema.CoerceValue(test.Input)

			if gotErrObj == nil {
				if test.WantErr != "" {
					t.Fatalf("coersion succeeded; want error: %q", test.WantErr)
				}
			} else {
				gotErr := tfdiags.FormatError(gotErrObj)
				if gotErr != test.WantErr {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", gotErr, test.WantErr)
				}
				return
			}

			if !gotValue.RawEquals(test.WantValue) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, gotValue, test.WantValue)
			}
		})
	}
}
