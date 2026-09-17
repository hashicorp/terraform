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
		requirements   map[string]*configs.RequiredProvider
		validationFunc func(t *testing.T, r []*addrs.Reference)
	}{
		"No references": {
			requirements: map[string]*configs.RequiredProvider{
				"testProvider": {},
			},
			validationFunc: func(t *testing.T, r []*addrs.Reference) {
				if len(r) != 0 {
					t.Fatalf("got %d references, want 0", len(r))
				}
			},
		},
		"Resolve references for version": {
			requirements: map[string]*configs.RequiredProvider{
				"testProvider": {
					RequirementExpr: testMockExprWith("var.some_version"),
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
			requirements: map[string]*configs.RequiredProvider{
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
			requirements: map[string]*configs.RequiredProvider{
				"testProvider_1": {
					RequirementExpr: testMockExprWith("var.version_1"),
					SourceExpr:      testMockExprWith("var.source_1"),
				},
				"testProvider_2": {
					RequirementExpr: testMockExprWith("var.version_2"),
					SourceExpr:      testMockExprWith("var.source_2"),
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
			n := nodeResolveProviderRequirements{
				Module: &configs.Module{
					ProviderRequirements: &configs.RequiredProviders{
						RequiredProviders: tc.requirements,
					},
				},
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
		addr        addrs.ModuleInstance
		wantVersion string
	}{
		"Resolve root required providers successfully": {
			addr:        addrs.RootModuleInstance,
			wantVersion: "0.0.7-james",
		},
		"Resolve children required providers successfully": {
			addr:        addrs.RootModuleInstance.Child("child", addrs.NoKey),
			wantVersion: "0.0.8-bill",
		},
	} {
		t.Run(name, func(t *testing.T) {
			n := nodeResolveProviderRequirements{
				Addr:   tc.addr,
				Module: testRequiredProvidersModule(t),
			}
			req, ok := n.Module.ProviderRequirements.RequiredProviders["testprovider"]
			if !ok {
				t.Fatal("provider testprovider not found")
			}

			evalCtx := &hcl.EvalContext{
				Variables: map[string]cty.Value{
					"var": cty.ObjectVal(map[string]cty.Value{
						"testprovider_src": cty.StringVal("hashicorp/testprovider"),
						"testprovider_ver": cty.StringVal(tc.wantVersion),
					}),
				},
			}
			ctx := &MockEvalContext{
				EvaluateExprResultFunc: func(expr hcl.Expression, _ cty.Type, _ addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
					value, diags := expr.Value(evalCtx)
					return value, tfdiags.Diagnostics{}.Append(diags)
				},
			}

			diags := n.Execute(ctx, walkInit)
			if diags.HasErrors() {
				t.Fatalf("got errors, expected none: %v", diags)
			}
			providers := n.Module.ProviderRequirements.RequiredProviders
			if len(providers) != 1 {
				t.Fatalf("got %d providers, expected 1", len(providers))
			}
			if providers["testprovider"] != req {
				t.Error("provider requirement was not updated in place")
			}
			if req.Source != "hashicorp/testprovider" {
				t.Errorf("got source %q, expected hashicorp/testprovider", req.Source)
			}
			wantType := addrs.NewDefaultProvider("testprovider")
			if !req.Type.Equals(wantType) {
				t.Errorf("got provider type %s, expected %s", req.Type, wantType)
			}
			if got := req.Requirement.Required.String(); got != tc.wantVersion {
				t.Errorf("got version %q, expected %q", got, tc.wantVersion)
			}
			if got := n.Module.ProviderLocalNames[wantType]; got != "testprovider" {
				t.Errorf("got provider local name %q, expected testprovider", got)
			}
		})
	}
}

func testRequiredProvidersModule(t *testing.T) *configs.Module {
	t.Helper()
	// Leave the expressions unresolved so Execute is responsible for evaluating them.
	return testRootModuleInline(t,
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
`}, false)
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
