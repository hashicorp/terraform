// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
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

					if diff := cmp.Diff(req.Meta, expectedMeta, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
			name:        "subject metadata carries address and provider source",
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

				resource "test_instance" "a" {
					count = 1
					value = "foo"
				}
				`,
			policyConfig: `# policy config is not read by Terraform`,
			prepareExpectations: func(t *testing.T, data *data) {
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					if diff := cmp.Diff(req.Meta, &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
						ProviderType:   "test",
						Operation:      proto.Operation_CREATE,
						ModulePath:     "module.child",
						Address:        "module.child.test_instance.a[0]",
						ProviderSource: "registry.terraform.io/hashicorp/test",
					}, protocmp.Transform()); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}
					// Without the relationships capability there is no run.
					if req.RunID != "" {
						t.Errorf("Expected empty run id, got %q", req.RunID)
					}
					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}
			},
			assertPolicyResults: func(t *testing.T, d *data) {
				if d.policy.BeginRunCalled {
					t.Error("Expected no relationship run without the capability")
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
					if diff := cmp.Diff(req.Meta, expectedMeta, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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

					if diff := cmp.Diff(req.Meta, expectedMeta, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
					}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
					}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
					}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
					}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
			name:        "replace resource with cbd. policy is evaluated with create and update operation",
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
				ops := make([]proto.Operation, 0, 2)
				data.policy.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
					data.policyEvalCalls++
					ops = append(ops, req.Meta.Operation)
					if diff := cmp.Diff(req.Meta.ProviderType, "test"); diff != "" {
						t.Errorf("Invalid resource metadata: %s", diff)
					}

					return policy.EvaluationResponse{Overall: policy.AllowResult}
				}

				t.Cleanup(func() {
					sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })
					want := []proto.Operation{proto.Operation_CREATE, proto.Operation_DELETE}
					if diff := cmp.Diff(want, ops, protocmp.Transform()); diff != "" {
						t.Errorf("wrong policy operations (-want +got):\n%s", diff)
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
					}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
	}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
				}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
		}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
		}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
		}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
	// This test asserts how policy evaluations work in partial plans.
	// Policy evaluations still need to run when some resource nodes fail,
	// and the way we do that is by tolerating failures from the policy eval node's
	// dependencies.
	// The policy eval node depends on all resource nodes.
	// However, due to transitive dependencies, there may be no dependency edges between
	// a resource node and the policy node, making the graph dependency failure tolerance insufficient.
	type testCase struct {
		config   string
		expected map[string]struct{}
	}
	configMap := map[string]testCase{

		// Here, policy node depends on both resource nodes.
		"simple config, one fails": {
			config: `terraform {
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
			`,
			expected: map[string]struct{}{"ok": {}},
		},

		// In this scenario, the policy dependency on the "ok"
		// resource is omitted in the graph due to the transitivity
		// via the "fail" resource.
		"failed resource depends on other resource": {
			config: `
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
				depends_on = [test_resource.ok]
			}
			`,
			expected: map[string]struct{}{"ok": {}},
		},
		// This is similar to the above case, but the dependency
		// goes through an intermediate object.
		"failed dependency through intermediate object": {
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
					intermediate = test_resource.ok.value
				}
		
				resource "test_resource" "ok" {
					value = "ok"
				}
		
				resource "test_resource" "fail" {
					value = "fail"
					defer = local.intermediate == "not_ok"
					
				}
				`,
			expected: map[string]struct{}{"ok": {}},
		},
		"failed resource is depended on by other resource": {
			config: `
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
						depends_on = [test_resource.fail]
					}
			
					resource "test_resource" "fail" {
						value = "fail"
					}
					`,
			expected: map[string]struct{}{},
		},
	}

	for key, testCase := range configMap {
		t.Run(key, func(t *testing.T) {
			mod := testModuleInline(t, map[string]string{
				"main.tf":           testCase.config,
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

			if diff := cmp.Diff(evaluatedPolicyValues, testCase.expected); diff != "" {
				t.Errorf("unexpected evaluated policy values: %s", diff)
			}
			if len(policyDiags) != 0 {
				t.Fatalf("expected no policy diagnostics, got %d", len(policyDiags))
			}

			if !provider.CloseCalled {
				t.Fatal("Expected provider to be closed, but it was not")
			}
		})
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
		}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
		}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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
		}, protocmp.Transform(), ignoreSubjectIdentity); diff != "" {
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

// ignoreSubjectIdentity ignores the subject's address and provider source in
// resource metadata comparisons of tests that predate those fields. They are
// asserted by the tests that are about them.
var ignoreSubjectIdentity = protocmp.IgnoreFields(&proto.PolicyEvaluateResourceRequest_ResourceMetadata{}, "address", "provider_source")

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

// relationshipsTestProvider returns a provider with the managed resource types
// test_net and test_vm and the data source test_info, for the relationship
// run tests. Computed ids are unknown when planned and set to "<name>-id"
// when applied, or "" if the name isn't set.
func relationshipsTestProvider() *testing_provider.MockProvider {
	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			}},
			ResourceTypes: map[string]providers.Schema{
				"test_net": {Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":     {Type: cty.String, Computed: true},
						"name":   {Type: cty.String, Optional: true},
						"secret": {Type: cty.String, Optional: true, Sensitive: true},
						"token":  {Type: cty.String, Optional: true, WriteOnly: true},
						"tags":   {Type: cty.Map(cty.String), Optional: true},
						// The mock provider defers a change when defer is true.
						"defer": {Type: cty.Bool, Optional: true},
					},
				}},
				"test_vm": {Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":      {Type: cty.String, Computed: true},
						"name":    {Type: cty.String, Optional: true},
						"net_id":  {Type: cty.String, Optional: true},
						"net_ids": {Type: cty.List(cty.String), Optional: true},
						"zone":    {Type: cty.String, Optional: true, Computed: true},
						"disks": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"size":     {Type: cty.Number, Optional: true},
									"password": {Type: cty.String, Optional: true, WriteOnly: true},
								},
							},
							Optional: true,
						},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"nic": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"net_id": {Type: cty.String, Optional: true},
									"key":    {Type: cty.String, Optional: true, WriteOnly: true},
								},
							},
						},
					},
				}},
			},
			DataSources: map[string]providers.Schema{
				"test_info": {Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":     {Type: cty.String, Computed: true},
						"net_id": {Type: cty.String, Optional: true},
					},
				}},
			},
		},
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{State: cty.ObjectVal(map[string]cty.Value{
			"id":     cty.StringVal("info"),
			"net_id": req.Config.GetAttr("net_id"),
		})}
	}
	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		return providers.ImportResourceStateResponse{ImportedResources: []providers.ImportedResource{{
			TypeName: req.TypeName,
			State: cty.ObjectVal(map[string]cty.Value{
				"id":     cty.StringVal(req.ID),
				"name":   cty.StringVal("imported"),
				"secret": cty.NullVal(cty.String),
				"token":  cty.NullVal(cty.String),
				"tags":   cty.NullVal(cty.Map(cty.String)),
				"defer":  cty.NullVal(cty.Bool),
			}),
		}}}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		if req.PlannedState.IsNull() {
			return providers.ApplyResourceChangeResponse{NewState: req.PlannedState}
		}
		id := ""
		if name := req.PlannedState.GetAttr("name"); name.IsKnown() && !name.IsNull() {
			id = name.AsString() + "-id"
		}
		newVal, err := cty.Transform(req.PlannedState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if v.IsKnown() {
				return v, nil
			}
			if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "id"}) {
				return cty.StringVal(id), nil
			}
			return cty.NullVal(v.Type()), nil
		})
		if err != nil {
			panic(err)
		}
		return providers.ApplyResourceChangeResponse{NewState: newVal, Private: req.PlannedPrivate}
	}
	return p
}

// relationshipRun records the relationship run RPCs and resource
// evaluations a MockClient receives, in call order.
type relationshipRun struct {
	events   []string
	begins   []*proto.BeginRunRequest
	reports  []*proto.ReportInstancesRequest
	evals    []policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]
	finishes []*proto.FinishRunRequest

	// beginErr is returned by BeginRun when set.
	beginErr error
	// reportErr is called with the 0-based index of each ReportInstances
	// call; a non-nil result is returned as the call's error.
	reportErr func(i int) error
}

// newRelationshipsPolicyClient returns a MockClient that announces the
// relationships capability and answers BeginRun with a spec for the given
// types.
func newRelationshipsPolicyClient(t *testing.T, types ...*proto.TypeSpec) (*policy.MockClient, *relationshipRun) {
	t.Helper()
	run := &relationshipRun{}
	client := policy.NewTestMockClient(t)
	client.RelationshipsSupportedResponse = true
	// The mock client calls these functions while holding its lock, so they
	// don't need any synchronization of their own.
	client.BeginRunFn = func(_ context.Context, req *proto.BeginRunRequest) (*proto.BeginRunResponse, error) {
		run.events = append(run.events, "begin")
		run.begins = append(run.begins, req)
		if run.beginErr != nil {
			return nil, run.beginErr
		}
		return &proto.BeginRunResponse{Spec: &proto.CollectionSpec{Types: types}}, nil
	}
	client.ReportInstancesFn = func(_ context.Context, req *proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error) {
		run.events = append(run.events, "report")
		run.reports = append(run.reports, req)
		if run.reportErr != nil {
			if err := run.reportErr(len(run.reports) - 1); err != nil {
				return nil, err
			}
		}
		return &proto.ReportInstancesResponse{}, nil
	}
	client.EvaluateFn = func(_ context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		run.events = append(run.events, "evaluate")
		run.evals = append(run.evals, req)
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}
	client.FinishRunFn = func(_ context.Context, req *proto.FinishRunRequest) (*proto.FinishRunResponse, error) {
		run.events = append(run.events, "finish")
		run.finishes = append(run.finishes, req)
		return &proto.FinishRunResponse{}, nil
	}
	return client, run
}

// relDefinitionDiagnostics returns BeginRun diagnostics like the policy
// engine returns for wrong relationship definitions: a warning for a
// definition that no policy uses, and an error for one that a policy uses.
func relDefinitionDiagnostics() []*proto.Diagnostic {
	return []*proto.Diagnostic{
		{
			Severity: proto.Severity_WARNING,
			Summary:  "Unknown resource type",
			Detail:   `The provider has no resource type "test_nett". No policy uses relationship "net_vms", so this is a warning. Fix or remove the relationship before a policy uses it.`,
			Subject: &proto.Range{
				Filename: "policies/net.policy.hcl",
				Start:    &proto.Position{Line: 3, Column: 14, Byte: 40},
				End:      &proto.Position{Line: 3, Column: 25, Byte: 51},
			},
			Snippet: &proto.Snippet{
				Context:              protobuf.String(`relationship "test_nett" "test_vm" "net_vms"`),
				Code:                 `relationship "test_nett" "test_vm" "net_vms" {`,
				StartLine:            3,
				HighlightStartOffset: 13,
				HighlightEndOffset:   24,
			},
			PolicySet: &proto.PolicySet{Name: "local", Path: "policies"},
		},
		{
			Severity: proto.Severity_ERROR,
			Summary:  "Unknown attribute",
			Detail:   `The resource type "test_vm" has no attribute "net".`,
			Subject: &proto.Range{
				Filename: "policies/vm.policy.hcl",
				Start:    &proto.Position{Line: 5, Column: 3, Byte: 80},
				End:      &proto.Position{Line: 5, Column: 6, Byte: 83},
			},
			PolicySet: &proto.PolicySet{Name: "local", Path: "policies"},
		},
	}
}

// assertRelDefinitionWarning checks that diags are exactly the warning of
// relDefinitionDiagnostics, with its policy information.
func assertRelDefinitionWarning(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if len(diags) != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %d: %v", len(diags), diags.ErrWithWarnings())
	}
	want := relDefinitionDiagnostics()[0]
	diag := diags[0]
	if diag.Severity() != tfdiags.Warning {
		t.Errorf("expected a warning, got %s", diag.Severity())
	}
	if desc := diag.Description(); desc.Summary != want.Summary || desc.Detail != want.Detail {
		t.Errorf("wrong description %#v", desc)
	}
	// The warning is about the policy files, not about the configuration.
	if src := diag.Source(); src.Subject != nil || src.Context != nil {
		t.Errorf("expected no configuration source, got %#v", src)
	}
	extra := tfdiags.ExtraInfo[*policy.PolicyExtra](diag)
	if extra == nil {
		t.Fatalf("expected policy extra info")
	}
	if extra.Severity != hcl.DiagWarning {
		t.Errorf("wrong policy extra severity %v", extra.Severity)
	}
	if extra.PolicySetName != "local" || extra.Directory != "policies" {
		t.Errorf("wrong policy set %q in %q", extra.PolicySetName, extra.Directory)
	}
	if extra.Range == nil || !protobuf.Equal(extra.Range.Subject, want.Subject) {
		t.Errorf("wrong policy range %v", extra.Range)
	}
	if !protobuf.Equal(extra.Snippet, want.Snippet) {
		t.Errorf("wrong policy snippet %v", extra.Snippet)
	}
}

// relTypeSpec returns a TypeSpec for a resource type of the default "test"
// provider, with dot-separated key paths.
func relTypeSpec(typeName string, keyPaths ...string) *proto.TypeSpec {
	spec := &proto.TypeSpec{
		ProviderSource: addrs.NewDefaultProvider("test").String(),
		Type:           typeName,
	}
	for _, kp := range keyPaths {
		spec.KeyPaths = append(spec.KeyPaths, relAttrPath(kp))
	}
	return spec
}

func relAttrPath(dotted string) *proto.AttributePath {
	path := &proto.AttributePath{}
	for _, name := range strings.Split(dotted, ".") {
		path.Steps = append(path.Steps, &proto.AttributePath_Step{
			Selector: &proto.AttributePath_Step_AttributeName{AttributeName: name},
		})
	}
	return path
}

func relPathString(path *proto.AttributePath) string {
	names := make([]string, 0, len(path.GetSteps()))
	for _, step := range path.GetSteps() {
		names = append(names, step.GetAttributeName())
	}
	return strings.Join(names, ".")
}

// assertRunSequence checks the sequence rules of a relationship run that
// started: one BeginRun, then the ReportInstances calls, then every resource
// evaluation, then one FinishRun, all with the same run id. It returns the
// run id.
func (r *relationshipRun) assertRunSequence(t *testing.T) string {
	t.Helper()
	if len(r.begins) != 1 {
		t.Fatalf("expected exactly 1 BeginRun call, got %d", len(r.begins))
	}
	runID := r.begins[0].RunId
	if _, err := uuid.ParseUUID(runID); err != nil {
		t.Fatalf("expected the run id to be a UUID, got %q: %s", runID, err)
	}
	if len(r.reports) == 0 {
		t.Fatal("expected at least 1 ReportInstances call")
	}
	if len(r.finishes) != 1 {
		t.Fatalf("expected exactly 1 FinishRun call, got %d", len(r.finishes))
	}
	if r.finishes[0].RunId != runID || r.finishes[0].Aborted {
		t.Fatalf("wrong FinishRun request: %v", r.finishes[0])
	}
	for i, report := range r.reports {
		if report.RunId != runID {
			t.Fatalf("ReportInstances call %d has run id %q, want %q", i, report.RunId, runID)
		}
	}
	for _, eval := range r.evals {
		if eval.RunID != runID {
			t.Fatalf("evaluation of %s has run id %q, want %q", eval.Meta.GetAddress(), eval.RunID, runID)
		}
	}

	// begin, report..., evaluate..., finish
	want := []string{"begin"}
	for range r.reports {
		want = append(want, "report")
	}
	for range r.evals {
		want = append(want, "evaluate")
	}
	want = append(want, "finish")
	if diff := cmp.Diff(want, r.events); diff != "" {
		t.Fatalf("wrong call order (-want +got):\n%s", diff)
	}
	return runID
}

// assertNoRun checks that no relationship RPC was called and that no
// evaluation carries a run id.
func (r *relationshipRun) assertNoRun(t *testing.T) {
	t.Helper()
	if len(r.begins) != 0 || len(r.reports) != 0 || len(r.finishes) != 0 {
		t.Fatalf("expected no relationship run, got %d BeginRun, %d ReportInstances and %d FinishRun calls", len(r.begins), len(r.reports), len(r.finishes))
	}
	for _, eval := range r.evals {
		if eval.RunID != "" {
			t.Fatalf("expected no run id, got %q for %s", eval.RunID, eval.Meta.GetAddress())
		}
	}
}

// records returns the records of all ReportInstances calls by address.
func (r *relationshipRun) records(t *testing.T) map[string]*proto.InstanceRecord {
	t.Helper()
	ret := make(map[string]*proto.InstanceRecord)
	for _, report := range r.reports {
		for _, rec := range report.Records {
			if _, exists := ret[rec.Address]; exists {
				t.Fatalf("duplicate record for %s", rec.Address)
			}
			ret[rec.Address] = rec
		}
	}
	return ret
}

// statuses returns the type statuses by type name, checking that they are
// all in the last ReportInstances call.
func (r *relationshipRun) statuses(t *testing.T) map[string]*proto.TypeStatus {
	t.Helper()
	ret := make(map[string]*proto.TypeStatus)
	for i, report := range r.reports {
		if len(report.Statuses) > 0 && i != len(r.reports)-1 {
			t.Fatalf("ReportInstances call %d of %d has statuses", i+1, len(r.reports))
		}
		for _, status := range report.Statuses {
			ret[status.Type] = status
		}
	}
	return ret
}

// providers returns the provider instances by id, checking that they are all
// in the first ReportInstances call and that every record's provider is
// among them.
func (r *relationshipRun) providers(t *testing.T) map[uint32]*proto.ProviderInstance {
	t.Helper()
	ret := make(map[uint32]*proto.ProviderInstance)
	for i, report := range r.reports {
		if len(report.Providers) > 0 && i != 0 {
			t.Fatalf("ReportInstances call %d has providers", i+1)
		}
		for _, p := range report.Providers {
			ret[p.Id] = p
		}
	}
	for addr, rec := range r.records(t) {
		if _, ok := ret[rec.ProviderInstanceId]; !ok {
			t.Fatalf("record %s refers to provider instance %d, which wasn't reported", addr, rec.ProviderInstanceId)
		}
	}
	return ret
}

func TestContext2Plan_PolicyRelationships_noCapability(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_net" "a" {
				name = "a"
			}
		`,
	})
	client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
	client.RelationshipsSupportedResponse = false

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(relationshipsTestProvider()),
		},
	})
	_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		PolicyClient: client,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	run.assertNoRun(t)
	if len(run.evals) != 1 {
		t.Fatalf("expected 1 resource evaluation, got %d", len(run.evals))
	}
}

func TestContext2Plan_PolicyRelationships_beginRun(t *testing.T) {
	netAddr := mustResourceInstanceAddr("test_net.a")
	priorState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(netAddr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"a-id","name":"a"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	tests := map[string]struct {
		opts         *PlanOpts
		state        *states.State
		wantMode     proto.PlanMode
		wantTargeted bool
	}{
		"normal": {
			opts:     &PlanOpts{Mode: plans.NormalMode},
			wantMode: proto.PlanMode_NORMAL_PLAN_MODE,
		},
		"destroy": {
			opts:     &PlanOpts{Mode: plans.DestroyMode},
			state:    priorState,
			wantMode: proto.PlanMode_DESTROY_PLAN_MODE,
		},
		"refresh-only": {
			opts:     &PlanOpts{Mode: plans.RefreshOnlyMode},
			state:    priorState,
			wantMode: proto.PlanMode_REFRESH_ONLY_PLAN_MODE,
		},
		"targeted": {
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{netAddr.ContainingResource()},
			},
			wantMode:     proto.PlanMode_NORMAL_PLAN_MODE,
			wantTargeted: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						name = "a"
					}
					resource "test_vm" "b" {
						net_id = test_net.a.id
					}
				`,
			})
			state := test.state
			if state == nil {
				state = states.NewState()
			}
			client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(relationshipsTestProvider()),
				},
			})
			test.opts.PolicyClient = client
			_, diags := ctx.Plan(mod, state, test.opts)
			tfdiags.AssertNoErrors(t, diags)

			run.assertRunSequence(t)
			if len(run.evals) == 0 {
				t.Fatal("expected resource evaluations")
			}
			begin := run.begins[0]
			if begin.Stage != proto.EvaluationStage_PLAN_EVALUATION_STAGE {
				t.Errorf("wrong stage %s", begin.Stage)
			}
			if begin.PlanMode != test.wantMode {
				t.Errorf("wrong plan mode %s, want %s", begin.PlanMode, test.wantMode)
			}
			if begin.Runtime != proto.RunRuntime_CLI_RUN_RUNTIME {
				t.Errorf("wrong runtime %s", begin.Runtime)
			}
			if begin.Targeted != test.wantTargeted {
				t.Errorf("wrong targeted %t, want %t", begin.Targeted, test.wantTargeted)
			}
			assertRelationshipsTestSchemas(t, begin.ProviderSchemas)
		})
	}
}

// assertRelationshipsTestSchemas checks that the given provider schemas are
// those of relationshipsTestProvider.
func assertRelationshipsTestSchemas(t *testing.T, got []*proto.ProviderSchema) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("expected 1 provider schema, got %d", len(got))
	}
	schema := got[0]
	if schema.Type != "test" || schema.Source != "registry.terraform.io/hashicorp/test" {
		t.Fatalf("wrong provider schema type %q and source %q", schema.Type, schema.Source)
	}
	want := relationshipsTestProvider().GetProviderSchemaResponse
	for name, rs := range want.ResourceTypes {
		ty, err := ctyjson.UnmarshalType(schema.Resources[name])
		if err != nil {
			t.Fatalf("invalid type for %s: %s", name, err)
		}
		if !ty.Equals(rs.Body.ImpliedType()) {
			t.Fatalf("wrong type for %s: %#v", name, ty)
		}
	}
	if len(schema.Resources) != len(want.ResourceTypes) {
		t.Fatalf("expected %d resource types, got %d", len(want.ResourceTypes), len(schema.Resources))
	}
	if _, ok := schema.DataSources["test_info"]; !ok || len(schema.DataSources) != 1 {
		t.Fatalf("expected the test_info data source, got %v", schema.DataSources)
	}
	gotWriteOnly := make(map[string][]string)
	for name, paths := range schema.WriteOnlyPaths {
		for _, path := range paths.Paths {
			gotWriteOnly[name] = append(gotWriteOnly[name], relPathString(path))
		}
	}
	wantWriteOnly := map[string][]string{
		"test_net": {"token"},
		"test_vm":  {"disks.password", "nic.key"},
	}
	if diff := cmp.Diff(wantWriteOnly, gotWriteOnly); diff != "" {
		t.Fatalf("wrong write-only paths (-want +got):\n%s", diff)
	}
}

func TestContext2Plan_PolicyRelationships_queryHasNoRun(t *testing.T) {
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
		`,
		"main.tfquery.hcl": `
			list "test_resource" "test1" {
				provider = test
				include_resource = true

				config {
					filter = {
						attr = "foo"
					}
				}
			}
		`,
	}, configs.MatchQueryFiles())

	provider := testProvider("test")
	provider.GetProviderSchemaResponse = getListProviderSchemaResp()
	provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
		return providers.ListResourceResponse{Result: cty.ObjectVal(map[string]cty.Value{
			"data": cty.TupleVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"identity":     cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")}),
				"display_name": cty.StringVal("Instance 1"),
				"state":        cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-1")}),
			})}),
			"config": request.Config.GetAttr("config"),
		})}
	}

	client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_resource", "id"))
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
		},
	})
	_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		Query:        true,
		PolicyClient: client,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	run.assertNoRun(t)
	if len(run.evals) == 0 {
		t.Fatal("expected the query results to be evaluated")
	}
}

func TestContext2Plan_PolicyRelationships_providers(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
			provider "test" {
				region = "us-east-1"
			}
			provider "test" {
				alias  = "same"
				region = "us-east-1"
			}
			provider "test" {
				alias  = "other"
				region = "eu-west-1"
			}
			provider "test" {
				alias  = "unknown"
				region = test_net.a.id
			}

			resource "test_net" "a" {
				name = "a"
			}
			resource "test_net" "b" {
				provider = test.same
				name     = "b"
			}
			resource "test_net" "c" {
				provider = test.other
				name     = "c"
			}
			resource "test_net" "d" {
				provider = test.unknown
				name     = "d"
			}
		`,
	})
	client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(relationshipsTestProvider()),
		},
	})
	_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		PolicyClient: client,
	})
	tfdiags.AssertNoDiagnostics(t, diags)
	run.assertRunSequence(t)

	byAddr := make(map[string]*proto.ProviderInstance)
	for _, p := range run.providers(t) {
		byAddr[p.ConfigAddress] = p
	}
	def := byAddr[`provider["registry.terraform.io/hashicorp/test"]`]
	same := byAddr[`provider["registry.terraform.io/hashicorp/test"].same`]
	other := byAddr[`provider["registry.terraform.io/hashicorp/test"].other`]
	unknown := byAddr[`provider["registry.terraform.io/hashicorp/test"].unknown`]
	if def == nil || same == nil || other == nil || unknown == nil {
		t.Fatalf("missing provider instances, got %v", byAddr)
	}
	for _, p := range []*proto.ProviderInstance{def, same, other} {
		if !p.Known || len(p.ConfigClass) != 32 {
			t.Fatalf("expected %s to be known with a class, got %v", p.ConfigAddress, p)
		}
		if p.Source != "registry.terraform.io/hashicorp/test" {
			t.Fatalf("wrong source for %s: %s", p.ConfigAddress, p.Source)
		}
	}
	if !bytes.Equal(def.ConfigClass, same.ConfigClass) {
		t.Error("expected identical provider configurations to have the same class")
	}
	if bytes.Equal(def.ConfigClass, other.ConfigClass) {
		t.Error("expected different provider configurations to have different classes")
	}
	if unknown.Known || len(unknown.ConfigClass) != 0 {
		t.Errorf("expected the provider configuration with an unknown value not to be known, got %v", unknown)
	}

	wantProviders := map[string]*proto.ProviderInstance{
		"test_net.a": def,
		"test_net.b": same,
		"test_net.c": other,
		"test_net.d": unknown,
	}
	records := run.records(t)
	for addr, want := range wantProviders {
		rec, ok := records[addr]
		if !ok {
			t.Errorf("missing record for %s", addr)
			continue
		}
		if rec.ProviderInstanceId != want.Id {
			t.Errorf("wrong provider for %s: got id %d, want %d (%s)", addr, rec.ProviderInstanceId, want.Id, want.ConfigAddress)
		}
	}
}

// planRelationships plans mod with a policy client that announces the
// relationships capability and answers BeginRun with a spec of the given
// types. It fails the test if the plan has errors.
func planRelationships(t *testing.T, mod *configs.Config, state *states.State, opts *PlanOpts, provider *testing_provider.MockProvider, types ...*proto.TypeSpec) (*plans.Plan, *relationshipRun) {
	t.Helper()
	client, run := newRelationshipsPolicyClient(t, types...)
	if provider == nil {
		provider = relationshipsTestProvider()
	}
	if state == nil {
		state = states.NewState()
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
		},
	})
	opts.PolicyClient = client
	if opts.SetVariables == nil {
		opts.SetVariables = testInputValuesUnset(mod.Module.Variables)
	}
	plan, diags := ctx.Plan(mod, state, opts)
	tfdiags.AssertNoErrors(t, diags)
	return plan, run
}

// wantRelRecord describes an expected InstanceRecord. Attrs and PriorAttrs
// list expected attribute values; nil means the record must not have them.
type wantRelRecord struct {
	Action      proto.ResourceAction
	Source      proto.RecordSource
	Attrs       map[string]cty.Value
	PriorAttrs  map[string]cty.Value
	Redacted    []string // redacted paths of attrs, dot-separated; not checked when nil
	Importing   bool
	PrevAddress string
	ModulePath  string
	// Provider is the config address of the record's provider instance;
	// the default provider configuration when empty.
	Provider string
	// Origins are the expected origins by dot-separated key path, each as
	// the referenced address and dot-separated path; nil means none.
	Origins map[string][]string
}

// relOrigins returns the origins of a record by dot-separated key path, each
// as the referenced address and dot-separated path, or nil if there are none.
func relOrigins(t *testing.T, rec *proto.InstanceRecord) map[string][]string {
	t.Helper()
	if len(rec.Origins) == 0 {
		return nil
	}
	ret := make(map[string][]string)
	for _, ko := range rec.Origins {
		kp := relPathString(ko.KeyPath)
		if _, exists := ret[kp]; exists {
			t.Fatalf("%s: duplicate origins for key path %s", rec.Address, kp)
		}
		if len(ko.Origins) == 0 {
			t.Fatalf("%s: empty origins for key path %s", rec.Address, kp)
		}
		for _, o := range ko.Origins {
			ret[kp] = append(ret[kp], o.Address+"."+relPathString(o.Path))
		}
		sort.Strings(ret[kp])
	}
	return ret
}

func relDecodeAttrs(t *testing.T, attrs *proto.ResourceAttributes) cty.Value {
	t.Helper()
	val, err := ctymsgpack.Unmarshal(attrs.Raw, cty.DynamicPseudoType)
	if err != nil {
		t.Fatalf("invalid attrs: %s", err)
	}
	return val
}

func relAssertAttrs(t *testing.T, what string, got *proto.ResourceAttributes, want map[string]cty.Value) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("%s: expected no value, got %#v", what, relDecodeAttrs(t, got))
		}
		return
	}
	if got == nil {
		t.Fatalf("%s: expected a value, got none", what)
	}
	val := relDecodeAttrs(t, got)
	for name, wantVal := range want {
		if !val.Type().IsObjectType() || !val.Type().HasAttribute(name) {
			t.Fatalf("%s: value has no attribute %q: %#v", what, name, val)
		}
		gotVal := val.GetAttr(name)
		if !wantVal.IsKnown() {
			if gotVal.IsKnown() || !gotVal.Type().Equals(wantVal.Type()) {
				t.Fatalf("%s: expected %s to be unknown, got %#v", what, name, gotVal)
			}
			continue
		}
		if !gotVal.RawEquals(wantVal) {
			t.Fatalf("%s: wrong value for %s\ngot:  %#v\nwant: %#v", what, name, gotVal, wantVal)
		}
	}
}

func relRedactedPaths(attrs *proto.ResourceAttributes) []string {
	var ret []string
	for _, path := range attrs.GetRedactedPaths() {
		var parts []string
		for _, step := range path.Steps {
			switch s := step.Selector.(type) {
			case *proto.AttributePath_Step_AttributeName:
				parts = append(parts, s.AttributeName)
			case *proto.AttributePath_Step_ElementKeyInt:
				parts = append(parts, fmt.Sprint(s.ElementKeyInt))
			case *proto.AttributePath_Step_ElementKeyString:
				parts = append(parts, s.ElementKeyString)
			}
		}
		ret = append(ret, strings.Join(parts, "."))
	}
	sort.Strings(ret)
	return ret
}

// assertRelRecords checks that the run reported exactly the wanted records.
func (r *relationshipRun) assertRecords(t *testing.T, want map[string]wantRelRecord) {
	t.Helper()
	got := r.records(t)
	providers := r.providers(t)
	var gotAddrs, wantAddrs []string
	for addr := range got {
		gotAddrs = append(gotAddrs, addr)
	}
	for addr := range want {
		wantAddrs = append(wantAddrs, addr)
	}
	sort.Strings(gotAddrs)
	sort.Strings(wantAddrs)
	if diff := cmp.Diff(wantAddrs, gotAddrs); diff != "" {
		t.Fatalf("wrong records (-want +got):\n%s", diff)
	}

	for addr, w := range want {
		rec := got[addr]
		instAddr := mustResourceInstanceAddr(addr)
		if rec.Type != instAddr.Resource.Resource.Type {
			t.Errorf("%s: wrong type %q", addr, rec.Type)
		}
		if rec.ProviderSource != "registry.terraform.io/hashicorp/test" {
			t.Errorf("%s: wrong provider source %q", addr, rec.ProviderSource)
		}
		if rec.ModulePath != w.ModulePath {
			t.Errorf("%s: wrong module path %q, want %q", addr, rec.ModulePath, w.ModulePath)
		}
		if rec.Action != w.Action {
			t.Errorf("%s: wrong action %s, want %s", addr, rec.Action, w.Action)
		}
		if rec.Source != w.Source {
			t.Errorf("%s: wrong source %s, want %s", addr, rec.Source, w.Source)
		}
		if rec.Importing != w.Importing {
			t.Errorf("%s: wrong importing %t, want %t", addr, rec.Importing, w.Importing)
		}
		if rec.PrevAddress != w.PrevAddress {
			t.Errorf("%s: wrong prev address %q, want %q", addr, rec.PrevAddress, w.PrevAddress)
		}
		wantProvider := w.Provider
		if wantProvider == "" {
			wantProvider = `provider["registry.terraform.io/hashicorp/test"]`
		}
		if p := providers[rec.ProviderInstanceId]; p == nil || p.ConfigAddress != wantProvider {
			t.Errorf("%s: wrong provider instance %d (%v), want %s", addr, rec.ProviderInstanceId, p, wantProvider)
		}
		if diff := cmp.Diff(w.Origins, relOrigins(t, rec)); diff != "" {
			t.Errorf("%s: wrong origins (-want +got):\n%s", addr, diff)
		}
		relAssertAttrs(t, addr+" attrs", rec.Attrs, w.Attrs)
		relAssertAttrs(t, addr+" prior attrs", rec.PriorAttrs, w.PriorAttrs)
		if rec.Attrs != nil && w.Redacted != nil {
			if diff := cmp.Diff(w.Redacted, relRedactedPaths(rec.Attrs)); diff != "" {
				t.Errorf("%s: wrong redacted paths (-want +got):\n%s", addr, diff)
			}
		}
	}
}

// assertStatuses checks the type statuses of the run. Deferred addresses
// are compared exactly.
func (r *relationshipRun) assertStatuses(t *testing.T, want map[string]*proto.TypeStatus) {
	t.Helper()
	got := r.statuses(t)
	for _, status := range want {
		if status.ProviderSource == "" {
			status.ProviderSource = "registry.terraform.io/hashicorp/test"
		}
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatalf("wrong statuses (-want +got):\n%s", diff)
	}
}

func relComplete(typeName string) *proto.TypeStatus {
	return &proto.TypeStatus{Type: typeName, Completeness: proto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS}
}

func relIncompleteError(typeName string) *proto.TypeStatus {
	return &proto.TypeStatus{Type: typeName, Completeness: proto.TypeCompleteness_INCOMPLETE_ERROR_TYPE_COMPLETENESS}
}

func relIncompleteDeferred(typeName string, addrs ...string) *proto.TypeStatus {
	return &proto.TypeStatus{Type: typeName, Completeness: proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS, DeferredAddresses: addrs}
}

func relNetState(s *states.SyncState, addr, attrsJSON string) {
	s.SetResourceInstanceCurrent(mustResourceInstanceAddr(addr), &states.ResourceInstanceObjectSrc{
		AttrsJSON: []byte(attrsJSON),
		Status:    states.ObjectReady,
	}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
}

var (
	relCreate  = proto.ResourceAction_CREATE_RESOURCE_ACTION
	relUpdate  = proto.ResourceAction_UPDATE_RESOURCE_ACTION
	relNoOp    = proto.ResourceAction_NO_OP_RESOURCE_ACTION
	relDelete  = proto.ResourceAction_DELETE_RESOURCE_ACTION
	relForget  = proto.ResourceAction_FORGET_RESOURCE_ACTION
	relPlanned = proto.RecordSource_PLANNED_RECORD_SOURCE
	relState   = proto.RecordSource_STATE_RECORD_SOURCE
	relUnknown = cty.UnknownVal(cty.String)
)

func TestContext2Plan_PolicyRelationships_records(t *testing.T) {
	tests := map[string]struct {
		files        map[string]string
		state        func(*states.SyncState)
		opts         *PlanOpts
		types        []*proto.TypeSpec
		wantRecords  map[string]wantRelRecord
		wantStatuses map[string]*proto.TypeStatus
	}{
		"create": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "b" {
					net_id = test_net.a.id
				}
			`},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id"), relTypeSpec("test_vm", "net_id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{
					"id": relUnknown, "name": cty.StringVal("a"),
				}},
				"test_vm.b": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{
					"id": relUnknown, "net_id": relUnknown,
				}, Origins: map[string][]string{
					"net_id": {"test_net.a.id"},
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{
				"test_net": relComplete("test_net"),
				"test_vm":  relComplete("test_vm"),
			},
		},
		"update and no-op": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "new"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
				relNetState(s, "test_net.b", `{"id":"b-id","name":"old"}`)
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relNoOp, Source: relPlanned, Attrs: map[string]cty.Value{
					"id": cty.StringVal("a-id"), "name": cty.StringVal("a"),
				}},
				"test_net.b": {Action: relUpdate, Source: relPlanned, Attrs: map[string]cty.Value{
					"id": cty.StringVal("b-id"), "name": cty.StringVal("new"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"replace": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
					lifecycle {
						create_before_destroy = true
					}
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
				relNetState(s, "test_net.b", `{"id":"b-id","name":"b"}`)
			},
			opts: &PlanOpts{
				Mode: plans.NormalMode,
				ForceReplace: []addrs.AbsResourceInstance{
					mustResourceInstanceAddr("test_net.a"),
					mustResourceInstanceAddr("test_net.b"),
				},
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {
					Action:     proto.ResourceAction_DELETE_THEN_CREATE_RESOURCE_ACTION,
					Source:     relPlanned,
					Attrs:      map[string]cty.Value{"id": relUnknown, "name": cty.StringVal("a")},
					PriorAttrs: map[string]cty.Value{"id": cty.StringVal("a-id"), "name": cty.StringVal("a")},
				},
				"test_net.b": {
					Action:     proto.ResourceAction_CREATE_THEN_DELETE_RESOURCE_ACTION,
					Source:     relPlanned,
					Attrs:      map[string]cty.Value{"id": relUnknown, "name": cty.StringVal("b")},
					PriorAttrs: map[string]cty.Value{"id": cty.StringVal("b-id"), "name": cty.StringVal("b")},
				},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"orphan": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
				relNetState(s, "test_net.gone", `{"id":"gone-id","name":"gone"}`)
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relNoOp, Source: relPlanned, Attrs: map[string]cty.Value{"id": cty.StringVal("a-id")}},
				"test_net.gone": {Action: relDelete, Source: relPlanned, PriorAttrs: map[string]cty.Value{
					"id": cty.StringVal("gone-id"), "name": cty.StringVal("gone"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"forget": {
			files: map[string]string{"main.tf": `
				removed {
					from = test_net.f
					lifecycle {
						destroy = false
					}
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.f", `{"id":"f-id","name":"f"}`)
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.f": {Action: relForget, Source: relPlanned, PriorAttrs: map[string]cty.Value{
					"id": cty.StringVal("f-id"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"import": {
			files: map[string]string{"main.tf": `
				import {
					to = test_net.i
					id = "i-id"
				}
				resource "test_net" "i" {
					name = "imported"
				}
			`},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.i": {Action: relNoOp, Source: relPlanned, Importing: true, Attrs: map[string]cty.Value{
					"id": cty.StringVal("i-id"), "name": cty.StringVal("imported"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"moved": {
			files: map[string]string{"main.tf": `
				moved {
					from = test_net.old
					to   = test_net.new
				}
				resource "test_net" "new" {
					name = "a"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.old", `{"id":"a-id","name":"a"}`)
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.new": {Action: relNoOp, Source: relPlanned, PrevAddress: "test_net.old", Attrs: map[string]cty.Value{
					"id": cty.StringVal("a-id"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"modules and expansion": {
			files: map[string]string{
				"main.tf": `
					module "child" {
						source = "./child"
						count  = 2
					}
				`,
				"child/main.tf": `
					resource "test_net" "a" {
						for_each = toset(["x", "y"])
						name     = each.key
					}
				`,
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				`module.child[0].test_net.a["x"]`: {Action: relCreate, Source: relPlanned, ModulePath: "module.child[0]", Attrs: map[string]cty.Value{"name": cty.StringVal("x")}},
				`module.child[0].test_net.a["y"]`: {Action: relCreate, Source: relPlanned, ModulePath: "module.child[0]", Attrs: map[string]cty.Value{"name": cty.StringVal("y")}},
				`module.child[1].test_net.a["x"]`: {Action: relCreate, Source: relPlanned, ModulePath: "module.child[1]", Attrs: map[string]cty.Value{"name": cty.StringVal("x")}},
				`module.child[1].test_net.a["y"]`: {Action: relCreate, Source: relPlanned, ModulePath: "module.child[1]", Attrs: map[string]cty.Value{"name": cty.StringVal("y")}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"types not in the spec, data sources and types without instances": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "b" {
					name = "b"
				}
				data "test_info" "c" {
					net_id = "x"
				}
			`},
			types: []*proto.TypeSpec{
				relTypeSpec("test_net", "id"),
				relTypeSpec("test_info", "net_id"),
				relTypeSpec("test_missing", "id"),
			},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{"name": cty.StringVal("a")}},
			},
			wantStatuses: map[string]*proto.TypeStatus{
				"test_net":     relComplete("test_net"),
				"test_info":    relComplete("test_info"),
				"test_missing": relComplete("test_missing"),
			},
		},
		"sensitive values": {
			files: map[string]string{"main.tf": `
				variable "name" {
					type      = string
					sensitive = true
					default   = "hidden"
				}
				resource "test_net" "a" {
					name   = var.name
					secret = "s"
				}
			`},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "name")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {
					Action:   relCreate,
					Source:   relPlanned,
					Attrs:    map[string]cty.Value{"name": cty.StringVal("hidden"), "secret": cty.StringVal("s")},
					Redacted: []string{"name", "secret"},
				},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"sensitive values in state": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name   = "a"
					secret = "s"
				}
				resource "test_net" "b" {
					name = "b"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a","secret":"s"}`)
			},
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_net.b")},
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "name")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {
					Action:   relNoOp,
					Source:   relState,
					Attrs:    map[string]cty.Value{"id": cty.StringVal("a-id"), "secret": cty.StringVal("s")},
					Redacted: []string{"secret"},
				},
				"test_net.b": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{"name": cty.StringVal("b")}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"targeted": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b-new"
				}
				resource "test_net" "c" {
					name = "c"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.b", `{"id":"b-id","name":"b"}`)
				relNetState(s, "test_net.gone", `{"id":"gone-id","name":"gone"}`)
			},
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_net.a")},
			},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{"id": relUnknown}},
				// Outside the target: the state value, not the configuration.
				"test_net.b": {Action: relNoOp, Source: relState, Attrs: map[string]cty.Value{
					"id": cty.StringVal("b-id"), "name": cty.StringVal("b"),
				}},
				// test_net.gone isn't in the configuration and test_net.c has
				// no object, so neither has a record.
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"refresh-only": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a-new"
				}
				resource "test_net" "b" {
					name = "b"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
			},
			opts:  &PlanOpts{Mode: plans.RefreshOnlyMode},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relNoOp, Source: relState, Attrs: map[string]cty.Value{
					"id": cty.StringVal("a-id"), "name": cty.StringVal("a-refreshed"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{"test_net": relComplete("test_net")},
		},
		"destroy": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "b" {
					net_id = test_net.a.id
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_vm.b"), &states.ResourceInstanceObjectSrc{
					AttrsJSON:    []byte(`{"id":"b-id","net_id":"a-id"}`),
					Status:       states.ObjectReady,
					Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_net.a")},
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			},
			opts:  &PlanOpts{Mode: plans.DestroyMode},
			types: []*proto.TypeSpec{relTypeSpec("test_net", "id"), relTypeSpec("test_vm", "net_id")},
			wantRecords: map[string]wantRelRecord{
				"test_net.a": {Action: relDelete, Source: relPlanned, PriorAttrs: map[string]cty.Value{"id": cty.StringVal("a-id")}},
				"test_vm.b": {Action: relDelete, Source: relPlanned, PriorAttrs: map[string]cty.Value{
					"id": cty.StringVal("b-id"), "net_id": cty.StringVal("a-id"),
				}},
			},
			wantStatuses: map[string]*proto.TypeStatus{
				"test_net": relComplete("test_net"),
				"test_vm":  relComplete("test_vm"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, test.files)
			state := states.NewState()
			if test.state != nil {
				state = states.BuildState(test.state)
			}
			opts := test.opts
			if opts == nil {
				opts = &PlanOpts{Mode: plans.NormalMode}
			}
			provider := relationshipsTestProvider()
			provider.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
				if opts.Mode != plans.RefreshOnlyMode {
					return providers.ReadResourceResponse{NewState: req.PriorState}
				}
				// Refresh-only plans see a changed name, so their
				// records must have the refreshed value.
				newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
					if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "name"}) && v.IsKnown() && !v.IsNull() {
						return cty.StringVal(v.AsString() + "-refreshed"), nil
					}
					return v, nil
				})
				if err != nil {
					t.Fatal(err)
				}
				return providers.ReadResourceResponse{NewState: newVal}
			}
			_, run := planRelationships(t, mod, state, opts, provider, test.types...)

			run.assertRunSequence(t)
			run.assertRecords(t, test.wantRecords)
			run.assertStatuses(t, test.wantStatuses)
		})
	}
}

func TestContext2Plan_PolicyRelationships_deferrals(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		# Deferred directly.
		resource "test_net" "deferred" {
			name  = "deferred"
			defer = true
		}

		# Partially expanded, because its for_each isn't known.
		resource "test_net" "partial" {
			for_each = toset([test_vm.a.id])
			name     = each.key
		}

		# Deferred because it depends on a deferred resource.
		resource "test_vm" "dependent" {
			net_id = test_net.deferred.id
		}

		resource "test_vm" "a" {
			name = "a"
		}

		resource "test_net" "existing" {
			name  = "existing"
			defer = true
		}

		data "test_info" "deferred" {
			net_id = test_net.deferred.id
		}
	`})
	state := states.BuildState(func(s *states.SyncState) {
		relNetState(s, "test_net.existing", `{"id":"existing-id","name":"existing","defer":true}`)
	})
	_, run := planRelationships(t, mod, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
	}, nil, relTypeSpec("test_net", "id"), relTypeSpec("test_vm", "net_id"), relTypeSpec("test_info", "net_id"))

	run.assertRunSequence(t)
	run.assertRecords(t, map[string]wantRelRecord{
		// Deferred instances have no records, even if they have a state
		// object.
		"test_vm.a": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{"name": cty.StringVal("a")}},
	})
	run.assertStatuses(t, map[string]*proto.TypeStatus{
		"test_net":  relIncompleteDeferred("test_net", "test_net.deferred", "test_net.existing", "test_net.partial[*]"),
		"test_vm":   relIncompleteDeferred("test_vm", "test_vm.dependent"),
		"test_info": relComplete("test_info"),
	})
}

func TestContext2Plan_PolicyRelationships_incomplete(t *testing.T) {
	t.Run("planning an instance failed", func(t *testing.T) {
		mod := testModuleInline(t, map[string]string{"main.tf": `
			resource "test_net" "ok" {
				name = "ok"
			}
			resource "test_net" "bad" {
				name = "bad"
			}
			resource "test_vm" "ok" {
				name = "ok"
			}
		`})
		provider := relationshipsTestProvider()
		provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			if name := req.Config.GetAttr("name"); name.IsKnown() && name.AsString() == "bad" {
				var resp providers.PlanResourceChangeResponse
				resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("planning failed"))
				return resp
			}
			return testDiffFn(req)
		}
		client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"), relTypeSpec("test_vm", "net_id"))
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
			},
		})
		_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
			Mode:         plans.NormalMode,
			PolicyClient: client,
		})
		if !diags.HasErrors() {
			t.Fatal("expected the plan to fail")
		}

		run.assertRunSequence(t)
		records := run.records(t)
		if _, ok := records["test_net.bad"]; ok {
			t.Fatal("expected no record for the instance that failed to plan")
		}
		if _, ok := records["test_net.ok"]; !ok {
			t.Fatal("expected a record for test_net.ok")
		}
		run.assertStatuses(t, map[string]*proto.TypeStatus{
			"test_net": relIncompleteError("test_net"),
			"test_vm":  relComplete("test_vm"),
		})
	})

	t.Run("state object with another schema version", func(t *testing.T) {
		mod := testModuleInline(t, map[string]string{"main.tf": `
			resource "test_net" "a" {
				name = "a"
			}
			resource "test_net" "old" {
				name = "old"
			}
			resource "test_vm" "b" {
				name = "b"
			}
		`})
		state := states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_net.old"), &states.ResourceInstanceObjectSrc{
				AttrsJSON:     []byte(`{"id":"old-id","name":"old"}`),
				SchemaVersion: 1,
				Status:        states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		})
		_, run := planRelationships(t, mod, state, &PlanOpts{
			Mode:    plans.NormalMode,
			Targets: []addrs.Targetable{mustResourceInstanceAddr("test_net.a")},
		}, nil, relTypeSpec("test_net", "id"), relTypeSpec("test_vm", "net_id"))

		run.assertRunSequence(t)
		run.assertRecords(t, map[string]wantRelRecord{
			"test_net.a": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{"name": cty.StringVal("a")}},
		})
		run.assertStatuses(t, map[string]*proto.TypeStatus{
			"test_net": relIncompleteError("test_net"),
			"test_vm":  relComplete("test_vm"),
		})
	})
}

func TestContext2Plan_PolicyRelationships_chunks(t *testing.T) {
	oldMax := relationshipChunkMaxRecords
	relationshipChunkMaxRecords = 2
	t.Cleanup(func() { relationshipChunkMaxRecords = oldMax })

	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_net" "a" {
			count = 5
			name  = "a${count.index}"
		}
	`})
	_, run := planRelationships(t, mod, nil, &PlanOpts{Mode: plans.NormalMode}, nil, relTypeSpec("test_net", "id"))

	// All chunks are reported before the first evaluation.
	run.assertRunSequence(t)
	if len(run.reports) != 3 {
		t.Fatalf("expected 3 ReportInstances calls, got %d", len(run.reports))
	}
	for i, report := range run.reports {
		if len(report.Records) > 2 {
			t.Errorf("ReportInstances call %d has %d records", i+1, len(report.Records))
		}
	}
	if got := len(run.records(t)); got != 5 {
		t.Errorf("expected 5 records, got %d", got)
	}
	if got := len(run.providers(t)); got != 1 {
		t.Errorf("expected 1 provider instance, got %d", got)
	}
	run.assertStatuses(t, map[string]*proto.TypeStatus{
		"test_net": relComplete("test_net"),
	})
	if len(run.evals) != 5 {
		t.Errorf("expected 5 evaluations, got %d", len(run.evals))
	}
}

func TestContext2Plan_PolicyRelationships_rpcErrors(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_net" "a" {
			count = 3
			name  = "a${count.index}"
		}
	`})
	plan := func(t *testing.T, client policy.Client) {
		t.Helper()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(relationshipsTestProvider()),
			},
		})
		_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
			Mode:         plans.NormalMode,
			PolicyClient: client,
		})
		// Relationship RPC failures never add diagnostics.
		tfdiags.AssertNoDiagnostics(t, diags)
	}

	t.Run("BeginRun fails", func(t *testing.T) {
		client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
		run.beginErr = fmt.Errorf("BeginRun failed")
		plan(t, client)

		if len(run.begins) != 1 {
			t.Fatalf("expected 1 BeginRun call, got %d", len(run.begins))
		}
		if len(run.reports) != 0 || len(run.finishes) != 0 {
			t.Fatalf("expected no ReportInstances or FinishRun calls, got %d and %d", len(run.reports), len(run.finishes))
		}
		if len(run.evals) != 3 {
			t.Fatalf("expected 3 evaluations, got %d", len(run.evals))
		}
		for _, eval := range run.evals {
			if eval.RunID != "" {
				t.Errorf("expected no run id, got %q for %s", eval.RunID, eval.Meta.GetAddress())
			}
		}
	})

	t.Run("ReportInstances fails", func(t *testing.T) {
		oldMax := relationshipChunkMaxRecords
		relationshipChunkMaxRecords = 2
		t.Cleanup(func() { relationshipChunkMaxRecords = oldMax })

		client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
		run.reportErr = func(i int) error {
			if i == 0 {
				return fmt.Errorf("ReportInstances failed")
			}
			return nil
		}
		plan(t, client)

		// The remaining chunk isn't sent, but the run continues.
		run.assertRunSequence(t)
		if len(run.reports) != 1 {
			t.Fatalf("expected 1 ReportInstances call, got %d", len(run.reports))
		}
		if len(run.evals) != 3 {
			t.Fatalf("expected 3 evaluations, got %d", len(run.evals))
		}
	})

	t.Run("responses with error diagnostics", func(t *testing.T) {
		client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
		errDiag := []*proto.Diagnostic{{Severity: proto.Severity_ERROR, Summary: "plugin error"}}
		beginRun := client.BeginRunFn
		client.BeginRunFn = func(ctx context.Context, req *proto.BeginRunRequest) (*proto.BeginRunResponse, error) {
			resp, err := beginRun(ctx, req)
			resp.Diagnostics = errDiag
			return resp, err
		}
		report := client.ReportInstancesFn
		client.ReportInstancesFn = func(ctx context.Context, req *proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error) {
			resp, err := report(ctx, req)
			resp.Diagnostics = errDiag
			return resp, err
		}
		finish := client.FinishRunFn
		client.FinishRunFn = func(ctx context.Context, req *proto.FinishRunRequest) (*proto.FinishRunResponse, error) {
			resp, err := finish(ctx, req)
			resp.Diagnostics = errDiag
			return resp, err
		}
		plan(t, client)

		run.assertRunSequence(t)
		if got := len(run.records(t)); got != 3 {
			t.Fatalf("expected 3 records, got %d", got)
		}
	})

	t.Run("FinishRun fails", func(t *testing.T) {
		client, run := newRelationshipsPolicyClient(t, relTypeSpec("test_net", "id"))
		finish := client.FinishRunFn
		client.FinishRunFn = func(ctx context.Context, req *proto.FinishRunRequest) (*proto.FinishRunResponse, error) {
			finish(ctx, req)
			return nil, fmt.Errorf("FinishRun failed")
		}
		plan(t, client)
		run.assertRunSequence(t)
	})
}

func TestContext2Plan_PolicyRelationships_emptySpec(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_net" "a" {
			name = "a"
		}
	`})
	// No types in the spec.
	_, run := planRelationships(t, mod, nil, &PlanOpts{Mode: plans.NormalMode}, nil)

	run.assertRunSequence(t)
	if len(run.reports) != 1 {
		t.Fatalf("expected 1 ReportInstances call, got %d", len(run.reports))
	}
	if report := run.reports[0]; len(report.Records) != 0 || len(report.Statuses) != 0 {
		t.Fatalf("expected no records and no statuses, got %v", report)
	}
}

func TestContext2Plan_PolicyRelationships_unconfiguredProvider(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		provider "test" {
			alias = "other"
		}
		resource "test_net" "a" {
			name = "a"
		}
		resource "test_net" "b" {
			provider = test.other
			name     = "b"
		}
	`})
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_net.b"), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"b-id","name":"b"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].other`))
	})
	// The other provider configuration isn't configured, because -target
	// excludes it.
	_, run := planRelationships(t, mod, state, &PlanOpts{
		Mode:    plans.NormalMode,
		Targets: []addrs.Targetable{mustResourceInstanceAddr("test_net.a")},
	}, nil, relTypeSpec("test_net", "id"))

	run.assertRunSequence(t)
	run.assertRecords(t, map[string]wantRelRecord{
		"test_net.a": {Action: relCreate, Source: relPlanned, Attrs: map[string]cty.Value{"name": cty.StringVal("a")}},
		"test_net.b": {
			Action:   relNoOp,
			Source:   relState,
			Attrs:    map[string]cty.Value{"id": cty.StringVal("b-id")},
			Provider: `provider["registry.terraform.io/hashicorp/test"].other`,
		},
	})
	providers := run.providers(t)
	other := providers[run.records(t)["test_net.b"].ProviderInstanceId]
	if other.Known || len(other.ConfigClass) != 0 || other.Source != "registry.terraform.io/hashicorp/test" {
		t.Fatalf("expected the unconfigured provider instance not to be known, got %v", other)
	}
}
