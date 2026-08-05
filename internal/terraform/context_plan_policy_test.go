// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_PolicyEvaluation(t *testing.T) {
	type data struct {
		config          *configs.Config
		plan            *plans.Plan
		viewHook        *testHook
		state           *states.State
		diags           tfdiags.Diagnostics
		policy          *policy.MockClient
		policyEvalCalls int
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
		expectCalls         int
		prepareExpectations func(*testing.T, *data)
		assertPolicyResults func(*testing.T, *data)
	}{
		{
			name:        "make policy evaluation calls",
			expectCalls: 2,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					var actual cty.Value
					if !req.Attrs.Raw.IsNull() {
						mp := req.Attrs.Raw.AsValueMap()
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

					expectedMeta := &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType: "test",
						Operation:    proto.Operation_CREATE,
					}
					if req.Target == "test_instance" {
						expectedMeta.ModulePath = "module.child"
					}

					if diff := cmp.Diff(req.Meta, expectedMeta, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					// Both resources are being created, so PriorAttrs should be null.
					if !req.PriorAttrs.Raw.IsNull() {
						t.Errorf("Expected null PriorAttrs for newly created %s, got non-null", req.Target)
						return policy.EvaluationResponse{}
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				data.policy.EvaluateModuleFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) policy.EvaluationResponse {
					if req.Meta != nil {
						if req.Meta.Address != "module.child" {
							t.Errorf(`Expected module address to be "module.child", got "%s"`, req.Meta.Address)
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
			name:        "deferred resource: policy is skipped",
			expectCalls: 0,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
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
			name:        "orphaned resource instance: policy is evaluated",
			expectCalls: 2,
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

				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					var actual cty.Value
					if !req.Attrs.Raw.IsNull() {
						mp := req.Attrs.Raw.AsValueMap()
						actual = cty.ObjectVal(map[string]cty.Value{
							"value":           mp["value"],
							"sensitive_value": mp["sensitive_value"],
						})
					}

					if diff := cmp.Diff(actual, expected[req.Target], cmp.Comparer(cty.Value.RawEquals)); diff != "" {
						t.Errorf("Unexpected diff (-got +want):\n%s", diff)
					}

					expectedMeta := &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType: "test",
						Operation:    proto.Operation_NO_OP,
					}
					if req.Target == "test_instance" {
						expectedMeta.Operation = proto.Operation_DELETE
					}
					if diff := cmp.Diff(req.Meta, expectedMeta, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					// Both resources have prior state, so PriorAttrs should be non-null.
					if req.PriorAttrs.Raw.IsNull() {
						t.Errorf("Expected non-null PriorAttrs for %s, got null", req.Target)
						return policy.EvaluationResponse{}
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
			},
		},
		{
			name:        "parent resource policy succeeds, child module resource policy fails",
			expectCalls: 2,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					expectedMeta := &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType: "test",
						Operation:    proto.Operation_CREATE,
					}
					if req.Target == "test_instance" {
						expectedMeta.ModulePath = "module.child"
					}

					if diff := cmp.Diff(req.Meta, expectedMeta, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					if !req.PriorAttrs.Raw.IsNull() {
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
			name:        "destroy plan: policy is evaluated with null attrs",
			expectCalls: 1,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					called++
					if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType: "test",
						Operation:    proto.Operation_DELETE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					if !req.Attrs.Raw.IsNull() {
						t.Errorf("Expected null attrs for destroy evaluation")
					}

					if req.PriorAttrs.Raw.IsNull() {
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
			name:        "destroy plan: policy denies destruction",
			expectCalls: 1,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					if diff := cmp.Diff(req.Meta, proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType: "test",
						Operation:    proto.Operation_DELETE,
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					if req.PriorAttrs.Raw.IsNull() {
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
			name:        "create resource with cbd. policy is evaluated with create operation",
			expectCalls: 1,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					called++
					if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
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
			name:        "update resource with cbd. policy is evaluated with update operation",
			expectCalls: 1,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					called++
					if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
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
			name:        "replace resource with cbd. policy is evaluated with update operation",
			expectCalls: 1,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					called++
					if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
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
			name:        "normal plan: removed config should send null attrs to policy",
			expectCalls: 1,
			mainConfig: `
						terraform {
							required_providers {
								test = {
									source = "hashicorp/test"
									version = "1.0.0"
								}
							}
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
			planMode: plans.NormalMode,
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
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					if req.PriorAttrs.Raw.IsNull() {
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
				exp = exp.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Destruction not allowed",
					Detail:   "Policy prevents destruction of test_resource.test",
				})
				tfdiags.AssertDiagnosticsMatch(t, d.diags, exp)
			},
		},
		{
			// This test uses a configuration that would result in cyclic errors
			// if module inputs were resolved and sent for module policy evaluation.
			name:        "module inputs omitted from module policy evaluation",
			expectCalls: 2,
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				module "child" {
					source = "./child"
					input  = "child-value"
					input2 = module.child.output
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
					type    = string
					default = "default"
				}

				variable "input2" {
					type    = string
					default = "default"
				}

				resource "test_instance" "test" {
					value = var.input
				}
				
				resource "test_instance" "test2" {
					value = var.input2
				}

				output "output" {
					value = resource.test_instance.test.value
				}
			`,
			prepareExpectations: func(t *testing.T, data *data) {
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				data.policy.EvaluateModuleFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) policy.EvaluationResponse {
					if !req.Attrs.Raw.RawEquals(cty.DynamicVal) {
						t.Fatalf("expected module policy evaluation for %s to omit attrs, got %#v", req.Target, req.Attrs)
					}
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if !d.policy.EvaluateModuleCalled {
					t.Fatal("Expected policyClient.EvaluateModule to be called")
				}
				tfdiags.AssertNoDiagnostics(t, d.diags)
			},
		},
		{
			name:        "normal plan: removed child module config still evaluates policy with nil resource config",
			expectCalls: 1,
			mainConfig: `
						terraform {
							required_providers {
								test = {
									source = "hashicorp/test"
									version = "1.0.0"
								}
							}
						}
						`,
			childConfig: "",
			policyConfig: `
						resource_policy "test_resource" "allow_destroy" {
							enforce {
								condition = true
							}
						}
						`,
			planMode: plans.NormalMode,
			state: states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("module.child.test_resource.test"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bar","type":"test_resource","sensitive_value":"secret"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			prepareExpectations: func(t *testing.T, data *data) {
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					if req.Target != "test_resource" {
						t.Fatalf("Expected target test_resource, got %q", req.Target)
					}
					if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType: "test",
						Operation:    proto.Operation_DELETE,
						ModulePath:   "module.child",
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}
					if !req.Attrs.Raw.IsNull() {
						t.Errorf("Expected null attrs for destroy evaluation")
					}
					if req.PriorAttrs.Raw.IsNull() {
						t.Errorf("Expected non-null PriorAttrs for destroy evaluation")
						return policy.EvaluationResponse{}
					}

					return policy.EvaluationResponse{
						Overall: policy.AllowResult,
						Enforcements: []policy.EnforcementResult{{
							Result:  policy.AllowResult,
							Message: "allowed",
						}},
					}
				}
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if !d.policy.EvaluateCalled {
					t.Error("Expected policyClient.Evaluate to be called for destroy plan")
				}
				tfdiags.AssertNoDiagnostics(t, d.diags)

				var gotResults int
				for addr, result := range d.viewHook.PolicyResults {
					gotResults++
					if addr != "module.child.test_resource.test" {
						t.Fatalf("Expected policy result for module.child.test_resource.test, got %q", addr)
					}
					if len(result.Enforcements) != 1 {
						t.Fatalf("Expected 1 enforcement result, got %d", len(result.Enforcements))
					}
					if result.Enforcements[0].LocalRange != nil {
						t.Fatalf("Expected empty local range for removed config, got %#v", result.Enforcements[0].LocalRange)
					}
				}
				if gotResults != 1 {
					t.Fatalf("Expected 1 stored policy result, got %d", gotResults)
				}

				for _, rc := range d.plan.Changes.Resources {
					if rc.Addr.String() == "module.child.test_resource.test" {
						if rc.Action != plans.Delete {
							t.Errorf("Expected delete action for module.child.test_resource.test, got %s", rc.Action)
						}
						return
					}
				}
				t.Error("Expected module.child.test_resource.test in plan changes")
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
				config:   mod,
				state:    state,
				policy:   policyClient,
				viewHook: &testHook{},
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
				Hooks:       []Hook{data.viewHook},
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

			if data.policyEvalCalls != tc.expectCalls {
				t.Fatalf("expected %d resource policy evaluation call(s), got %d", tc.expectCalls, data.policyEvalCalls)
			}

			for _, result := range data.viewHook.PolicyResults {
				data.diags = data.diags.Append(result.Diagnostics.AsTerraformDiags())
			}
			if tc.assertPolicyResults != nil {
				tc.assertPolicyResults(t, data)
			} else {
				tfdiags.AssertNoDiagnostics(t, data.diags)
			}
		})
	}
}

func TestContext2Plan_PolicyEvaluation_RedactedPaths(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}

			variable "current_secret" {
				type      = string
				sensitive = true
			}

			resource "test_resource" "test" {
				schema_sensitive = "from-config"
				current_only     = var.current_secret
			}
		`,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"schema_sensitive": {
						Type:      cty.String,
						Optional:  true,
						Sensitive: true,
					},
					"current_only": {
						Type:     cty.String,
						Optional: true,
					},
					"prior_only": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	})

	state := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_resource.test"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"existing","schema_sensitive":"from-state","prior_only":"prior-secret"}`),
				AttrSensitivePaths: []cty.Path{
					cty.GetAttrPath("prior_only"),
				},
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	policyClient := policy.NewTestMockClient(t)
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"current_secret": &InputValue{
				Value:      cty.StringVal("current-secret").Mark(marks.Sensitive),
				SourceType: ValueFromCaller,
			},
		},
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if !policyClient.EvaluateCalled {
		t.Fatal("expected resource policy evaluation to be called")
	}

	if policyClient.EvaluateRequest.Target != "test_resource" {
		t.Fatalf("unexpected policy target %q", policyClient.EvaluateRequest.Target)
	}
	if diff := cmp.Diff(policyClient.EvaluateRequest.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
		ProviderType: "test",
		Operation:    proto.Operation_UPDATE,
	}, protocmp.Transform()); diff != "" {
		t.Fatalf("invalid resource metadata: %s", diff)
	}

	wantAttrs := []cty.Path{
		cty.GetAttrPath("schema_sensitive"),
		cty.GetAttrPath("current_only"),
	}
	wantPriorAttrs := []cty.Path{
		cty.GetAttrPath("schema_sensitive"),
		cty.GetAttrPath("prior_only"),
	}

	assertPathsEqual(t, policyClient.EvaluateRequest.Attrs.RedactedPaths, wantAttrs)
	assertPathsEqual(t, policyClient.EvaluateRequest.PriorAttrs.RedactedPaths, wantPriorAttrs)
}

func TestContext2Plan_PolicyEvaluation_WriteOnly(t *testing.T) {
	providerAddr := addrs.NewDefaultProvider("ephem")
	provider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"ephem_write_only": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"normal": {
								Type:     cty.String,
								Required: true,
							},
							"write_only": {
								Type:      cty.String,
								Required:  true,
								WriteOnly: true,
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name             string
		planResourceFn   func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse
		expectPolicyCall bool
		assertPolicyReq  func(*testing.T, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata])
		expectDiags      tfdiags.Diagnostics
	}{
		{
			name: "policy receives null write-only attrs",
			planResourceFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				return providers.PlanResourceChangeResponse{
					PlannedState: cty.ObjectVal(map[string]cty.Value{
						"normal":     req.ProposedNewState.GetAttr("normal"),
						"write_only": cty.NullVal(cty.String),
					}),
				}
			},
			expectPolicyCall: true,
			assertPolicyReq: func(t *testing.T, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) {
				t.Helper()

				if req.Target != "ephem_write_only" {
					t.Fatalf("unexpected policy target %q", req.Target)
				}
				if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
					ProviderType: "ephem",
					Operation:    proto.Operation_UPDATE,
				}, protocmp.Transform()); diff != "" {
					t.Fatalf("invalid resource metadata: %s", diff)
				}

				if req.Attrs.Raw.IsNull() {
					t.Fatal("expected non-null attrs for policy evaluation")
				}
				if req.PriorAttrs.Raw.IsNull() {
					t.Fatal("expected non-null prior attrs for policy evaluation")
				}

				if got := req.Attrs.Raw.GetAttr("normal").AsString(); got != "updated" {
					t.Fatalf("expected attrs.normal to be updated, got %q", got)
				}
				if got := req.PriorAttrs.Raw.GetAttr("normal").AsString(); got != "outdated" {
					t.Fatalf("expected prior_attrs.normal to be outdated, got %q", got)
				}
				if got := req.Attrs.Raw.GetAttr("write_only"); !got.IsNull() {
					t.Fatalf("expected attrs.write_only to be null, got %v", got)
				}
				if got := req.PriorAttrs.Raw.GetAttr("write_only"); !got.IsNull() {
					t.Fatalf("expected prior_attrs.write_only to be null, got %v", got)
				}
			},
		},
		{
			name: "provider returning write-only value fails before policy",
			planResourceFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				return providers.PlanResourceChangeResponse{
					PlannedState: cty.ObjectVal(map[string]cty.Value{
						"normal":     req.ProposedNewState.GetAttr("normal"),
						"write_only": cty.StringVal("should not be returned by the provider"),
					}),
				}
			},
			expectPolicyCall: false,
			expectDiags: tfdiags.Diagnostics{}.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid plan",
				`Provider "provider[\"registry.terraform.io/hashicorp/ephem\"]" returned a value for the write-only attribute "ephem_write_only.wo.write_only" during planning. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.`,
			)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModuleInline(t, map[string]string{
				"main.tf": `
					variable "ephem" {
						type      = string
						ephemeral = true
					}

					resource "ephem_write_only" "wo" {
						normal     = "updated"
						write_only = var.ephem
					}
				`,
				"main.tfpolicy.hcl": `
					resource_policy "ephem_write_only" "policy_name" {
						enforce {
							condition = true
						}
					}
				`,
			})

			provider.PlanResourceChangeFn = tc.planResourceFn

			priorState := states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("ephem_write_only.wo"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"normal":"outdated","write_only":null}`),
					},
					addrs.AbsProviderConfig{
						Provider: providerAddr,
						Module:   addrs.RootModule,
					},
				)
			})

			policyClient := policy.NewTestMockClient(t)
			policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				if !tc.expectPolicyCall {
					t.Fatalf("expected policy evaluation to be skipped, got request for %s", req.Target)
				}
				if tc.assertPolicyReq != nil {
					tc.assertPolicyReq(t, req)
				}
				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
			})

			plan, diags := ctx.Plan(m, priorState, &PlanOpts{
				Mode: plans.NormalMode,
				SetVariables: InputValues{
					"ephem": {
						Value:      cty.StringVal("ephemeral-secret"),
						SourceType: ValueFromCLIArg,
					},
				},
				PolicyClient: policyClient,
			})

			if tc.expectDiags != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectDiags)
			} else {
				if plan == nil {
					t.Fatal("expected non-nil plan")
				}
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			if policyClient.EvaluateCalled != tc.expectPolicyCall {
				t.Fatalf("expected policy evaluation called=%t, got %t", tc.expectPolicyCall, policyClient.EvaluateCalled)
			}
		})
	}
}

func TestContext2Plan_PolicyEvaluation_NoResourceRunsAfterPolicy(t *testing.T) {
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

	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": samplePolicyConfig,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")

	var policyRan atomic.Bool
	var planCalls atomic.Int32

	provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		callNum := planCalls.Add(1)
		if callNum == 2 {
			time.Sleep(150 * time.Millisecond)
		}

		if policyRan.Load() {
			t.Fatalf("resource plan for %s ran after policy evaluation", req.TypeName)
		}

		resp.PlannedState = req.ProposedNewState
		return resp
	}

	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		policyRan.Store(true)

		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType: "test",
			Operation:    proto.Operation_CREATE,
		}, protocmp.Transform()); diff != "" {
			t.Errorf("Invalid resource metadata: %s", diff)
		}

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	h := &testHook{}
	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 4,
		Hooks:       []Hook{h},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if !policyClient.EvaluateCalled {
		t.Fatal("expected policy evaluation to be called during plan")
	}

	if len(plan.Changes.Resources) != 2 {
		t.Fatalf("expected 2 planned resource changes, got %d", len(plan.Changes.Resources))
	}

	var policyDiags tfdiags.Diagnostics
	for _, result := range h.PolicyResults {
		policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
	}
	tfdiags.AssertNoDiagnostics(t, policyDiags)
}

func TestContext2Plan_PolicyEvaluation_ManagedResourcesOnly(t *testing.T) {
	// This tests that only managed resources are sent for policy evaluation.
	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		data "test_data_source" "lookup" {
			foo = "from-data"
		}

		resource "test_resource" "test" {
			sensitive_value = data.test_data_source.lookup.foo
		}
	`

	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": "",
	})

	var policyEvalCalls int
	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		policyEvalCalls++
		if req.Target != "test_resource" {
			t.Fatalf("expected policy evaluation only for managed resource test_resource, got %q", req.Target)
		}

		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType: "test",
			Operation:    proto.Operation_CREATE,
		}, protocmp.Transform()); diff != "" {
			t.Errorf("Invalid resource metadata: %s", diff)
		}

		if req.Attrs.Raw.IsNull() {
			t.Fatal("expected non-null attrs for managed resource policy evaluation")
		}
		if got := req.Attrs.Raw.GetAttr("sensitive_value").AsString(); got != "from-data" {
			t.Fatalf("expected managed resource attrs to include sensitive_value=from-data, got %q", got)
		}

		return policy.EvaluationResponse{
			Overall: policy.AllowResult,
			Enforcements: []policy.EnforcementResult{{
				Result:  policy.AllowResult,
				Message: "allowed",
			}},
		}
	}

	provider := testProvider("test")
	provider.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		resp.State = req.Config
		return resp
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
		},
		Parallelism: 1,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if policyEvalCalls != 1 {
		t.Fatalf("expected exactly 1 policy evaluation call for managed resources, got %d", policyEvalCalls)
	}
}

func TestContext2Plan_PolicyEvaluation_ImportBlock(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "a" {
  id = "importable"
}

import {
  to = test_resource.a
  id = "importable"
}
`,
		"main.tfpolicy.hcl": `
resource_policy "test_resource" "policy_name" {
  enforce {
    condition = true
  }
}
`,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Required: true,
					},
					"imported_only": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		planned := req.ProposedNewState.AsValueMap()
		if !req.PriorState.IsNull() {
			if got := req.PriorState.GetAttr("id"); got.IsKnown() && !got.IsNull() {
				planned["id"] = got
			}
			if got := req.PriorState.GetAttr("imported_only"); got.IsKnown() && !got.IsNull() {
				planned["imported_only"] = got
			}
		}
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(planned),
		}
	}
	provider.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: "test_resource",
					State: cty.ObjectVal(map[string]cty.Value{
						"id":            cty.StringVal("importable"),
						"imported_only": cty.StringVal("from-import"),
					}),
				},
			},
		}
	}

	policyClient := policy.NewTestMockClient(t)
	var evalCount int
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		evalCount++
		if req.Target != "test_resource" {
			t.Fatalf("unexpected policy target %q", req.Target)
		}

		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType: "test",
			Operation:    proto.Operation_NO_OP,
		}, protocmp.Transform()); diff != "" {
			t.Errorf("Invalid resource metadata: %s", diff)
		}

		if req.Attrs.Raw.IsNull() {
			t.Fatal("expected non-null attrs for import policy evaluation")
		}
		if req.PriorAttrs.Raw.IsNull() {
			t.Fatal("expected non-null prior attrs for import policy evaluation")
		}
		if got := req.Attrs.Raw.GetAttr("imported_only"); !got.RawEquals(cty.StringVal("from-import")) {
			t.Fatalf("expected attrs.imported_only to come from imported state, got %#v", got)
		}
		if got := req.PriorAttrs.Raw.GetAttr("imported_only"); !got.RawEquals(cty.StringVal("from-import")) {
			t.Fatalf("expected prior_attrs.imported_only to come from imported state, got %#v", got)
		}

		return policy.EvaluationResponse{
			Overall: policy.AllowResult,
			Enforcements: []policy.EnforcementResult{{
				Result: policy.AllowResult,
			}},
		}
	}

	h := &testHook{}
	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 1,
		Hooks:       []Hook{h},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if !policyClient.EvaluateCalled {
		t.Fatal("expected policy evaluation to be called for import block planning")
	}

	var policyDiags tfdiags.Diagnostics
	for _, result := range h.PolicyResults {
		policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
	}
	tfdiags.AssertNoDiagnostics(t, policyDiags)
}

func TestContext2Plan_PolicyEvaluation_PartialPlan(t *testing.T) {
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

	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": samplePolicyConfig,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		planned := req.ProposedNewState.AsValueMap()
		if planned["value"].AsString() == "fail" {
			resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"plan failed",
				"simulated provider plan failure",
			))
			return resp
		}
		planned["id"] = cty.UnknownVal(cty.String)
		resp.PlannedState = cty.ObjectVal(planned)
		return resp
	}

	policyClient := policy.NewTestMockClient(t)
	evaluatedPolicyValues := map[string]struct{}{}
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		evaluatedPolicyValues[req.Attrs.Raw.GetAttr("value").AsString()] = struct{}{}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	h := &testHook{}
	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Hooks: []Hook{h},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	if !diags.HasErrors() {
		t.Fatal("expected plan to fail")
	}

	var policyDiags tfdiags.Diagnostics
	for _, result := range h.PolicyResults {
		policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
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

func TestContext2Plan_PolicyEvaluation_RefreshOnly(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}

			provider "test" {}

			resource "test_object" "a" {
				arg = "after"
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := simpleMockProvider()
	provider.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Body: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	provider.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "arg"}) {
				return cty.StringVal("current"), nil
			}
			return v, nil
		})
		if err != nil {
			t.Fatalf("ReadResourceFn transform failed: %s", err)
		}
		return providers.ReadResourceResponse{NewState: newVal}
	}
	provider.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
		return providers.UpgradeResourceStateResponse{
			UpgradedState: cty.ObjectVal(map[string]cty.Value{
				"arg": cty.StringVal("before"),
			}),
		}
	}

	policyClient := policy.NewTestMockClient(t)
	evalCount := 0
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		evalCount++
		if req.Target != "test_object" {
			t.Fatalf("expected resource policy target %q, got %q", "test_object", req.Target)
		}
		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType: "test",
			Operation:    proto.Operation_NO_OP,
		}, protocmp.Transform()); diff != "" {
			t.Fatalf("invalid resource metadata: %s", diff)
		}
		if req.Attrs.Raw.IsNull() {
			t.Fatal("expected non-null attrs for refresh-only policy evaluation")
		}
		if req.PriorAttrs.Raw.IsNull() {
			t.Fatal("expected non-null PriorAttrs for refresh-only policy evaluation")
		}
		if !req.Attrs.Raw.RawEquals(req.PriorAttrs.Raw) {
			t.Fatalf("expected refresh-only policy attrs and prior attrs to match, got attrs=%#v prior=%#v", req.Attrs.Raw, req.PriorAttrs.Raw)
		}
		if got := req.Attrs.Raw.GetAttr("arg"); !got.RawEquals(cty.StringVal("current")) {
			t.Fatalf("expected refreshed arg value %q, got %#v", "current", got)
		}
		return policy.EvaluationResponse{
			Overall: policy.AllowResult,
			Enforcements: []policy.EnforcementResult{{
				Result: policy.AllowResult,
			}},
		}
	}
	policyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) policy.EvaluationResponse {
		if req.Target != "test" {
			t.Fatalf("expected provider policy target %q, got %q", "test", req.Target)
		}
		return policy.EvaluationResponse{
			Overall: policy.AllowResult,
			Enforcements: []policy.EnforcementResult{{
				Result: policy.AllowResult,
			}},
		}
	}

	h := &testHook{}
	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Parallelism: 1,
		Hooks:       []Hook{h},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	plan, diags := ctx.Plan(mod, state, &PlanOpts{
		Mode:         plans.RefreshOnlyMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if evalCount != 1 {
		t.Fatalf("expected exactly 3 policy evaluations, got %d", evalCount)
	}
	if !policyClient.EvaluateCalled {
		t.Fatal("expected resource policy evaluation during refresh-only planning")
	}
	if !policyClient.EvaluateProviderCalled {
		t.Fatal("expected provider policy evaluation during refresh-only planning")
	}

	if got := len(plan.Changes.Resources); got != 0 {
		t.Fatalf("expected refresh-only plan to contain no resource changes, got %d", got)
	}

	if _, ok := h.PolicyResults[addr.String()]; !ok {
		t.Fatalf("expected resource policy result for %s during refresh-only planning", addr)
	}
	if _, ok := h.PolicyResults[`provider["registry.terraform.io/hashicorp/test"]`]; !ok {
		t.Fatal("expected provider policy result to be streamed through hooks")
	}
	if got := len(h.PolicyResults); got != 2 {
		t.Fatalf("expected exactly 2 policy results (resource and provider), got %d", got)
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

		resource "test_instance" "mixed" {
			count = 2
			ami = count.index == 0 ? "unknown" : "booper"
			compute = count.index == 0 ? uuid() : "known"
			depends_on = [test_instance.baz]
		}
	`

	mod := testModuleInline(t, map[string]string{
		"main.tf":           mainConfig,
		"main.tfpolicy.hcl": samplePolicyConfig,
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
		matchAllResults  []cty.Value
		filteredResults  []cty.Value
		noMatchCount     int
		nonExistentCount int
		foundUnknown     bool
	}

	var mu sync.Mutex
	results := make(map[string]callbackResult)

	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
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

		// 3. Match with an attribute filter that will never match any planned resource.
		noMatch, _, err := req.Callbacks.GetResources(t.Context(), "test_instance", cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("nonexistent"),
		}))
		if err != nil {
			t.Errorf("GetResources(test_instance, ami=nonexistent): %v", err)
		} else {
			cr.noMatchCount = len(noMatch)
		}

		// 4. Query for a resource type that doesn't exist in the config.
		nonExistentMatch, _, err := req.Callbacks.GetResources(t.Context(), "nonexistent_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources(nonexistent_resource): %v", err)
		} else {
			cr.nonExistentCount = len(nonExistentMatch)
		}

		// 5. Query for a resource type where the filtered attribute is unknown.
		_, unknown, err := req.Callbacks.GetResources(t.Context(), "test_instance", cty.ObjectVal(map[string]cty.Value{
			"compute": cty.StringVal("foo"),
		}))
		if err != nil {
			t.Errorf("Error when filtering by unknown attribute: %v", err)
		} else {
			cr.foundUnknown = unknown
		}

		// Key by the ami attribute of the resource being evaluated.
		ami := req.Attrs.Raw.GetAttr("ami").AsString()
		mu.Lock()
		results[ami] = cr
		mu.Unlock()

		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	h := &testHook{}
	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
		Hooks: []Hook{h},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	var policyDiags tfdiags.Diagnostics
	for _, result := range h.PolicyResults {
		policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
	}
	tfdiags.AssertNoDiagnostics(t, policyDiags)

	// We expect exactly 4 evaluations (one per test_instance resource).
	if len(results) != 4 {
		t.Fatalf("expected 4 policy evaluations, got %d", len(results))
	}

	for ami, cr := range results {
		expectedTotal := 4
		filteredCount := 1
		if len(cr.matchAllResults) != expectedTotal {
			t.Errorf("evaluation[%s]: expected %d result for matchAll, got %d", ami, expectedTotal, len(cr.matchAllResults))
		}

		// Filtering by ami="nonexistent" should always return 0 for all evaluations.
		if cr.noMatchCount != 0 {
			t.Errorf("evaluation[%s]: expected 0 results for ami=nonexistent filter, got %d", ami, cr.noMatchCount)
		}

		// Querying for a non-existent resource type should always return 0.
		if cr.nonExistentCount != 0 {
			t.Errorf("evaluation[%s]: expected 0 results for nonexistent_resource, got %d", ami, cr.nonExistentCount)
		}

		// Querying for a resource type where one candidate has an unknown filtered attribute
		// should report the callback result as incomplete, even if later candidates are definite non-matches.
		if !cr.foundUnknown {
			t.Errorf("evaluation[%s]: expected compute filter to report unknown=true when any candidate has an unknown value", ami)
		}

		// The filtered result should only match one resource "bar", except when evaluating "bar" itself.
		if len(cr.filteredResults) != filteredCount {
			t.Errorf("evaluation[%s]: expected filtered count %d, got %d", ami, filteredCount, len(cr.filteredResults))
		}
	}
}

func TestContext2Plan_PolicyCallback_RelatedResources(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		config      string
		pairs       []callback.RelatedAttributePair
		wantRelated []string
		wantPartial bool
	}{
		"direct traversal with pair conjunction": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "source" {
			sensitive_value = "west"
			random          = "source"
		}

		resource "test_resource" "direct" {
			value           = test_resource.source.id
			sensitive_value = test_resource.source.sensitive_value
			random          = "direct"
		}

		resource "test_resource" "mismatch" {
			value           = test_resource.source.id
			sensitive_value = "east"
			random          = "mismatch"
		}
		`,
			pairs: []callback.RelatedAttributePair{
				{SourceAttribute: "id", RelatedAttribute: "value"},
				{SourceAttribute: "sensitive_value", RelatedAttribute: "sensitive_value"},
			},
			wantRelated: []string{"direct"},
			wantPartial: false,
		},
		"literal static equality": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "source" {
			value  = "literal-source"
			random = "source"
		}

		resource "test_resource" "literal" {
			value  = "literal-source"
			random = "literal"
		}
		`,
			pairs: []callback.RelatedAttributePair{
				{SourceAttribute: "value", RelatedAttribute: "value"},
			},
			wantRelated: []string{"literal"},
			wantPartial: false,
		},
		"indirect local reference is partial": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		locals {
			source_id = test_resource.source.id
		}

		resource "test_resource" "source" {
			random = "source"
		}

		resource "test_resource" "indirect" {
			value  = local.source_id
			random = "indirect"
		}
		`,
			pairs: []callback.RelatedAttributePair{
				{SourceAttribute: "id", RelatedAttribute: "value"},
			},
			wantRelated: []string{},
			wantPartial: true,
		},
		"known match with indeterminate candidate returns partial": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		locals {
			source_id = test_resource.source.id
		}

		resource "test_resource" "source" {
			random = "source"
		}

		resource "test_resource" "direct" {
			value  = test_resource.source.id
			random = "direct"
		}

		resource "test_resource" "indirect" {
			value  = local.source_id
			random = "indirect"
		}
		`,
			pairs: []callback.RelatedAttributePair{
				{SourceAttribute: "id", RelatedAttribute: "value"},
			},
			wantRelated: []string{"direct"},
			wantPartial: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mod := testModuleInline(t, map[string]string{
				"main.tf": tc.config,
				"main.tfpolicy.hcl": `
					resource_policy "test_resource" "policy_name" {
						enforce {
							condition = true
						}
					}
				`,
			})

			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")

			policyClient := policy.NewTestMockClient(t)
			var mu sync.Mutex
			callbackCalled := false
			gotRelatedRandom := make([]string, 0)
			gotPartial := false

			policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				if req.Target != "test_resource" {
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
				if req.Attrs.Raw.IsNull() || !req.Attrs.Raw.Type().HasAttribute("random") {
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				random := req.Attrs.Raw.GetAttr("random")
				if random.IsNull() || !random.IsKnown() || random.AsString() != "source" {
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				if req.Callbacks.RelatedResources == nil {
					t.Errorf("RelatedResources callback was nil")
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				related, partial, err := req.Callbacks.RelatedResources(t.Context(), "test_resource", tc.pairs)
				if err != nil {
					t.Errorf("RelatedResources callback failed: %v", err)
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				relatedRandom := make([]string, 0, len(related))
				for _, result := range related {
					if result.Type().HasAttribute("random") {
						attr := result.GetAttr("random")
						if attr.IsKnown() && !attr.IsNull() {
							relatedRandom = append(relatedRandom, attr.AsString())
						}
					}
				}
				sort.Strings(relatedRandom)

				mu.Lock()
				callbackCalled = true
				gotPartial = partial
				gotRelatedRandom = relatedRandom
				mu.Unlock()

				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			h := &testHook{}
			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
				Hooks: []Hook{h},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: testInputValuesUnset(mod.Module.Variables),
				PolicyClient: policyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			var policyDiags tfdiags.Diagnostics
			for _, result := range h.PolicyResults {
				policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
			}
			tfdiags.AssertNoDiagnostics(t, policyDiags)

			mu.Lock()
			defer mu.Unlock()
			if !callbackCalled {
				t.Fatal("expected RelatedResources callback to be called for source resource")
			}

			wantRelatedRandom := append([]string{}, tc.wantRelated...)
			sort.Strings(wantRelatedRandom)
			if diff := cmp.Diff(wantRelatedRandom, gotRelatedRandom); diff != "" {
				t.Fatalf("unexpected related resources (-want +got):\n%s", diff)
			}
			if gotPartial != tc.wantPartial {
				t.Fatalf("unexpected partial result: got %t, want %t", gotPartial, tc.wantPartial)
			}
		})
	}
}

func TestContext2Plan_PolicyCallback_RelatedResources_KnownValuePrecedesTraversal(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		config              string
		configurePlanChange func(*testing_provider.MockProvider)
	}{
		"ignore_changes preserves prior value": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "source" {
			sensitive_value = "expected"
			random          = "source"
		}

		resource "test_resource" "candidate" {
			value  = test_resource.source.sensitive_value
			random = "candidate"

			lifecycle {
				ignore_changes = [value]
			}
		}
		`,
		},
		"provider plan preserves prior value": {
			config: `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}

		resource "test_resource" "source" {
			sensitive_value = "expected"
			random          = "source"
		}

		resource "test_resource" "candidate" {
			value  = test_resource.source.sensitive_value
			random = "candidate"
		}
		`,
			configurePlanChange: func(provider *testing_provider.MockProvider) {
				provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					return providers.PlanResourceChangeResponse{PlannedState: req.PriorState}
				}
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mod := testModuleInline(t, map[string]string{
				"main.tf": tc.config,
				"main.tfpolicy.hcl": `
					resource_policy "test_resource" "policy_name" {
						enforce {
							condition = true
						}
					}
				`,
			})

			priorState := states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.source"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"source-id","sensitive_value":"expected","random":"source"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
				ss.SetResourceInstanceCurrent(
					mustResourceInstanceAddr("test_resource.candidate"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"candidate-id","value":"stale","random":"candidate"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			})

			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			if tc.configurePlanChange != nil {
				tc.configurePlanChange(provider)
			}

			policyClient := policy.NewTestMockClient(t)
			var mu sync.Mutex
			callbackCalled := false
			gotRelatedRandom := make([]string, 0)
			gotPartial := false

			policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				if req.Target != "test_resource" {
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
				if req.Attrs.Raw.IsNull() || !req.Attrs.Raw.Type().HasAttribute("random") {
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				random := req.Attrs.Raw.GetAttr("random")
				if random.IsNull() || !random.IsKnown() || random.AsString() != "source" {
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				related, partial, err := req.Callbacks.RelatedResources(t.Context(), "test_resource", []callback.RelatedAttributePair{
					{SourceAttribute: "sensitive_value", RelatedAttribute: "value"},
				})
				if err != nil {
					t.Errorf("RelatedResources callback failed: %v", err)
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				relatedRandom := make([]string, 0, len(related))
				for _, result := range related {
					if result.Type().HasAttribute("random") {
						attr := result.GetAttr("random")
						if attr.IsKnown() && !attr.IsNull() {
							relatedRandom = append(relatedRandom, attr.AsString())
						}
					}
				}
				sort.Strings(relatedRandom)

				mu.Lock()
				callbackCalled = true
				gotRelatedRandom = relatedRandom
				gotPartial = partial
				mu.Unlock()

				return policy.EvaluationResponse{Overall: policy.AllowResult}
			}

			h := &testHook{}
			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
				Hooks: []Hook{h},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			_, diags = ctx.Plan(mod, priorState, &PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: testInputValuesUnset(mod.Module.Variables),
				PolicyClient: policyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			var policyDiags tfdiags.Diagnostics
			for _, result := range h.PolicyResults {
				policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
			}
			tfdiags.AssertNoDiagnostics(t, policyDiags)

			mu.Lock()
			defer mu.Unlock()
			if !callbackCalled {
				t.Fatal("expected RelatedResources callback to be called for source resource")
			}
			if diff := cmp.Diff([]string{}, gotRelatedRandom); diff != "" {
				t.Fatalf("unexpected related resources (-want +got):\n%s", diff)
			}
			if gotPartial {
				t.Fatal("expected full result when planned literal mismatch is known")
			}
		})
	}
}

func TestContext2Plan_PolicyCallback_GetDataSource(t *testing.T) {
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
		"getdatasource returns deferred incorrectly": {
			targetDataSource: "test_data_source",
			dataSourceReqConfig: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.NullVal(cty.String),
				"foo": cty.StringVal("test val"),
			}),
			deferralAllowed: false,
			// Returning this data would be a provider bug, but we still want to provide some information about the problem
			// to the policy engine.
			deferralResponse: &providers.Deferred{
				Reason: providers.DeferredReasonAbsentPrereq,
			},
			expectedErr: `The provider signaled a deferred action for test_data_source, ` +
				`but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.`,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockPolicyClient := policy.NewTestMockClient(t)

			var gotResults []callbackResult
			mockPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
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

			h := &testHook{}
			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider),
				},
				Hooks: []Hook{h},
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
			_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: tc.deferralAllowed,
				PolicyClient:    mockPolicyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			var policyDiags tfdiags.Diagnostics
			for _, result := range h.PolicyResults {
				policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
			}
			tfdiags.AssertNoDiagnostics(t, policyDiags)

			if diff := cmp.Diff(tc.expectedCallbackResults, gotResults, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
				t.Errorf("unexpected policy callback results\n%s", diff)
			}
		})
	}
}

func TestContext2Plan_PolicyCallback_GetDataSource_ProviderMeta(t *testing.T) {
	// This tests that we pass a null provider meta to the ReadDataSource callback
	// when the provider schema defines a provider meta block.
	// Policy callbacks do not need to report metadata to the provider.
	t.Parallel()

	providerMetaSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"baz": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}
	providerMetaType := providerMetaSchema.ImpliedType()

	mockPolicyClient := policy.NewTestMockClient(t)
	var gotResult cty.Value
	mockPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		result, deferred, err := req.Callbacks.GetDataSource(t.Context(), "test_data_source", cty.ObjectVal(map[string]cty.Value{
			"id":  cty.NullVal(cty.String),
			"foo": cty.StringVal("test val"),
		}))
		if err != nil {
			t.Fatalf("unexpected GetDataSource error: %v", err)
		}
		if deferred {
			t.Fatal("expected GetDataSource callback not to be deferred")
		}
		gotResult = result
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	testProvider := testProvider("test")
	schema := getProviderSchema(testProvider)
	schema.ProviderMeta = providerMetaSchema
	testProvider.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(schema)
	testProvider.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		if !req.ProviderMeta.IsNull() {
			t.Fatalf("expected null ProviderMeta for policy GetDataSource callback, got %#v", req.ProviderMeta)
		}
		if got := req.ProviderMeta.Type(); !got.Equals(providerMetaType) {
			t.Fatalf("unexpected ProviderMeta type: got %s, want %s", got.FriendlyName(), providerMetaType.FriendlyName())
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

	_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: mockPolicyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if !testProvider.ReadDataSourceCalled {
		t.Fatal("expected ReadDataSource to be called by policy callback")
	}
	wantResult := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("computed val"),
		"foo": cty.StringVal("test val"),
	})
	if diff := cmp.Diff(gotResult, wantResult, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
		t.Fatalf("unexpected GetDataSource result (-got +want):\n%s", diff)
	}
}

func TestContext2Plan_PolicyCallback_GetResources_Deferral(t *testing.T) {
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
							"id":              cty.UnknownVal(cty.String),
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
							"id":            cty.UnknownVal(cty.String),
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
							"type":          cty.UnknownVal(cty.String),
							"unknown":       cty.UnknownVal(cty.String),
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

			mockPolicyClient := policy.NewTestMockClient(t)
			gotResults := make([]callbackResult, 0)

			mockPolicyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
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

			h := &testHook{}
			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider("test")),
				},
				Hooks: []Hook{h},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			mod := testModuleInline(t, map[string]string{
				"main.tf":           tc.config,
				"main.tfpolicy.hcl": `# policy config is not read by Terraform`,
			})
			_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: true,
				PolicyClient:    mockPolicyClient,
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			var policyDiags tfdiags.Diagnostics
			for _, result := range h.PolicyResults {
				policyDiags = policyDiags.Append(result.Diagnostics.AsTerraformDiags())
			}
			tfdiags.AssertNoDiagnostics(t, policyDiags)

			if diff := cmp.Diff(tc.expectedCallbackResults, gotResults, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
				t.Errorf("unexpected policy callback results\n%s", diff)
			}
		})
	}
}

func TestContext2Plan_PolicyEvaluation_NoOpOperation(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}

			resource "test_resource" "test" {
				sensitive_value = "same"
			}
		`,
		"main.tfpolicy.hcl": `
			resource_policy "test_resource" "policy_name" {
				enforce {
					condition = true
				}
			}
		`,
	})

	state := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_resource.test"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"existing","sensitive_value":"same"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		planned := req.ProposedNewState.AsValueMap()
		if priorID, ok := req.PriorState.AsValueMap()["id"]; ok && priorID.IsKnown() && !priorID.IsNull() {
			planned["id"] = priorID
		}
		resp.PlannedState = cty.ObjectVal(planned)
		return resp
	}

	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType: "test",
			Operation:    proto.Operation_NO_OP,
		}, protocmp.Transform()); diff != "" {
			t.Fatalf("unexpected resource metadata (-got +want):\n%s", diff)
		}

		actualAttrs := req.Attrs.Raw
		if actualAttrs.IsNull() {
			t.Fatal("expected non-null attrs for no-op evaluation")
		}
		actualAttrs = cty.ObjectVal(map[string]cty.Value{
			"id":              actualAttrs.GetAttr("id"),
			"sensitive_value": actualAttrs.GetAttr("sensitive_value"),
		})
		wantAttrs := cty.ObjectVal(map[string]cty.Value{
			"id":              cty.StringVal("existing"),
			"sensitive_value": cty.StringVal("same"),
		})
		if diff := cmp.Diff(actualAttrs, wantAttrs, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Fatalf("unexpected attrs (-got +want):\n%s", diff)
		}

		actualPrior := req.PriorAttrs.Raw
		if actualPrior.IsNull() {
			t.Fatal("expected non-null prior attrs for no-op evaluation")
		}
		actualPrior = cty.ObjectVal(map[string]cty.Value{
			"id":              actualPrior.GetAttr("id"),
			"sensitive_value": actualPrior.GetAttr("sensitive_value"),
		})
		if diff := cmp.Diff(actualPrior, wantAttrs, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Fatalf("unexpected prior attrs (-got +want):\n%s", diff)
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

	_, diags = ctx.Plan(mod, state, &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if !policyClient.EvaluateCalled {
		t.Fatal("expected policy evaluation to be called for no-op resource")
	}
}

func TestContext2Plan_PolicyEvaluation_RefreshOnlyOperation(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}

			resource "test_resource" "test" {
				sensitive_value = "config"
			}
		`,
		"main.tfpolicy.hcl": `
			resource_policy "test_resource" "policy_name" {
				enforce {
					condition = true
				}
			}
		`,
	})

	state := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_resource.test"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"existing","sensitive_value":"stale"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		resp.NewState = cty.ObjectVal(map[string]cty.Value{
			"id":              cty.StringVal("existing"),
			"value":           cty.NullVal(cty.String),
			"sensitive_value": cty.StringVal("current"),
			"defer":           cty.NullVal(cty.Bool),
			"random":          cty.NullVal(cty.String),
			"nesting_single": cty.NullVal(cty.Object(map[string]cty.Type{
				"value":           cty.String,
				"sensitive_value": cty.String,
			})),
		})
		return resp
	}

	policyClient := policy.NewTestMockClient(t)
	var evaluateCalls int
	policyClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		evaluateCalls++
		if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType: "test",
			Operation:    proto.Operation_NO_OP,
		}, protocmp.Transform()); diff != "" {
			t.Fatalf("unexpected resource metadata (-got +want):\n%s", diff)
		}

		actualAttrs := req.Attrs.Raw
		if actualAttrs.IsNull() {
			t.Fatal("expected non-null attrs for refresh-only evaluation")
		}
		actualAttrs = cty.ObjectVal(map[string]cty.Value{
			"id":              actualAttrs.GetAttr("id"),
			"sensitive_value": actualAttrs.GetAttr("sensitive_value"),
		})
		wantAttrs := cty.ObjectVal(map[string]cty.Value{
			"id":              cty.StringVal("existing"),
			"sensitive_value": cty.StringVal("current"),
		})
		if diff := cmp.Diff(actualAttrs, wantAttrs, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Fatalf("unexpected attrs (-got +want):\n%s", diff)
		}

		actualPrior := req.PriorAttrs.Raw
		if actualPrior.IsNull() {
			t.Fatal("expected non-null prior attrs for refresh-only evaluation")
		}
		actualPrior = cty.ObjectVal(map[string]cty.Value{
			"id":              actualPrior.GetAttr("id"),
			"sensitive_value": actualPrior.GetAttr("sensitive_value"),
		})
		if diff := cmp.Diff(actualPrior, wantAttrs, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Fatalf("unexpected prior attrs (-got +want):\n%s", diff)
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
		Mode:         plans.RefreshOnlyMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		PolicyClient: policyClient,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if got := len(plan.Changes.Resources); got != 0 {
		t.Fatalf("expected refresh-only plan to record no resource changes, got %d", got)
	}
	if evaluateCalls != 1 {
		t.Fatalf("expected 1 policy evaluation call for refresh-only resource, got %d", evaluateCalls)
	}
}

func assertPathsEqual(t *testing.T, got, want []cty.Path) {
	t.Helper()

	if diff := cmp.Diff(pathStrings(want), pathStrings(got)); diff != "" {
		t.Fatalf("unexpected redacted paths (-want +got):\n%s", diff)
	}
}

func pathStrings(paths []cty.Path) []string {
	ret := make([]string, 0, len(paths))
	for _, path := range paths {
		ret = append(ret, format.CtyPath(path))
	}
	sort.Strings(ret)
	return ret
}
