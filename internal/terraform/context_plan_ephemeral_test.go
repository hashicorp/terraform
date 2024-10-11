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
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_ephemeralValues(t *testing.T) {
	for name, tc := range map[string]struct {
		toBeImplemented                             bool // Skip the test
		module                                      map[string]string
		expectValidateDiagnostics                   func(m *configs.Config) tfdiags.Diagnostics
		expectPlanDiagnostics                       func(m *configs.Config) tfdiags.Diagnostics
		expectOpenEphemeralResourceCalled           bool
		expectValidateEphemeralResourceConfigCalled bool
		expectCloseEphemeralResourceCalled          bool
		assertTestProviderConfigure                 func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse)
		assertPlan                                  func(*testing.T, *plans.Plan)
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

		"normal attribute": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {
}

resource "test_object" "test" {
  test_string = ephemeral.ephem_resource.data.value
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid use of ephemeral value",
					Detail:   "Ephemeral values are not valid in resource arguments, because resource instances must persist between Terraform phases.",
				})
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
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each argument",
					Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
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
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
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

			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each argument",
					Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
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
  for_each = toset(ephemeral.ephem_resource.data.list)
  id = each.value
  to = test_object.test[each.value]
}

resource "test_object" "test" {
    for_each = toset(ephemeral.ephem_resource.data.list)
    test_string = each.value
}
`,
			},
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid for_each argument",
						Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
					},
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid for_each argument",
						Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
					},
				)
			},
		},

		"functions": {
			toBeImplemented: true,
			module: map[string]string{
				"child/main.tf": `
ephemeral "ephem_resource" "data" {}

# We expect this to error since it should be an ephemeral value
output "value" {
    value = max(42, length(ephemeral.ephem_resource.data.list))
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

		"provider-defined functions": {
			toBeImplemented: true,
			module: map[string]string{
				"child/main.tf": `
				
terraform {
    required_providers {
        ephem = {
            source = "hashicorp/ephem"
        }
    }
}
ephemeral "ephem_resource" "data" {}

# We expect this to error since it should be an ephemeral value
output "value" {
    value = provider::ephem::either(ephemeral.ephem_resource.data.value, "b")
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

		"check blocks": {
			toBeImplemented: true,
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {}

check "check_using_ephemeral_value" {
  assert {
    condition = ephemeral.ephem_resource.data.bool
    error_message = "This should not fail"
  }
}
				`,
			},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,

			assertPlan: func(t *testing.T, p *plans.Plan) {
				// Checks using ephemeral values should not be included in the plan
				if p.Checks.ConfigResults.Len() > 0 {
					t.Fatalf("Expected no checks to be included in the plan, but got %d", p.Checks.ConfigResults.Len())
				}
			},
		},

		"function ephemeralasnull": {
			module: map[string]string{
				"child/main.tf": `
ephemeral "ephem_resource" "data" {}

output "value" {
    value = ephemeralasnull(ephemeral.ephem_resource.data.value)
}
`,
				"main.tf": `
module "child" {
    source = "./child"
}
				`,
			},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,
		},

		"function ephemeral": {
			toBeImplemented: true,
			module: map[string]string{
				"child/main.tf": `

# We expect this to error since it should be an ephemeral value
output "value" {
    value = ephemeral("hello world")
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
	} {
		t.Run(name, func(t *testing.T) {
			if tc.toBeImplemented {
				t.Skip("To be implemented")
			}
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

									"bool": {
										Type:     cty.Bool,
										Computed: true,
									},
								},
							},
						},
					},
					Functions: map[string]providers.FunctionDecl{
						"either": providers.FunctionDecl{
							Parameters: []providers.FunctionParam{
								{
									Name: "a",
									Type: cty.String,
								},
								{
									Name: "b",
									Type: cty.String,
								},
							},
							ReturnType: cty.String,
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
					"bool": cty.True,
				})
				return resp
			}

			ephem.CallFunctionFn = func(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
				resp.Result = cty.StringVal(req.Arguments[0].AsString())
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

			plan, diags := ctx.Plan(m, nil, DefaultPlanOpts)
			if tc.expectPlanDiagnostics != nil {
				assertDiagnosticsSummaryAndDetailMatch(t, diags, tc.expectPlanDiagnostics(m))
			} else {
				assertNoDiagnostics(t, diags)
			}

			if tc.assertPlan != nil {
				tc.assertPlan(t, plan)
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
