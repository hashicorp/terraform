// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"path/filepath"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var _ configs.ModuleWalker = (*MockModuleWalker)(nil)

type MockModuleWalker struct {
	Calls         []*configs.ModuleRequest
	DefaultModule *configs.Module
	// the string key refers to ModuleSource.String()
	MockedCalls map[string]*configs.Module
}

func (m *MockModuleWalker) LoadModule(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
	m.Calls = append(m.Calls, req)

	if mod, ok := m.MockedCalls[req.SourceAddr.String()]; ok {
		return mod, nil, nil
	}

	return m.DefaultModule, nil, nil
}

func (m *MockModuleWalker) MockModuleCalls(t *testing.T, calls map[string]*configs.Module) {
	t.Helper()
	if m.MockedCalls == nil {
		m.MockedCalls = make(map[string]*configs.Module)
	}
	for k, v := range calls {
		// Make sure we can parse the module source
		ms := mustModuleSource(t, k)
		m.MockedCalls[ms.String()] = v
	}
}

func TestInit(t *testing.T) {
	for name, tc := range map[string]struct {
		module                map[string]string
		vars                  InputValues
		mockedLoadModuleCalls map[string]map[string]string
		// m -> root module
		// mc -> module calls
		expectDiags           func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics
		expectLoadModuleCalls []*configs.ModuleRequest
	}{
		"empty config": {
			module: map[string]string{"main.tf": ``},
		},
		"local - no variables": {
			module: map[string]string{
				"main.tf": `
module "example" {
  source = "./modules/example"
}
`,
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./modules/example"),
			}},
		},

		"remote - no variables": {
			module: map[string]string{
				"main.tf": `
module "example" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "6.6.0"
}

module "example2" {
  source  = "terraform-iaac/cert-manager/kubernetes"
}
				`,
			},

			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "terraform-iaac/cert-manager/kubernetes"),
			}, {
				SourceAddr:        mustModuleSource(t, "terraform-aws-modules/vpc/aws"),
				VersionConstraint: mustVersionContraint(t, "= 6.6.0"),
			}},
		},

		"local - with variables": {
			module: map[string]string{
				"main.tf": `
variable "name" {
  type = string
  const = true
}
module "example" {
    source = "./modules/${var.name}"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("example"), SourceType: ValueFromCLIArg},
			},

			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./modules/example"),
			}},
		},

		"local with non-const variables": {
			module: map[string]string{
				"main.tf": `
variable "name" {
  type = string
}
module "example" {
    source = "./modules/${var.name}"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("example"), SourceType: ValueFromCLIArg},
			},

			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				// TODO: We should try to somehow add an "extra" into the diagnostics to indicate
				// that this may be caused by a non-const variable used during init.
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid module source`,
					Detail:   `The value of a reference in the module source is unknown.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 6, Column: 27, Byte: 82},
						End:      hcl.Pos{Line: 6, Column: 35, Byte: 90},
					},
				})
			},
		},

		"remote - with variable in source": {
			module: map[string]string{
				"main.tf": `
variable "name" {
  type = string
  const = true
}
module "example2" {
  source  = "terraform-iaac/${var.name}/kubernetes"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("cert-manager"), SourceType: ValueFromCLIArg},
			},

			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "terraform-iaac/cert-manager/kubernetes"),
			}},
		},
		"remote - with variable in constraint": {
			module: map[string]string{
				"main.tf": `
variable "name" {
  type = string
  const = true
}
module "example2" {
  source  = "terraform-iaac/cert-manager/kubernetes"
  version = ">= ${var.name}"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("1.2.3"), SourceType: ValueFromCLIArg},
			},

			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr:        mustModuleSource(t, "terraform-iaac/cert-manager/kubernetes"),
				VersionConstraint: mustVersionContraint(t, ">= 1.2.3"),
			}},
		},

		"locals in module sources": {
			module: map[string]string{
				"main.tf": `
variable "name" {
  type = string
  const = true
}

locals {
  org_and_repo = "terraform-iaac/${var.name}"
}

module "example2" {
  source  = "${local.org_and_repo}/kubernetes"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("cert-manager"), SourceType: ValueFromCLIArg},
			},

			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr:        mustModuleSource(t, "terraform-iaac/cert-manager/kubernetes"),
				VersionConstraint: mustVersionContraint(t, ">= 1.2.3"),
			}},
		},

		"each in module sources": {
			module: map[string]string{
				"main.tf": `
module "example" {
  for_each = toset(["cert-manager", "helm"])
  source  = "terraform-iaac/${each.key}/kubernetes"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("cert-manager"), SourceType: ValueFromCLIArg},
			},

			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid module source`,
					Detail:   `The module source can only reference input variables and local values.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 4, Column: 31, Byte: 95},
						End:      hcl.Pos{Line: 4, Column: 39, Byte: 103},
					},
				})
			},
		},

		"module variables in source": {
			module: map[string]string{
				"main.tf": `
module "mod" {
  source = "./mod"
  name   = "cert-manager"
}
`,
			},
			vars: InputValues{
				"name": &InputValue{Value: cty.StringVal("cert-manager"), SourceType: ValueFromCLIArg},
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"./mod": {
					"main.tf": `
variable "name" {
  type = string
  const = true
}
module "example" {
  source  = "terraform-iaac/${var.name}/kubernetes"
}
					`,
				},
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./mod"),
			}, {
				SourceAddr: mustModuleSource(t, "terraform-iaac/cert-manager/kubernetes"),
			}},
		},

		"undefined variable in module source": {
			module: map[string]string{
				"main.tf": `
variable "name" {
  type = string
  const = true
}
module "example2" {
  source  = "terraform-iaac/${var.name}/kubernetes"
}
`,
			},
			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Required variable not set",
					Detail:   `The variable "name" is required, but is not set.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
					},
				})
			},
		},

		"resource reference in module source": {
			module: map[string]string{
				"main.tf": `
resource "null_resource" "example" {}

module "example" {
    source  = "terraform-iaac/${null_resource.example.id}/kubernetes"
}
`,
			},
			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid module source",
					Detail:   "The module source can only reference input variables and local values.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 5, Column: 33, Byte: 91},
						End:      hcl.Pos{Line: 5, Column: 54, Byte: 112},
					},
				})
			},
		},
		"resource reference in module call": {
			module: map[string]string{
				"main.tf": `
variable "name" {
    type = string
    default = "aws"
    const = true
}
resource "null_resource" "example" {}

module "example" {
    source  = "./${var.name}"

    name = var.name
    this_should_be_unknown_and_not_cause_error = null_resource.example.id
}
`,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"./aws": {
					"main.tf": `
variable "name" {
    type = string
    const = true
}

variable "this_should_be_unknown_and_not_cause_error" {
    type = string
}

module "example" {
    source = "terraform-iaac/${var.name}/kubernetes"
}


output "foo" {
    value = var.this_should_be_unknown_and_not_cause_error
}
			`,
				},
			},

			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "terraform-iaac/aws/kubernetes"),
			}, {
				SourceAddr: mustModuleSource(t, "./aws"),
			}},
		},

		"module output reference in module source": {
			module: map[string]string{
				"main.tf": `
module "example" {
    source = "./module/example"
}

module "example2" {
    source  = "terraform-iaac/${module.example.id}/kubernetes"
}
    `,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"./module/example": {
					"main.tf": `
output "id" {
  value = "example-id"
}
							`,
				}},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./module/example"),
			}},
			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid module source",
					Detail:   "The module source can only reference input variables and local values.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 7, Column: 33, Byte: 107},
						End:      hcl.Pos{Line: 7, Column: 50, Byte: 124},
					},
				})
			},
		},

		"nested module loading - no variables": {
			module: map[string]string{
				"main.tf": `
module "parent" {
  source = "hashicorp/parent/aws"
}
`,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"hashicorp/parent/aws": {
					"main.tf": `
module "child" {
  source = "hashicorp/child/aws"
}
					`,
				},
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "hashicorp/parent/aws"),
			}, {
				SourceAddr: mustModuleSource(t, "hashicorp/child/aws"),
			}},
		},

		"nested module loading - with variables": {
			module: map[string]string{
				"main.tf": `
module "parent" {
  source = "hashicorp/parent/aws"
  name = "child"
}
`,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"hashicorp/parent/aws": {
					"main.tf": `
variable "name" {
    type = string
    const = true
}
module "child" {
  source = "hashicorp/${var.name}/aws"
  name = "grand${var.name}"
}
					`,
				},
				"hashicorp/child/aws": {
					"main.tf": `
variable "name" {
    type = string
    const = true
}
module "grandchild" {
  source = "hashicorp/${var.name}/aws"
}
					`,
				},
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "hashicorp/parent/aws"),
			}, {
				SourceAddr: mustModuleSource(t, "hashicorp/child/aws"),
			}, {
				SourceAddr: mustModuleSource(t, "hashicorp/grandchild/aws"),
			}},
		},
		"module nested expansion": {
			module: map[string]string{
				"main.tf": `
module "fromdisk" {
  source    = "./mod"
  namespace = "terraform-iaac"
}
`,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"./mod": {
					"main.tf": `
locals {
  source = var.namespace
}
variable "namespace" {
  type      = string
  const = true
}
module "terraform" {
  source = "${var.namespace}/helm/kubernetes"
}
output "name" {
  value = "fooo"
}
`,
				},
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./mod"),
			}, {
				SourceAddr: mustModuleSource(t, "terraform-iaac/helm/kubernetes"),
			}},
		},

		"const variable with no value and no default": {
			module: map[string]string{"main.tf": `
variable "name" {
  type = string
  const = true
}
module "example" {
    source = "./modules/${var.name}"
}
`,
			},
			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Required variable not set`,
					Detail:   `The variable "name" is required, but is not set.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
					},
				})
			},
		},

		"const variable with default": {
			module: map[string]string{"main.tf": `
variable "name" {
  type = string
  const = true
  default = "example"
}
module "example" {
    source = "./modules/${var.name}"
}
`,
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./modules/example"),
			}},
		},

		"non-const variable passed into const module variable": {
			module: map[string]string{"main.tf": `
variable "name" {
  type = string
  default = "example"
}
module "example" {
  source = "./modules/example"
  name = "./modules/${var.name}2"
}
`,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"./modules/example": {
					"main.tf": `
variable "name" {
  type = string
  const = true
}
					`,
				},
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./modules/example"),
			}},
			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Constant variables must be known`,
					Detail:   `Only a constant value can be passed into a constant module variable.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 8, Column: 10, Byte: 118},
						End:      hcl.Pos{Line: 8, Column: 34, Byte: 142},
					},
				})
			},
		},

		"non-const module variable used as const": {
			module: map[string]string{"main.tf": `
module "example" {
  source = "./modules/example"

  name = "foo"
}
`,
			},
			mockedLoadModuleCalls: map[string]map[string]string{
				"./modules/example": {
					"main.tf": `
variable "name" {
  type = string
}

module "nested" {
    source = "./modules/${var.name}"
}
					`,
				},
			},
			expectLoadModuleCalls: []*configs.ModuleRequest{{
				SourceAddr: mustModuleSource(t, "./modules/example"),
			}},
			expectDiags: func(m *configs.Module, mc map[string]*configs.Module) tfdiags.Diagnostics {
				// TODO: We should try to somehow add an "extra" into the diagnostics to indicate
				// that this may be caused by a non-constant variable used during init.
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid module source`,
					Detail:   `The value of a reference in the module source is unknown.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(mc["./modules/example"].SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 7, Column: 27, Byte: 82},
						End:      hcl.Pos{Line: 7, Column: 35, Byte: 90},
					},
				})
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := testRootModuleInline(t, tc.module)

			ctx := testContext2(t, &ContextOpts{})
			moduleWalker := MockModuleWalker{
				DefaultModule: testRootModuleInline(t, map[string]string{"main.tf": `// empty`}),
			}
			mockedModules := make(map[string]*configs.Module)
			if tc.mockedLoadModuleCalls != nil {
				for k, v := range tc.mockedLoadModuleCalls {
					mockedModules[k] = testRootModuleInline(t, v)
				}
				moduleWalker.MockModuleCalls(t, mockedModules)
			}
			_, diags := ctx.Init(m, InitOpts{
				SetVariables: tc.vars,
				Walker:       &moduleWalker,
			})
			if tc.expectDiags != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectDiags(m, mockedModules))
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			if len(moduleWalker.Calls) != len(tc.expectLoadModuleCalls) {
				t.Fatalf("expected %d LoadModule calls, got %d", len(tc.expectLoadModuleCalls), len(moduleWalker.Calls))
			}

			// Create a map of expected sources for easier comparison
			expectedSources := make(map[string]bool)
			foundSources := []string{}
			for _, expected := range tc.expectLoadModuleCalls {
				expectedSources[expected.SourceAddr.String()] = false
			}

			// Mark sources as found
			for _, call := range moduleWalker.Calls {
				source := call.SourceAddr.String()
				foundSources = append(foundSources, source)
				if _, exists := expectedSources[source]; !exists {
					t.Errorf("unexpected LoadModule call for source %q", source)
				} else {
					expectedSources[source] = true
				}
			}

			// Check all expected sources were called
			for source, found := range expectedSources {
				if !found {
					t.Errorf("expected LoadModule call for source %q but it was not called. Calls that were made: \n %s", source, strings.Join(foundSources, ", "))
				}
			}
		})
	}
}

func mustModuleSource(t *testing.T, rawStr string) addrs.ModuleSource {
	src, err := moduleaddrs.ParseModuleSource(rawStr)
	if err != nil {
		t.Fatalf("failed to parse module source %q: %s", rawStr, err)
	}
	return src
}

func mustVersionContraint(t *testing.T, rawStr string) configs.VersionConstraint {
	constraints, err := version.NewConstraint(rawStr)
	if err != nil {
		t.Fatalf("failed to parse version constraint %q: %s", rawStr, err)
	}
	return configs.VersionConstraint{
		Required: constraints,
	}
}
