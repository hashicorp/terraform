// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Apply_ephemeralProviderRef(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
ephemeral "ephem_resource" "data" {
}

provider "test" {
  test_string = ephemeral.ephem_resource.data.value
}

resource "test_object" "test" {
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			EphemeralResourceTypes: map[string]providers.Schema{
				"ephem_resource": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}

	ephem.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		resp.Result = cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("test string"),
		})
		resp.RenewAt = time.Now().Add(11 * time.Millisecond)
		resp.Private = []byte("private data")
		return resp
	}

	// make sure we can wait for renew to be called
	renewed := make(chan bool)
	renewDone := sync.OnceFunc(func() { close(renewed) })

	ephem.RenewEphemeralResourceFn = func(req providers.RenewEphemeralResourceRequest) (resp providers.RenewEphemeralResourceResponse) {
		defer renewDone()
		if string(req.Private) != "private data" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("invalid private data %q", req.Private))
			return resp
		}

		resp.RenewAt = time.Now().Add(10 * time.Millisecond)
		resp.Private = req.Private
		return resp
	}

	p := simpleMockProvider()
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		// wait here for the ephemeral value to be renewed at least once
		<-renewed
		if req.Config.GetAttr("test_string").AsString() != "test string" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, nil, DefaultPlanOpts)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.RenewEphemeralResourceCalled {
		t.Error("RenewEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}

	// reset the ephemeral call flags and the gate
	ephem.OpenEphemeralResourceCalled = false
	ephem.RenewEphemeralResourceCalled = false
	ephem.CloseEphemeralResourceCalled = false
	renewed = make(chan bool)
	renewDone = sync.OnceFunc(func() { close(renewed) })

	_, diags = ctx.Apply(plan, m, nil)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.RenewEphemeralResourceCalled {
		t.Error("RenewEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}
}

func TestContext2Apply_ephemeralApplyAndDestroy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "test" {
  for_each = toset(["a"])
  source = "./mod"
  input = each.value
}

provider "test" {
  test_string = module.test["a"].data
}

resource "test_object" "test" {
}
`,
		"./mod/main.tf": `
variable input {
}

ephemeral "ephem_resource" "data" {
}

output "data" {
  ephemeral = true
  value = ephemeral.ephem_resource.data.value
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			EphemeralResourceTypes: map[string]providers.Schema{
				"ephem_resource": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}

	ephemeralData := struct {
		sync.Mutex
		data string
	}{}

	ephem.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		ephemeralData.Lock()
		defer ephemeralData.Unlock()
		// open sets the data
		ephemeralData.data = "test string"

		resp.Result = cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal(ephemeralData.data),
		})
		return resp
	}

	// closing with invalidate the ephemeral data
	ephem.CloseEphemeralResourceFn = func(providers.CloseEphemeralResourceRequest) (resp providers.CloseEphemeralResourceResponse) {
		ephemeralData.Lock()
		defer ephemeralData.Unlock()

		// close invalidates the data
		ephemeralData.data = ""
		return resp
	}

	p := simpleMockProvider()
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		// wait here for the ephemeral value to be renewed at least once
		if req.Config.GetAttr("test_string").AsString() != "test string" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))

			// check if the ephemeral data is actually valid, as if we were
			// using something like a temporary authentication token which gets
			// revoked.
			ephemeralData.Lock()
			defer ephemeralData.Unlock()
			if ephemeralData.data == "" {
				resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("ephemeralData from config not valid: %#v", req.Config))
			}
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, nil, DefaultPlanOpts)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}

	// reset the ephemeral call flags and data
	ephem.OpenEphemeralResourceCalled = false
	ephem.CloseEphemeralResourceCalled = false

	state, diags := ctx.Apply(plan, m, nil)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}

	// now reverse the process
	ephem.OpenEphemeralResourceCalled = false
	ephem.CloseEphemeralResourceCalled = false

	plan, diags = ctx.Plan(m, state, &PlanOpts{Mode: plans.DestroyMode})
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}

	ephem.OpenEphemeralResourceCalled = false
	ephem.CloseEphemeralResourceCalled = false

	_, diags = ctx.Apply(plan, m, nil)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}
}

func TestContext2Apply_ephemeralChecks(t *testing.T) {
	// test the full validate-plan-apply lifecycle for ephemeral conditions
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "input" {
  type = string
}

ephemeral "ephem_resource" "data" {
  for_each = toset(["a", "b"])
  lifecycle {
    precondition {
      condition = var.input == "ok"
      error_message = "input not ok"
    }
    postcondition {
      condition = self.value != null
      error_message = "value is null"
    }
  }
}

provider "test" {
  test_string = ephemeral.ephem_resource.data["a"].value
}

resource "test_object" "test" {
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			EphemeralResourceTypes: map[string]providers.Schema{
				"ephem_resource": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}

	ephem.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		resp.Result = cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("test string"),
		})
		return resp
	}

	p := simpleMockProvider()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, &ValidateOpts{})
	assertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(m, nil, &PlanOpts{
		SetVariables: InputValues{
			"input": &InputValue{
				Value:      cty.StringVal("ok"),
				SourceType: ValueFromConfig,
			},
		},
	})
	assertNoDiagnostics(t, diags)

	// reset the ephemeral call flags
	ephem.ConfigureProviderCalled = false

	_, diags = ctx.Apply(plan, m, nil)
	assertNoDiagnostics(t, diags)
}

func TestContext2Apply_write_only_attribute_not_in_plan_and_state(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true
}

resource "ephem_write_only" "wo" {
  normal     = "normal"
  write_only = var.ephem
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								WriteOnly: true,
								Required:  true,
							},
						},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
		},
	})

	ephemVar := &InputValue{
		Value:      cty.StringVal("ephemeral_value"),
		SourceType: ValueFromCLIArg,
	}
	plan, diags := ctx.Plan(m, nil, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})
	assertNoDiagnostics(t, diags)

	if len(plan.Changes.Resources) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.Changes.Resources))
	}

	schemas, schemaDiags := ctx.Schemas(m, plan.PriorState)
	assertNoDiagnostics(t, schemaDiags)
	planChanges, err := plan.Changes.Decode(schemas)
	if err != nil {
		t.Fatalf("Failed to decode plan changes: %v.", err)
	}

	if !planChanges.Resources[0].After.GetAttr("write_only").IsNull() {
		t.Fatalf("Expected write_only to be null, got %v", planChanges.Resources[0].After.GetAttr("write_only"))
	}

	state, diags := ctx.Apply(plan, m, &ApplyOpts{
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})
	assertNoDiagnostics(t, diags)

	resource := state.Resource(addrs.AbsResource{
		Module: addrs.RootModuleInstance,
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "ephem_write_only",
			Name: "wo",
		},
	})

	if resource == nil {
		t.Fatalf("Resource not found")
	}

	resourceInstance := resource.Instances[addrs.NoKey]
	if resourceInstance == nil {
		t.Fatalf("Resource instance not found")
	}

	attrs, err := resourceInstance.Current.Decode(cty.Object(map[string]cty.Type{
		"normal":     cty.String,
		"write_only": cty.String,
	}))
	if err != nil {
		t.Fatalf("Failed to decode attributes: %v", err)
	}

	if attrs.Value.GetAttr("normal").AsString() != "normal" {
		t.Fatalf("normal attribute not as expected")
	}

	if !attrs.Value.GetAttr("write_only").IsNull() {
		t.Fatalf("write_only attribute should be null")
	}
}

func TestContext2Apply_update_write_only_attribute_not_in_plan_and_state(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true
}

resource "ephem_write_only" "wo" {
  normal     = "normal"
  write_only = var.ephem
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								WriteOnly: true,
								Required:  true,
							},
						},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
		},
	})

	ephemVar := &InputValue{
		Value:      cty.StringVal("ephemeral_value"),
		SourceType: ValueFromCLIArg,
	}

	priorState := states.BuildState(func(state *states.SyncState) {
		state.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("ephem_write_only.wo"),
			&states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady,
				AttrsJSON: mustParseJson(map[string]interface{}{
					"normal": "outdated",
				}),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("ephem"),
				Module:   addrs.RootModule,
			})
	})

	plan, diags := ctx.Plan(m, priorState, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})
	assertNoDiagnostics(t, diags)

	if len(plan.Changes.Resources) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.Changes.Resources))
	}

	schemas, schemaDiags := ctx.Schemas(m, plan.PriorState)
	assertNoDiagnostics(t, schemaDiags)
	planChanges, err := plan.Changes.Decode(schemas)
	if err != nil {
		t.Fatalf("Failed to decode plan changes: %v.", err)
	}

	if !planChanges.Resources[0].After.GetAttr("write_only").IsNull() {
		t.Fatalf("Expected write_only to be null, got %v", planChanges.Resources[0].After.GetAttr("write_only"))
	}

	state, diags := ctx.Apply(plan, m, &ApplyOpts{
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})
	assertNoDiagnostics(t, diags)

	resource := state.Resource(addrs.AbsResource{
		Module: addrs.RootModuleInstance,
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "ephem_write_only",
			Name: "wo",
		},
	})

	if resource == nil {
		t.Fatalf("Resource not found")
	}

	resourceInstance := resource.Instances[addrs.NoKey]
	if resourceInstance == nil {
		t.Fatalf("Resource instance not found")
	}

	attrs, err := resourceInstance.Current.Decode(cty.Object(map[string]cty.Type{
		"normal":     cty.String,
		"write_only": cty.String,
	}))
	if err != nil {
		t.Fatalf("Failed to decode attributes: %v", err)
	}

	if attrs.Value.GetAttr("normal").AsString() != "normal" {
		t.Fatalf("normal attribute not as expected")
	}

	if !attrs.Value.GetAttr("write_only").IsNull() {
		t.Fatalf("write_only attribute should be null")
	}
}

func TestContext2Apply_normal_attributes_becomes_write_only_attribute(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true
}

resource "ephem_write_only" "wo" {
  normal     = "normal"
  write_only = var.ephem
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								WriteOnly: true,
								Required:  true,
							},
						},
					},
				},
			},
		},
	}
	// Below we force the write_only attribute's returned state to be Null, mimicking what the plugin-framework would
	// return during an UpgradeResourceState RPC
	ephem.UpgradeResourceStateFn = func(ursr providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
		return providers.UpgradeResourceStateResponse{
			UpgradedState: cty.ObjectVal(map[string]cty.Value{
				"normal":     cty.StringVal("normal"),
				"write_only": cty.NullVal(cty.String),
			}),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
		},
	})

	ephemVar := &InputValue{
		Value:      cty.StringVal("ephemeral_value"),
		SourceType: ValueFromCLIArg,
	}

	priorState := states.BuildState(func(state *states.SyncState) {
		state.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("ephem_write_only.wo"),
			&states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady,
				AttrsJSON: mustParseJson(map[string]interface{}{
					"normal":     "normal",
					"write_only": "this was not ephemeral but now is",
				}),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("ephem"),
				Module:   addrs.RootModule,
			})
	})

	plan, diags := ctx.Plan(m, priorState, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})
	assertNoDiagnostics(t, diags)

	if len(plan.Changes.Resources) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.Changes.Resources))
	}

	schemas, schemaDiags := ctx.Schemas(m, plan.PriorState)
	assertNoDiagnostics(t, schemaDiags)
	planChanges, err := plan.Changes.Decode(schemas)
	if err != nil {
		t.Fatalf("Failed to decode plan changes: %v.", err)
	}

	if !planChanges.Resources[0].After.GetAttr("write_only").IsNull() {
		t.Fatalf("Expected write_only to be null, got %v", planChanges.Resources[0].After.GetAttr("write_only"))
	}

	state, diags := ctx.Apply(plan, m, &ApplyOpts{
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})
	assertNoDiagnostics(t, diags)

	resource := state.Resource(addrs.AbsResource{
		Module: addrs.RootModuleInstance,
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "ephem_write_only",
			Name: "wo",
		},
	})

	if resource == nil {
		t.Fatalf("Resource not found")
	}

	resourceInstance := resource.Instances[addrs.NoKey]
	if resourceInstance == nil {
		t.Fatalf("Resource instance not found")
	}

	attrs, err := resourceInstance.Current.Decode(cty.Object(map[string]cty.Type{
		"normal":     cty.String,
		"write_only": cty.String,
	}))
	if err != nil {
		t.Fatalf("Failed to decode attributes: %v", err)
	}

	if attrs.Value.GetAttr("normal").AsString() != "normal" {
		t.Fatalf("normal attribute not as expected")
	}

	if !attrs.Value.GetAttr("write_only").IsNull() {
		t.Fatalf("write_only attribute should be null")
	}
}

func TestContext2Apply_write_only_attribute_provider_applies_with_non_null_value(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true
}

resource "ephem_write_only" "wo" {
  normal     = "normal"
  write_only = var.ephem
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								WriteOnly: true,
								Required:  true,
							},
						},
					},
				},
			},
		},
		ApplyResourceChangeResponse: &providers.ApplyResourceChangeResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"normal":     cty.StringVal("normal"),
				"write_only": cty.StringVal("the provider should have set this to null"),
			}),
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
		},
	})

	ephemVar := &InputValue{
		Value:      cty.StringVal("ephemeral_value"),
		SourceType: ValueFromCLIArg,
	}

	plan, planDiags := ctx.Plan(m, nil, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})

	assertNoDiagnostics(t, planDiags)

	_, diags := ctx.Apply(plan, m, &ApplyOpts{
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})

	var expectedDiags tfdiags.Diagnostics

	expectedDiags = append(expectedDiags, tfdiags.Sourceless(
		tfdiags.Error,
		"Provider produced invalid object",
		`Provider "provider[\"registry.terraform.io/hashicorp/ephem\"]" returned a value for the write-only attribute "ephem_write_only.wo.write_only" after apply. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.`,
	))

	tfdiags.AssertDiagnosticsMatch(t, diags, expectedDiags)
}

func TestContext2Apply_write_only_attribute_provider_plan_with_non_null_value(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true
}

resource "ephem_write_only" "wo" {
  normal     = "normal"
  write_only = var.ephem
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								WriteOnly: true,
								Required:  true,
							},
						},
					},
				},
			},
		},
		PlanResourceChangeResponse: &providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"normal":     cty.StringVal("normal"),
				"write_only": cty.StringVal("the provider should have set this to null"),
			}),
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
		},
	})

	ephemVar := &InputValue{
		Value:      cty.StringVal("ephemeral_value"),
		SourceType: ValueFromCLIArg,
	}

	_, diags := ctx.Plan(m, nil, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
	})

	var expectedDiags tfdiags.Diagnostics

	expectedDiags = append(expectedDiags, tfdiags.Sourceless(
		tfdiags.Error,
		"Provider produced invalid plan",
		`Provider "provider[\"registry.terraform.io/hashicorp/ephem\"]" returned a value for the write-only attribute "ephem_write_only.wo.write_only" during planning. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.`,
	))

	tfdiags.AssertDiagnosticsMatch(t, diags, expectedDiags)
}

func TestContext2Apply_write_only_attribute_provider_read_with_non_null_value(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true
}

resource "ephem_write_only" "wo" {
  normal     = "normal"
  write_only = var.ephem
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								WriteOnly: true,
								Required:  true,
							},
						},
					},
				},
			},
		},
		ReadResourceResponse: &providers.ReadResourceResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"normal":     cty.StringVal("normal"),
				"write_only": cty.StringVal("the provider should have set this to null"),
			}),
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
		},
	})

	ephemVar := &InputValue{
		Value:      cty.StringVal("ephemeral_value"),
		SourceType: ValueFromCLIArg,
	}
	priorState := states.BuildState(func(state *states.SyncState) {
		state.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("ephem_write_only.wo"),
			&states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady,
				AttrsJSON: mustParseJson(map[string]interface{}{
					"normal": "outdated",
				}),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("ephem"),
				Module:   addrs.RootModule,
			})
	})

	_, diags := ctx.Plan(m, priorState, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"ephem": ephemVar,
		},
		SkipRefresh: false,
	})

	var expectedDiags tfdiags.Diagnostics

	expectedDiags = append(expectedDiags, tfdiags.Sourceless(
		tfdiags.Error,
		"Provider produced invalid object",
		`Provider "provider[\"registry.terraform.io/hashicorp/ephem\"]" returned a value for the write-only attribute "ephem_write_only.wo.write_only" during refresh. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.`,
	))

	tfdiags.AssertDiagnosticsMatch(t, diags, expectedDiags)
}
