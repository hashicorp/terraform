// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestContext2Plan_PolicyEvaluation(t *testing.T) {
	type data struct {
		config *configs.Config
		plan   *plans.Plan
		state  *states.State
		diags  tfdiags.Diagnostics
		policy *policy.MockClient
	}
	cases := []struct {
		name                string
		mainConfig          string
		childConfig         string
		policyConfig        string
		state               *states.State
		planMode            plans.Mode
		forceReplace        []addrs.AbsResourceInstance
		deferralAllowed     bool
		prepareExpectations func(*testing.T, *data)
		assertPolicyResults func(*testing.T, *data)
	}{
		{
			name: "make policy evaluation calls",
			mainConfig: `
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

				variable "input2" {
					type = string
					default = "bar"
				}

				resource "test_resource" "test" {
					sensitive_value = "foo"
				}

				module "child" {
					source = "./child"
				}

				`,
			childConfig: `
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
					default = "child-foo"
				}

				resource "test_instance" "test" {
					value = "foo"
				}

				`,
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
							enforce {
									condition = attrs.sensitive_value == "foo"
					}
				}
				`,
			prepareExpectations: func(t *testing.T, data *data) {

				// The expected values to be sent for policy evaluation.
				expected := map[string]cty.Value{
					"test_resource": cty.ObjectVal(map[string]cty.Value{
						"value":           cty.NullVal(cty.String),
						"sensitive_value": cty.StringVal("foo"),
					}),

					"test_instance": cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("foo"),
					}),
				}
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					var actual cty.Value
					if !req.Attrs.IsNull() {
						mp := req.Attrs.AsValueMap()
						retMP := map[string]cty.Value{
							"value": mp["value"],
						}
						if sv, ok := mp["sensitive_value"]; ok {
							retMP["sensitive_value"] = sv
						}
						actual = cty.ObjectVal(retMP)
					}

					if diff := cmp.Diff(actual, expected[req.Target], cmp.Comparer(cty.Value.RawEquals)); diff != "" {
						t.Errorf("Unexpected diff (-got +want):\n%s", diff)
					}

					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         req.Target,
						ProviderType: "test",
						Operation:    proto.Operation_CREATE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					// Both resources are being created, so PriorAttrs should be null.
					if !req.PriorAttrs.IsNull() {
						t.Errorf("Expected null PriorAttrs for newly created %s, got non-null", req.Target)
						return policy.EvaluationResponse{}
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				data.policy.EvaluateModuleFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ModuleMetadata]) policy.EvaluationResponse {
					if req.Meta != nil {
						if req.Meta.Address != "module.child" {
							t.Errorf(`Expected module address to be "module.child", got "%s"`, req.Meta.Address)
						}
						if req.Meta.Source != "./child" {
							t.Errorf(`Expected module source to be "./child", got "%s"`, req.Meta.Source)
						}
					}

					if req.Target != "./child" {
						t.Errorf(`Expected target to be "./child", got %s`, req.Target)
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if !d.policy.EvaluateProviderCalled {
					t.Error("Expected policyClient.EvaluateProvider to be called")
				}
				if !d.policy.EvaluateModuleCalled {
					t.Error("Expected policyClient.EvaluateModule to be called")
				}
				if !d.policy.EvaluateCalled {
					t.Error("Expected policyClient.Evaluate to be called")
				}
				tfdiags.AssertNoDiagnostics(t, d.diags)
			},
		},

		{
			name: "deferred resource: policy is skipped",
			mainConfig: `
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

				variable "input2" {
					type = string
					default = "bar"
				}

				resource "test_resource" "test" {
					sensitive_value = "foo"
					defer = true
				}
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
							enforce {
									condition = attrs.sensitive_value == "foo"
					}
				}
				`,
			deferralAllowed: true,
			prepareExpectations: func(t *testing.T, data *data) {
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					t.Fatalf("Expected policy evaluation to be skipped for deferred resource, but got request for %s", req.Target)
					return policy.EvaluationResponse{}
				}
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if d.policy.EvaluateCalled {
					t.Error("Expected policyClient.Evaluate not to be called for deferred resource")
				}
				tfdiags.AssertNoDiagnostics(t, d.diags)

				if len(d.plan.DeferredResources) != 1 {
					t.Fatalf("Expected 1 deferred resource, got %d", len(d.plan.DeferredResources))
				}
			},
		},
		{
			name: "orphaned resource instance: policy is evaluated",
			mainConfig: `
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

				variable "input2" {
					type = string
					default = "bar"
				}

				resource "test_resource" "test" {
					sensitive_value = "foo"
				}
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
							enforce {
									condition = attrs.sensitive_value == "foo"
					}
				}
				`,
			state: states.BuildState(func(ss *states.SyncState) {
				testAddr := mustResourceInstanceAddr("test_resource.test")
				orphanAddr := mustResourceInstanceAddr("test_instance.child")
				ss.SetResourceInstanceCurrent(
					testAddr,
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bin","type":"test_resource","sensitive_value":"foo"}`),
						Dependencies: []addrs.ConfigResource{
							orphanAddr.ContainingResource().Config(),
						},
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
				ss.SetResourceInstanceCurrent(
					orphanAddr,
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bin","type":"test_instance","sensitive_value":"foo-child"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			prepareExpectations: func(t *testing.T, data *data) {
				// The expected values to be sent for policy evaluation.
				expected := map[string]cty.Value{
					"test_resource": cty.ObjectVal(map[string]cty.Value{
						"value":           cty.NullVal(cty.String),
						"sensitive_value": cty.StringVal("foo"),
					}),

					// orphaned resource, so a nil set would be sent for policy evaluation.
					"test_instance": cty.NilVal,
				}

				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					var actual cty.Value
					if !req.Attrs.IsNull() {
						mp := req.Attrs.AsValueMap()
						actual = cty.ObjectVal(map[string]cty.Value{
							"value":           mp["value"],
							"sensitive_value": mp["sensitive_value"],
						})
					}

					if diff := cmp.Diff(actual, expected[req.Target], cmp.Comparer(cty.Value.RawEquals)); diff != "" {
						t.Errorf("Unexpected diff (-got +want):\n%s", diff)
					}

					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         req.Target,
						ProviderType: "test",
						Operation:    proto.Operation_DELETE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					// Both resources have prior state, so PriorAttrs should be non-null.
					if req.PriorAttrs.IsNull() {
						t.Errorf("Expected non-null PriorAttrs for %s, got null", req.Target)
						return policy.EvaluationResponse{}
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
			},
		},
		{
			name: "parent resource policy succeeds, child module resource policy fails",
			mainConfig: `
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
				}
				`,
			childConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "test_instance" "test" {
					value = "forbidden_value"
				}
				`,
			policyConfig: `
				resource_policy "test_resource" "parent_policy" {
					enforce {
						condition = attrs.sensitive_value == "foo"
					}
				}

				resource_policy "test_instance" "child_policy" {
					enforce {
						condition = attrs.value != "forbidden_value"
					}
				}
				`,
			prepareExpectations: func(t *testing.T, data *data) {
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         req.Target,
						ProviderType: "test",
						Operation:    proto.Operation_CREATE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					if !req.PriorAttrs.IsNull() {
						t.Errorf("Expected null PriorAttrs for newly created %s, got non-null", req.Target)
						return policy.EvaluationResponse{}
					}

					// Child module resource policy fails
					if req.Target == "test_instance" {
						return policy.EvaluationResponse{
							Overall:      policy.DenyResult,
							Enforcements: []policy.EnforcementResult{},
							Diagnostics: policy.DiagsFromProto([]*proto.Diagnostic{
								{
									Severity: proto.Severity_ERROR,
									Summary:  "Child module policy violation",
									Detail:   "Resource test_instance.test violates policy: forbidden value detected",
									Result: &proto.DiagnosticResult{
										Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
									},
									Subject: &proto.Range{
										Filename: "child_policy.tfpolicy.hcl",
										Start: &proto.Position{
											Line:   1,
											Column: 1,
										},
										End: &proto.Position{
											Line:   4,
											Column: 10,
										},
									},
								},
							}, nil),
						}
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
			},
			assertPolicyResults: func(t *testing.T, data *data) {
				tfdiags.AssertDiagnosticCount(t, data.diags, 1)
				var exp tfdiags.Diagnostics
				// We want to test that the diagnostic subject is set to the terraform file,
				// with an internal extra data for the policy file.
				// This allows us to display both source information in the diagnostic.
				policyClientDiag := data.diags[0]
				policyExtra, ok := data.diags[0].ExtraInfo().(*policy.PolicyExtra)
				if !ok {
					t.Fatalf("Expected diagnostic extra info to be a *policy.PolicyExtra, got %T", policyClientDiag.ExtraInfo())
				}
				tfSubject := policyClientDiag.Source().Subject.ToHCL().Ptr()
				if filepath.Ext(tfSubject.Filename) != ".tf" {
					t.Fatalf("Expected diagnostic subject filename to end with .tf, got %q", tfSubject.Filename)
				}
				if !strings.HasSuffix(policyExtra.Range.Subject.Filename, ".tfpolicy.hcl") {
					t.Fatalf("Expected policy diagnostic subject filename to end with .tfpolicy.hcl, got %q", policyExtra.Range.Subject.Filename)
				}

				exp = exp.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Child module policy violation",
					Detail:   "Resource test_instance.test violates policy: forbidden value detected",
					Subject:  tfSubject,
				})
				tfdiags.AssertDiagnosticsMatch(t, data.diags, exp)

				// Check that parent resource was planned successfully but child resource was not
				resourceChanges := data.plan.Changes.Resources
				var parentFound, childFound bool
				for _, change := range resourceChanges {
					if change.Addr.String() == "test_resource.test" {
						parentFound = true
					}
					if change.Addr.String() == "module.child.test_instance.test" {
						childFound = true
					}
				}

				if !parentFound {
					t.Error("Expected parent resource test_resource.test to be planned")
				}
				if !childFound {
					t.Error("Expected child resource module.child.test_instance.test to be planned due to policy failure")
				}
			},
		},
		{
			name: "destroy plan: policy is evaluated with null attrs",
			mainConfig: `
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
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
					enforce {
						condition = true
					}
				}
				`,
			planMode: plans.DestroyMode,
			state: states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.test"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bar","type":"test_resource","sensitive_value":"foo"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			prepareExpectations: func(t *testing.T, data *data) {
				// EvalPolicy should be called during the actual destroy plan with null attrs
				var called int
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					called++
					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         "test_resource",
						ProviderType: "test",
						Operation:    proto.Operation_DELETE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					if !req.Attrs.IsNull() {
						t.Errorf("Expected null attrs for destroy evaluation")
					}

					if req.PriorAttrs.IsNull() {
						t.Errorf("Expected non-null PriorAttrs for destroy evaluation")
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				t.Cleanup(func() {
					if called != 1 {
						t.Errorf("Expected EvalPolicy to be called once got %d", called)
					}
				})
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if !d.policy.EvaluateCalled {
					t.Error("Expected policyClient.Evaluate to be called for destroy plan")
				}
				tfdiags.AssertNoDiagnostics(t, d.diags)

				// Verify the plan contains a delete action
				for _, rc := range d.plan.Changes.Resources {
					if rc.Addr.String() == "test_resource.test" {
						if rc.Action != plans.Delete {
							t.Errorf("Expected delete action for test_resource.test, got %s", rc.Action)
						}
						return
					}
				}
				t.Error("Expected test_resource.test in plan changes")
			},
		},
		{
			name: "destroy plan: policy denies destruction",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "test_resource" "test" {
					sensitive_value = "secret"
				}
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "no_destroy" {
					enforce {
						condition = false
					}
				}
				`,
			planMode: plans.DestroyMode,
			state: states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.test"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bar","type":"test_resource","sensitive_value":"secret"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			prepareExpectations: func(t *testing.T, data *data) {
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					if diff := cmp.Diff(req.Meta, proto.ResourceMetadata{
						Type:         "test_resource",
						ProviderType: "test",
						Operation:    proto.Operation_DELETE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					if req.PriorAttrs.IsNull() {
						t.Errorf("Expected non-null PriorAttrs for destroy evaluation")
						return policy.EvaluationResponse{}
					}

					return policy.EvaluationResponse{
						Overall:      policy.DenyResult,
						Enforcements: []policy.EnforcementResult{},
						Diagnostics: policy.DiagsFromProto([]*proto.Diagnostic{
							{
								Severity: proto.Severity_ERROR,
								Summary:  "Destruction not allowed",
								Detail:   "Policy prevents destruction of test_resource.test",
								Result: &proto.DiagnosticResult{
									Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
								},
								Subject: &proto.Range{
									Filename: "no_destroy.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   4,
										Column: 10,
									},
								},
							},
						}, nil),
					}
				}
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if !d.policy.EvaluateCalled {
					t.Error("Expected policyClient.Evaluate to be called for destroy plan")
				}
				tfdiags.AssertDiagnosticCount(t, d.diags, 1)

				var exp tfdiags.Diagnostics
				policyClientDiag := d.diags[0]
				tfSubject := policyClientDiag.Source().Subject.ToHCL().Ptr()
				exp = exp.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Destruction not allowed",
					Detail:   "Policy prevents destruction of test_resource.test",
					Subject:  tfSubject,
				})
				tfdiags.AssertDiagnosticsMatch(t, d.diags, exp)
			},
		},
		{
			name: "create resource with cbd. policy is evaluated with create operation",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "test_resource" "test" {
					sensitive_value = "after"

					lifecycle {
						create_before_destroy = true
					}
				}
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
					enforce {
						condition = true
					}
				}
				`,
			state: states.NewState(),
			prepareExpectations: func(t *testing.T, data *data) {
				var called int
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					called++
					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         "test_resource",
						ProviderType: "test",
						Operation:    proto.Operation_CREATE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				t.Cleanup(func() {
					if called != 1 {
						t.Errorf("Expected EvalPolicy to be called once got %d", called)
					}
				})
			},
		},
		{
			name: "update resource with cbd. policy is evaluated with update operation",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "test_resource" "test" {
					sensitive_value = "after"

					lifecycle {
						create_before_destroy = true
					}
				}
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
					enforce {
						condition = true
					}
				}
				`,
			state: states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.test"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bar","type":"test_resource","sensitive_value":"secret"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			prepareExpectations: func(t *testing.T, data *data) {
				var called int
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					called++
					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         "test_resource",
						ProviderType: "test",
						Operation:    proto.Operation_UPDATE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				t.Cleanup(func() {
					if called != 1 {
						t.Errorf("Expected EvalPolicy to be called once got %d", called)
					}
				})
			},
		},
		{
			name: "replace resource with cbd. policy is evaluated with update operation",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "test_resource" "test" {
					sensitive_value = "after"

					lifecycle {
						create_before_destroy = true
					}
				}
				`,
			childConfig: "",
			policyConfig: `
				resource_policy "test_resource" "policy_name" {
					enforce {
						condition = true
					}
				}
				`,
			state: states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.test"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bar","type":"test_resource","sensitive_value":"secret"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			forceReplace: []addrs.AbsResourceInstance{mustResourceInstanceAddr("test_resource.test")},
			prepareExpectations: func(t *testing.T, data *data) {
				var called int
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
					called++
					if diff := cmp.Diff(req.Meta, &proto.ResourceMetadata{
						Type:         "test_resource",
						ProviderType: "test",
						Operation:    proto.Operation_UPDATE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				t.Cleanup(func() {
					if called != 1 {
						t.Errorf("Expected EvalPolicy to be called once got %d", called)
					}
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configFiles := map[string]string{"main.tf": tc.mainConfig}
			if tc.childConfig != "" {
				configFiles["child/child.tf"] = tc.childConfig
			}
			if tc.policyConfig != "" {
				configFiles["main.tfpolicy.hcl"] = tc.policyConfig
			}

			mod := testModuleInline(t, configFiles)
			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			state := states.NewState()
			if tc.state != nil {
				state = tc.state
			}

			// mock expectations
			policyClient := policy.NewTestMockClient(t)
			data := &data{
				config: mod,
				state:  state,
				policy: policyClient,
			}
			planMode := tc.planMode
			if planMode == 0 {
				planMode = plans.NormalMode
			}

			if tc.prepareExpectations != nil {
				tc.prepareExpectations(t, data)
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
				Parallelism: 1,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			plan, diags := ctx.Plan(mod, state, &PlanOpts{
				Mode:            planMode,
				SetVariables:    testInputValuesUnset(mod.Module.Variables),
				PolicyClient:    policyClient,
				DeferralAllowed: tc.deferralAllowed,
				ForceReplace:    tc.forceReplace,
			})
			// The plan itself should not have diagnostics. Policy diagnostics are propagated via
			// the PolicyResults object.
			tfdiags.AssertNoDiagnostics(t, diags)

			data.plan = plan
			for _, result := range plan.PolicyResults.Iter() {
				data.diags = data.diags.Append(result.EvaluationResponse.Diagnostics.AsTerraformDiags())
			}
			if tc.assertPolicyResults != nil {
				tc.assertPolicyResults(t, data)
			} else {
				tfdiags.AssertNoDiagnostics(t, data.diags)
			}
		})
	}
}

func TestContext2Plan_PolicyCallback(t *testing.T) {
	// This test verifies that the GetResources callback provided during policy
	// evaluation works correctly: matching all resources, filtering by
	// attributes, returning nothing for non-matching filters, and returning
	// nothing for non-existent resource types.
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

	policyClient := policy.NewTestMockClient(t)

	type callbackResult struct {
		matchAllCount    int
		matchAllResults  []cty.Value
		filteredCount    int
		filteredResults  []cty.Value
		noMatchCount     int
		unknownTypeCount int
	}

	var mu sync.Mutex
	results := make(map[string]callbackResult)

	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		cr := callbackResult{}

		if req.Callbacks.GetResources == nil {
			t.Errorf("GetResources callback was nil")
			return policy.EvaluationResponse{Overall: policy.AllowResult}
		}

		// 1. Match all test_instance resources with null attrs (no filter).
		all, err := req.Callbacks.GetResources("test_instance", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources(test_instance, null): %v", err)
		} else {
			cr.matchAllCount = len(all)
			cr.matchAllResults = all
		}

		// 2. Match resources with ami="bar" filter.
		filtered, err := req.Callbacks.GetResources("test_instance", cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}))
		if err != nil {
			t.Errorf("GetResources(test_instance, ami=bar): %v", err)
		} else {
			cr.filteredCount = len(filtered)
			cr.filteredResults = filtered
		}

		// 3. Match with an attribute filter that will never match any planned resource.
		noMatch, err := req.Callbacks.GetResources("test_instance", cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("nonexistent"),
		}))
		if err != nil {
			t.Errorf("GetResources(test_instance, ami=nonexistent): %v", err)
		} else {
			cr.noMatchCount = len(noMatch)
		}

		// 4. Query for a resource type that doesn't exist in the config.
		unknown, err := req.Callbacks.GetResources("nonexistent_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources(nonexistent_resource): %v", err)
		} else {
			cr.unknownTypeCount = len(unknown)
		}

		// Key by the ami attribute of the resource being evaluated.
		ami := req.Attrs.GetAttr("ami").AsString()
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
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	var policyDiags tfdiags.Diagnostics
	for _, result := range plan.PolicyResults.Iter() {
		policyDiags = policyDiags.Append(result.EvaluationResponse.Diagnostics.AsTerraformDiags())
	}
	tfdiags.AssertNoDiagnostics(t, policyDiags)

	// We expect exactly 3 evaluations (one per test_instance resource).
	if len(results) != 3 {
		t.Fatalf("expected 3 policy evaluations, got %d", len(results))
	}

	for ami, cr := range results {
		var expectedTotal int
		filteredCount := 1
		switch ami {
		case "bar":
			expectedTotal = 0
			filteredCount = 0
		case "qux":
			expectedTotal = 1
		case "booper":
			expectedTotal = 2
		}
		if cr.matchAllCount != expectedTotal {
			t.Errorf("evaluation[%s]: expected %d result for matchAll, got %d", ami, expectedTotal, cr.matchAllCount)
		}

		// Filtering by ami="nonexistent" should always return 0 for all evaluations.
		if cr.noMatchCount != 0 {
			t.Errorf("evaluation[%s]: expected 0 results for ami=nonexistent filter, got %d", ami, cr.noMatchCount)
		}

		// Querying for a non-existent resource type should always return 0.
		if cr.unknownTypeCount != 0 {
			t.Errorf("evaluation[%s]: expected 0 results for nonexistent_resource, got %d", ami, cr.unknownTypeCount)
		}

		// The filtered result should only match one resource "bar", except when evaluating "bar" itself.
		if cr.filteredCount != filteredCount {
			t.Errorf("evaluation[%s]: expected filtered count %d, got %d", ami, filteredCount, cr.filteredCount)
		}
	}
}
