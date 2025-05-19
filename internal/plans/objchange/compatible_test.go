// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package objchange

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestAssertObjectCompatible(t *testing.T) {
	schemaWithFoo := configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {Type: cty.String, Optional: true},
		},
	}
	fooBlockValue := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})
	schemaWithFooBar := configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {Type: cty.String, Optional: true},
			"bar": {Type: cty.String, Optional: true},
		},
	}
	fooBarBlockValue := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
		"bar": cty.NullVal(cty.String), // simulating the situation where bar isn't set in the config at all
	})

	tests := []struct {
		Schema   *configschema.Block
		Planned  cty.Value
		Actual   cty.Value
		WantErrs []string
	}{
		{
			&configschema.Block{},
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy"),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.UnknownVal(cty.String),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy"),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("wotsit"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy"),
			}),
			[]string{
				`.name: was cty.StringVal("wotsit"), but now cty.StringVal("thingy")`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.UnknownVal(cty.String),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.Zero,
			}),
			[]string{
				`.name: wrong final value type: string required`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.UnknownVal(cty.String).RefineNotNull(),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.NullVal(cty.String),
			}),
			[]string{
				`.name: final value cty.NullVal(cty.String) does not conform to planning placeholder cty.UnknownVal(cty.String).RefineNotNull()`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.UnknownVal(cty.String).Refine().
					StringPrefix("boop:").
					NewValue(),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("thingy"),
			}),
			[]string{
				`.name: final value cty.StringVal("thingy") does not conform to planning placeholder cty.UnknownVal(cty.String).Refine().StringPrefixFull("boop:").NewValue()`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:      cty.String,
						Required:  true,
						Sensitive: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("wotsit"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy"),
			}),
			[]string{
				`.name: inconsistent values for sensitive attribute`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("wotsit").Mark(marks.Sensitive),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy"),
			}),
			[]string{
				`.name: inconsistent values for sensitive attribute`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("wotsit"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("thingy").Mark(marks.Sensitive),
			}),
			[]string{
				`.name: inconsistent values for sensitive attribute`,
			},
		},
		{
			// This tests the codepath that leads to couldHaveUnknownBlockPlaceholder,
			// where a set may be sensitive and need to be unmarked before it
			// is iterated upon
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"configuration": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"sensitive_fields": {
									Nesting: configschema.NestingSet,
									Block:   schemaWithFoo,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"configuration": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"sensitive_fields": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("secret"),
							}),
						}).Mark(marks.Sensitive),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"configuration": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"sensitive_fields": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("secret"),
							}),
						}).Mark(marks.Sensitive),
					}),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"stuff": {
						Type:     cty.DynamicPseudoType,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.DynamicVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.StringVal("thingy"),
			}),
			[]string{},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"obj": {
						Type: cty.Object(map[string]cty.Type{
							"stuff": cty.DynamicPseudoType,
						}),
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"obj": cty.ObjectVal(map[string]cty.Value{
					"stuff": cty.DynamicVal,
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"obj": cty.ObjectVal(map[string]cty.Value{
					"stuff": cty.NumberIntVal(3),
				}),
			}),
			[]string{},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"stuff": {
						Type:     cty.DynamicPseudoType,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.StringVal("wotsit"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.StringVal("thingy"),
			}),
			[]string{
				`.stuff: was cty.StringVal("wotsit"), but now cty.StringVal("thingy")`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"stuff": {
						Type:     cty.DynamicPseudoType,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.StringVal("true"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.True,
			}),
			[]string{
				`.stuff: wrong final value type: string required`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"stuff": {
						Type:     cty.DynamicPseudoType,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.DynamicVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.EmptyObjectVal,
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"stuff": {
						Type:     cty.DynamicPseudoType,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"stuff": cty.ObjectVal(map[string]cty.Value{
					"nonsense": cty.StringVal("yup"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"stuff": cty.EmptyObjectVal,
			}),
			[]string{
				`.stuff: wrong final value type: attribute "nonsense" is required`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"tags": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"tags": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"tags": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("wotsit"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
				}),
			}),
			[]string{
				`.tags["Name"]: was cty.StringVal("wotsit"), but now cty.StringVal("thingy")`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"tags": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
					"Env":  cty.StringVal("production"),
				}),
			}),
			[]string{
				`.tags: new element "Env" has appeared`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"tags": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"tags": cty.MapValEmpty(cty.String),
			}),
			[]string{
				`.tags: element "Name" has vanished`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"tags": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.NullVal(cty.String),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"zones": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"zones": cty.SetVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"zones": cty.SetVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"zones": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"zones": cty.SetVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"zones": cty.SetVal([]cty.Value{
					cty.StringVal("thingy"),
					cty.StringVal("wotsit"),
				}),
			}),
			[]string{
				`.zones: actual set element cty.StringVal("wotsit") does not correlate with any element in plan`,
				`.zones: length changed from 1 to 2`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"zones": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"zones": cty.SetVal([]cty.Value{
					cty.UnknownVal(cty.String),
					cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"zones": cty.SetVal([]cty.Value{
					// Imagine that both of our unknown values ultimately resolved to "thingy",
					// causing them to collapse into a single element. That's valid,
					// even though it's also a little confusing and counter-intuitive.
					cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"names": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"names": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"names": cty.UnknownVal(cty.List(cty.String)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"names": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"names": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
					cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
					cty.StringVal("wotsit"),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"names": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
					cty.StringVal("thingy"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
					cty.StringVal("wotsit"),
				}),
			}),
			[]string{
				`.names[1]: was cty.StringVal("thingy"), but now cty.StringVal("wotsit")`,
			},
		},
		{
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"names": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.String),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"names": cty.ListVal([]cty.Value{
					cty.StringVal("thingy"),
					cty.StringVal("wotsit"),
				}),
			}),
			[]string{
				`.names: new element 1 has appeared`,
			},
		},

		// NestingSingle blocks
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingSingle,
						Block:   configschema.Block{},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.EmptyObjectVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.EmptyObjectVal,
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingSingle,
						Block:   configschema.Block{},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.UnknownVal(cty.EmptyObject),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.EmptyObjectVal,
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.NullVal(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				})),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("hello"),
				}),
			}),
			[]string{
				`.key: was absent, but now present`,
			},
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("hello"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.NullVal(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				})),
			}),
			[]string{
				`.key: was present, but now absent`,
			},
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"key": cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.NullVal(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				})),
			}),
			nil,
		},

		// NestingList blocks
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingList,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ListVal([]cty.Value{
					fooBlockValue,
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ListVal([]cty.Value{
					fooBlockValue,
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingList,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.TupleVal([]cty.Value{
					fooBlockValue,
					fooBlockValue,
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.TupleVal([]cty.Value{
					fooBlockValue,
				}),
			}),
			[]string{
				`.key: block count changed from 2 to 1`,
			},
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingList,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.TupleVal([]cty.Value{}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.TupleVal([]cty.Value{
					fooBlockValue,
					fooBlockValue,
				}),
			}),
			[]string{
				`.key: block count changed from 0 to 2`,
			},
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingList,
						Block:   schemaWithFooBar,
					},
				},
			},
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"key": cty.List(fooBarBlockValue.Type()),
			})),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			nil, // an unknown block is allowed to expand into multiple, because that's how dynamic blocks behave when for_each is unknown
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingList,
						Block:   schemaWithFooBar,
					},
				},
			},
			// While we must make an exception for empty strings in sets due to
			// the legacy SDK, lists should be compared more strictly.
			// This does not count as a dynamic block placeholder
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ListVal([]cty.Value{
					fooBarBlockValue,
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
						"bar": cty.StringVal(""),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ListVal([]cty.Value{
					fooBlockValue,
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			[]string{".key: block count changed from 2 to 3"},
		},

		// NestingSet blocks
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					// This is testing the scenario where the two unknown values
					// turned out to be equal after we learned their values,
					// and so they coalesced together into a single element.
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.UnknownVal(cty.Set(
					cty.Object(map[string]cty.Type{
						"foo": cty.String,
					}),
				)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("nope"),
					}),
				}),
			}),
			// there is no error here, because the presence of unknowns
			// indicates this may be a dynamic block, and the length is unknown
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("howdy"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("world"),
					}),
				}),
			}),
			[]string{
				`.block: planned set element cty.ObjectVal(map[string]cty.Value{"foo":cty.StringVal("hello")}) does not correlate with any element in actual`,
			},
		},
		{
			// This one is an odd situation where the value representing the
			// block itself is unknown. This is never supposed to be true,
			// but in legacy SDK mode we allow such things to pass through as
			// a warning, and so we must tolerate them for matching purposes.
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.UnknownVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.UnknownVal(cty.Set(fooBlockValue.Type())),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("a"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("b"),
					}),
				}),
			}),
			nil,
		},
		// test a set with an unknown dynamic count going to 0 values
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block2": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block2": cty.UnknownVal(cty.Set(fooBlockValue.Type())),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block2": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				})),
			}),
			nil,
		},
		// test a set with a patially known dynamic count reducing it's values
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block3": {
						Nesting: configschema.NestingSet,
						Block:   schemaWithFoo,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block3": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("a"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block3": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("a"),
					}),
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.EmptyObjectVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"block": cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d: %#v and %#v", i, test.Planned, test.Actual), func(t *testing.T) {
			errs := AssertObjectCompatible(test.Schema, test.Planned, test.Actual)

			wantErrs := make(map[string]struct{})
			gotErrs := make(map[string]struct{})
			for _, err := range errs {
				gotErrs[tfdiags.FormatError(err)] = struct{}{}
			}
			for _, msg := range test.WantErrs {
				wantErrs[msg] = struct{}{}
			}

			t.Logf("\nplanned: %sactual:  %s", ctydebug.ValueString(test.Planned), ctydebug.ValueString(test.Actual))
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
