// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_importResourceBasic(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  test_string = "foo"
}

import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	hook := new(MockHook)
	ctx := testContext2(t, &ContextOpts{
		Hooks: []Hook{hook},
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addr.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
		if instPlan.Importing.ID != "123" {
			t.Errorf("expected import change from \"123\", got non-import change")
		}

		if !hook.PrePlanImportCalled {
			t.Fatalf("PostPlanImport hook not called")
		}
		if addr, wantAddr := hook.PrePlanImportAddr, instPlan.Addr; !addr.Equal(wantAddr) {
			t.Errorf("expected addr to be %s, but was %s", wantAddr, addr)
		}

		if !hook.PostPlanImportCalled {
			t.Fatalf("PostPlanImport hook not called")
		}
		if addr, wantAddr := hook.PostPlanImportAddr, instPlan.Addr; !addr.Equal(wantAddr) {
			t.Errorf("expected addr to be %s, but was %s", wantAddr, addr)
		}
	})
}

func TestContext2Plan_importResourceAlreadyInState(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  test_string = "foo"
}

import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addr.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
		if instPlan.Importing != nil {
			t.Errorf("expected non-import change, got import change %#v", instPlan.Importing)
		}
	})
}

func TestContext2Plan_importResourceUpdate(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  test_string = "bar"
}

import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addr.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Update; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
		if instPlan.Importing.ID != "123" {
			t.Errorf("expected import change from \"123\", got non-import change")
		}
	})
}

func TestContext2Plan_importResourceReplace(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  test_string = "bar"
}

import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addr,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addr.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.DeleteThenCreate; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if instPlan.Importing.ID != "123" {
			t.Errorf("expected import change from \"123\", got non-import change")
		}
	})
}

func TestContext2Plan_importRefreshOnce(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  test_string = "bar"
}

import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	readCalled := 0
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		readCalled++
		state, _ := simpleTestSchema().CoerceValue(cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}))

		return providers.ReadResourceResponse{
			NewState: state,
		}
	}

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addr,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	if readCalled > 1 {
		t.Error("ReadResource called multiple times for import")
	}
}

func TestContext2Plan_importTargetWithKeyDoesNotExist(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  count = 1
  test_string = "bar"
}

import {
  to   = test_object.a[42]
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatalf("expected error but got none")
	}
}

func TestContext2Plan_importIdVariable(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-id-variable")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		SetVariables: InputValues{
			"the_id": &InputValue{
				// let var take its default value
				Value: cty.NilVal,
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_importIdFunc(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-id-func")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_importIdDataSource(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-id-data-source")

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_subnet": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_subnet": {
				Attributes: map[string]*configschema.Attribute{
					"vpc_id": {
						Type:     cty.String,
						Required: true,
					},
					"cidr_block": {
						Type:     cty.String,
						Computed: true,
					},
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"vpc_id":     cty.StringVal("abc"),
			"cidr_block": cty.StringVal("10.0.1.0/24"),
			"id":         cty.StringVal("123"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_subnet",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_importIdModule(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-id-module")

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_lb": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_lb",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_importIdInvalidNull(t *testing.T) {
	p := testProvider("test")
	m := testModule(t, "import-id-invalid-null")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		SetVariables: InputValues{
			"the_id": &InputValue{
				Value: cty.NullVal(cty.String),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), "The import ID cannot be null"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_importIdInvalidEmptyString(t *testing.T) {
	p := testProvider("test")
	m := testModule(t, "import-id-invalid-null")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		SetVariables: InputValues{
			"the_id": &InputValue{
				Value: cty.StringVal(""),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), "The import ID value evaluates to an empty string, please provide a non-empty value."; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_importIdInvalidUnknown(t *testing.T) {
	p := testProvider("test")
	m := testModule(t, "import-id-invalid-unknown")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),
		}
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_resource",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), `The import block "id" argument depends on resource attributes that cannot be determined until apply`; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_importIntoModuleWithGeneratedConfig(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
import {
  to = test_object.a
  id = "123"
}

import {
  to = module.mod.test_object.a
  id = "456"
}

module "mod" {
  source = "./mod"
}
`,
		"./mod/main.tf": `
resource "test_object" "a" {
  test_string = "bar"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		GenerateConfigPath: "generated.tf", // Actual value here doesn't matter, as long as it is not empty.
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	one := mustResourceInstanceAddr("test_object.a")
	two := mustResourceInstanceAddr("module.mod.test_object.a")

	onePlan := plan.Changes.ResourceInstance(one)
	twoPlan := plan.Changes.ResourceInstance(two)

	// This test is just to make sure things work e2e with modules and generated
	// config, so we're not too careful about the actual responses - we're just
	// happy nothing panicked. See the other import tests for actual validation
	// of responses and the like.
	if twoPlan.Action != plans.Update {
		t.Errorf("expected nested item to be updated but was %s", twoPlan.Action)
	}

	if len(onePlan.GeneratedConfig) == 0 {
		t.Errorf("expected root item to generate config but it didn't")
	}
}

func TestContext2Plan_importResourceConfigGen(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		GenerateConfigPath: "generated.tf", // Actual value here doesn't matter, as long as it is not empty.
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addr.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
		if instPlan.Importing.ID != "123" {
			t.Errorf("expected import change from \"123\", got non-import change")
		}

		want := `resource "test_object" "a" {
  test_bool   = null
  test_list   = null
  test_map    = null
  test_number = null
  test_string = "foo"
}`
		got := instPlan.GeneratedConfig
		if diff := cmp.Diff(want, got); len(diff) > 0 {
			t.Errorf("got:\n%s\nwant:\n%s\ndiff:\n%s", got, want, diff)
		}
	})
}

func TestContext2Plan_importResourceConfigGenWithAlias(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
provider "test" {
  alias = "backup"
}

import {
  provider = test.backup
  to       = test_object.a
  id       = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		GenerateConfigPath: "generated.tf", // Actual value here doesn't matter, as long as it is not empty.
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addr.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
		if instPlan.Importing.ID != "123" {
			t.Errorf("expected import change from \"123\", got non-import change")
		}

		want := `resource "test_object" "a" {
  provider    = test.backup
  test_bool   = null
  test_list   = null
  test_map    = null
  test_number = null
  test_string = "foo"
}`
		got := instPlan.GeneratedConfig
		if diff := cmp.Diff(want, got); len(diff) > 0 {
			t.Errorf("got:\n%s\nwant:\n%s\ndiff:\n%s", got, want, diff)
		}
	})
}

func TestContext2Plan_importResourceConfigGenExpandedResource(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
import {
  to       = test_object.a[0]
  id       = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		GenerateConfigPath: "generated.tf",
	})
	if !diags.HasErrors() {
		t.Fatalf("expected plan to error, but it did not")
	}
}

// config generation still succeeds even when planning fails
func TestContext2Plan_importResourceConfigGenWithError(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
import {
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.NullVal(cty.DynamicPseudoType),
		Diagnostics:  tfdiags.Diagnostics(nil).Append(errors.New("plan failed")),
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		GenerateConfigPath: "generated.tf", // Actual value here doesn't matter, as long as it is not empty.
	})
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}

	instPlan := plan.Changes.ResourceInstance(addr)
	if instPlan == nil {
		t.Fatalf("no plan for %s at all", addr)
	}

	want := `resource "test_object" "a" {
  test_bool   = null
  test_list   = null
  test_map    = null
  test_number = null
  test_string = "foo"
}`
	got := instPlan.GeneratedConfig
	if diff := cmp.Diff(want, got); len(diff) > 0 {
		t.Errorf("got:\n%s\nwant:\n%s\ndiff:\n%s", got, want, diff)
	}
}

func TestContext2Plan_importForEach(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  things = {
    first = "first_id"
    second = "second_id"
  }
}

resource "test_object" "a" {
  for_each = local.things
  test_string = "foo"
}

import {
  for_each = local.things
  to = test_object.a[each.key]
  id = each.value
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	firstAddr := mustResourceInstanceAddr(`test_object.a["first"]`)
	secondAddr := mustResourceInstanceAddr(`test_object.a["second"]`)

	for _, instPlan := range plan.Changes.Resources {
		switch {
		case instPlan.Addr.Equal(firstAddr):
			if instPlan.Importing.ID != "first_id" {
				t.Errorf("expected import ID of \"first_id\", got %q", instPlan.Importing.ID)
			}
		case instPlan.Addr.Equal(secondAddr):
			if instPlan.Importing.ID != "second_id" {
				t.Errorf("expected import ID of \"second_id\", got %q", instPlan.Importing.ID)
			}
		default:
			t.Errorf("unexpected change for %s", instPlan.Addr)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	}
}

func TestContext2Plan_importForEachmodule(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  things = {
    brown = "brown_id"
    blue = "blue_id"
  }
}

module "sub" {
  for_each = local.things
  source = "./sub"
  things = local.things
}

import {
  for_each = [
	{
      mod = "brown"
      res = "brown"
      id = "brown_brown_id"
	},
    {
      mod = "brown"
      res = "blue"
      id = "brown_blue_id"
    },
    {
      mod = "blue"
      res = "brown"
      id = "blue_brown_id"
    },
    {
      mod = "blue"
      res = "blue"
      id = "blue_blue_id"
    },
  ]
  to = module.sub[each.value.mod].test_object.a[each.value.res]
  id = each.value.id
}
`,

		"./sub/main.tf": `
variable things {
  type = map(string)
}

locals {
  static_id = "foo"
}

resource "test_object" "a" {
  for_each = var.things
  test_string = local.static_id
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	brownBlueAddr := mustResourceInstanceAddr(`module.sub["brown"].test_object.a["brown"]`)
	brownBrownAddr := mustResourceInstanceAddr(`module.sub["brown"].test_object.a["blue"]`)
	blueBlueAddr := mustResourceInstanceAddr(`module.sub["blue"].test_object.a["brown"]`)
	blueBrownAddr := mustResourceInstanceAddr(`module.sub["blue"].test_object.a["blue"]`)

	for _, instPlan := range plan.Changes.Resources {
		switch {
		case instPlan.Addr.Equal(brownBlueAddr):
			if instPlan.Importing.ID != "brown_brown_id" {
				t.Errorf("expected import ID of \"brown_brown_id\", got %q", instPlan.Importing.ID)
			}
		case instPlan.Addr.Equal(brownBrownAddr):
			if instPlan.Importing.ID != "brown_blue_id" {
				t.Errorf("expected import ID of \"brown_blue_id\", got %q", instPlan.Importing.ID)
			}
		case instPlan.Addr.Equal(blueBlueAddr):
			if instPlan.Importing.ID != "blue_brown_id" {
				t.Errorf("expected import ID of \"blue_brown_id\", got %q", instPlan.Importing.ID)
			}
		case instPlan.Addr.Equal(blueBrownAddr):
			if instPlan.Importing.ID != "blue_blue_id" {
				t.Errorf("expected import ID of \"blue_blue_id\", got %q", instPlan.Importing.ID)
			}
		default:
			t.Errorf("unexpected change for %s", instPlan.Addr)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	}
}

func TestContext2Plan_importForEachPartial(t *testing.T) {
	// one of the imported instances already exists in the state, which should
	// result in a non-import, NoOp change
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  things = {
    first = "first_id"
    second = "second_id"
  }
}

resource "test_object" "a" {
  for_each = local.things
  test_string = "foo"
}

import {
  for_each = local.things
  to = test_object.a[each.key]
  id = each.value
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(`test_object.a["first"]`).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	firstAddr := mustResourceInstanceAddr(`test_object.a["first"]`)
	secondAddr := mustResourceInstanceAddr(`test_object.a["second"]`)

	for _, instPlan := range plan.Changes.Resources {
		switch {
		case instPlan.Addr.Equal(firstAddr):
			if instPlan.Importing != nil {
				t.Errorf("expected no import for %s, got %#v", firstAddr, instPlan.Importing)
			}
		case instPlan.Addr.Equal(secondAddr):
			if instPlan.Importing.ID != "second_id" {
				t.Errorf("expected import ID of \"second_id\", got %q", instPlan.Importing.ID)
			}
		default:
			t.Errorf("unexpected change for %s", instPlan.Addr)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	}
}

func TestContext2Plan_importForEachFromData(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_object" "d" {
}

resource "test_object" "a" {
  count = 2
  test_string = "foo"
}

import {
  for_each = data.test_object.d.objects
  to = test_object.a[each.key]
  id = each.value
}
`,
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": providers.Schema{Block: simpleTestSchema()},
		},
		DataSources: map[string]providers.Schema{
			"test_object": providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"objects": {
							Type:     cty.List(cty.String),
							Computed: true,
						},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"objects": cty.ListVal([]cty.Value{
				cty.StringVal("first_id"), cty.StringVal("second_id"),
			}),
		}),
	}

	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("foo"),
		}),
	}
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("foo"),
				}),
			},
		},
	}

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	firstAddr := mustResourceInstanceAddr(`test_object.a[0]`)
	secondAddr := mustResourceInstanceAddr(`test_object.a[1]`)

	for _, instPlan := range plan.Changes.Resources {
		switch {
		case instPlan.Addr.Equal(firstAddr):
			if instPlan.Importing.ID != "first_id" {
				t.Errorf("expected import ID of \"first_id\", got %q", instPlan.Importing.ID)
			}
		case instPlan.Addr.Equal(secondAddr):
			if instPlan.Importing.ID != "second_id" {
				t.Errorf("expected import ID of \"second_id\", got %q", instPlan.Importing.ID)
			}
		default:
			t.Errorf("unexpected change for %s", instPlan.Addr)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	}
}

func TestContext2Plan_importGenerateNone(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
import {
  for_each = []
  to   = test_object.a
  id   = "123"
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		GenerateConfigPath: "generated.tf",
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	if len(plan.Changes.Resources) != 0 {
		t.Fatal("expected no resource changes")
	}
}
