// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Ported from OpenTofu internal/tofu/bench_many_instances_test.go at
// 0fb8b8f8a9da58dfab95e992dd17b92f30361c7a for local performance comparison.
// The configurations, instance counts, provider behavior, parallelism, and
// timed regions are intentionally unchanged.

package terraform

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func benchmarkAssertNoDiagnostics(b testing.TB, diags tfdiags.Diagnostics) {
	b.Helper()
	if len(diags) == 0 {
		return
	}
	for _, diag := range diags {
		desc := diag.Description()
		b.Logf("%s: %s", desc.Summary, desc.Detail)
	}
	b.FailNow()
}

func BenchmarkManyResourceInstances(b *testing.B) {
	const instanceCount = 2500
	m := testModuleInline(b, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "terraform.io/builtin/test"
					}
				}
			}

			variable "instance_count" {
				type = number
			}

			resource "test" "a" {
				count = var.instance_count

				num = count.index
			}

			resource "test" "b" {
				count = length(test.a)

				num = test.a[count.index].num
			}
		`,
	})
	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"num": {
								Type:     cty.Number,
								Required: true,
							},
						},
					},
				},
			},
		},
		PlanResourceChangeFn: func(prcr providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: prcr.ProposedNewState,
			}
		},
		ApplyResourceChangeFn: func(arcr providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			return providers.ApplyResourceChangeResponse{
				NewState: arcr.PlannedState,
			}
		},
	}
	tfCtx := testContext2(b, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("test"): testProviderFuncFixed(p),
		},
		Parallelism: instanceCount * 3,
	})
	priorStateBase := states.BuildState(func(ss *states.SyncState) {
		providerAddr := addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewBuiltInProvider("test"),
		}
		resourceAddrA := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: "a",
		}.Absolute(addrs.RootModuleInstance)
		resourceAddrB := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: "b",
		}.Absolute(addrs.RootModuleInstance)
		for i := range instanceCount {
			instAddrA := resourceAddrA.Instance(addrs.IntKey(i))
			instAddrB := resourceAddrB.Instance(addrs.IntKey(i))
			rawStateAttrs := `{"num":` + strconv.Itoa(i) + `}`
			ss.SetResourceInstanceCurrent(
				instAddrA,
				&states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(rawStateAttrs),
				},
				providerAddr,
			)
			ss.SetResourceInstanceCurrent(
				instAddrB,
				&states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(rawStateAttrs),
				},
				providerAddr,
			)
		}
	})
	planOpts := &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"instance_count": {
				Value: cty.NumberIntVal(instanceCount),
			},
		},
	}
	b.ResetTimer()

	for range b.N {
		priorState := priorStateBase.DeepCopy()

		plan, planDiags := tfCtx.Plan(m, priorState, planOpts)
		benchmarkAssertNoDiagnostics(b, planDiags)

		_, applyDiags := tfCtx.Apply(plan, m, nil)
		benchmarkAssertNoDiagnostics(b, applyDiags)
	}
}

func BenchmarkManyModuleInstances(b *testing.B) {
	const instanceCount = 2500
	m := testModuleInline(b, map[string]string{
		"main.tf": `
			variable "instance_count" {
				type = number
			}

			module "a" {
				source = "./child"
				count  = var.instance_count

				num = count.index
			}

			module "b" {
				source = "./child"
				count  = length(module.a)

				num = module.a[count.index].num
			}
		`,
		"child/child.tf": `
			variable "num" {
				type = number
			}

			output "num" {
				value = var.num
			}
		`,
	})
	tfCtx := testContext2(b, &ContextOpts{
		Providers:   nil,
		Parallelism: instanceCount * 2 * 8,
	})
	planOpts := &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"instance_count": {
				Value: cty.NumberIntVal(instanceCount),
			},
		},
	}
	b.ResetTimer()

	for range b.N {
		plan, planDiags := tfCtx.Plan(m, states.NewState(), planOpts)
		benchmarkAssertNoDiagnostics(b, planDiags)

		_, applyDiags := tfCtx.Apply(plan, m, nil)
		benchmarkAssertNoDiagnostics(b, applyDiags)
	}
}

// The benchmarks below are not part of the OpenTofu port above. Each one
// reproduces a specific evaluation bottleneck reported against Terraform, and
// leaves Parallelism at its default to match a normal CLI run.

// BenchmarkManyEvaluatedBlocks measures the fixed cost of evaluating many
// small, independent blocks. Every expression evaluation creates a new
// lang.Scope, and each scope builds its own table of built-in functions, so
// the cost scales with the number of evaluated blocks rather than with the
// size of the plan.
//
// https://github.com/hashicorp/terraform/issues/39445
func BenchmarkManyEvaluatedBlocks(b *testing.B) {
	const blockCount = 2500

	var cfg strings.Builder
	cfg.WriteString("locals {\n")
	for i := range blockCount {
		fmt.Fprintf(&cfg, "  l%d = %d\n", i, i)
	}
	cfg.WriteString("}\n")
	for i := range blockCount {
		fmt.Fprintf(&cfg, "output \"o%d\" {\n  value = local.l%d\n}\n", i, i)
	}

	m := testModuleInline(b, map[string]string{
		"main.tf": cfg.String(),
	})
	tfCtx := testContext2(b, &ContextOpts{})
	b.ResetTimer()

	for range b.N {
		plan, planDiags := tfCtx.Plan(m, states.NewState(), DefaultPlanOpts)
		benchmarkAssertNoDiagnostics(b, planDiags)

		_, applyDiags := tfCtx.Apply(plan, m, nil)
		benchmarkAssertNoDiagnostics(b, applyDiags)
	}
}

// BenchmarkManyResourceInstancesInlineForEach measures a no-op plan and apply
// of a resource whose for_each is an inline expression that is itself
// expensive to evaluate. The expression is evaluated once to expand the
// resource, but then again by every instance to find its own each.key and
// each.value, so total cost grows with the instance count multiplied by the
// cost of the expression.
//
// The configuration is the reproduction case from the issue.
//
// https://github.com/hashicorp/terraform/issues/39456
func BenchmarkManyResourceInstancesInlineForEach(b *testing.B) {
	const instanceCount = 250
	m := testModuleInline(b, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "terraform.io/builtin/test"
					}
				}
			}

			variable "instance_count" {
				type = number
			}

			locals {
				users = { for i in range(var.instance_count) : "u${i}" => i }
				pairs = { for i in range(var.instance_count) : "p${i}" => { member = "u${i}" } }
			}

			resource "test" "a" {
				for_each = { for k, v in local.pairs : k => v if contains(keys(local.users), v.member) }

				num = local.users[each.value.member]
			}
		`,
	})
	tfCtx := testContext2(b, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("test"): testProviderFuncFixed(benchmarkNumProvider()),
		},
	})
	priorStateBase := benchmarkNumState("a", instanceCount, func(i int) addrs.InstanceKey {
		return addrs.StringKey("p" + strconv.Itoa(i))
	})
	planOpts := &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"instance_count": {
				Value: cty.NumberIntVal(instanceCount),
			},
		},
	}
	b.ResetTimer()

	for range b.N {
		priorState := priorStateBase.DeepCopy()

		plan, planDiags := tfCtx.Plan(m, priorState, planOpts)
		benchmarkAssertNoDiagnostics(b, planDiags)

		_, applyDiags := tfCtx.Apply(plan, m, nil)
		benchmarkAssertNoDiagnostics(b, applyDiags)
	}
}

// BenchmarkManyResourceInstancesSelfPostcondition measures a no-op plan and
// apply of a counted resource with a postcondition referring to self. Each
// instance only needs its own object, but self is resolved by first
// evaluating the containing resource, which decodes every sibling instance.
//
// https://github.com/hashicorp/terraform/issues/39343
func BenchmarkManyResourceInstancesSelfPostcondition(b *testing.B) {
	const instanceCount = 500
	m := testModuleInline(b, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "terraform.io/builtin/test"
					}
				}
			}

			variable "instance_count" {
				type = number
			}

			resource "test" "a" {
				count = var.instance_count

				num = count.index

				lifecycle {
					postcondition {
						condition     = self.num == count.index
						error_message = "Unexpected num."
					}
				}
			}
		`,
	})
	tfCtx := testContext2(b, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("test"): testProviderFuncFixed(benchmarkNumProvider()),
		},
	})
	priorStateBase := benchmarkNumState("a", instanceCount, func(i int) addrs.InstanceKey {
		return addrs.IntKey(i)
	})
	planOpts := &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"instance_count": {
				Value: cty.NumberIntVal(instanceCount),
			},
		},
	}
	b.ResetTimer()

	for range b.N {
		priorState := priorStateBase.DeepCopy()

		plan, planDiags := tfCtx.Plan(m, priorState, planOpts)
		benchmarkAssertNoDiagnostics(b, planDiags)

		_, applyDiags := tfCtx.Apply(plan, m, nil)
		benchmarkAssertNoDiagnostics(b, applyDiags)
	}
}

// benchmarkNumProvider returns a mock provider with a single "test" resource
// type, which has one required "num" attribute and accepts any change as-is.
func benchmarkNumProvider() *testing_provider.MockProvider {
	return &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"num": {
								Type:     cty.Number,
								Required: true,
							},
						},
					},
				},
			},
		},
		PlanResourceChangeFn: func(prcr providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: prcr.ProposedNewState,
			}
		},
		ApplyResourceChangeFn: func(arcr providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			return providers.ApplyResourceChangeResponse{
				NewState: arcr.PlannedState,
			}
		},
	}
}

// benchmarkNumState returns a state containing count instances of the root
// module resource test.<name>, as created by benchmarkNumProvider. Instance i
// has the key returned by key(i) and a num attribute of i.
func benchmarkNumState(name string, count int, key func(i int) addrs.InstanceKey) *states.State {
	return states.BuildState(func(ss *states.SyncState) {
		providerAddr := addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewBuiltInProvider("test"),
		}
		resourceAddr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: name,
		}.Absolute(addrs.RootModuleInstance)
		for i := range count {
			ss.SetResourceInstanceCurrent(
				resourceAddr.Instance(key(i)),
				&states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"num":` + strconv.Itoa(i) + `}`),
				},
				providerAddr,
			)
		}
	})
}
