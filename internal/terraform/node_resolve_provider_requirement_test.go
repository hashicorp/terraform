// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeResolveProviderRequirements_References(t *testing.T) {
	for name, tc := range map[string]struct {
		nodeExprs      map[string]*configs.ProviderRequirementExpr
		validationFunc func(t *testing.T, r []*addrs.Reference)
	}{
		"No references": {
			nodeExprs: map[string]*configs.ProviderRequirementExpr{
				"testProvider": {},
			},
			validationFunc: func(t *testing.T, r []*addrs.Reference) {
				if len(r) != 0 {
					t.Fatalf("got %d references, want 0", len(r))
				}
			},
		},
		"Resolve references for version": {
			nodeExprs: map[string]*configs.ProviderRequirementExpr{
				"testProvider": {
					VersionExpr: testMockExprWith("var.some_version"),
				},
			},
			validationFunc: func(t *testing.T, r []*addrs.Reference) {
				if len(r) != 1 {
					t.Fatalf("got %d references, expected 1", len(r))
				}
				if r[0].Subject.String() != "var.some_version" {
					t.Errorf(
						"got %s, expected var.some_version",
						r[0].Subject,
					)
				}
			},
		},
		"Resolve references for source": {
			nodeExprs: map[string]*configs.ProviderRequirementExpr{
				"testProvider": {
					SourceExpr: testMockExprWith("var.some_source"),
				},
			},
			validationFunc: func(t *testing.T, r []*addrs.Reference) {
				if len(r) != 1 {
					t.Fatalf("got %d references, expected 1", len(r))
				}
				if r[0].Subject.String() != "var.some_source" {
					t.Errorf(
						"got %s, expected var.some_source",
						r[0].Subject,
					)
				}
			},
		},
		"Resolve all references for multiple providers": {
			nodeExprs: map[string]*configs.ProviderRequirementExpr{
				"testProvider_1": {
					VersionExpr: testMockExprWith("var.version_1"),
					SourceExpr:  testMockExprWith("var.source_1"),
				},
				"testProvider_2": {
					VersionExpr: testMockExprWith("var.version_2"),
					SourceExpr:  testMockExprWith("var.source_2"),
				},
			},
			validationFunc: func(t *testing.T, r []*addrs.Reference) {
				if len(r) != 4 {
					t.Fatalf("got %d references, expected 4", len(r))
				}

				expected := []*addrs.Reference{
					mustReference("var.source_1"),
					mustReference("var.source_2"),
					mustReference("var.version_1"),
					mustReference("var.version_2"),
				}

				if eq := cmp.Equal(r, expected,
					cmpopts.SortSlices(func(a, b *addrs.Reference) bool {
						return a.Subject.String() < b.Subject.String()
					}),
					cmp.Comparer(func(a, b *addrs.Reference) bool {
						return a.Subject.String() == b.Subject.String()
					}),
				); !eq {
					t.Fatalf(
						"references not equal\n got: %v\nwant: %v",
						r,
						expected,
					)
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			mod := testModule(t, "empty")
			n := nodeResolveProviderRequirements{
				Module: mod.Module,
				Exprs:  tc.nodeExprs,
			}

			r := n.References()

			tc.validationFunc(t, r)
		})
	}
}

func testMockExprWith(variable string) mockHCLExpression {
	varChunks := strings.Split(variable, ".")
	if len(varChunks) != 2 {
		panic("variable has to consist of two parts separated by '.'" +
			"\nex: var.test_var")
	}
	return mockHCLExpression{
		variablesFunc: func() []hcl.Traversal {
			return []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: varChunks[0],
					},
					hcl.TraverseAttr{
						Name: varChunks[1],
					},
				},
			}
		},
	}
}

func TestNodeResolveProviderRequirements_Execute(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx            EvalContext
		module         *configs.Module
		nodeExprs      map[string]*configs.ProviderRequirementExpr
		validationFunc func(
			t *testing.T,
			n nodeResolveProviderRequirements,
			diags tfdiags.Diagnostics,
		)
	}{
		"Resolve root required providers successfully": {
			ctx:       &MockEvalContext{},
			module:    testRequiredProvidersConfig(t).Module,
			nodeExprs: map[string]*configs.ProviderRequirementExpr{},
			validationFunc: func(
				t *testing.T,
				n nodeResolveProviderRequirements,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatalf("got errors, expected none: %v", diags)
				}
				providers := n.Module.ProviderRequirements.RequiredProviders
				if len(providers) != 1 {
					t.Fatalf("got %d providers, expected 1", len(providers))
				}

				rp, ok := providers["testprovider"]
				if !ok {
					t.Fatalf("provider testprovider not found")
				}
				if rp.Requirement.Required.String() != "0.0.7-james" {
					t.Errorf("got %s, expected 0.0.7-james", rp.Requirement.Required)
				}
			},
		},
		"Resolve children required providers successfully": {
			ctx:       &MockEvalContext{},
			module:    testRequiredProvidersConfig(t).Children["child"].Module,
			nodeExprs: map[string]*configs.ProviderRequirementExpr{},
			validationFunc: func(
				t *testing.T,
				n nodeResolveProviderRequirements,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatalf("got errors, expected none: %v", diags)
				}
				providers := n.Module.ProviderRequirements.RequiredProviders
				if len(providers) != 1 {
					t.Fatalf("got %d providers, expected 1", len(providers))
				}

				rp, ok := providers["testprovider"]
				if !ok {
					t.Fatalf("provider testprovider not found")
				}
				if rp.Requirement.Required.String() != "0.0.8-bill" {
					t.Errorf("got %s, expected 0.0.8-bill", rp.Requirement.Required)
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			n := nodeResolveProviderRequirements{
				Module: tc.module,
				Exprs:  tc.nodeExprs,
			}

			diags := n.Execute(tc.ctx, walkInit)

			tc.validationFunc(t, n, diags)
		})
	}
}

func testRequiredProvidersConfig(t *testing.T) *configs.Config {
	return testModuleInlineWithVars(t,
		map[string]string{
			"main.tf": `
terraform {
	required_providers {
		testprovider = {
			source  = "${var.testprovider_src}"
			version = "${var.testprovider_ver}"
		}
	}
}

variable "testprovider_src" {
	type = string
	const = true
}

variable "testprovider_ver" {
	type = string
	const = true
}

module "child" {
	source = "./local_module"
	testprovider_src = var.testprovider_src
	testprovider_ver = "0.0.8-bill"
}
`,
			"local_module/main.tf": `
terraform {
	required_providers {
		testprovider = {
			source  = "${var.testprovider_src}"
			version = "${var.testprovider_ver}"
		}
	}
}

variable "testprovider_src" {
	type = string
	const = true
}

variable "testprovider_ver" {
	type = string
	const = true
}
`},
		map[string]*InputValue{
			"testprovider_src": {
				Value: cty.StringVal("hashicorp/testprovider"),
			},
			"testprovider_ver": {
				Value: cty.StringVal("0.0.7-james"),
			},
		})
}

type mockHCLExpression struct {
	rangeFunc      func() hcl.Range
	startRangeFunc func() hcl.Range
	variablesFunc  func() []hcl.Traversal
	valuesFunc     func(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics)
}

func (e mockHCLExpression) Range() hcl.Range {
	return e.rangeFunc()
}

func (e mockHCLExpression) StartRange() hcl.Range {
	return e.startRangeFunc()
}

func (e mockHCLExpression) Variables() []hcl.Traversal {
	return e.variablesFunc()
}

func (e mockHCLExpression) Value(
	ctx *hcl.EvalContext,
) (cty.Value, hcl.Diagnostics) {
	return e.valuesFunc(ctx)
}
