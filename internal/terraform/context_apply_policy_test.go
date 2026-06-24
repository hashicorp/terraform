// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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

		provider "test" {
			value = sensitive("foo")
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
	provider.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":              {Type: cty.String, Computed: true},
					"value":           {Type: cty.String, Optional: true},
					"sensitive_value": {Type: cty.String, Optional: true, Sensitive: true},
				},
			},
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":              {Type: cty.String, Computed: true},
					"value":           {Type: cty.String, Optional: true},
					"sensitive_value": {Type: cty.String, Optional: true, Sensitive: true},
				},
			},
		},
	})

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
		"test": cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("foo"),
		}),
		"test_resource": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("foo"),
		}),
		"test_instance": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.UnknownVal(cty.String),
			"sensitive_value": cty.NullVal(cty.String),
		}),
	}
	actualPlan := make(map[string]cty.Value)

	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		var actualVal cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.Raw.IsNull() {
			mp := attrs.Raw.AsValueMap()
			actualVal = cty.ObjectVal(map[string]cty.Value{
				"value":           mp["value"],
				"sensitive_value": mp["sensitive_value"],
			})
		}
		actualPlan[target] = actualVal

		if actualVal.Type().HasAttribute("sensitive_value") {
			if !actualVal.GetAttr("sensitive_value").IsNull() && len(attrs.RedactedPaths) == 0 {
				t.Errorf("Expected redacted paths for sensitive attributes to be included in the request")
			}

			for _, path := range attrs.RedactedPaths {
				if !path.Equals(cty.Path{cty.GetAttrStep{Name: "sensitive_value"}}) {
					t.Errorf("Unexpected redacted path: %s", path)
				}
			}
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	planPolicyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) policy.EvaluationResponse {
		var actualVal cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.Raw.IsNull() {
			mp := attrs.Raw.AsValueMap()
			actualVal = cty.ObjectVal(map[string]cty.Value{
				"value": mp["value"],
			})
		}
		actualPlan[target] = actualVal

		if actualVal.Type().HasAttribute("value") {
			if !actualVal.GetAttr("value").IsNull() && len(attrs.RedactedPaths) == 0 {
				t.Errorf("Expected redacted paths for sensitive attributes to be included in the request")
			}

			for _, path := range attrs.RedactedPaths {
				if !path.Equals(cty.GetAttrPath("value")) {
					t.Errorf("Unexpected redacted path: %s", path)
				}
			}
		}
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
		"test": cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("foo"),
		}),
		"test_resource": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("foo"),
		}),
		"test_instance": cty.ObjectVal(map[string]cty.Value{
			"value":           cty.StringVal("parent"), // was unknown in the plan
			"sensitive_value": cty.NullVal(cty.String),
		}),
	}
	actualApply := make(map[string]cty.Value)

	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.Raw.IsNull() {
			mp := attrs.Raw.AsValueMap()
			actual = cty.ObjectVal(map[string]cty.Value{
				"value":           mp["value"],
				"sensitive_value": mp["sensitive_value"],
			})
		}
		actualApply[target] = actual

		if actual.Type().HasAttribute("sensitive_value") {
			if !actual.GetAttr("sensitive_value").IsNull() && len(attrs.RedactedPaths) == 0 {
				t.Errorf("Expected redacted paths for sensitive attributes to be included in the request")
			}

			for _, path := range attrs.RedactedPaths {
				if !path.Equals(cty.Path{cty.GetAttrStep{Name: "sensitive_value"}}) {
					t.Errorf("Unexpected redacted path: %s", path)
				}
			}
		}

		// this return does not actually do anything
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	applyPolicyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.Raw.IsNull() {
			mp := attrs.Raw.AsValueMap()
			actual = cty.ObjectVal(map[string]cty.Value{
				"value": mp["value"],
			})
		}
		actualApply[target] = actual

		if actual.Type().HasAttribute("value") {
			if !actual.GetAttr("value").IsNull() && len(attrs.RedactedPaths) == 0 {
				t.Errorf("Expected redacted paths for sensitive attributes to be included in the request")
			}

			for _, path := range attrs.RedactedPaths {
				if !path.Equals(cty.GetAttrPath("value")) {
					t.Errorf("Unexpected redacted path: %s", path)
				}
			}
		}

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

	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.Raw.IsNull() {
			mp := attrs.Raw.AsValueMap()
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
	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		var actual cty.Value
		attrs := req.Attrs
		target := req.Target
		if !attrs.Raw.IsNull() {
			mp := attrs.Raw.AsValueMap()
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
	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		policyRan.Store(true)

		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
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
			applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				called++
				if req.Target != tc.expectTarget {
					t.Fatalf("expected target %s, got %s", tc.expectTarget, req.Target)
				}
				if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
					ProviderType: "test",
					Operation:    tc.expectOp,
				}, protocmp.Transform()); diff != "" {
					t.Fatalf("unexpected resource metadata (-got +want):\n%s", diff)
				}

				actualAttrs := req.Attrs
				if !actualAttrs.Raw.IsNull() {
					mp := actualAttrs.Raw.AsValueMap()
					actualAttrs.Raw = cty.ObjectVal(map[string]cty.Value{
						"id":              mp["id"],
						"sensitive_value": mp["sensitive_value"],
					})
				}
				if diff := cmp.Diff(actualAttrs.Raw, tc.expectFinalAttr, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
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

func TestContext2Apply_PolicyEvaluation_PartialApply(t *testing.T) {
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "ok" {
			value = "ok"
		}

		resource "test_resource" "fail" {
			value = "fail"
		}
		`
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
		newState := req.PlannedState.AsValueMap()
		if newState["value"].AsString() == "fail" {
			resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"create failed",
				"simulated provider create failure",
			))
			return resp
		}

		newState["id"] = cty.StringVal("ok")
		resp.NewState = cty.ObjectVal(newState)
		return resp
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	applyPolicyClient := policy.NewTestMockClient(t)
	evaluatedPolicyValues := map[string]struct{}{}
	applyResults := plans.NewPolicyResults()
	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if req.Attrs.Raw.Type().IsObjectType() && !req.Attrs.Raw.IsNull() {
			evaluatedPolicyValues[req.Attrs.Raw.GetAttr("value").AsString()] = struct{}{}
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	_, diags = ctx.Apply(plan, mod, &ApplyOpts{
		PolicyClient:  applyPolicyClient,
		PolicyResults: applyResults,
	})
	if !diags.HasErrors() {
		t.Fatal("expected apply to fail")
	}

	var policyDiags tfdiags.Diagnostics
	for _, result := range applyResults.Iter() {
		policyDiags = policyDiags.Append(result.EvaluationResponse.Diagnostics.AsTerraformDiags())
	}

	// now check that the policy evaluation results match our expectations
	// we only expect evaluation for the "ok" resource, not the "fail" resource
	expectedValues := map[string]struct{}{"ok": {}}
	if diff := cmp.Diff(evaluatedPolicyValues, expectedValues); diff != "" {
		t.Errorf("unexpected evaluated policy values: %s", diff)
	}
	if len(policyDiags) != 0 {
		t.Fatalf("expected no policy diagnostics, got %d", len(policyDiags))
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

	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		planEvalCalled++

		if req.Target != "test_resource" {
			t.Errorf("Plan: expected target to be test_resource, got %s", req.Target)
		}

		// For a destroy plan, attrs (the "after" value) should be null.
		if !req.Attrs.Raw.IsNull() {
			t.Errorf("Plan: expected null attrs for destroy evaluation, got %#v", req.Attrs)
		}

		// PriorAttrs should contain the state being destroyed.
		if req.PriorAttrs.Raw.IsNull() {
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

	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		applyEvalCalled++

		if req.Target != "test_resource" {
			t.Errorf("Apply: expected target to be test_resource, got %s", req.Target)
		}

		if !req.Attrs.Raw.IsNull() {
			t.Errorf("Apply: expected null attrs for destroy evaluation, got %#v", req.Attrs)
		}

		if req.PriorAttrs.Raw.IsNull() {
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

func TestContext2Apply_PolicyCallback(t *testing.T) {
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_instance" "foo" {
			ami = "bar"
		}

		resource "test_instance" "baz" {
			ami = "qux"
			depends_on = [test_instance.foo]
		}

		resource "test_instance" "boop" {
			ami = "booper"
			depends_on = [test_instance.baz]
		}
	`

	policyConfig := `
		resource_policy "test_instance" "policy_name" {
			enforce {
				condition = core::getresources("some_resource_type", {})[0].value != null
			}
		}
	`

	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": policyConfig,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	// Apply echoes the planned config back as the new state, so the applied
	// instances carry concrete ami values in state for the callback to read.
	provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		resp.NewState = req.PlannedState
		return resp
	}

	type callbackResult struct {
		matchAllResults  []cty.Value
		filteredResults  []cty.Value
		noMatchCount     int
		unknownTypeCount int
	}

	// The plan policy client allows everything; the callback assertions run
	// during apply only.
	planPolicyClient := policy.NewTestMockClient(t)
	planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	var mu sync.Mutex
	results := make(map[string]callbackResult)

	applyPolicyClient := policy.NewTestMockClient(t)
	applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		cr := callbackResult{}

		if req.Callbacks.GetResources == nil {
			t.Errorf("GetResources callback was nil")
			return policy.EvaluationResponse{Overall: policy.AllowResult}
		}

		// 1. Match all test_instance resources with null attrs (no filter).
		all, _, err := req.Callbacks.GetResources(t.Context(), "test_instance", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources(test_instance, null): %v", err)
		} else {
			cr.matchAllResults = all
		}

		// 2. Match resources with ami="bar" filter.
		filtered, _, err := req.Callbacks.GetResources(t.Context(), "test_instance", cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}))
		if err != nil {
			t.Errorf("GetResources(test_instance, ami=bar): %v", err)
		} else {
			cr.filteredResults = filtered
		}

		// 3. Match with an attribute filter that will never match.
		noMatch, _, err := req.Callbacks.GetResources(t.Context(), "test_instance", cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("nonexistent"),
		}))
		if err != nil {
			t.Errorf("GetResources(test_instance, ami=nonexistent): %v", err)
		} else {
			cr.noMatchCount = len(noMatch)
		}

		// 4. Query for a resource type that doesn't exist in the config.
		unknown, _, err := req.Callbacks.GetResources(t.Context(), "nonexistent_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources(nonexistent_resource): %v", err)
		} else {
			cr.unknownTypeCount = len(unknown)
		}

		ami := req.Attrs.Raw.GetAttr("ami").AsString()
		mu.Lock()
		results[ami] = cr
		mu.Unlock()

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: planPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Apply(plan, mod, &ApplyOpts{
		PolicyClient: applyPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	// We expect exactly 3 evaluations (one per test_instance resource) during
	// apply.
	if len(results) != 3 {
		t.Fatalf("expected 3 policy evaluations during apply, got %d", len(results))
	}

	for ami, cr := range results {
		// match-all reads every test_instance from state: all 3 instances.
		if len(cr.matchAllResults) != 3 {
			t.Errorf("evaluation[%s]: expected 3 results for matchAll from state, got %d", ami, len(cr.matchAllResults))
		}
		// ami="bar" matches exactly the one foo instance.
		if len(cr.filteredResults) != 1 {
			t.Errorf("evaluation[%s]: expected 1 result for ami=bar filter, got %d", ami, len(cr.filteredResults))
		}
		// ami="nonexistent" matches nothing.
		if cr.noMatchCount != 0 {
			t.Errorf("evaluation[%s]: expected 0 results for ami=nonexistent filter, got %d", ami, cr.noMatchCount)
		}
		// An unknown resource type returns nothing.
		if cr.unknownTypeCount != 0 {
			t.Errorf("evaluation[%s]: expected 0 results for nonexistent_resource, got %d", ami, cr.unknownTypeCount)
		}
	}
}

func TestContext2Apply_PolicyCallback_GetDataSource(t *testing.T) {
	t.Parallel()

	type callbackResult struct {
		DataSourceResult   cty.Value
		DataSourceDeferred bool
	}

	testCases := map[string]struct {
		targetDataSource        string
		dataSourceReqConfig     cty.Value
		deferralAllowed         bool
		deferralResponse        *providers.Deferred
		expectedCallbackResults []callbackResult
		expectedErr             string
	}{
		"getdatasource returns result": {
			targetDataSource: "test_data_source",
			dataSourceReqConfig: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.NullVal(cty.String), // computed
				"foo": cty.StringVal("test val"),
			}),
			expectedCallbackResults: []callbackResult{
				{
					DataSourceResult: cty.ObjectVal(map[string]cty.Value{
						"id":  cty.StringVal("computed val"),
						"foo": cty.StringVal("test val"),
					}),
					DataSourceDeferred: false,
				},
			},
		},
		"getdatasource not found": {
			targetDataSource: "test_non_existent",
			dataSourceReqConfig: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.NullVal(cty.String),
				"foo": cty.StringVal("test val"),
			}),
			expectedErr: `no data source found for test_non_existent`,
		},
		"getdatasource returns deferred": {
			targetDataSource: "test_data_source",
			dataSourceReqConfig: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.NullVal(cty.String),
				"foo": cty.StringVal("test val"),
			}),
			deferralAllowed: true,
			deferralResponse: &providers.Deferred{
				Reason: providers.DeferredReasonAbsentPrereq,
			},
			expectedCallbackResults: []callbackResult{
				{
					DataSourceResult: cty.ObjectVal(map[string]cty.Value{
						"id":  cty.NullVal(cty.String),
						"foo": cty.StringVal("test val"),
					}),
					DataSourceDeferred: true,
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The plan policy client allows everything; the callback assertions run during apply only.
			planPolicyClient := policy.NewTestMockClient(t)
			planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			var gotResults []callbackResult
			applyPolicyClient := policy.NewTestMockClient(t)
			applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				result, deferred, err := req.Callbacks.GetDataSource(t.Context(), tc.targetDataSource, tc.dataSourceReqConfig)
				if err != nil {
					if !strings.Contains(err.Error(), tc.expectedErr) {
						t.Errorf("Unexpected error in callback GetDataSource(test_data_source): %v", err)
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				gotResults = append(gotResults, callbackResult{
					DataSourceResult:   result,
					DataSourceDeferred: deferred,
				})

				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			testProvider := testProvider("test")
			testProvider.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
				// We aren't checking the client capability here to enable testing an invalid provider implementation
				if tc.deferralResponse != nil {
					return providers.ReadDataSourceResponse{
						State:    req.Config,
						Deferred: tc.deferralResponse,
					}
				}

				stateVal := req.Config.AsValueMap()
				stateVal["id"] = cty.StringVal("computed val")
				return providers.ReadDataSourceResponse{
					State: cty.ObjectVal(stateVal),
				}
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider),
				},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			mod := testModuleInline(t, map[string]string{
				// Config isn't as important to this test, since we're just testing the getdatasource callback
				// which will directly call a configured provider instance.
				"main.tf": `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "foo" {
			value = "foo"
			defer = false
		}
	`,
				"main.tfpolicy.hcl": `# policy config is not read by Terraform`,
			})
			plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: tc.deferralAllowed,
				PolicyClient:    planPolicyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			policyResults := plans.NewPolicyResults()
			_, diags = ctx.Apply(plan, mod, &ApplyOpts{
				PolicyClient:  applyPolicyClient,
				PolicyResults: policyResults,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			var policyDiags tfdiags.Diagnostics
			for _, result := range policyResults.Iter() {
				policyDiags = policyDiags.Append(result.EvaluationResponse.Diagnostics.AsTerraformDiags())
			}
			tfdiags.AssertNoDiagnostics(t, policyDiags)

			if diff := cmp.Diff(tc.expectedCallbackResults, gotResults, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
				t.Errorf("unexpected policy callback results\n%s", diff)
			}
		})
	}
}

func TestContext2Apply_PolicyCallback_GetResources_Deferral(t *testing.T) {
	t.Parallel()

	type callbackResult struct {
		TestResourceMatches []cty.Value
		TestResourcePartial bool

		TestInstanceMatches []cty.Value
		TestInstancePartial bool
	}

	testCases := map[string]struct {
		config                  string
		expectedCallbackResults []callbackResult
	}{
		"resource type lookup with deferral return partial result": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		# Deferred (directly)
		resource "test_resource" "foo" {
			value = "foo"
			defer = true
		}

		# Deferred (by dependency)
		resource "test_instance" "foo" {
			value = test_resource.foo.value
		}

		# Not deferred, policy is evaluated
		resource "test_resource" "bar" {
			value = "bar"
			defer = false
		}
	`,
			expectedCallbackResults: []callbackResult{
				{
					// This is the only non-deferred test_resource we can match
					TestResourceMatches: []cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"id":              cty.StringVal(""),
							"value":           cty.StringVal("bar"),
							"sensitive_value": cty.NullVal(cty.String),
							"defer":           cty.False,
							"random":          cty.NullVal(cty.String),
							"nesting_single": cty.NullVal(cty.Object(map[string]cty.Type{
								"value":           cty.String,
								"sensitive_value": cty.String,
							})),
						}),
					},
					TestResourcePartial: true,

					// test_instance is deferred, so the response is partial with no matches
					TestInstanceMatches: []cty.Value{},
					TestInstancePartial: true,
				},
			},
		},
		"resource type lookup without deferral return full result": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		# Deferred (directly)
		resource "test_resource" "foo" {
			value = "foo"
			defer = true
		}

		# Not deferred, policy is evaluated
		resource "test_instance" "foo" {
			value = "foo"
		}
	`,
			expectedCallbackResults: []callbackResult{
				{
					// test_resource is deferred, so the response is partial with no matches
					TestResourceMatches: []cty.Value{},
					TestResourcePartial: true,

					// There are no test_instance resources being deferred so the response is not partial.
					TestInstanceMatches: []cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"id":            cty.StringVal(""),
							"ami":           cty.NullVal(cty.String),
							"dep":           cty.NullVal(cty.String),
							"num":           cty.NullVal(cty.Number),
							"require_new":   cty.NullVal(cty.String),
							"var":           cty.NullVal(cty.String),
							"foo":           cty.NullVal(cty.String),
							"bar":           cty.NullVal(cty.String),
							"compute":       cty.NullVal(cty.String),
							"compute_value": cty.NullVal(cty.String),
							"value":         cty.StringVal("foo"),
							"output":        cty.NullVal(cty.String),
							"write":         cty.NullVal(cty.String),
							"instance":      cty.NullVal(cty.String),
							"vpc_id":        cty.NullVal(cty.String),
							"type":          cty.StringVal(""),
							"unknown":       cty.StringVal(""),
						}),
					},
					TestInstancePartial: false,
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The plan policy client allows everything; the callback assertions run during apply only.
			planPolicyClient := policy.NewTestMockClient(t)
			planPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			gotResults := make([]callbackResult, 0)
			applyPolicyClient := policy.NewTestMockClient(t)
			applyPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				cr := callbackResult{}

				matches, partial, err := req.Callbacks.GetResources(t.Context(), "test_resource", cty.NullVal(cty.DynamicPseudoType))
				if err != nil {
					t.Errorf("Unexpected error in callback GetResources(test_resource): %v", err)
				} else {
					cr.TestResourceMatches = matches
					cr.TestResourcePartial = partial
				}

				matches, partial, err = req.Callbacks.GetResources(t.Context(), "test_instance", cty.NullVal(cty.DynamicPseudoType))
				if err != nil {
					t.Errorf("Unexpected error in callback GetResources(test_instance): %v", err)
				} else {
					cr.TestInstanceMatches = matches
					cr.TestInstancePartial = partial
				}

				gotResults = append(gotResults, cr)

				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider("test")),
				},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			mod := testModuleInline(t, map[string]string{
				"main.tf":           tc.config,
				"main.tfpolicy.hcl": `# policy config is not read by Terraform`,
			})
			plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: true,
				PolicyClient:    planPolicyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			policyResults := plans.NewPolicyResults()
			_, diags = ctx.Apply(plan, mod, &ApplyOpts{
				PolicyClient:  applyPolicyClient,
				PolicyResults: policyResults,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			var policyDiags tfdiags.Diagnostics
			for _, result := range policyResults.Iter() {
				policyDiags = policyDiags.Append(result.EvaluationResponse.Diagnostics.AsTerraformDiags())
			}
			tfdiags.AssertNoDiagnostics(t, policyDiags)

			if diff := cmp.Diff(tc.expectedCallbackResults, gotResults, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
				t.Errorf("unexpected policy callback results\n%s", diff)
			}
		})
	}
}

// TestContext2Apply_PolicySpanParentage verifies the OpenTelemetry wiring that
// makes per-resource policy spans children of the enclosing "terraform apply"
// command span.
func TestContext2Apply_PolicySpanParentage(t *testing.T) {
	// Collect spans into an in-memory exporter. The tracer provider is a
	// global, so this test must not run in parallel. We restore the previous
	// provider (rather than setting nil) so later tests in this package that
	// emit spans -- e.g. the policy-phase span -- don't hit a nil provider.
	prevProvider := otel.GetTracerProvider()
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		provider.Shutdown(context.Background())
		otel.SetTracerProvider(prevProvider)
	})

	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_instance" "foo" {
			ami = "bar"
		}
	`
	policyConfig := `
		resource_policy "test_instance" "policy_name" {
			enforce {
				condition = attrs.ami != ""
			}
		}
	`
	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": policyConfig,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	prov := testProvider("test")
	prov.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}
	prov.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		resp.NewState = req.PlannedState
		return resp
	}

	// Plan with an allow-all client (no spans needed during plan here).
	planClient := policy.NewTestMockClient(t)
	planClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	// The apply client starts a span named like the real client's, using the
	// context the engine passes in. That context is ctx.StopCtx() (the run
	// context), which acquireRun parents on the caller context we set below.
	var evaluateSpanContext trace.SpanContext
	applyClient := policy.NewTestMockClient(t)
	applyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		_, span := otel.Tracer("test").Start(ctx, "policy.client.evaluate_resource")
		evaluateSpanContext = span.SpanContext()
		span.End()
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	commandCtx, commandSpan := otel.Tracer("test").Start(context.Background(), "terraform apply")
	commandSpanContext := commandSpan.SpanContext()

	tfCtx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(prov),
		},
		TracingContext: commandCtx,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := tfCtx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: planClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = tfCtx.Apply(plan, mod, &ApplyOpts{
		PolicyClient: applyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)
	commandSpan.End()

	if !evaluateSpanContext.IsValid() {
		t.Fatal("policy evaluate span was never started during apply")
	}

	// The evaluate span must belong to the same trace as the command span...
	if evaluateSpanContext.TraceID() != commandSpanContext.TraceID() {
		t.Errorf("policy evaluate span is in a different trace than the terraform apply span\n  apply trace:    %s\n  evaluate trace: %s",
			commandSpanContext.TraceID(), evaluateSpanContext.TraceID())
	}

	// ...and it must be a descendant of the command span. We find the recorded
	// evaluate span in the exporter and walk its parent chain up to the command
	// span.
	spans := exp.GetSpans()
	byID := make(map[trace.SpanID]tracetest.SpanStub, len(spans))
	for _, s := range spans {
		byID[s.SpanContext.SpanID()] = s
	}

	cur, ok := byID[evaluateSpanContext.SpanID()]
	if !ok {
		t.Fatalf("evaluate span %s was not recorded by the exporter", evaluateSpanContext.SpanID())
	}
	foundCommandAncestor := false
	for {
		parentID := cur.Parent.SpanID()
		if parentID == commandSpanContext.SpanID() {
			foundCommandAncestor = true
			break
		}
		next, ok := byID[parentID]
		if !ok {
			break
		}
		cur = next
	}
	if !foundCommandAncestor {
		t.Errorf("policy.client.evaluate_resource span is not nested under the terraform apply span")
	}
}

// TestContext2Apply_PolicyPhaseSpanOutlivesEvaluations verifies that the
// policy-phase span ("terraform.policy.evaluate") is ended only after every
// policy evaluation, and any spans nested below them, has finished. The span is
// ended by the nodePolicyEvalFinish sentinel node, which must depend on every
// policy node in the subgraph; otherwise it closes the span while evaluations
// are still running and child spans outlive their parent.
//
// It mutates the global OpenTelemetry TracerProvider, so it must not run in
// parallel.
func TestContext2Apply_PolicyPhaseSpanOutlivesEvaluations(t *testing.T) {
	// Each evaluation sleeps for this long so an out-of-order finish node closes
	// the phase span observably early.
	const evalDelay = 100 * time.Millisecond

	// Record spans into an in-memory exporter. Restore the previous (global)
	// provider rather than setting nil so other tests don't hit a nil provider.
	prevProvider := otel.GetTracerProvider()
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		provider.Shutdown(context.Background())
		otel.SetTracerProvider(prevProvider)
	})

	// With the prior state below, this yields one update, one create and one
	// delete evaluation during apply.
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_instance" "to_update" {
			ami = "new"
		}

		resource "test_instance" "to_create" {
			ami = "fresh"
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
	prov := testProvider("test")
	prov.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}
	prov.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		resp.NewState = req.PlannedState
		return resp
	}

	// Prior state so that to_update is updated and to_delete is destroyed.
	priorState := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_instance.to_update"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"update-id","ami":"old"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		ss.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_instance.to_delete"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"delete-id","ami":"obsolete"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	// Plan uses a plain allow-all client; the span assertions concern apply.
	planClient := policy.NewTestMockClient(t)

	// The apply client mirrors the real client: per evaluation it starts a
	// per-resource span under the phase span, invokes a callback to nest a
	// further span, then does some work.
	applyClient := policy.NewTestMockClient(t)
	applyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		spanCtx, span := otel.Tracer("test").Start(ctx, "policy.client.evaluate_resource")
		defer span.End()

		if req.Callbacks.GetResources != nil {
			req.Callbacks.GetResources(spanCtx, "test_instance", cty.NullVal(cty.DynamicPseudoType))
		}

		time.Sleep(evalDelay)

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	tfCtx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(prov),
		},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := tfCtx.Plan(mod, priorState, &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: planClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	// Sanity check that the plan really covers all three evaluation kinds.
	gotActions := map[string]plans.Action{}
	for _, rc := range plan.Changes.Resources {
		gotActions[rc.Addr.String()] = rc.Action
	}
	for addr, want := range map[string]plans.Action{
		"test_instance.to_update": plans.Update,
		"test_instance.to_create": plans.Create,
		"test_instance.to_delete": plans.Delete,
	} {
		if gotActions[addr] != want {
			t.Fatalf("expected %s to have action %s, got %s", addr, want, gotActions[addr])
		}
	}

	_, diags = tfCtx.Apply(plan, mod, &ApplyOpts{
		PolicyClient: applyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	spans := exp.GetSpans()
	byID := make(map[trace.SpanID]tracetest.SpanStub, len(spans))
	for _, s := range spans {
		byID[s.SpanContext.SpanID()] = s
	}

	// isDescendantOf reports whether s is nested (at any depth) under ancestorID.
	isDescendantOf := func(s tracetest.SpanStub, ancestorID trace.SpanID) bool {
		parentID := s.Parent.SpanID()
		for parentID.IsValid() {
			if parentID == ancestorID {
				return true
			}
			parent, ok := byID[parentID]
			if !ok {
				return false
			}
			parentID = parent.Parent.SpanID()
		}
		return false
	}

	var phaseSpans []tracetest.SpanStub
	var evaluateSpans int
	for _, s := range spans {
		switch s.Name {
		case "terraform.policy.evaluate":
			phaseSpans = append(phaseSpans, s)
		case "policy.client.evaluate_resource":
			evaluateSpans++
		}
	}

	if len(phaseSpans) == 0 {
		t.Fatal("no terraform.policy.evaluate phase span was recorded")
	}
	if evaluateSpans < 3 {
		t.Fatalf("expected at least 3 policy.client.evaluate_resource spans (create, update, delete), got %d", evaluateSpans)
	}

	// Every span nested under a phase span must have finished no later than the
	// phase span itself.
	checkedDescendants := 0
	for _, phase := range phaseSpans {
		for _, s := range spans {
			if s.SpanContext.SpanID() == phase.SpanContext.SpanID() {
				continue
			}
			if !isDescendantOf(s, phase.SpanContext.SpanID()) {
				continue
			}
			checkedDescendants++
			if s.EndTime.After(phase.EndTime) {
				t.Errorf("span %q finished %s after its ancestor phase span %q; the phase span was closed before policy evaluation completed",
					s.Name, s.EndTime.Sub(phase.EndTime), phase.Name)
			}
		}
	}

	if checkedDescendants == 0 {
		t.Fatal("no descendant spans were found under any policy phase span; the test did not exercise nested policy evaluation as intended")
	}
}
