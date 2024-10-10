// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_ephemeralValues(t *testing.T) {
	for name, tc := range map[string]struct {
		module                                      map[string]string
		expectValidateDiagnostics                   func(m *configs.Config) tfdiags.Diagnostics
		expectPlanDiagnostics                       func(m *configs.Config) tfdiags.Diagnostics
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

		"terraform.applying": {
			module: map[string]string{
				"child/main.tf": `
output "value" {
    value = terraform.applying
    # Testing that this errors in the best way to ensure the symbol is ephemeral
    ephemeral = false 
}
`,
				"main.tf": `
module "child" {
    source = "./child"
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral value not allowed",
					Detail:   "This output value is not declared as returning an ephemeral value, so it cannot be set to a result derived from an ephemeral value.",
				})
			},
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
				attr := req.Config.GetAttr("test_string")
				if attr.AsString() != "test string" {
					resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))
				}
				return resp
			},
		},

		"provider reference through module": {
			module: map[string]string{
				"child/main.tf": `
ephemeral "ephem_resource" "data" {
}

output "value" {
    value = ephemeral.ephem_resource.data.value
    ephemeral = true
}
`,
				"main.tf": `
module "child" {
    source = "./child"
}

provider "test" {
  test_string = module.child.value
}

resource "test_object" "test" {
}
`,
			},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,
			assertTestProviderConfigure: func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
				attr := req.Config.GetAttr("test_string")
				if attr.AsString() != "test string" {
					resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))
				}
				return resp
			},
		},

		"resource expansion - for_each": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {}
resource "test_object" "test" {
  for_each = toset(ephemeral.ephem_resource.data.list)
  test_string = each.value
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral output not allowed",
					Detail:   "Ephemeral outputs are not allowed in for_each expressions",
				})
			},
		},

		"resource expansion - count": {
			module: map[string]string{

				"main.tf": `
ephemeral "ephem_resource" "data" {}
resource "test_object" "test" {
  count = length(ephemeral.ephem_resource.data.list)
  test_string = count.index
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid count argument",
					Detail:   `The given "count" is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify the number of resource instances.`,
				})
			},
		},

		"module expansion - for_each": {
			module: map[string]string{
				"child/main.tf": `
output "value" {
    value = "static value"
}
`,
				"main.tf": `
ephemeral "ephem_resource" "data" {
}
module "child" {
    for_each = toset(ephemeral.ephem_resource.data.list)
    source = "./child"
}
`,
			},

			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral output not allowed",
					Detail:   "Ephemeral outputs are not allowed in for_each expressions",
				})
			},
		},

		"module expansion - count": {
			module: map[string]string{
				"child/main.tf": `
output "value" {
    value = "static value"
}
`,
				"main.tf": `
ephemeral "ephem_resource" "data" {}
module "child" {
    count = length(ephemeral.ephem_resource.data.list)
    source = "./child"
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid count argument",
					Detail:   `The given "count" is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify the number of resource instances.`,
				})
			},
		},

		"import expansion": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {}

import {
  for_each = toset(ephemeral.ephem_resource.data.value)
  id = each.value.id
  to = each.value.to
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral output not allowed",
					Detail:   "Ephemeral outputs are not allowed in for_each expressions",
				})
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

									"list": {
										Type:     cty.List(cty.String),
										Computed: true,
									},

									"map": {
										Type:     cty.List(cty.Map(cty.String)),
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
					"list":  cty.ListVal([]cty.Value{cty.StringVal("test string 1"), cty.StringVal("test string 2")}),
					"map": cty.ListVal([]cty.Value{
						cty.MapVal(map[string]cty.Value{
							"id": cty.StringVal("id-0"),
							"to": cty.StringVal("aws_instance.a"),
						}),
						cty.MapVal(map[string]cty.Value{
							"id": cty.StringVal("id-1"),
							"to": cty.StringVal("aws_instance.b"),
						}),
					}),
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
			if tc.expectValidateDiagnostics != nil {
				assertDiagnosticsSummaryAndDetailMatch(t, diags, tc.expectValidateDiagnostics(m))
				// If we expect diagnostics, we should not continue with the plan
				// as it will fail.
				return
			} else {
				assertNoDiagnostics(t, diags)
			}

			if tc.expectValidateEphemeralResourceConfigCalled {
				if !ephem.ValidateEphemeralResourceConfigCalled {
					t.Fatal("ValidateEphemeralResourceConfig not called")
				}
			}

			_, diags = ctx.Plan(m, nil, DefaultPlanOpts)
			if tc.expectPlanDiagnostics != nil {
				assertDiagnosticsMatch(t, diags, tc.expectPlanDiagnostics(m))
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
