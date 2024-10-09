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
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_ephemeralValues(t *testing.T) {
	for name, tc := range map[string]struct {
		module                                      map[string]string
		expectValidateDiagnostics                   []tfdiags.Diagnostic
		expectPlanDiagnostics                       []tfdiags.Diagnostic
		expectOpenEphemeralResourceCalled           bool
		expectValidateEphemeralResourceConfigCalled bool
		expectCloseEphemeralResourceCalled          bool
		assertTestProviderConfigure                 func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse)
	}{
		"basic": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {
}
`},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,
		},

		"provider reference": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {
}

provider "test" {
  test_string = ephemeral.ephem_resource.data.value
}

resource "test_object" "test" {
}
`,
			},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,
			assertTestProviderConfigure: func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
				if req.Config.GetAttr("test_string").AsString() != "test string" {
					resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))
				}
				return resp
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, tc.module)

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
				if tc.assertTestProviderConfigure != nil {
					return tc.assertTestProviderConfigure(req)
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
			if len(tc.expectValidateDiagnostics) > 0 {
				assertDiagnosticsMatch(t, diags, tc.expectValidateDiagnostics)
			} else {
				assertNoDiagnostics(t, diags)
			}

			if tc.expectValidateEphemeralResourceConfigCalled {
				if !ephem.ValidateEphemeralResourceConfigCalled {
					t.Fatal("ValidateEphemeralResourceConfig not called")
				}
			}

			_, diags = ctx.Plan(m, nil, DefaultPlanOpts)
			if len(tc.expectPlanDiagnostics) > 0 {
				assertDiagnosticsMatch(t, diags, tc.expectPlanDiagnostics)
			} else {
				assertNoDiagnostics(t, diags)
			}

			if tc.expectOpenEphemeralResourceCalled {
				if !ephem.OpenEphemeralResourceCalled {
					t.Fatal("OpenEphemeralResource not called")
				}
			}

			if tc.expectCloseEphemeralResourceCalled {
				if !ephem.CloseEphemeralResourceCalled {
					t.Fatal("CloseEphemeralResource not called")
				}
			}
		})
	}
}
