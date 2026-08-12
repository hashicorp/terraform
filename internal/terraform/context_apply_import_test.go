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
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
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

	assertImportedId(t, state, "module.child.test_object.bar[\"first\"]", "testa")
	assertImportedId(t, state, "module.child.test_object.bar[\"second\"]", "testb")
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

	// the parent module import takes precedence (confirming the comment in refactoring/import_statement.go)
	assertImportedId(t, state, "module.child.test_object.foo", "rootimport")
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

func TestContextApply_import_expressions(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "child" {
  source = "./child"
}
resource "test_object" "root_foo" {}
resource "test_object" "root_bar" {}

locals {
  root_foo_id = "foo-123"
  root_bar_id = "bar-123"
}
import {
  to = test_object.root_foo
  id = local.root_foo_id
}
import {
  to = test_object.root_bar
  identity = {
    id = local.root_bar_id
  }
}
		`,
		"child/main.tf": `
resource "test_object" "child_foo" {}
resource "test_object" "child_bar" {}

locals {
  child_foo_id = "foo-456"
  child_bar_id = "bar-456"
}
import {
  to = test_object.child_foo
  id = local.child_foo_id
}
import {
  to = test_object.child_bar
  identity = {
    id = local.child_bar_id
  }
}

		`,
	})

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Body: &configschema.Block{}},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"id":          {Type: cty.String, Computed: true},
							"test_string": {Type: cty.String, Optional: true},
						},
					},
					Identity: &configschema.Object{
						Nesting: configschema.NestingSingle,
						Attributes: map[string]*configschema.Attribute{
							"id": {Type: cty.String, Required: true},
						},
					},
				},
			},
		},
	}

	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		if !req.Identity.IsNull() {
			return providers.ImportResourceStateResponse{
				ImportedResources: []providers.ImportedResource{
					{
						TypeName: "test_object",
						Identity: cty.ObjectVal(map[string]cty.Value{
							"id": req.Identity.GetAttr("id"),
						}),
						State: cty.ObjectVal(map[string]cty.Value{
							"test_string": cty.StringVal("importable"),
							"id":          req.Identity.GetAttr("id"),
						}),
					},
				},
			}
		}
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: "test_object",
					State: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("importable"),
						"id":          cty.StringVal(req.ID),
					}),
					Identity: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(req.ID),
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
			Identity: cty.ObjectVal(map[string]cty.Value{
				"id": id,
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
		t.Fatal("resources not imported")
	}

	assertImportedId(t, state, "test_object.root_foo", "foo-123")
	assertImportedId(t, state, "test_object.root_bar", "bar-123")
	assertImportedId(t, state, "module.child.test_object.child_foo", "foo-456")
	assertImportedId(t, state, "module.child.test_object.child_bar", "bar-456")
}

func TestContextApply_import_into_module_expressions(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "child" {
  source = "./child"
}

locals {
  root_foo_id = "foo-123"
  root_bar_id = "bar-123"
}
import {
  to = module.child.test_object.child_foo
  id = local.root_foo_id
}
import {
  to = module.child.test_object.child_bar
  identity = {
    id = local.root_bar_id
  }
}
		`,
		"child/main.tf": `
resource "test_object" "child_foo" {}
resource "test_object" "child_bar" {}

locals {
  child_foo_id = "foo-456"
  child_bar_id = "bar-456"
}
module "grandchild" {
  source = "./grandchild"
}
import {
  to = module.grandchild.test_object.grandchild_foo
  id = local.child_foo_id
}
import {
  to = module.grandchild.test_object.grandchild_bar
  identity = {
    id = local.child_bar_id
  }
}
		`,
		"child/grandchild/main.tf": `
resource "test_object" "grandchild_foo" {}
resource "test_object" "grandchild_bar" {}
		`,
	})

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Body: &configschema.Block{}},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"id":          {Type: cty.String, Computed: true},
							"test_string": {Type: cty.String, Optional: true},
						},
					},
					Identity: &configschema.Object{
						Nesting: configschema.NestingSingle,
						Attributes: map[string]*configschema.Attribute{
							"id": {Type: cty.String, Required: true},
						},
					},
				},
			},
		},
	}

	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		if !req.Identity.IsNull() {
			return providers.ImportResourceStateResponse{
				ImportedResources: []providers.ImportedResource{
					{
						TypeName: "test_object",
						Identity: cty.ObjectVal(map[string]cty.Value{
							"id": req.Identity.GetAttr("id"),
						}),
						State: cty.ObjectVal(map[string]cty.Value{
							"test_string": cty.StringVal("importable"),
							"id":          req.Identity.GetAttr("id"),
						}),
					},
				},
			}
		}
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: "test_object",
					State: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("importable"),
						"id":          cty.StringVal(req.ID),
					}),
					Identity: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(req.ID),
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
			Identity: cty.ObjectVal(map[string]cty.Value{
				"id": id,
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
		t.Fatal("resources not imported")
	}

	assertImportedId(t, state, "module.child.test_object.child_foo", "foo-123")
	assertImportedId(t, state, "module.child.test_object.child_bar", "bar-123")
	assertImportedId(t, state, "module.child.module.grandchild.test_object.grandchild_foo", "foo-456")
	assertImportedId(t, state, "module.child.module.grandchild.test_object.grandchild_bar", "bar-456")
}

func TestContextApply_import_into_module_expressions_foreach(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "child" {
  for_each = local.rep_data
  source = "./child"
  rep_data = toset([each.key])
}

locals {
  num = 123
  rep_data = toset(["foo", "bar"])
}
import {
  for_each = local.rep_data
  to = module.child[each.key].test_object.child_obj[each.key]
  id = "${each.key}-${local.num}"
}
		`,
		"child/main.tf": `
variable "rep_data" {
  type = set(string)
}
resource "test_object" "child_obj" {
  for_each = var.rep_data
}
		`,
	})

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Body: &configschema.Block{}},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"id":          {Type: cty.String, Computed: true},
							"test_string": {Type: cty.String, Optional: true},
						},
					},
					Identity: &configschema.Object{
						Nesting: configschema.NestingSingle,
						Attributes: map[string]*configschema.Attribute{
							"id": {Type: cty.String, Required: true},
						},
					},
				},
			},
		},
	}

	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		if !req.Identity.IsNull() {
			return providers.ImportResourceStateResponse{
				ImportedResources: []providers.ImportedResource{
					{
						TypeName: "test_object",
						Identity: cty.ObjectVal(map[string]cty.Value{
							"id": req.Identity.GetAttr("id"),
						}),
						State: cty.ObjectVal(map[string]cty.Value{
							"test_string": cty.StringVal("importable"),
							"id":          req.Identity.GetAttr("id"),
						}),
					},
				},
			}
		}
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: "test_object",
					State: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("importable"),
						"id":          cty.StringVal(req.ID),
					}),
					Identity: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(req.ID),
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
			Identity: cty.ObjectVal(map[string]cty.Value{
				"id": id,
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
		t.Fatal("resources not imported")
	}

	assertImportedId(t, state, `module.child["foo"].test_object.child_obj["foo"]`, "foo-123")
	assertImportedId(t, state, `module.child["bar"].test_object.child_obj["bar"]`, "bar-123")
}

func assertImportedId(t *testing.T, state *states.State, resourceAddr, expectedID string) {
	t.Helper()

	rs := state.ResourceInstance(mustResourceInstanceAddr(resourceAddr))
	if rs == nil {
		t.Errorf("imported resource %q not found in module", resourceAddr)
		return
	}
	var attrs map[string]interface{}
	err := json.Unmarshal(rs.Current.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}
	if got := attrs["id"]; got != expectedID {
		t.Errorf("wrong id for %q got:  %#v  want: %#v\n", resourceAddr, got, expectedID)
	}
}
