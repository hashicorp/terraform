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
