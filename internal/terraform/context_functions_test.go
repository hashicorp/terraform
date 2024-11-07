// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
)

func TestContext2Plan_providerFunctionBasic(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
	}
  }
}

locals {
  input = {
    key = "value"
  }

  expected = {
    key = "value"
  }
}

output "noop_equals" {
  // The false branch will fail to evaluate entirely if our condition doesn't
  // hold true. This is not a normal way to check a condition, but it's been
  // seen in the wild, so adding it here for variety.
  value = provider::test::noop(local.input) == local.expected ? "ok" : {}["fail"]
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Functions: map[string]providers.FunctionDecl{
			"noop": providers.FunctionDecl{
				Parameters: []providers.FunctionParam{
					{
						Name: "any",
						Type: cty.DynamicPseudoType,
					},
				},
				ReturnType: cty.DynamicPseudoType,
			},
		},
	}
	p.CallFunctionFn = func(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
		resp.Result = req.Arguments[0]
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	expect, err := msgpack.Marshal(cty.StringVal("ok"), cty.DynamicPseudoType)
	if err != nil {
		t.Fatal(err)
	}

	// there is exactly one output, which is a dynamically typed string
	if !bytes.Equal(expect, plan.Changes.Outputs[0].After) {
		t.Fatalf("got output dynamic value of %q", plan.Changes.Outputs[0].After)
	}
}

// check that provider functions called multiple times during validate and plan
// return consistent results
func TestContext2Plan_providerFunctionImpurePlan(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
	}
  }
}

output "first" {
  value = provider::test::echo("input")
}

output "second" {
  value = provider::test::echo("input")
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Functions: map[string]providers.FunctionDecl{
			"echo": providers.FunctionDecl{
				Parameters: []providers.FunctionParam{
					{
						Name: "arg",
						Type: cty.String,
					},
				},
				ReturnType: cty.String,
			},
		},
	}

	inc := 0
	p.CallFunctionFn = func(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
		// this broken echo adds a counter to the argument
		resp.Result = cty.StringVal(fmt.Sprintf("%s-%d", req.Arguments[0].AsString(), inc))
		inc++
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}

	errs := diags.Err().Error()
	if !strings.Contains(errs, "provider function returned an inconsistent result") {
		t.Fatalf("expected error with %q, got %q", "provider function returned an inconsistent result", errs)
	}
	_, diags = ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}

	errs = diags.Err().Error()
	if !strings.Contains(errs, "provider function returned an inconsistent result") {
		t.Fatalf("expected error with %q, got %q", "provider function returned an inconsistent result", errs)
	}
}

// check that we catch provider functions which return inconsistent results
// during apply
func TestContext2Plan_providerFunctionImpureApply(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "provider-function-echo")

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: simpleTestSchema()},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{Block: simpleTestSchema()},
			},
			DataSources: map[string]providers.Schema{
				"test_object": providers.Schema{Block: simpleTestSchema()},
			},
			Functions: map[string]providers.FunctionDecl{
				"echo": providers.FunctionDecl{
					Parameters: []providers.FunctionParam{
						{
							Name: "arg",
							Type: cty.String,
						},
					},
					ReturnType: cty.String,
				},
			},
		},
	}

	inc := 0
	p.CallFunctionFn = func(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
		// this broken echo adds a counter to the argument
		resp.Result = cty.StringVal(fmt.Sprintf("%s-%d", req.Arguments[0].AsString(), inc))
		inc++
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	// Write / Read plan to simulate running it through a Plan file
	ctxOpts, m, plan, err := contextOptsForPlanViaFile(t, snap, plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}

	ctxOpts.Providers = map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}
	ctx = testContext2(t, ctxOpts)

	_, diags = ctx.Apply(plan, m, nil)
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}

	errs := diags.Err().Error()
	if !strings.Contains(errs, "provider function returned an inconsistent result") {
		t.Fatalf("expected error with %q, got %q", "provider function returned an inconsistent result", errs)
	}
}

func TestContext2Validate_providerFunctionDiagnostics(t *testing.T) {
	provider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: simpleTestSchema()},
			Functions: map[string]providers.FunctionDecl{
				"echo": providers.FunctionDecl{
					Parameters: []providers.FunctionParam{
						{
							Name: "arg",
							Type: cty.String,
						},
					},
					ReturnType: cty.String,
				},
			},
		},
	}

	tests := []struct {
		name         string
		cfg          *configs.Config
		expectedDiag string
	}{
		{
			"missing provider",
			testModuleInline(t, map[string]string{
				"main.tf": `
			output "first" {
				value = provider::test::echo("input")
			}`}),
			`Ensure that provider name "test" is declared in this module's required_providers block, and that this provider offers a function named "echo"`,
		},
		{
			"invalid namespace",
			testModuleInline(t, map[string]string{
				"main.tf": `
			output "first" {
				value = test::echo("input")
			}`}),
			`The function namespace "test" is not valid. Provider function calls must use the "provider::" namespace prefix`,
		},
		{
			"missing namespace",
			testModuleInline(t, map[string]string{
				"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "registry.terraform.io/hashicorp/test"
					}
				}
			}
			output "first" {
				value = echo("input")
			}`}),
			`There is no function named "echo". Did you mean "provider::test::echo"?`,
		},
		{
			"no function from provider",
			testModuleInline(t, map[string]string{
				"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "registry.terraform.io/hashicorp/test"
					}
				}
			}
			output "first" {
				value = provider::test::missing("input")
			}`}),
			`Unknown provider function: The function "missing" is not available from the provider "test".`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
				},
			})

			diags := ctx.Validate(test.cfg, nil)
			if !diags.HasErrors() {
				t.Fatal("expected diagnsotics, got none")
			}
			got := diags.Err().Error()
			if !strings.Contains(got, test.expectedDiag) {
				t.Fatalf("expected %q, got %q", test.expectedDiag, got)
			}
		})
	}
}
