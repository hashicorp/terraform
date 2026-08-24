// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	viewjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
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
			name: "multiple list blocks",
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
				if got["@level"] != "info" {
					t.Fatalf("record[%d] @level = %v, want info", i, got["@level"])
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
	return tfdiags.Diagnostics{tfdiags.SourcelessWithExtra(
		tfdiags.Warning,
		"Policy evaluation skipped",
		"1 resource(s) in list block "+addr+" have no state (include_resource = false). Policy evaluation cannot be performed without resource state.",
		&tfdiags.ListBlockAddrExtra{ListBlockAddr: addr},
	)}
}

// ---------------------------------------------------------------------------
// CORE-7 Regression tests
// The original bug: QueryHuman.Hooks() and QueryJSON.Hooks() returned raw
// UiHook/jsonHook wired to the base view. When nodeQueryResourcePolicy fired
// the PolicyResult hook it landed in the generic renderer, so query policy
// results were emitted as "policy_result" records instead of being buffered
// into the queryPolicyView and flushed as a single "policy_query_summary".
// ---------------------------------------------------------------------------

// TestNewQueryJSON_hooksRouteToOperation is the primary regression test for
// CORE-7. It drives the full public surface (NewQuery → Hooks → PolicyResult
// hook call → Operation().Plan) and asserts that:
//  1. Exactly one policy_query_summary record is emitted.
//  2. No generic policy_result records are emitted.
//  3. The summary record carries the correct shape and field values.
func TestNewQueryJSON_hooksRouteToOperation(t *testing.T) {
	listBlockAddr := "aws_instance.example"
	targetAddr := "aws_instance.example_0"

	streams, done := terminal.StreamsForTesting(t)
	q := NewQuery(arguments.ViewJSON, NewView(streams))

	// Fire the PolicyResult hook as the graph walker would during query execution.
	for _, hook := range q.Hooks() {
		action, err := hook.PolicyResult(targetAddr, queryEvalResp(
			listBlockAddr,
			map[string]string{"id": "i-1"},
			policy.AllowResult,
			[]policyResultSpec{{address: "policy.allow", result: policy.AllowResult}},
		))
		if err != nil {
			t.Fatalf("PolicyResult hook returned error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Fatalf("PolicyResult hook returned unexpected action: %v", action)
		}
	}

	// Trigger flush (Plan is the point where Flush is called).
	q.Operation().Plan(nil, nil)

	output := done(t).Stdout()
	var summaries []map[string]any
	var genericResults []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("failed to decode JSON line %q: %s", line, err)
		}
		switch decoded["type"] {
		case string(viewjson.MessagePolicyQuerySummary):
			summaries = append(summaries, decoded)
		case string(viewjson.MessagePolicyEvaluationResult):
			genericResults = append(genericResults, decoded)
		}
	}

	// AC: exactly one policy_query_summary record.
	if len(summaries) != 1 {
		t.Fatalf("want 1 policy_query_summary record, got %d\nfull output:\n%s", len(summaries), output)
	}
	// AC: no duplicate generic policy_result records.
	if len(genericResults) != 0 {
		t.Fatalf("want 0 policy_result records, got %d (duplicate emission detected)\nfull output:\n%s", len(genericResults), output)
	}

	rec := summaries[0]
	if rec["@level"] != "info" {
		t.Errorf("@level = %v, want info", rec["@level"])
	}
	if rec["@policy"] != "true" {
		t.Errorf("@policy = %v, want true", rec["@policy"])
	}
	if rec["type"] != "policy_query_summary" {
		t.Errorf("type = %v, want policy_query_summary", rec["type"])
	}
	if rec["list_block_address"] != listBlockAddr {
		t.Errorf("list_block_address = %v, want %s", rec["list_block_address"], listBlockAddr)
	}
	if rec["overall_result"] != "pass" {
		t.Errorf("overall_result = %v, want pass", rec["overall_result"])
	}
	results := rec["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results length = %d, want 1", len(results))
	}
	passedPolicies := rec["passed_policies"].([]any)
	if len(passedPolicies) != 1 {
		t.Fatalf("passed_policies length = %d, want 1", len(passedPolicies))
	}
}

// TestNewQueryHuman_hooksRouteToOperation verifies that the human view's hooks
// also route PolicyResult through the query operation, so the human summary is
// rendered by Plan() and no raw policy output leaks.
func TestNewQueryHuman_hooksRouteToOperation(t *testing.T) {
	listBlockAddr := "aws_instance.example"
	targetAddr := "aws_instance.example_0"

	streams, done := terminal.StreamsForTesting(t)
	q := NewQuery(arguments.ViewHuman, NewView(streams))

	for _, hook := range q.Hooks() {
		if _, err := hook.PolicyResult(targetAddr, queryEvalResp(
			listBlockAddr,
			map[string]string{"id": "i-1"},
			policy.DenyResult,
			[]policyResultSpec{{address: "policy.deny", result: policy.DenyResult,
				diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied", "detail", policy.DenyResult)},
			}},
		)); err != nil {
			t.Fatalf("PolicyResult hook returned error: %s", err)
		}
	}

	q.Operation().Plan(nil, nil)

	output := done(t).All()
	if !strings.Contains(output, "Policy results for aws_instance.example (FAIL)") {
		t.Errorf("expected human summary in output\nfull output:\n%s", output)
	}
	// The raw "Policy Result" log line must not appear (that would be the base UiHook path).
	if strings.Contains(output, "Policy Result") {
		t.Errorf("unexpected raw policy result line in human output\nfull output:\n%s", output)
	}
}

// TestNewQueryJSON_noDuplicateRecords verifies that when a result has no
// ListBlockAddr (i.e. not a query policy result), it falls through to the
// generic emitter and does NOT produce a policy_query_summary.
func TestNewQueryJSON_noDuplicateRecords(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	q := NewQuery(arguments.ViewJSON, NewView(streams))

	// A response with no ListBlockAddr should NOT be buffered as query policy.
	for _, hook := range q.Hooks() {
		hook.PolicyResult("some.resource", queryEvalResp( //nolint:errcheck
			"", // empty ListBlockAddr — should NOT be buffered
			nil,
			policy.AllowResult,
			[]policyResultSpec{{address: "policy.allow", result: policy.AllowResult}},
		))
	}

	q.Operation().Plan(nil, nil)

	output := done(t).Stdout()
	var summaries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			continue
		}
		if decoded["type"] == string(viewjson.MessagePolicyQuerySummary) {
			summaries = append(summaries, decoded)
		}
	}
	if len(summaries) != 0 {
		t.Fatalf("expected no policy_query_summary for result with empty ListBlockAddr, got %d", len(summaries))
	}
}

// TestQueryJSONHook_policyResultDelegates verifies that queryJSONHook.PolicyResult
// calls op.PolicyResult and does not emit a generic policy_result record.
func TestQueryJSONHook_policyResultDelegates(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))
	qpv := newQueryPolicyView()
	op := &QueryOperationJSON{view: jv, queryPolicy: qpv}
	hook := &queryJSONHook{jsonHook: newJSONHook(jv), op: op}

	action, err := hook.PolicyResult("addr.target", queryEvalResp(
		"aws_instance.block",
		map[string]string{"id": "i-99"},
		policy.AllowResult,
		[]policyResultSpec{{address: "policy.p", result: policy.AllowResult}},
	))

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("unexpected action: %v", action)
	}
	if !qpv.HasResults() {
		t.Fatal("expected queryPolicyView to have results after hook call")
	}

	op.Plan(nil, nil)

	output := done(t).Stdout()
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			continue
		}
		if decoded["type"] == string(viewjson.MessagePolicyEvaluationResult) {
			t.Fatalf("unexpected generic policy_result record emitted\nfull output:\n%s", output)
		}
	}
}

// TestQueryUiHook_policyResultDelegates verifies that queryUiHook.PolicyResult
// calls op.PolicyResult without leaking anything to the raw stream immediately.
func TestQueryUiHook_policyResultDelegates(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	qpv := newQueryPolicyView()
	op := &QueryOperationHuman{view: view, queryPolicy: qpv}
	hook := &queryUiHook{UiHook: NewUiHook(view), op: op}

	action, err := hook.PolicyResult("addr.target", queryEvalResp(
		"aws_instance.block",
		map[string]string{"id": "i-99"},
		policy.AllowResult,
		[]policyResultSpec{{address: "policy.p", result: policy.AllowResult}},
	))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("unexpected action: %v", action)
	}
	if !qpv.HasResults() {
		t.Fatal("expected queryPolicyView to have results after hook call")
	}

	// Nothing should be written to the stream yet — output only arrives on Plan().
	midOutput := done(t).All()
	if midOutput != "" {
		t.Errorf("unexpected output before Plan(): %q", midOutput)
	}
}

// TestQueryPolicySummaryOverall covers each branch of the roll-up logic.
func TestQueryPolicySummaryOverall(t *testing.T) {
	tests := []struct {
		name    string
		results []queryPolicyIdentityResult
		want    queryPolicyResult
	}{
		{
			name:    "no results → pass",
			results: []queryPolicyIdentityResult{},
			want:    queryPolicyResultPass,
		},
		{
			name:    "all pass",
			results: []queryPolicyIdentityResult{{Result: queryPolicyResultPass}, {Result: queryPolicyResultPass}},
			want:    queryPolicyResultPass,
		},
		{
			name:    "any fail → fail",
			results: []queryPolicyIdentityResult{{Result: queryPolicyResultPass}, {Result: queryPolicyResultFail}},
			want:    queryPolicyResultFail,
		},
		{
			name:    "unknown with pass → unknown",
			results: []queryPolicyIdentityResult{{Result: queryPolicyResultPass}, {Result: queryPolicyResultUnknown}},
			want:    queryPolicyResultUnknown,
		},
		{
			name:    "unknown does not override fail",
			results: []queryPolicyIdentityResult{{Result: queryPolicyResultFail}, {Result: queryPolicyResultUnknown}},
			want:    queryPolicyResultFail,
		},
		{
			name:    "error short-circuits",
			results: []queryPolicyIdentityResult{{Result: queryPolicyResultFail}, {Result: queryPolicyResultError}},
			want:    queryPolicyResultError,
		},
		{
			name:    "error short-circuits even before fail",
			results: []queryPolicyIdentityResult{{Result: queryPolicyResultError}, {Result: queryPolicyResultFail}},
			want:    queryPolicyResultError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := queryPolicySummaryOverall(tc.results)
			if got != tc.want {
				t.Errorf("queryPolicySummaryOverall = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestQueryPolicyResultFromEvaluation verifies every EvaluateResult → queryPolicyResult mapping.
func TestQueryPolicyResultFromEvaluation(t *testing.T) {
	tests := []struct {
		input policy.EvaluateResult
		want  queryPolicyResult
	}{
		{policy.AllowResult, queryPolicyResultPass},
		{policy.DenyResult, queryPolicyResultFail},
		{policy.PolicyErrorResult, queryPolicyResultError},
		{policy.SetupErrorResult, queryPolicyResultError},
		{policy.EvaluateResult(999), queryPolicyResultUnknown},
	}
	for _, tc := range tests {
		got := queryPolicyResultFromEvaluation(tc.input)
		if got != tc.want {
			t.Errorf("queryPolicyResultFromEvaluation(%v) = %s, want %s", tc.input, got, tc.want)
		}
	}
}

// TestQueryPolicyView_addResultNoListBlock verifies that AddResult returns false
// and does not buffer a block when ListBlockAddr is empty.
func TestQueryPolicyView_addResultNoListBlock(t *testing.T) {
	v := newQueryPolicyView()
	resp := queryEvalResp("", nil, policy.AllowResult, []policyResultSpec{{address: "p", result: policy.AllowResult}})
	if v.AddResult("target", resp) {
		t.Fatal("AddResult should return false when ListBlockAddr is empty")
	}
	if v.HasResults() {
		t.Fatal("HasResults should be false when no results with a list block have been added")
	}
}

// TestQueryPolicyView_flushEmptiesBuffer verifies that calling Flush a second
// time does not re-emit records from a prior flush.
func TestQueryPolicyView_flushEmptiesBuffer(t *testing.T) {
	v := newQueryPolicyView()
	v.AddResult("target", queryEvalResp("aws_instance.block", nil, policy.AllowResult, []policyResultSpec{{address: "p", result: policy.AllowResult}}))

	var count int
	flush := func(_ queryPolicySummary) { count++ }
	v.Flush(flush, nil)
	if count != 1 {
		t.Fatalf("first Flush: got %d calls, want 1", count)
	}
	v.Flush(flush, nil)
	if count != 1 {
		t.Fatalf("second Flush: got %d total calls, want still 1 (buffer should have been drained)", count)
	}
}

// TestQueryPolicyView_concurrentAddResult verifies that concurrent AddResult
// calls from multiple goroutines do not race (use with -race).
func TestQueryPolicyView_concurrentAddResult(t *testing.T) {
	const goroutines = 20
	v := newQueryPolicyView()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			addr := "aws_instance.block"
			target := "aws_instance.target_" + string(rune('a'+i))
			v.AddResult(target, queryEvalResp(addr, map[string]string{"id": string(rune('a' + i))}, policy.AllowResult, []policyResultSpec{{address: "p.allow", result: policy.AllowResult}}))
		}(i)
	}
	wg.Wait()

	var summaries []queryPolicySummary
	v.Flush(func(s queryPolicySummary) { summaries = append(summaries, s) }, nil)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary (one block), got %d", len(summaries))
	}
	if len(summaries[0].Results) != goroutines {
		t.Fatalf("expected %d results in summary, got %d", goroutines, len(summaries[0].Results))
	}
}

// TestQueryPolicyEvaluatedCount verifies that queryPolicyEvaluatedCount counts
// unique policy addresses across all blocks.
func TestQueryPolicyEvaluatedCount(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*queryPolicyView)
		want  int
	}{
		{
			name:  "nil view",
			setup: func(_ *queryPolicyView) {},
			want:  0,
		},
		{
			name: "single block single policy",
			setup: func(v *queryPolicyView) {
				v.AddResult("t0", queryEvalResp("block.a", nil, policy.AllowResult, []policyResultSpec{{address: "p.one", result: policy.AllowResult}}))
			},
			want: 1,
		},
		{
			name: "two blocks same policy address",
			setup: func(v *queryPolicyView) {
				v.AddResult("t0", queryEvalResp("block.a", nil, policy.AllowResult, []policyResultSpec{{address: "p.one", result: policy.AllowResult}}))
				v.AddResult("t1", queryEvalResp("block.b", nil, policy.AllowResult, []policyResultSpec{{address: "p.one", result: policy.AllowResult}}))
			},
			want: 1, // same address; deduplicated
		},
		{
			name: "two blocks different policies",
			setup: func(v *queryPolicyView) {
				v.AddResult("t0", queryEvalResp("block.a", nil, policy.AllowResult, []policyResultSpec{{address: "p.one", result: policy.AllowResult}}))
				v.AddResult("t1", queryEvalResp("block.b", nil, policy.AllowResult, []policyResultSpec{{address: "p.two", result: policy.AllowResult}}))
			},
			want: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := newQueryPolicyView()
			tc.setup(v)
			got := queryPolicyEvaluatedCount(v)
			if got != tc.want {
				t.Errorf("queryPolicyEvaluatedCount = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestQueryOperationJSON_recordShape verifies the exact JSON shape of the
// policy_query_summary record against the RFC wire contract.
func TestQueryOperationJSON_recordShape(t *testing.T) {
	listBlockAddr := "aws_instance.web"

	streams, done := terminal.StreamsForTesting(t)
	v := &QueryOperationJSON{view: NewJSONView(NewView(streams)), queryPolicy: newQueryPolicyView()}

	// Two identities: one pass, one fail with a diagnostic.
	v.PolicyResult("aws_instance.web_0", queryEvalResp(
		listBlockAddr,
		map[string]string{"account": "111", "id": "i-aaa"},
		policy.AllowResult,
		[]policyResultSpec{{address: "policy.allow", result: policy.AllowResult}},
	))
	v.PolicyResult("aws_instance.web_1", queryEvalResp(
		listBlockAddr,
		map[string]string{"account": "222", "id": "i-bbb"},
		policy.DenyResult,
		[]policyResultSpec{{
			address:     "policy.deny",
			result:      policy.DenyResult,
			diagnostics: []policy.Diagnostic{policy.NewErrorDiagnostic("denied summary", "denied detail", policy.DenyResult)},
		}},
	))
	v.Plan(nil, nil)

	output := done(t).Stdout()
	var rec map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("failed to decode JSON: %s", err)
		}
		if decoded["type"] == "policy_query_summary" {
			rec = decoded
			break
		}
	}
	if rec == nil {
		t.Fatalf("no policy_query_summary record found\nfull output:\n%s", output)
	}

	// Top-level field presence and values.
	requiredFields := []string{"@level", "@policy", "type", "list_block_address", "overall_result", "results", "passed_policies"}
	for _, f := range requiredFields {
		if _, ok := rec[f]; !ok {
			t.Errorf("missing required field %q in policy_query_summary record", f)
		}
	}
	if rec["@level"] != "info" {
		t.Errorf("@level = %v, want info", rec["@level"])
	}
	if rec["@policy"] != "true" {
		t.Errorf("@policy = %v, want true", rec["@policy"])
	}
	if rec["type"] != "policy_query_summary" {
		t.Errorf("type = %v, want policy_query_summary", rec["type"])
	}
	if rec["list_block_address"] != listBlockAddr {
		t.Errorf("list_block_address = %v, want %s", rec["list_block_address"], listBlockAddr)
	}
	if rec["overall_result"] != "fail" {
		t.Errorf("overall_result = %v, want fail", rec["overall_result"])
	}

	// results[] shape.
	results := rec["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}
	for idx, rawResult := range results {
		r := rawResult.(map[string]any)
		if r["target_address"] == "" {
			t.Errorf("results[%d] missing target_address", idx)
		}
		if r["result"] == "" {
			t.Errorf("results[%d] missing result", idx)
		}
		if _, ok := r["identity"]; !ok {
			t.Errorf("results[%d] missing identity", idx)
		}
		policies := r["policies"].([]any)
		if len(policies) == 0 {
			t.Errorf("results[%d] policies is empty", idx)
		}
		for pidx, rawPol := range policies {
			pol := rawPol.(map[string]any)
			if _, ok := pol["policy_metadata"].(map[string]any); !ok {
				t.Errorf("results[%d].policies[%d] missing policy_metadata object", idx, pidx)
			}
			if pol["result"] == "" {
				t.Errorf("results[%d].policies[%d] missing result", idx, pidx)
			}
			if _, ok := pol["diagnostics"]; !ok {
				t.Errorf("results[%d].policies[%d] missing diagnostics field", idx, pidx)
			}
		}
	}

	// passed_policies[] shape: 2 distinct policy addresses.
	passedPolicies := rec["passed_policies"].([]any)
	if len(passedPolicies) != 2 {
		t.Fatalf("passed_policies length = %d, want 2", len(passedPolicies))
	}
	for pidx, rawMeta := range passedPolicies {
		if _, ok := rawMeta.(map[string]any); !ok {
			t.Errorf("passed_policies[%d] is not an object", pidx)
		}
	}
}

// TestQueryPolicyView_sortedOutput verifies that both results[] and
// passed_policies[] are emitted in deterministic sorted order.
func TestQueryPolicyView_sortedOutput(t *testing.T) {
	v := newQueryPolicyView()
	// Add targets in reverse alphabetical order.
	v.AddResult("z_target", queryEvalResp("block.x", nil, policy.AllowResult, []policyResultSpec{{address: "p.z", result: policy.AllowResult}}))
	v.AddResult("a_target", queryEvalResp("block.x", nil, policy.AllowResult, []policyResultSpec{{address: "p.a", result: policy.AllowResult}}))

	var summaries []queryPolicySummary
	v.Flush(func(s queryPolicySummary) { summaries = append(summaries, s) }, nil)

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	s := summaries[0]

	if s.Results[0].TargetAddress != "a_target" || s.Results[1].TargetAddress != "z_target" {
		t.Errorf("results not sorted: got [%s, %s], want [a_target, z_target]",
			s.Results[0].TargetAddress, s.Results[1].TargetAddress)
	}
	if s.PassedPolicies[0].PolicyName != "p.a" || s.PassedPolicies[1].PolicyName != "p.z" {
		t.Errorf("passed_policies not sorted: got [%s, %s], want [p.a, p.z]",
			s.PassedPolicies[0].PolicyName, s.PassedPolicies[1].PolicyName)
	}
}

// TestQueryPolicyView_multiBlock verifies that results for distinct list block
// addresses are emitted as separate records, sorted by list_block_address.
func TestQueryPolicyView_multiBlock(t *testing.T) {
	v := newQueryPolicyView()
	v.AddResult("t1", queryEvalResp("block.z", nil, policy.AllowResult, []policyResultSpec{{address: "p.z", result: policy.AllowResult}}))
	v.AddResult("t2", queryEvalResp("block.a", nil, policy.DenyResult, []policyResultSpec{{address: "p.a", result: policy.DenyResult}}))

	var summaries []queryPolicySummary
	v.Flush(func(s queryPolicySummary) { summaries = append(summaries, s) }, nil)

	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries (one per block), got %d", len(summaries))
	}
	// Emitted in sorted order: block.a before block.z.
	if summaries[0].ListBlockAddress != "block.a" {
		t.Errorf("summaries[0] ListBlockAddress = %q, want block.a", summaries[0].ListBlockAddress)
	}
	if summaries[1].ListBlockAddress != "block.z" {
		t.Errorf("summaries[1] ListBlockAddress = %q, want block.z", summaries[1].ListBlockAddress)
	}
	if summaries[0].OverallResult != queryPolicyResultFail {
		t.Errorf("summaries[0] OverallResult = %s, want fail", summaries[0].OverallResult)
	}
	if summaries[1].OverallResult != queryPolicyResultPass {
		t.Errorf("summaries[1] OverallResult = %s, want pass", summaries[1].OverallResult)
	}
}

func TestQueryOperation_nilQueryPolicyPolicyResult(t *testing.T) {
	// Test that QueryOperationHuman and QueryOperationJSON do not panic
	// when PolicyResult is called with queryPolicy == nil. This can occur if
	// a test constructs the operation directly without going through NewQuery.
	streams, _ := terminal.StreamsForTesting(t)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "QueryOperationHuman with nil queryPolicy",
			test: func(t *testing.T) {
				op := &QueryOperationHuman{
					view:        NewView(streams),
					queryPolicy: nil,
				}
				// This should not panic. It should delegate to the base view.
				resp := policy.EvaluationResponse{
					ListBlockAddr: "",
					Identity:      map[string]string{"id": "test"},
					Overall:       policy.AllowResult,
					Policies:      []*policy.Policy{},
				}
				// Call should complete without panic
				op.PolicyResult("aws_instance.example", resp)
			},
		},
		{
			name: "QueryOperationJSON with nil queryPolicy",
			test: func(t *testing.T) {
				op := &QueryOperationJSON{
					view:        NewJSONView(NewView(streams)),
					queryPolicy: nil,
				}
				// This should not panic. It should delegate to the base view.
				resp := policy.EvaluationResponse{
					ListBlockAddr: "",
					Identity:      map[string]string{"id": "test"},
					Overall:       policy.AllowResult,
					Policies:      []*policy.Policy{},
				}
				// Call should complete without panic
				op.PolicyResult("aws_instance.example", resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
