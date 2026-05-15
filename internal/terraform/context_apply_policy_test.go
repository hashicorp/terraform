// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestContext2Apply_PolicyEvaluation_Full(t *testing.T) {
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		variable "input" {
			type = string
			default = "foo"
		}

		resource "test_resource" "test" {
			sensitive_value = "foo"
		}

		module "child" {
			source = "./child"
			// this is a computed value in the parent, so will not be available until apply.
			input = test_resource.test.id
		}

		`
	childConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		variable "input" {
			type = string
		}

		resource "test_instance" "child" {
			value = var.input
		}

		`
	policyConfig := `
		resource_policy "test_resource" "policy_name" {
					enforce {
							condition = attrs.sensitive_value == "foo"
			}
		}
		`
	configFiles := map[string]string{
		"main.tf":           mainConfig,
		"child/child.tf":    childConfig,
		"main.tfpolicy.hcl": policyConfig,
	}

	mod := testModuleInline(t, configFiles)
	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")

	provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		cfg := req.Config.AsValueMap()
		if req.TypeName == "test_resource" {
			cfg["id"] = cty.StringVal("parent")
		}
		resp.NewState = cty.ObjectVal(cfg)
		return resp
	}
	state := states.NewState()

	// mock the policy expectations during plan
	planPolicyClient := policy.NewTestMockClient(t)

	// The expected values to be sent for policy evaluation.
	expectedPlan := map[string]cty.Value{
		"test_resource": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("foo"),
		}),
		"test_instance": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.UnknownVal(cty.String),
			"sensitive_value": cty.NilVal,
		}),
	}
	actualPlan := make(map[string]cty.Value)

	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		var actualVal cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.IsNull() {
			mp := attrs.AsValueMap()
			actualVal = cty.ObjectVal(map[string]cty.Value{
				"value":           mp["value"],
				"sensitive_value": mp["sensitive_value"],
			})
		}
		actualPlan[target] = actualVal
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}
	t.Cleanup(func() {
		if diff := cmp.Diff(actualPlan, expectedPlan, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("Unexpected diff (-got +want):\n%s", diff)
		}
	})

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 1,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, state, &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: planPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	// mock the policy expectations during apply
	applyPolicyClient := policy.NewTestMockClient(t)

	// The expected values to be sent for policy evaluation.
	expectedApply := map[string]cty.Value{
		"test_resource": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("foo"),
		}),
		"test_instance": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.StringVal("parent"), // was unknown in the plan
			"sensitive_value": cty.NilVal,
		}),
	}
	actualApply := make(map[string]cty.Value)

	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.IsNull() {
			mp := attrs.AsValueMap()
			actual = cty.ObjectVal(map[string]cty.Value{
				"value":           mp["value"],
				"sensitive_value": mp["sensitive_value"],
			})
		}
		actualApply[target] = actual

		// this return does not actually do anything
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	t.Cleanup(func() {
		if diff := cmp.Diff(actualApply, expectedApply, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("Unexpected diff (-got +want):\n%s", diff)
		}
	})

	_, diags = ctx.Apply(plan, mod, &ApplyOpts{
		PolicyClient: applyPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

}

// TestContext2Apply_PolicyEvaluationError tests that the apply operation returns policy diagnostics
// when the policy evaluation returns an error.
func TestContext2Apply_PolicyEvaluationError(t *testing.T) {
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		variable "input" {
			type = string
			default = "foo"
		}

		resource "test_resource" "test" {
			sensitive_value = "foo"
		}

		module "child" {
			source = "./child"
			// this is a computed value in the parent, so will not be available until apply.
			input = test_resource.test.id
		}

		`
	childConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		variable "input" {
			type = string
		}

		resource "test_instance" "child" {
			value = var.input
		}

		`
	policyConfig := `
		resource_policy "test_resource" "policy_name" {
					enforce {
							condition = attrs.sensitive_value == "foo"
			}
		}
		`
	configFiles := map[string]string{
		"main.tf":           mainConfig,
		"child/child.tf":    childConfig,
		"main.tfpolicy.hcl": policyConfig,
	}

	mod := testModuleInline(t, configFiles)
	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")

	provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		cfg := req.Config.AsValueMap()
		if req.TypeName == "test_resource" {
			cfg["id"] = cty.StringVal("parent")
		}
		resp.NewState = cty.ObjectVal(cfg)
		return resp
	}
	state := states.NewState()

	// mock the policy expectations during plan
	planPolicyClient := policy.NewTestMockClient(t)

	// The expected values to be sent for policy evaluation.
	expected := map[string]cty.Value{
		"test_resource": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("foo"),
		}),
		"test_instance": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.UnknownVal(cty.String),
			"sensitive_value": cty.NilVal,
		}),
	}

	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.IsNull() {
			mp := attrs.AsValueMap()
			actual = cty.ObjectVal(map[string]cty.Value{
				"value":           mp["value"],
				"sensitive_value": mp["sensitive_value"],
			})
		}

		if diff := cmp.Diff(actual, expected[target], cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("Unexpected diff (-got +want):\n%s", diff)
		}

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 1,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, state, &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: planPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	// mock the policy expectations during apply
	applyPolicyClient := policy.NewTestMockClient(t)

	// The expected values to be sent for policy evaluation.
	expected = map[string]cty.Value{
		"test_resource": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("foo"),
		}),
		"test_instance": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.StringVal("parent"), // was unknown in the plan
			"sensitive_value": cty.NilVal,
		}),
	}

	// Track which resource we're evaluating for different responses
	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.IsNull() {
			mp := attrs.AsValueMap()
			actual = cty.ObjectVal(map[string]cty.Value{
				"value":           mp["value"],
				"sensitive_value": mp["sensitive_value"],
			})
		}

		if diff := cmp.Diff(actual, expected[target], cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Errorf("Unexpected diff (-got +want):\n%s", diff)
		}

		if target == "test_resource" {
			return policy.EvaluationResponse{
				Overall: policy.DenyResult,
				Diagnostics: policy.DiagsFromProto([]*proto.Diagnostic{
					{
						Severity: proto.Severity_ERROR,
						Summary:  "error message",
					},
				}, nil),
			}
		}

		// test_instance should still be evaluated despite the error in test_resource
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	applyResults := plans.NewPolicyResults()
	state, diags = ctx.Apply(plan, mod, &ApplyOpts{
		PolicyClient:  applyPolicyClient,
		PolicyResults: applyResults,
	})
	tfdiags.AssertDiagnosticCount(t, diags, 0)

	var policyDiags tfdiags.Diagnostics
	for _, res := range applyResults.Iter() {
		policyDiags = policyDiags.Append(res.EvaluationResponse.Diagnostics.AsTerraformDiags())
	}
	var exp tfdiags.Diagnostics
	exp = exp.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "error message",
		Subject:  policyDiags[0].Source().Subject.ToHCL().Ptr(),
		Extra:    policyDiags[0].ExtraInfo(),
	})
	tfdiags.AssertDiagnosticsMatch(t, policyDiags, exp)

	addrs := state.AllManagedResourceInstanceObjectAddrs()
	if len(addrs) != 2 {
		t.Fatalf("expected 1 managed resource in the state, got %d", len(addrs))
	}

	rs := state.Resource(mustAbsResourceAddr("test_resource.test"))
	if rs == nil {
		t.Fatal("expected resource to be in the state")
	}
}

func TestContext2Apply_PolicyEvaluation_NoResourceAfterPolicy(t *testing.T) {
	// This verifies that no resource instance node is run after policy evaluation
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_instance" "test" {
			count = 2
			value = tostring(count.index)
		}
	`

	policyConfig := `
		resource_policy "test_instance" "policy_name" {
			enforce {
				condition = true
			}
		}
	`

	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": policyConfig,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")

	var policyRan atomic.Bool
	var applyCalls atomic.Int32

	provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		callNum := applyCalls.Add(1)
		if callNum == 2 {
			time.Sleep(150 * time.Millisecond)
		}

		if policyRan.Load() {
			t.Fatalf("resource apply for %s ran after policy evaluation", req.TypeName)
		}

		newState := req.PlannedState.AsValueMap()
		newState["id"] = cty.StringVal(req.PlannedState.GetAttr("value").AsString())
		newState["type"] = cty.StringVal(req.TypeName)
		newState["unknown"] = cty.StringVal("known")
		resp.NewState = cty.ObjectVal(newState)
		return resp
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 4,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	applyPolicyClient := policy.NewTestMockClient(t)
	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		policyRan.Store(true)

		if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
			Type:         "test_instance",
			ProviderType: "test",
			Operation:    proto.Operation_CREATE,
		}, protocmp.Transform()); diff != "" {
			t.Errorf("Invalid resource metadata: %s", diff)
		}

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	resultState, diags := ctx.Apply(plan, mod, &ApplyOpts{
		PolicyClient: applyPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if !applyPolicyClient.EvaluateCalled {
		t.Fatal("expected policy evaluation to be called during apply")
	}

	remainingAddrs := resultState.AllManagedResourceInstanceObjectAddrs()
	if len(remainingAddrs) != 2 {
		t.Fatalf("expected 2 managed resources in the state after apply, got %d: %v", len(remainingAddrs), remainingAddrs)
	}
}

func TestContext2Apply_PolicyEvaluation_ChangedResourceCount(t *testing.T) {
	cases := []struct {
		name            string
		state           *states.State
		configBody      string
		expectTarget    string
		expectOp        proto.Operation
		expectCalls     int
		expectFinalAttr cty.Value
	}{
		{
			name:  "create",
			state: states.NewState(),
			configBody: `
resource "test_resource" "test" {
  sensitive_value = "foo"
}
`,
			expectTarget: "test_resource",
			expectOp:     proto.Operation_CREATE,
			expectCalls:  1,
			expectFinalAttr: cty.ObjectVal(map[string]cty.Value{
				"id":              cty.StringVal("created"),
				"sensitive_value": cty.StringVal("foo"),
			}),
		},
		{
			name: "update",
			state: states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.test"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"existing","type":"test_resource","sensitive_value":"before"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			configBody: `
resource "test_resource" "test" {
  sensitive_value = "after"
}
`,
			expectTarget: "test_resource",
			expectOp:     proto.Operation_UPDATE,
			expectCalls:  1,
			expectFinalAttr: cty.ObjectVal(map[string]cty.Value{
				"id":              cty.StringVal("existing"),
				"sensitive_value": cty.StringVal("after"),
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}
` + tc.configBody

			policyConfig := `
		resource_policy "test_resource" "policy_name" {
			enforce {
				condition = true
			}
		}
`
			mod := testModuleInline(t, map[string]string{
				"main.tf":           mainConfig,
				"main.tfpolicy.hcl": policyConfig,
			})

			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
				cfg := req.Config.AsValueMap()
				if req.TypeName == "test_resource" {
					if id, ok := cfg["id"]; ok && !id.IsNull() && id.IsKnown() {
						cfg["id"] = id
					} else if tc.name == "create" {
						cfg["id"] = cty.StringVal("created")
					} else {
						cfg["id"] = cty.StringVal("existing")
					}
				}
				resp.NewState = cty.ObjectVal(cfg)
				return resp
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
				Parallelism: 1,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			planPolicyClient := policy.NewTestMockClient(t)
			plan, diags := ctx.Plan(mod, tc.state, &PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: testInputValuesUnset(mod.Module.Variables),
				PolicyClient: planPolicyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			applyPolicyClient := policy.NewTestMockClient(t)
			var called int
			applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
				called++
				if req.Target != tc.expectTarget {
					t.Fatalf("expected target %s, got %s", tc.expectTarget, req.Target)
				}
				if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
					Type:         tc.expectTarget,
					ProviderType: "test",
					Operation:    tc.expectOp,
				}, protocmp.Transform()); diff != "" {
					t.Fatalf("unexpected resource metadata (-got +want):\n%s", diff)
				}

				actualAttrs := req.Attrs
				if !actualAttrs.IsNull() {
					mp := actualAttrs.AsValueMap()
					actualAttrs = cty.ObjectVal(map[string]cty.Value{
						"id":              mp["id"],
						"sensitive_value": mp["sensitive_value"],
					})
				}
				if diff := cmp.Diff(actualAttrs, tc.expectFinalAttr, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
					t.Fatalf("unexpected attrs (-got +want):\n%s", diff)
				}
				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			_, diags = ctx.Apply(plan, mod, &ApplyOpts{
				PolicyClient: applyPolicyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			if called != tc.expectCalls {
				t.Fatalf("expected %d policy evaluation call(s), got %d", tc.expectCalls, called)
			}
		})
	}
}

func TestContext2Apply_PolicyEvaluation_Destroy(t *testing.T) {
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "test" {
			sensitive_value = "foo"
		}
		`
	policyConfig := `
		resource_policy "test_resource" "policy_name" {
			enforce {
				condition = true
			}
		}
		`
	configFiles := map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": policyConfig,
	}

	mod := testModuleInline(t, configFiles)
	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")

	// Build a pre-existing state with the resource already created.
	state := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_resource.test"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"bar","sensitive_value":"foo"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	planPolicyClient := policy.NewTestMockClient(t)
	var planEvalCalled int

	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		planEvalCalled++

		if req.Target != "test_resource" {
			t.Errorf("Plan: expected target to be test_resource, got %s", req.Target)
		}

		// For a destroy plan, attrs (the "after" value) should be null.
		if !req.Attrs.IsNull() {
			t.Errorf("Plan: expected null attrs for destroy evaluation, got %#v", req.Attrs)
		}

		// PriorAttrs should contain the state being destroyed.
		if req.PriorAttrs.IsNull() {
			t.Errorf("Plan: expected non-null PriorAttrs for destroy evaluation")
		}

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 1,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, state, &PlanOpts{
		Mode:         plans.DestroyMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: planPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if planEvalCalled != 1 {
		t.Fatalf("Plan: expected policy Evaluate to be called 1 time, got %d", planEvalCalled)
	}

	// Verify the plan contains a delete action.
	var foundDelete bool
	for _, rc := range plan.Changes.Resources {
		if rc.Addr.String() == "test_resource.test" {
			if rc.Action != plans.Delete {
				t.Errorf("Expected delete action for test_resource.test, got %s", rc.Action)
			}
			foundDelete = true
		}
	}
	if !foundDelete {
		t.Fatal("Expected test_resource.test in plan changes")
	}

	// --- Apply phase ---
	applyPolicyClient := policy.NewTestMockClient(t)
	var applyEvalCalled int

	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		applyEvalCalled++

		if req.Target != "test_resource" {
			t.Errorf("Apply: expected target to be test_resource, got %s", req.Target)
		}

		if !req.Attrs.IsNull() {
			t.Errorf("Apply: expected null attrs for destroy evaluation, got %#v", req.Attrs)
		}

		if req.PriorAttrs.IsNull() {
			t.Errorf("Apply: expected non-null PriorAttrs for destroy evaluation")
		}

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	resultState, diags := ctx.Apply(plan, mod, &ApplyOpts{
		PolicyClient: applyPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if applyEvalCalled != 1 {
		t.Fatalf("Apply: expected policy Evaluate to be called 1 time, got %d", applyEvalCalled)
	}

	// After a successful destroy, the resource should no longer be in state.
	remainingAddrs := resultState.AllManagedResourceInstanceObjectAddrs()
	if len(remainingAddrs) != 0 {
		t.Fatalf("expected 0 managed resources in the state after destroy, got %d: %v", len(remainingAddrs), remainingAddrs)
	}

	rs := resultState.Resource(mustAbsResourceAddr("test_resource.test"))
	if rs != nil {
		t.Fatal("expected test_resource.test to be removed from state after destroy")
	}
}
