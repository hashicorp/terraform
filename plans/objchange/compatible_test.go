package objchange

import (
	"fmt"
	"testing"

	"github.com/apparentlymart/go-dump/dump"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestAssertObjectCompatible(t *testing.T) {
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
				"id":  cty.UnknownVal(cty.String),
				"key": cty.EmptyObjectVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
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
				"id":  cty.UnknownVal(cty.String),
				"key": cty.UnknownVal(cty.EmptyObject),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
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
			cty.ObjectVal(map[string]cty.Value{
				"key": cty.ObjectVal(map[string]cty.Value{
					// One wholly unknown block is what "dynamic" blocks
					// generate when the for_each expression is unknown.
					"foo": cty.UnknownVal(cty.String),
				}),
			}),
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
						Block:   configschema.Block{},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"key": cty.ListVal([]cty.Value{
					cty.EmptyObjectVal,
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"key": cty.ListVal([]cty.Value{
					cty.EmptyObjectVal,
				}),
			}),
			nil,
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"key": {
						Nesting: configschema.NestingList,
						Block:   configschema.Block{},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"key": cty.TupleVal([]cty.Value{
					cty.EmptyObjectVal,
					cty.EmptyObjectVal,
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"key": cty.TupleVal([]cty.Value{
					cty.EmptyObjectVal,
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
						Block:   configschema.Block{},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"key": cty.TupleVal([]cty.Value{}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"key": cty.TupleVal([]cty.Value{
					cty.EmptyObjectVal,
					cty.EmptyObjectVal,
				}),
			}),
			[]string{
				`.key: block count changed from 0 to 2`,
			},
		},

		// NestingSet blocks
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
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
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("nope"),
					}),
				}),
			}),
			[]string{
				`.block: block set length changed from 2 to 3`,
			},
		},
		{
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Nesting: configschema.NestingSet,
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
				`.block: planned set element cty.Value{ty: cty.Object(map[string]cty.Type{"foo":cty.String}), v: map[string]interface {}{"foo":"hello"}} does not correlate with any element in actual`,
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
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v and %#v", test.Planned, test.Actual), func(t *testing.T) {
			errs := AssertObjectCompatible(test.Schema, test.Planned, test.Actual)

			wantErrs := make(map[string]struct{})
			gotErrs := make(map[string]struct{})
			for _, err := range errs {
				gotErrs[tfdiags.FormatError(err)] = struct{}{}
			}
			for _, msg := range test.WantErrs {
				wantErrs[msg] = struct{}{}
			}

			t.Logf("\nplanned: %sactual:  %s", dump.Value(test.Planned), dump.Value(test.Actual))
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
