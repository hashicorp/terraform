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

	// wantBlock carries per-record assertions for multi-block test cases.
	type wantBlock struct {
		addr         string
		wantOverall  string
		wantResults  int
		wantPolicies int
	}

	tests := []struct {
		name         string
		responses    []policy.EvaluationResponse
		targets      []string
		wantOverall  string
		wantResults  int
		wantPolicies int
		wantPerBlock []wantBlock
	}{
		{
			name: "all pass",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-2"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
			},
			targets:      []string{"aws_instance.example_0", "aws_instance.example_1"},
			wantOverall:  "pass",
			wantResults:  2,
			wantPolicies: 1,
		},
		{
			name: "all fail",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)}}}),
			},
			targets:      []string{"aws_instance.example_0"},
			wantOverall:  "fail",
			wantResults:  1,
			wantPolicies: 1,
		},
		{
			name: "mixed",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-2"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)}}}),
			},
			targets:      []string{"aws_instance.example_0", "aws_instance.example_1"},
			wantOverall:  "fail",
			wantResults:  2,
			wantPolicies: 2,
		},
		{
			name: "unknown",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.EvaluateResult(999), []policyResultSpec{{address: "policy.unknown", result: policy.EvaluateResult(999)}}),
			},
			targets:      []string{"aws_instance.example_0"},
			wantOverall:  "unknown",
			wantResults:  1,
			wantPolicies: 1,
		},
		{
			name: "error",
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.PolicyErrorResult, []policyResultSpec{{address: "policy.error", result: policy.PolicyErrorResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("errored", "detail", policy.PolicyErrorResult)}}}),
			},
			targets:      []string{"aws_instance.example_0"},
			wantOverall:  "error",
			wantResults:  1,
			wantPolicies: 1,
		},
		{
			name:         "multiple list blocks",
			wantPerBlock: []wantBlock{
				{
					addr:         listBlockAddr,
					wantOverall:  "pass",
					wantResults:  1,
					wantPolicies: 1,
				},
				{
					addr:         "aws_instance.other",
					wantOverall:  "pass",
					wantResults:  1,
					wantPolicies: 1,
				},
			},
			responses: []policy.EvaluationResponse{
				queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}),
				queryEvalResp("aws_instance.other", map[string]string{"id": "i-2"}, policy.AllowResult, []policyResultSpec{{address: "policy.other", result: policy.AllowResult}}),
			},
			targets: []string{"aws_instance.example_0", "aws_instance.other_0"},
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

			// For single-block test cases use the inline fields; for multi-block
			// cases use the per-block want slice.
			wants := tc.wantPerBlock
			if len(wants) == 0 {
				wants = []wantBlock{{
					addr:         "",
					wantOverall:  tc.wantOverall,
					wantResults:  tc.wantResults,
					wantPolicies: tc.wantPolicies,
				}}
			}

			if len(lines) != len(wants) {
				t.Fatalf("got %d policy_query_summary records, want %d", len(lines), len(wants))
			}

			for i, want := range wants {
				got := lines[i]
				if got["@level"] != "error" {
					t.Fatalf("record[%d] @level = %v, want error", i, got["@level"])
				}
				if got["@policy"] != "true" {
					t.Fatalf("record[%d] @policy = %v, want true", i, got["@policy"])
				}
				if want.addr != "" && got["list_block_address"] != want.addr {
					t.Fatalf("record[%d] list_block_address = %v, want %s", i, got["list_block_address"], want.addr)
				}
				if got["overall_result"] != want.wantOverall {
					t.Fatalf("record[%d] overall_result = %v, want %s", i, got["overall_result"], want.wantOverall)
				}
				results := got["results"].([]any)
				if len(results) != want.wantResults {
					t.Fatalf("record[%d] results length = %d, want %d", i, len(results), want.wantResults)
				}
				passedPolicies := got["passed_policies"].([]any)
				if len(passedPolicies) != want.wantPolicies {
					t.Fatalf("record[%d] passed_policies length = %d, want %d", i, len(passedPolicies), want.wantPolicies)
				}
				for _, rawResult := range results {
					result := rawResult.(map[string]any)
					if result["target_address"] == "" {
						t.Fatalf("record[%d] expected target_address", i)
					}
					if result["result"] == "" {
						t.Fatalf("record[%d] expected result", i)
					}
					for _, rawPolicy := range result["policies"].([]any) {
						pol := rawPolicy.(map[string]any)
						if pol["result"] == "" {
							t.Fatalf("record[%d] expected per-policy result", i)
						}
						if _, ok := pol["policy_metadata"].(map[string]any); !ok {
							t.Fatalf("record[%d] expected policy_metadata object", i)
						}
					}
				}
			}
		})
	}
}

func TestQueryOperationHuman_policySummary(t *testing.T) {
	listBlockAddr := "aws_instance.example"

	tests := []struct {
		name    string
		plan    *plans.Plan
		schemas *terraform.Schemas
		setup   func(v *QueryOperationHuman)
		want    []string
		notWant []string
	}{
		{
			name: "mixed_pass_and_fail",
			setup: func(v *QueryOperationHuman) {
				v.PolicyResult("aws_instance.example_0", queryEvalResp(listBlockAddr, map[string]string{"account": "123", "id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}))
				v.PolicyResult("aws_instance.example_1", queryEvalResp(listBlockAddr, map[string]string{"account": "123", "id": "i-2"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied summary", "detail", policy.DenyResult)}}}))
				v.Diagnostics(tfdiagsWarningForQueryTest("aws_instance.example"))
			},
			want: []string{
				"Evaluated 2 policies.",
				"Policy results for aws_instance.example (FAIL)",
				"account=123, id=i-1: PASS",
				"account=123, id=i-2: FAIL",
				"policy.deny: denied summary (mandatory)",
				"Policy evaluation skipped",
			},
		},
		{
			name: "all_pass",
			setup: func(v *QueryOperationHuman) {
				v.PolicyResult("aws_instance.example_0", queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}))
				v.PolicyResult("aws_instance.example_1", queryEvalResp(listBlockAddr, map[string]string{"id": "i-2"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}))
			},
			want: []string{
				"Evaluated 1 policies.",
				"Policy results for aws_instance.example (PASS)",
				"id=i-1: PASS",
				"id=i-2: PASS",
			},
		},
		{
			name: "all_fail",
			setup: func(v *QueryOperationHuman) {
				v.PolicyResult("aws_instance.example_0", queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)}}}))
				v.PolicyResult("aws_instance.example_1", queryEvalResp(listBlockAddr, map[string]string{"id": "i-2"}, policy.DenyResult, []policyResultSpec{{address: "policy.deny", result: policy.DenyResult, diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)}}}))
			},
			want: []string{
				"Evaluated 1 policies.",
				"Policy results for aws_instance.example (FAIL)",
				"id=i-1: FAIL",
				"id=i-2: FAIL",
				"policy.deny: denied (mandatory)",
			},
		},
		{
			name: "unknown_result",
			setup: func(v *QueryOperationHuman) {
				v.PolicyResult("aws_instance.example_0", queryEvalResp(listBlockAddr, map[string]string{"id": "i-1"}, policy.EvaluateResult(999), []policyResultSpec{{address: "policy.unknown", result: policy.EvaluateResult(999)}}))
			},
			want: []string{
				"Evaluated 1 policies.",
				"Policy results for aws_instance.example (UNKNOWN)",
				"id=i-1: UNKNOWN",
			},
		},
		{
			name:    "no_policies",
			plan:    &plans.Plan{Changes: &plans.ChangesSrc{}},
			schemas: &terraform.Schemas{},
			setup: func(v *QueryOperationHuman) {
				// No PolicyResult calls; Plan should not print a policy summary section.
			},
			notWant: []string{
				"Evaluated",
				"Policy results for",
			},
		},
		{
			name:    "nil_plan",
			plan:    nil,
			schemas: nil,
			setup: func(v *QueryOperationHuman) {
				// No PolicyResult calls; Plan(nil, nil) must not panic.
			},
			notWant: []string{
				"Evaluated",
				"Policy results for",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := &QueryOperationHuman{view: NewView(streams), queryPolicy: newQueryPolicyView()}
			tc.setup(v)
			v.Plan(tc.plan, tc.schemas)

			output := done(t).All()
			for _, want := range tc.want {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q\nfull output:\n%s", want, output)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("expected output NOT to contain %q\nfull output:\n%s", notWant, output)
				}
			}
		})
	}
}


func TestQueryPolicyListBlockAddr(t *testing.T) {
	tests := []struct {
		detail string
		want   string
	}{
		{
			detail: "1 resource(s) in list block aws_instance.example have no state",
			want:   "aws_instance.example",
		},
		{
			detail: "1 resource(s) in list block module.vpc.aws_instance.mylist have no state",
			want:   "module.vpc.aws_instance.mylist",
		},
		{
			detail: "1 resource(s) in list block aws_instance.example: extra text",
			want:   "aws_instance.example",
		},
		{
			detail: "no marker here",
			want:   "",
		},
		{
			detail: "list block ",
			want:   "",
		},
	}
	for _, tc := range tests {
		got := queryPolicyListBlockAddr(tc.detail)
		if got != tc.want {
			t.Errorf("queryPolicyListBlockAddr(%q) = %q, want %q", tc.detail, got, tc.want)
		}
	}
}

func TestAddWarningDiagsRoutesCorrectly(t *testing.T) {
	// Verify that a warning diagnostic for "aws_instance.example" is routed to
	// the block keyed "aws_instance.example", not a truncated "aws_instance".
	v := newQueryPolicyView()
	v.AddResult("aws_instance.example_0", queryEvalResp("aws_instance.example", map[string]string{"id": "i-1"}, policy.AllowResult, []policyResultSpec{{address: "policy.allow", result: policy.AllowResult}}))
	v.AddWarningDiags(tfdiagsWarningForQueryTest("aws_instance.example"))

	var summaries []queryPolicySummary
	v.Flush(func(s queryPolicySummary) { summaries = append(summaries, s) }, nil)

	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	if summaries[0].ListBlockAddress != "aws_instance.example" {
		t.Fatalf("ListBlockAddress = %q, want %q", summaries[0].ListBlockAddress, "aws_instance.example")
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

// queryEvalResp builds a policy.EvaluationResponse for tests. The target
// address is passed separately as the first argument to PolicyResult and is not
// part of the response itself.
func queryEvalResp(listBlockAddr string, identity map[string]string, overall policy.EvaluateResult, specs []policyResultSpec) policy.EvaluationResponse {
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
