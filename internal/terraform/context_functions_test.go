// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
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

	p := new(MockProvider)
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

	// there is exactly one output, which is a dynamically typed string
	if !bytes.Equal([]byte("\x92\xc4\b\"string\"\xa2ok"), plan.Changes.Outputs[0].After) {
		t.Fatalf("got output dynamic value of %q", plan.Changes.Outputs[0].After)
	}
}
