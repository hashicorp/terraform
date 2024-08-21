// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

// Benchmark that stresses resource instance evaluation during plan. The
// references to resources with large count values force the evaluator to
// repeated return all instances, so accessing those changes should be made as
// efficient as possible.
func BenchmarkPlanLargeCountRefs(b *testing.B) {
	m := testModuleInline(b, map[string]string{
		"main.tf": `
resource "test_resource" "a" {
  count = 512
  input = "ok"
}

resource "test_resource" "b" {
  count = length(test_resource.a)
  input = test_resource.a
}

module "mod" {
  count = length(test_resource.a)
  source = "./mod"
  in = [test_resource.a[count.index].id, test_resource.b[count.index].id]
}

output out {
  value = module.mod
}`,
		"./mod/main.tf": `
variable "in" {
}

output "out" {
  value = var.in
}
`})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":    {Type: cty.String, Computed: true},
					"input": {Type: cty.DynamicPseudoType, Optional: true},
				},
			},
		},
	})

	ctx := testContext2(b, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	b.ResetTimer()
	for range b.N {
		_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
		if diags.HasErrors() {
			b.Fatal(diags.Err())
		}
	}
}

// Similar to PlanLargeCountRefs, this runs through Apply to benchmark the
// caching of decoded state values.
func BenchmarkApplyLargeCountRefs(b *testing.B) {
	m := testModuleInline(b, map[string]string{
		"main.tf": `
resource "test_resource" "a" {
  count = 512
  input = "ok"
}

resource "test_resource" "b" {
  count = length(test_resource.a)
  input = test_resource.a
}

module "mod" {
  count = length(test_resource.a)
  source = "./mod"
  in = [test_resource.a[count.index].id, test_resource.b[count.index].id]
}

output out {
  value = module.mod
}`,
		"./mod/main.tf": `
variable "in" {
}

output "out" {
  value = var.in
}
`})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":    {Type: cty.String, Computed: true},
					"input": {Type: cty.DynamicPseudoType, Optional: true},
				},
			},
		},
	})

	ctx := testContext2(b, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		b.Fatal(diags.Err())
	}

	b.ResetTimer()
	for range b.N {
		ctx := testContext2(b, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		_, diags := ctx.Apply(plan, m, nil)
		if diags.HasErrors() {
			b.Fatal(diags.Err())
		}
	}
}
