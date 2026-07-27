// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	viewjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestQueryOperationJSON_policySummary(t *testing.T) {
	listBlockAddr := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "aws_instance", Name: "example"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance).String()

	tests := []struct {
		name         string
		responses    []policy.EvaluationResponse
		targets      []string
		wantOverall  string
		wantResults  int
		wantPolicies int
	}{
		{
			name: "all pass",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
				queryEvalResp(listBlockAddr, "aws_instance.example_1", map[string]string{"id": "i-2"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
			},
			targets:      []string{"aws_instance.example_0", "aws_instance.example_1"},
			wantOverall:  "pass",
			wantResults:  2,
			wantPolicies: 1,
		},
		{
			name: "all fail",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"id": "i-1"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)}}}),
			},
			targets:      []string{"aws_instance.example_0"},
			wantOverall:  "fail",
			wantResults:  1,
			wantPolicies: 1,
		},
		{
			name: "mixed",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
				queryEvalResp(listBlockAddr, "aws_instance.example_1", map[string]string{"id": "i-2"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)}}}),
			},
			targets:      []string{"aws_instance.example_0", "aws_instance.example_1"},
			wantOverall:  "fail",
			wantResults:  2,
			wantPolicies: 2,
		},
		{
			name: "unknown",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"id": "i-1"}, policy.EvaluateResult(999), []policyResultSpec{{address: "policy.unknown", result: policy.EvaluateResult(999)}}),
			},
			targets:      []string{"aws_instance.example_0"},
			wantOverall:  "unknown",
			wantResults:  1,
			wantPolicies: 1,
		},
		{
			name: "error",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"id": "i-1"}, policy.PolicyErrorResult, []policyResultSpec{{address: "policy.error", result: policy.PolicyErrorResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("errored", "detail", policy.PolicyErrorResult)}}}),
			},
			targets:      []string{"aws_instance.example_0"},
			wantOverall:  "error",
			wantResults:  1,
			wantPolicies: 1,
		},
		{
			name: "multiple list blocks",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
				queryEvalResp("aws_instance.other", "aws_instance.other_0", map[string]string{"id": "i-2"}, policy.AllowResult, []policyResultSpec{{address: "policy.other", result: policy.AllowResult}}),
			},
			targets:      []string{"aws_instance.example_0", "aws_instance.other_0"},
			wantOverall:  "pass",
			wantResults:  1,
			wantPolicies: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := &QueryOperationJSON{view: NewJSONView(NewView(streams)), queryPolicy: newQueryPolicyView()}
			for i, resp := range tc.responses {
				v.PolicyResult(tc.targets[i], resp)
			}
			v.Plan(nil, nil)

			var lines []map[string]any
			for _, line := range strings.Split(strings.TrimSpace(done(t).Stdout()), "\n") {
				var decoded map[string]any
				if err := json.Unmarshal([]byte(line), &decoded); err != nil {
					t.Fatalf("failed to decode json line %q: %s", line, err)
				}
				if decoded["type"] == string(viewjson.MessagePolicyQuerySummary) {
					lines = append(lines, decoded)
				}
			}

			if len(lines) == 0 {
				t.Fatal("expected policy query summary output")
			}

			got := lines[0]
			if got["@level"] != "error" {
				t.Fatalf("@level = %v, want error", got["@level"])
			}
			if got["@policy"] != "true" {
				t.Fatalf("@policy = %v, want true", got["@policy"])
			}
			if got["overall_result"] != tc.wantOverall {
				t.Fatalf("overall_result = %v, want %s", got["overall_result"], tc.wantOverall)
			}
			results := got["results"].([]any)
			if len(results) != tc.wantResults {
				t.Fatalf("results length = %d, want %d", len(results), tc.wantResults)
			}
			passedPolicies := got["passed_policies"].([]any)
			if len(passedPolicies) != tc.wantPolicies {
				t.Fatalf("passed_policies length = %d, want %d", len(passedPolicies), tc.wantPolicies)
			}
			for _, rawResult := range results {
				result := rawResult.(map[string]any)
				if result["target_address"] == "" {
					t.Fatal("expected target_address")
				}
				if result["result"] == "" {
					t.Fatal("expected result")
				}
				for _, rawPolicy := range result["policies"].([]any) {
					pol := rawPolicy.(map[string]any)
					if pol["result"] == "" {
						t.Fatal("expected per-policy result")
					}
					if _, ok := pol["policy_metadata"].(map[string]any); !ok {
						t.Fatal("expected policy_metadata object")
					}
				}
			}
		})
	}
}

func TestQueryOperationHuman_policySummary(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &QueryOperationHuman{view: NewView(streams), queryPolicy: newQueryPolicyView()}

	listBlockAddr := "aws_instance.example"
	v.PolicyResult("aws_instance.example_0", queryEvalResp(listBlockAddr, "aws_instance.example_0", map[string]string{"account": "123", "id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}))
	v.PolicyResult("aws_instance.example_1", queryEvalResp(listBlockAddr, "aws_instance.example_1", map[string]string{"account": "123", "id": "i-2"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied summary", "detail", policy.DenyResult)}}}))
	v.Diagnostics(tfdiagsWarningForQueryTest("aws_instance.example"))
	v.Plan(&plans.Plan{Changes: &plans.ChangesSrc{}}, &terraform.Schemas{})

	output := done(t).All()
	for _, want := range []string{
		"Evaluated 2 policies.",
		"Policy results for aws_instance.example (FAIL)",
		"account=123, id=i-1: PASS",
		"account=123, id=i-2: FAIL",
		"policy.deny: denied summary (mandatory)",
		"Policy evaluation skipped",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %s", want, output)
		}
	}
}

func TestRenderer_ignoresPolicyQuerySummary(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	renderer := jsonformat.Renderer{Streams: streams, Colorize: NewView(streams).colorize}

	err := renderer.RenderLog(&jsonformat.JSONLog{Type: jsonformat.JSONLogType(viewjson.MessagePolicyQuerySummary), Message: "ignored"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := done(t).All(); got != "" {
		t.Fatalf("expected no output, got %q", got)
	}
}

type policyResultSpec struct {
	address     string
	result      policy.EvaluateResult
	diagnostics []policy.Diagnostic
}

func queryEvalResp(listBlockAddr, _ string, identity map[string]string, overall policy.EvaluateResult, specs []policyResultSpec) policy.EvaluationResponse {
	resp := policy.EvaluationResponse{
		Overall:       overall,
		Identity:      identity,
		ListBlockAddr: listBlockAddr,
		Policies:      make([]*policy.Policy, 0, len(specs)),
	}
	for _, spec := range specs {
		pol := &policy.Policy{Address: spec.address, Filename: "policy.tfpolicy.hcl", EnforcementLevel: "mandatory", Result: spec.result}
		resp.Policies = append(resp.Policies, pol)
		for _, diag := range spec.diagnostics {
			resp.Diagnostics = append(resp.Diagnostics, diag)
		}
	}
	return resp
}

func tfdiagsWarningForQueryTest(addr string) tfdiags.Diagnostics {
	return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Warning, "Policy evaluation skipped", "1 resource(s) in list block "+addr+" have no state (include_resource = false). Policy evaluation cannot be performed without resource state.")}
}
