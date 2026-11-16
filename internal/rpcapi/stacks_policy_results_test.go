// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
)

// A replaced resource instance is evaluated once for its create and once for
// its delete, and both report against the same resource address.
func TestStackPolicyEvaluationHooks_multipleResultsForAddress(t *testing.T) {
	runtimeHooks := &stackruntime.Hooks{}
	results := stackPolicyEvaluationHooks(policy.NewTestMockClient(t), runtimeHooks)

	componentAddr := mustAbsComponentInstance(t, "component.test")
	for _, overall := range []policy.EvaluateResult{policy.AllowResult, policy.DenyResult} {
		runtimeHooks.ReportComponentInstancePolicyResult(context.Background(), nil, &hooks.ComponentInstancePolicyResult{
			ComponentAddr: componentAddr,
			ResourceAddr:  "test_instance.example",
			Result:        policy.EvaluationResponse{Overall: overall},
		})
	}

	got, ok := results.GetOk(componentAddr)
	if !ok {
		t.Fatalf("no policy results recorded for %s", componentAddr)
	}
	var gotOverall []policy.EvaluateResult
	for _, result := range got["test_instance.example"] {
		gotOverall = append(gotOverall, result.Overall)
	}
	want := []policy.EvaluateResult{policy.AllowResult, policy.DenyResult}
	if diff := cmp.Diff(want, gotOverall); diff != "" {
		t.Errorf("wrong results for test_instance.example\n%s", diff)
	}
}

func TestComponentInstancePolicyEvaluationProto_multipleResultsForAddress(t *testing.T) {
	componentAddr := mustAbsComponentInstance(t, "component.test")
	policyObj := func(result policy.EvaluateResult) *policy.Policy {
		return &policy.Policy{
			Address:          "resource_policy.example",
			PolicySetName:    "example",
			Filename:         "policy_file.tfpolicy.hcl",
			EnforcementLevel: "mandatory",
			Result:           result,
		}
	}

	got := componentInstancePolicyEvaluationProto(componentAddr, map[string][]policy.EvaluationResponse{
		"test_instance.example": {
			{
				Overall:  policy.AllowResult,
				Policies: []*policy.Policy{policyObj(policy.AllowResult)},
				Enforcements: []policy.EnforcementResult{{
					Result:     policy.AllowResult,
					Message:    "create allowed",
					BlockIndex: 1,
					Policy:     policyObj(policy.AllowResult),
				}},
			},
			{
				Overall:  policy.DenyResult,
				Policies: []*policy.Policy{policyObj(policy.DenyResult)},
				Diagnostics: policy.Diagnostics{
					policy.NewErrorDiagnostic(
						"delete denied",
						"test_instance.example may not be destroyed",
						policy.DenyResult,
					),
				},
			},
		},
	})

	if len(got.Results) != 2 || len(got.Infos) != 1 || len(got.Diagnostics) != 1 {
		t.Fatalf("got %d results, %d infos, %d diagnostics; want 2, 1, 1",
			len(got.Results), len(got.Infos), len(got.Diagnostics))
	}
	var gotResults []stacks.EvaluateResult
	for _, result := range got.Results {
		if result.TargetAddress != "test_instance.example" {
			t.Errorf("wrong target address %q", result.TargetAddress)
		}
		gotResults = append(gotResults, result.Result)
	}
	want := []stacks.EvaluateResult{
		stacks.EvaluateResult_ALLOW_EVALUATE_RESULT,
		stacks.EvaluateResult_DENY_EVALUATE_RESULT,
	}
	if diff := cmp.Diff(want, gotResults); diff != "" {
		t.Errorf("wrong results\n%s", diff)
	}
}
