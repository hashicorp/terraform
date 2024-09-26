// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_ephemeralBasic(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
ephemeral "test_resource" "data" {
}

output "data" {
  ephemeral = true
  value = ephemeral.test_resource.data.value
}
`,
	})

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			EphemeralResourceTypes: map[string]providers.Schema{
				"test_resource": {
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

	p.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		resp.Result = cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("test string"),
		})
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			// The providers never actually going to get called here, we should
			// catch the error long before anything happens.
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, &ValidateOpts{})
	assertNoDiagnostics(t, diags)

	if !p.ValidateEphemeralResourceConfigCalled {
		t.Fatal("ValidateEphemeralResourceConfig not called")
	}

	_, diags = ctx.Plan(m, nil, DefaultPlanOpts)
	assertNoDiagnostics(t, diags)

	if !p.OpenEphemeralResourceCalled {
		t.Fatal("OpenEphemeralResource not called")
	}

	if !p.CloseEphemeralResourceCalled {
		t.Fatal("CloseEphemeralResource not called")
	}
}

func TestContext2Plan_ephemeralProviderRef(t *testing.T) {
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
		return resp
	}

	p := simpleMockProvider()
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		if req.Config.GetAttr("test_string").AsString() != "test string" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			// The providers never actually going to get called here, we should
			// catch the error long before anything happens.
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, &ValidateOpts{})
	assertNoDiagnostics(t, diags)

	if !ephem.ValidateEphemeralResourceConfigCalled {
		t.Fatal("ValidateEphemeralResourceConfig not called")
	}

	_, diags = ctx.Plan(m, nil, DefaultPlanOpts)
	assertNoDiagnostics(t, diags)
}
