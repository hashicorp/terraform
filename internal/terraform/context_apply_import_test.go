// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// other import tests can be found in context_apply2_test.go
func TestContextApply_import_in_module(t *testing.T) {
	m := testModule(t, "import-block-in-module")

	p := mockProviderWithResourceTypeSchema("test_object", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":          {Type: cty.String, Computed: true},
			"test_string": {Type: cty.String, Optional: true},
		},
	})
	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: "test_object",
					State: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("importable"),
						"id":          cty.StringVal(req.ID),
					}),
				},
			},
		}
	}
	p.ReadResourceFn = func(r providers.ReadResourceRequest) providers.ReadResourceResponse {
		id := r.PriorState.GetAttr("id")
		return providers.ReadResourceResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"test_string": cty.StringVal("importable"),
				"id":          id,
			}),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	tfdiags.AssertNoErrors(t, diags)

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	if !p.ImportResourceStateCalled {
		t.Fatal("resource not imported")
	}

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.child.test_object.bar[\"first\"]"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}
	var attrs map[string]interface{}
	err := json.Unmarshal(rs.Current.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := attrs["id"], "testa"; got != want {
		t.Fatalf("wrong id for \"first\" got:  %#v  want: %#v\n", got, want)
	}

	rs = state.ResourceInstance(mustResourceInstanceAddr("module.child.test_object.bar[\"second\"]"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}
	err = json.Unmarshal(rs.Current.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := attrs["id"], "testb"; got != want {
		t.Fatalf("wrong id for \"second\" got:  %#v  want: %#v\n", got, want)
	}
}

func TestContextApply_import_in_nested_module(t *testing.T) { // more nested than the test above. nesteder.
	m := testModule(t, "import-block-in-nested-module")

	p := simpleMockProvider()
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("importable"),
				}),
			},
		},
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("importable"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	tfdiags.AssertNoErrors(t, diags)

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.child.module.kinder.test_object.bar"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("resources not imported")
	}
}

func TestContextApply_import_in_expanded_module(t *testing.T) { // count AND for each!
	m := testModule(t, "import-block-in-module-with-expansion")

	p := simpleMockProvider()
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("importable"),
				}),
			},
		},
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("importable"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	tfdiags.AssertNoErrors(t, diags)

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.count_child[0].test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	rs = state.ResourceInstance(mustResourceInstanceAddr("module.count_child[1].test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	rs = state.ResourceInstance(mustResourceInstanceAddr("module.for_each_child[\"a\"].test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("resources not imported")
	}
}

func TestContextApply_import_duplication(t *testing.T) {
	// two imports to the same resource - one in root, one in the child mod
	m := testModule(t, "import-block-duplication")

	p := mockProviderWithResourceTypeSchema("test_object", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":          {Type: cty.String, Computed: true},
			"test_string": {Type: cty.String, Optional: true},
		},
	})
	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("importable"),
					"id":          cty.StringVal(req.ID),
				}),
			}},
		}
	}
	p.ReadResourceFn = func(r providers.ReadResourceRequest) providers.ReadResourceResponse {
		id := r.PriorState.GetAttr("id")
		return providers.ReadResourceResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"test_string": cty.StringVal("importable"),
				"id":          id,
			}),
		}
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	tfdiags.AssertNoErrors(t, diags)

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.child.test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}
	var attrs map[string]interface{}
	err := json.Unmarshal(rs.Current.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}

	// the parent module import takes precedence (confirming the comment in refactoring/import_statement.go)
	if got, want := attrs["id"], "rootimport"; got != want {
		t.Fatalf("wrong id! got:  %#v  want: %#v\n", got, want)
	}
}

func TestContextApply_import_in_state_not_config(t *testing.T) {
	// regression test for https://github.com/hashicorp/terraform/issues/38660:
	// ensure a clear error when the requested import resource exists in state
	// but not config
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			import {
				to = test_object.foo
				id = "importable"
			}
		`,
	})

	p := simpleMockProvider()
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("importable"),
				}),
			},
		},
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("hi, mom!"),
		}),
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"foo"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	// verify the module is valid
	diags := ctx.Validate(m, &ValidateOpts{})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Plan(m, state, nil)
	if !strings.Contains(diags.Err().Error(), "Configuration for import target does not exist") {
		t.Fatalf("wrong error! got %s\n", diags.Err())
	}
}
