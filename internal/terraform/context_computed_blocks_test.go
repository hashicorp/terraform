// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_computedBlocks(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "computized" {
}
`,
	})

	block := configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"attr": {
				Type:     cty.String,
				Computed: true,
			},
		},
		Computed: true,
	}

	schema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"single": &configschema.NestedBlock{
				Block:   block,
				Nesting: configschema.NestingSingle,
			},
			"list": &configschema.NestedBlock{
				Block:   block,
				Nesting: configschema.NestingList,
			},
			"set": &configschema.NestedBlock{
				Block:   block,
				Nesting: configschema.NestingSet,
			},
			"map": &configschema.NestedBlock{
				Block:   block,
				Nesting: configschema.NestingMap,
			},
		},
	}

	testResourceType := schema.ImpliedType()

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": schema,
		},
	})

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		obj := req.ProposedNewState.AsValueMap()
		for attr, ty := range testResourceType.AttributeTypes() {
			// we only have blocks, and they are all computed, so mark all null
			// attributes as unknown in the plan.
			if req.Config.GetAttr(attr).IsNull() {
				obj[attr] = cty.UnknownVal(ty)
			}
		}
		resp.PlannedState = cty.ObjectVal(obj)
		return resp
	}

	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		obj := req.PlannedState.AsValueMap()
		// create a value for all block attributes which were planned as unknown
		for attr, ty := range testResourceType.AttributeTypes() {
			b := req.PlannedState.GetAttr(attr)
			if !b.IsKnown() {
				nestedObj := cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("test"),
				})
				switch {
				case ty.IsObjectType():
					obj[attr] = nestedObj
				case ty.IsListType():
					obj[attr] = cty.ListVal([]cty.Value{nestedObj})
				case ty.IsSetType():
					obj[attr] = cty.SetVal([]cty.Value{nestedObj})
				case ty.IsMapType():
					obj[attr] = cty.MapVal(map[string]cty.Value{
						"key": nestedObj,
					})
				}
			}
		}
		resp.NewState = cty.ObjectVal(obj)
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, nil, SimplePlanOpts(plans.NormalMode, InputValues{}))
	tfdiags.AssertNoDiagnostics(t, diags)

	after, err := plan.Changes.Resources[0].After.Decode(testResourceType)
	if err != nil {
		t.Fatal(err)
	}

	want := cty.ObjectVal(map[string]cty.Value{
		"list":   cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{"attr": cty.String}))),
		"map":    cty.UnknownVal(cty.Map(cty.Object(map[string]cty.Type{"attr": cty.String}))),
		"set":    cty.UnknownVal(cty.Set(cty.Object(map[string]cty.Type{"attr": cty.String}))),
		"single": cty.UnknownVal(cty.Object(map[string]cty.Type{"attr": cty.String})),
	})

	if !after.RawEquals(want) {
		t.Fatal(cmp.Diff(ctydebug.ValueString(want), ctydebug.ValueString(after)))
	}

	// now apply a value from that plan!
	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoDiagnostics(t, diags)
	inst := state.ResourceInstance(mustResourceInstanceAddr("test_resource.computized"))
	expectedState := `{"list":[{"attr":"test"}],"map":{"key":{"attr":"test"}},"set":[{"attr":"test"}],"single":{"attr":"test"}}`
	if string(inst.Current.AttrsJSON) != expectedState {
		fmt.Printf("expected: %s\ngot: %s\n", expectedState, inst.Current.AttrsJSON)
	}
}

func TestContext2Plan_partiallyKnownComputedBlock(t *testing.T) {
	// this test_resource will supply some information about a computed block
	// during plan, and fill in the rest during apply.

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "computized" {
}
`,
	})

	schema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"list": &configschema.NestedBlock{
				Block: configschema.Block{
					Computed: true,
					Attributes: map[string]*configschema.Attribute{
						"optional": {Type: cty.String, Optional: true, Computed: true},
						"computed": {Type: cty.String, Computed: true},
					},
				},
				Nesting: configschema.NestingList,
			},
		},
	}

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": schema,
		},
	})

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		obj := req.ProposedNewState.AsValueMap()
		obj["list"] = cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"optional": cty.NullVal(cty.String),
				"computed": cty.UnknownVal(cty.String),
			}),
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"optional": cty.String,
				"computed": cty.String,
			})),
		})
		resp.PlannedState = cty.ObjectVal(obj)
		return resp
	}

	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		obj := req.PlannedState.AsValueMap()
		obj["list"] = cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"optional": cty.NullVal(cty.String),
				"computed": cty.StringVal("first computed"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"optional": cty.NullVal(cty.String),
				"computed": cty.StringVal("second computed"),
			}),
		})
		resp.NewState = cty.ObjectVal(obj)
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, nil, SimplePlanOpts(plans.NormalMode, InputValues{}))
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Apply(plan, m, nil)
	tfdiags.AssertNoDiagnostics(t, diags)
}

func TestContext2Plan_partiallyKnownNestedComputedBlock(t *testing.T) {
	// this test_resource will supply some information about a computed block
	// during plan, and fill in the rest during apply.

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "computized" {
	partial {
		required = "test"
	}
}
`,
	})

	schema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"partial": &configschema.NestedBlock{
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"required": {Type: cty.String, Required: true},
						"computed": {Type: cty.String, Computed: true},
					},

					BlockTypes: map[string]*configschema.NestedBlock{
						"nested": &configschema.NestedBlock{
							Block: configschema.Block{
								Computed: true,
								Attributes: map[string]*configschema.Attribute{
									"optional": {Type: cty.String, Optional: true, Computed: true},
									"computed": {Type: cty.String, Computed: true},
								},
							},
							Nesting: configschema.NestingList,
						},
					},
				},
				Nesting: configschema.NestingList,
			},
		},
	}

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": schema,
		},
	})

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		obj := req.ProposedNewState.AsValueMap()
		obj["partial"] = cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"required": cty.StringVal("test"),
				"computed": cty.UnknownVal(cty.String),
				"nested": cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
					"optional": cty.String,
					"computed": cty.String,
				}))),
			}),
		})
		resp.PlannedState = cty.ObjectVal(obj)
		return resp
	}

	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		obj := req.PlannedState.AsValueMap()
		obj["partial"] = cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"required": cty.StringVal("test"),
				"computed": cty.StringVal("computed"),
				"nested": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"optional": cty.NullVal(cty.String),
					"computed": cty.StringVal("computed"),
				})}),
			}),
		})
		resp.NewState = cty.ObjectVal(obj)
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, nil, SimplePlanOpts(plans.NormalMode, InputValues{}))
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Apply(plan, m, nil)
	tfdiags.AssertNoDiagnostics(t, diags)
}
