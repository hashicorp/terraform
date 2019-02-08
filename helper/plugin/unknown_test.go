package plugin

import (
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestSetUnknowns(t *testing.T) {
	for n, tc := range map[string]struct {
		Schema   *configschema.Block
		Val      cty.Value
		Expected cty.Value
	}{
		"empty": {
			&configschema.Block{},
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
			cty.ObjectVal(map[string]cty.Value{}),
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.UnknownVal(cty.String),
			}),
		},
		"no prior with set": {
			// the set value should remain null
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
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.ObjectVal(map[string]cty.Value{}),
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
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bonjour"),
				"bar": cty.StringVal("petit dejeuner"),
				"baz": cty.StringVal("grande dejeuner"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bonjour"),
				"bar": cty.StringVal("petit dejeuner"),
				"baz": cty.StringVal("grande dejeuner"),
				"boz": cty.UnknownVal(cty.String),
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("beep"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("beep"),
					"baz": cty.UnknownVal(cty.String),
				}),
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bap"),
						"baz": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.UnknownVal(cty.String),
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.NullVal(cty.String),
						"baz": cty.StringVal("boop"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.UnknownVal(cty.String),
						"baz": cty.StringVal("boop"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.UnknownVal(cty.String),
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NullVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("blep"),
						"baz": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.UnknownVal(cty.String),
					}),
				}),
			}),
		},
		"sets differing only by unknown": {
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
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
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.UnknownVal(cty.String),
					}),
				}),
			}),
		},
	} {
		t.Run(n, func(t *testing.T) {
			// coerce the values because SetUnknowns expects the values to be
			// complete, and so we can take shortcuts writing them in the
			// test.
			v, err := tc.Schema.CoerceValue(tc.Val)
			if err != nil {
				t.Fatal(err)
			}

			expected, err := tc.Schema.CoerceValue(tc.Expected)
			if err != nil {
				t.Fatal(err)
			}

			got := SetUnknowns(v, tc.Schema)
			if !got.RawEquals(expected) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", expected, got)
			}
		})
	}
}
