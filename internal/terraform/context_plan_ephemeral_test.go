// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		toBeImplemented                             bool
		module                                      map[string]string
		expectValidateDiagnostics                   func(m *configs.Config) tfdiags.Diagnostics
		expectPlanDiagnostics                       func(m *configs.Config) tfdiags.Diagnostics
		expectOpenEphemeralResourceCalled           bool
		expectValidateEphemeralResourceConfigCalled bool
		expectCloseEphemeralResourceCalled          bool
		assertTestProviderConfigure                 func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse)
		assertPlan                                  func(*testing.T, *plans.Plan)
		inputs                                      InputValues
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
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "child", "main.tf"),
						Start:    hcl.Pos{Line: 3, Column: 13, Byte: 30},
						End:      hcl.Pos{Line: 3, Column: 31, Byte: 48},
					},
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
					Detail:   `Ephemeral values are not valid for "test_string", because it is not a write-only attribute and must be persisted to state.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 6, Column: 17, Byte: 88},
						End:      hcl.Pos{Line: 6, Column: 52, Byte: 123},
					},
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
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each argument",
					Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 4, Column: 14, Byte: 83},
						End:      hcl.Pos{Line: 4, Column: 55, Byte: 124},
					},
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
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 4, Column: 11, Byte: 80},
						End:      hcl.Pos{Line: 4, Column: 53, Byte: 122},
					},
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
					Summary:  "Invalid for_each argument",
					Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 5, Column: 16, Byte: 71},
						End:      hcl.Pos{Line: 5, Column: 57, Byte: 112},
					},
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
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 4, Column: 13, Byte: 67},
						End:      hcl.Pos{Line: 4, Column: 55, Byte: 109},
					},
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
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid for_each argument",
						Detail:   `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 11, Column: 16, Byte: 207},
							End:      hcl.Pos{Line: 11, Column: 57, Byte: 248},
						},
					},
				)
			},
		},

		"functions": {
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
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "child", "main.tf"),
						Start:    hcl.Pos{Line: 6, Column: 13, Byte: 132},
						End:      hcl.Pos{Line: 6, Column: 64, Byte: 183},
					},
				})
			},
		},

		"provider-defined functions": {
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
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "child", "main.tf"),
						Start:    hcl.Pos{Line: 14, Column: 13, Byte: 245},
						End:      hcl.Pos{Line: 14, Column: 78, Byte: 310},
					},
				})
			},
		},

		"check blocks": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {}

check "check_using_ephemeral_value" {
  assert {
    condition = ephemeral.ephem_resource.data.bool == false
    error_message = "Fine to persist"
  }
  assert {
    condition = ephemeral.ephem_resource.data.bool == false
    error_message = "Shall not be persisted ${ephemeral.ephem_resource.data.bool}"
  }
}
				`,
			},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,

			assertPlan: func(t *testing.T, p *plans.Plan) {
				key := addrs.ConfigCheck{
					Module: addrs.RootModule,
					Check: addrs.Check{
						Name: "check_using_ephemeral_value",
					},
				}
				result, ok := p.Checks.ConfigResults.GetOk(key)
				if !ok {
					t.Fatalf("expected to find check result for %q", key)
				}
				objKey := addrs.AbsCheck{
					Module: addrs.RootModuleInstance,
					Check: addrs.Check{
						Name: "check_using_ephemeral_value",
					},
				}
				obj, ok := result.ObjectResults.GetOk(objKey)
				if !ok {
					t.Fatalf("expected to find object for %q", objKey)
				}
				expectedMessages := []string{"Fine to persist"}
				if diff := cmp.Diff(expectedMessages, obj.FailureMessages); diff != "" {
					t.Fatalf("unexpected messages: %s", diff)
				}
			},
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Check block assertion failed",
					Detail:   "Fine to persist",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 6, Column: 17, Byte: 104},
						End:      hcl.Pos{Line: 6, Column: 60, Byte: 147},
					},
				})
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Check block assertion failed",
					Detail:   "This check failed, but has an invalid error message as described in the other accompanying messages.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 10, Column: 17, Byte: 217},
						End:      hcl.Pos{Line: 10, Column: 60, Byte: 260},
					},
				})
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Error message refers to ephemeral values",
					Detail: "The error expression used to explain this condition refers to ephemeral values, so Terraform will not display the resulting message." +
						"\n\nYou can correct this by removing references to ephemeral values, or by using the ephemeralasnull() function on the references to not reveal ephemeral data.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 11, Column: 21, Byte: 281},
						End:      hcl.Pos{Line: 11, Column: 83, Byte: 343},
					},
				})
				return diags
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

		"locals": {
			module: map[string]string{
				"child/main.tf": `
ephemeral "ephem_resource" "data" {}

locals {
  composedString = "prefix-${ephemeral.ephem_resource.data.value}-suffix"
  composedList = ["a", ephemeral.ephem_resource.data.value, "c"]
  composedObj = {
    key = ephemeral.ephem_resource.data.value
    foo = "bar"
  }
}

# We expect this to error since it should be an ephemeral value
output "composedString" {
    value = local.composedString
}
output "composedList" {
    value = local.composedList
}
output "composedObj" {
    value = local.composedObj
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
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "child", "main.tf"),
						Start:    hcl.Pos{Line: 15, Column: 13, Byte: 376},
						End:      hcl.Pos{Line: 15, Column: 33, Byte: 396},
					},
				}, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral value not allowed",
					Detail:   "This output value is not declared as returning an ephemeral value, so it cannot be set to a result derived from an ephemeral value.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "child", "main.tf"),
						Start:    hcl.Pos{Line: 18, Column: 13, Byte: 435},
						End:      hcl.Pos{Line: 18, Column: 31, Byte: 453},
					},
				}, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral value not allowed",
					Detail:   "This output value is not declared as returning an ephemeral value, so it cannot be set to a result derived from an ephemeral value.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "child", "main.tf"),
						Start:    hcl.Pos{Line: 21, Column: 13, Byte: 491},
						End:      hcl.Pos{Line: 21, Column: 30, Byte: 508},
					},
				})
			},
		},
		"resource precondition": {
			module: map[string]string{
				"main.tf": `
locals {
  test_value = 2
}
ephemeral "ephem_resource" "data" {
  lifecycle {
    precondition {
      condition = local.test_value != 2
	  error_message = "value should not be 2"
    }
  }
}
`,
			},
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Resource precondition failed",
					Detail:   "value should not be 2",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 8, Column: 19, Byte: 116},
						End:      hcl.Pos{Line: 8, Column: 40, Byte: 137},
					},
				})
			},
		},
		"resource postcondition": {
			module: map[string]string{
				"main.tf": `
locals {
  test_value = 2
}
ephemeral "ephem_resource" "data" {
  lifecycle {
    postcondition {
      condition = self.value == "pass"
	  error_message = "value should be \"pass\""
    }
  }
}
`,
			},
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Resource postcondition failed",
					Detail:   `value should be "pass"`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 8, Column: 19, Byte: 117},
						End:      hcl.Pos{Line: 8, Column: 39, Byte: 137},
					},
				})
			},
		},

		"variable validation": {
			module: map[string]string{
				"main.tf": `
variable "ephem" {
  type        = string
  ephemeral   = true

  validation {
    condition     = length(var.ephem) > 4
    error_message = "This should fail but not show the value: ${var.ephem}"
  }
}

output "out" {
  value = ephemeralasnull(var.ephem)
}
`,
			},
			inputs: InputValues{
				"ephem": &InputValue{
					Value: cty.StringVal("ami"),
				},
			},
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid value for variable",
					Detail: fmt.Sprintf(`The error message included a sensitive value, so it will not be displayed.

This was checked by the validation rule at %s.`, m.Module.Variables["ephem"].Validations[0].DeclRange.String()),
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 17, Byte: 17},
					},
				}).Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Error message refers to ephemeral values",
					Detail: `The error expression used to explain this condition refers to ephemeral values. Terraform will not display the resulting message.

You can correct this by removing references to ephemeral values, or by carefully using the ephemeralasnull() function if the expression will not reveal the ephemeral data.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 8, Column: 21, Byte: 142},
						End:      hcl.Pos{Line: 8, Column: 76, Byte: 197},
					},
				})
			},
		},

		"write_only attribute": {
			module: map[string]string{
				"main.tf": `
ephemeral "ephem_resource" "data" {
}
resource "ephem_write_only" "test" {
    write_only = ephemeral.ephem_resource.data.value
}
`,
			},
			expectOpenEphemeralResourceCalled:           true,
			expectValidateEphemeralResourceConfigCalled: true,
			expectCloseEphemeralResourceCalled:          true,
		},
		"write_only_sensitive_and_ephem": {
			module: map[string]string{
				"main.tf": `
variable "in" {
  sensitive = true
  ephemeral = true
}
resource "ephem_write_only" "test" {
    write_only = var.in
}
`,
			},
			inputs: InputValues{
				"in": &InputValue{
					Value: cty.StringVal("test"),
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, tc.module)

			ephem := &testing_provider.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					EphemeralResourceTypes: map[string]providers.Schema{
						"ephem_resource": {
							Body: &configschema.Block{
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
					ResourceTypes: map[string]providers.Schema{
						"ephem_write_only": {
							Body: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"write_only": {
										Type:      cty.String,
										WriteOnly: true,
										Optional:  true,
									},
								},
							},
						},
					},
					Functions: map[string]providers.FunctionDecl{
						"either": {
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
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectValidateDiagnostics(m))
				// If we expect diagnostics, we should not continue with the plan
				// as it will fail.
				return
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			if tc.expectValidateEphemeralResourceConfigCalled {
				if !ephem.ValidateEphemeralResourceConfigCalled {
					t.Fatal("ValidateEphemeralResourceConfig not called")
				}
			}

			inputs := tc.inputs
			if inputs == nil {
				inputs = InputValues{}
			}

			plan, diags := ctx.Plan(m, nil, SimplePlanOpts(plans.NormalMode, inputs))
			if tc.expectPlanDiagnostics != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectPlanDiagnostics(m))
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
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

func TestContext2Apply_ephemeralUnknownPlan(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "test" {
}

ephemeral "ephem_resource" "data" {
  input = test_instance.test.id
  lifecycle {
    postcondition {
      condition = self.value != nil
      error_message = "should return a value"
    }
  }
}

locals {
  value = ephemeral.ephem_resource.data.value
}

// create a sink for the ephemeral value to test
provider "sink" {
  test_string = local.value
}

// we need a resource to ensure the sink provider is configured
resource "sink_object" "empty" {
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			EphemeralResourceTypes: map[string]providers.Schema{
				"ephem_resource": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Computed: true,
							},
							"input": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
	}

	sink := simpleMockProvider()
	sink.GetProviderSchemaResponse.ResourceTypes = map[string]providers.Schema{
		"sink_object": {Body: simpleTestSchema()},
	}
	sink.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		if req.Config.GetAttr("test_string").IsKnown() {
			t.Error("sink provider config should not be known in this test")
		}
		return resp
	}

	p := testProvider("test")

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
			addrs.NewDefaultProvider("sink"):  testProviderFuncFixed(sink),
		},
	})

	_, diags := ctx.Plan(m, nil, DefaultPlanOpts)
	tfdiags.AssertNoDiagnostics(t, diags)

	if ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled called when config was not known")
	}
}
