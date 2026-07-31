// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

// policy_crash_test.go contains unit tests for fault tolerance during the
// policy evaluation phase when the policy plugin crashes or becomes unavailable.
//
// The scenarios covered are:
//
//   TestPolicyCrash_SingleResourceCrash         – one resource crashes; result is error
//   TestPolicyCrash_DiagnosticContent           – exact diagnostic format for crash
//   TestPolicyCrash_ContinuesAfterCrash         – sibling resources are still evaluated
//   TestPolicyCrash_PartialResults              – mix of pass/crash results
//   TestPolicyCrash_AllResourcesCrash           – every resource crashes
//   TestPolicyCrash_CrashThenTimeout            – crash and per-call timeout in same pass
//   TestPolicyCrash_RaceDetector                – concurrent crashes (run with -race)

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// crashEvaluateFn returns an EvaluateFn that immediately returns
// PolicyErrorResult with a "Policy plugin crashed" summary, simulating the
// behaviour of policy/client.go after isPluginCrashError returns true.
func crashEvaluateFn(t *testing.T) func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
	t.Helper()
	return func(_ context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"Policy plugin crashed",
				fmt.Sprintf(
					"The policy plugin process crashed or became unavailable while evaluating %s: EOF. %s",
					req.Target,
					"The policy plugin process may have crashed or been forcibly terminated. Policy evaluation cannot be completed for this resource.",
				),
				policy.PolicyErrorResult,
			)},
		}
	}
}

// ---------------------------------------------------------------------------
// Single-resource crash tests
// ---------------------------------------------------------------------------

// TestPolicyCrash_SingleResourceCrash verifies that when the policy plugin
// returns a crash error for a single resource:
//   - The hook is called with Overall = PolicyErrorResult.
//   - The diagnostic summary is "Policy plugin crashed".
//   - Execute() does not return an error diagnostic itself (the crash is
//     surfaced through the hook result, not as a graph-walk error).
func TestPolicyCrash_SingleResourceCrash(t *testing.T) {
	mockClient := &policy.MockClient{
		EvaluateFn: crashEvaluateFn(t),
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	n := makeTimeoutTestNode(0)
	diags := n.Execute(ctx, walkPlan)

	// Execute must not itself return an error — the crash is a result, not a
	// graph-walk failure. Only hook errors bubble up as diags.
	if diags.HasErrors() {
		t.Fatalf("unexpected error diags from Execute: %s", diags.Err())
	}

	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatalf("expected PolicyResult hook call for crashed resource, got none")
	}

	if resp.Overall != policy.PolicyErrorResult {
		t.Errorf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic for crashed resource")
	}
}

// TestPolicyCrash_DiagnosticContent verifies the exact diagnostic produced
// when the plugin crashes. The summary must be "Policy plugin crashed" and the
// detail must contain the pluginCrashDiagMsg sentinel string so that callers
// can distinguish crash errors from ordinary policy evaluation failures.
func TestPolicyCrash_DiagnosticContent(t *testing.T) {
	const pluginCrashSentinel = "The policy plugin process may have crashed or been forcibly terminated."

	mockClient := &policy.MockClient{
		EvaluateFn: crashEvaluateFn(t),
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	n := makeTimeoutTestNode(0)
	n.Execute(ctx, walkPlan) //nolint:errcheck

	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatalf("expected PolicyResult hook call")
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics on crash result, got none")
	}

	d := resp.Diagnostics[0]
	if d.Description().Summary != "Policy plugin crashed" {
		t.Errorf("Summary = %q, want \"Policy plugin crashed\"", d.Description().Summary)
	}
	if !strings.Contains(d.Description().Detail, pluginCrashSentinel) {
		t.Errorf("Detail %q does not contain crash sentinel %q", d.Description().Detail, pluginCrashSentinel)
	}
}

// ---------------------------------------------------------------------------
// Crash isolation tests
// ---------------------------------------------------------------------------

// TestPolicyCrash_ContinuesAfterCrash verifies that after one resource's
// evaluation crashes, the next resource in the pass is still evaluated.
// This is the critical fault-tolerance property: a single crash must not
// abort the entire policy pass.
//
// Note: n0 and n1 are executed directly and sequentially (not via the DAG
// walker) so that the test is deterministic. AllowUpstreamFailure on
// nodePolicyEval handles a different concern (resource-graph failures upstream
// of the policy phase); this test is specifically about crash containment
// within the policy evaluation itself.
func TestPolicyCrash_ContinuesAfterCrash(t *testing.T) {
	var callCount int32

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			// First resource: simulate a crash (Unavailable/EOF scenario).
			return policy.EvaluationResponse{
				Overall: policy.PolicyErrorResult,
				Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
					"Policy plugin crashed",
					"The policy plugin process crashed: EOF. The policy plugin process may have crashed or been forcibly terminated. Policy evaluation cannot be completed for this resource.",
					policy.PolicyErrorResult,
				)},
			}
		}
		// Subsequent resources: succeed normally.
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	n0 := makeTimeoutTestNode(0)
	n1 := makeTimeoutTestNode(1)

	n0.Execute(ctx, walkPlan) //nolint:errcheck
	n1.Execute(ctx, walkPlan) //nolint:errcheck

	// n0 must be error (crashed).
	if resp, ok := hook.PolicyResults[n0.ResourceAddr.String()]; !ok {
		t.Error("n0: no PolicyResult")
	} else if resp.Overall != policy.PolicyErrorResult {
		t.Errorf("n0: Overall = %v, want PolicyErrorResult", resp.Overall)
	}

	// n1 must be AllowResult — crash of n0 must not abort the pass.
	if resp, ok := hook.PolicyResults[n1.ResourceAddr.String()]; !ok {
		t.Error("n1: no PolicyResult — crash of n0 should not have prevented n1 from being evaluated")
	} else if resp.Overall != policy.AllowResult {
		t.Errorf("n1: Overall = %v, want AllowResult — crash of n0 must not abort the pass", resp.Overall)
	}
}

// TestPolicyCrash_PartialResults verifies that in a mix of passing and crashing
// resources, each resource's result is recorded correctly. This models the
// acceptance criterion: "partial results after crash".
func TestPolicyCrash_PartialResults(t *testing.T) {
	// r0: AllowResult, r1: crash, r2: DenyResult, r3: crash, r4: AllowResult
	responses := map[int]policy.EvaluationResponse{
		0: {Overall: policy.AllowResult},
		1: {
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"Policy plugin crashed",
				"crash: EOF. The policy plugin process may have crashed or been forcibly terminated. Policy evaluation cannot be completed for this resource.",
				policy.PolicyErrorResult,
			)},
		},
		2: {Overall: policy.DenyResult, Enforcements: []policy.EnforcementResult{{Result: policy.DenyResult, Message: "denied"}}},
		3: {
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"Policy plugin crashed",
				"crash: broken pipe. The policy plugin process may have crashed or been forcibly terminated. Policy evaluation cannot be completed for this resource.",
				policy.PolicyErrorResult,
			)},
		},
		4: {Overall: policy.AllowResult},
	}

	var idx int32
	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		i := int(atomic.AddInt32(&idx, 1)) - 1
		if r, ok := responses[i]; ok {
			return r
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	nodes := make([]*nodeQueryResourcePolicy, 5)
	for i := range nodes {
		nodes[i] = makeTimeoutTestNode(i)
		nodes[i].Execute(ctx, walkPlan) //nolint:errcheck
	}

	if len(hook.PolicyResults) != 5 {
		t.Fatalf("expected 5 hook calls, got %d", len(hook.PolicyResults))
	}

	wantResults := map[int]policy.EvaluateResult{
		0: policy.AllowResult,
		1: policy.PolicyErrorResult,
		2: policy.DenyResult,
		3: policy.PolicyErrorResult,
		4: policy.AllowResult,
	}

	for i, n := range nodes {
		resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
		if !ok {
			t.Errorf("node %d: missing PolicyResult", i)
			continue
		}
		if resp.Overall != wantResults[i] {
			t.Errorf("node %d: Overall = %v, want %v", i, resp.Overall, wantResults[i])
		}
	}
}

// TestPolicyCrash_AllResourcesCrash verifies that when every resource
// evaluation crashes, all results are recorded as PolicyErrorResult and the
// hook is called for every resource (no silent drops).
func TestPolicyCrash_AllResourcesCrash(t *testing.T) {
	mockClient := &policy.MockClient{
		EvaluateFn: crashEvaluateFn(t),
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	const count = 4
	for i := 0; i < count; i++ {
		makeTimeoutTestNode(i).Execute(ctx, walkPlan) //nolint:errcheck
	}

	if len(hook.PolicyResults) != count {
		t.Fatalf("expected %d hook calls, got %d", count, len(hook.PolicyResults))
	}
	for addr, resp := range hook.PolicyResults {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("%s: Overall = %v, want PolicyErrorResult", addr, resp.Overall)
		}
		if len(resp.Diagnostics) == 0 {
			t.Errorf("%s: expected diagnostics on crash result", addr)
		}
	}
}

// TestPolicyCrash_CrashThenTimeout verifies the combination scenario where
// one resource crashes (gRPC Unavailable/EOF) and a second resource hits the
// per-call timeout. Both must be recorded as PolicyErrorResult; the first with
// a "Policy plugin crashed" diagnostic, the second with a timeout diagnostic.
func TestPolicyCrash_CrashThenTimeout(t *testing.T) {
	var callCount int32

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			// First call: immediate crash response.
			return policy.EvaluationResponse{
				Overall: policy.PolicyErrorResult,
				Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
					"Policy plugin crashed",
					"crash: EOF. The policy plugin process may have crashed or been forcibly terminated. Policy evaluation cannot be completed for this resource.",
					policy.PolicyErrorResult,
				)},
			}
		}
		// Second call: block until the per-call context is cancelled.
		<-ctx.Done()
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"rpc error",
				fmt.Sprintf("rpc failed: %v", ctx.Err()),
				policy.PolicyErrorResult,
			)},
		}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(),
		policy.EvalTimeouts{PerCall: 1 * time.Millisecond, Overall: 0},
		mockClient,
	)
	evalCtx.HookHook = hook

	n0 := makeTimeoutTestNode(0) // crash
	n1 := makeTimeoutTestNode(1) // timeout

	n0.Execute(evalCtx, walkPlan) //nolint:errcheck
	n1.Execute(evalCtx, walkPlan) //nolint:errcheck

	// n0: crash
	if resp, ok := hook.PolicyResults[n0.ResourceAddr.String()]; !ok {
		t.Error("n0: missing PolicyResult")
	} else {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("n0: Overall = %v, want PolicyErrorResult (crash)", resp.Overall)
		}
		found := false
		for _, d := range resp.Diagnostics {
			if d.Description().Summary == "Policy plugin crashed" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("n0: no 'Policy plugin crashed' diagnostic; diags: %v", resp.Diagnostics)
		}
	}

	// n1: timeout — must carry the per-call timeout summary, not the crash summary.
	if resp, ok := hook.PolicyResults[n1.ResourceAddr.String()]; !ok {
		t.Error("n1: missing PolicyResult")
	} else {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("n1: Overall = %v, want PolicyErrorResult (timeout)", resp.Overall)
		}
		// Positive assertion: at least one diagnostic must carry the timeout summary.
		foundTimeout := false
		for _, d := range resp.Diagnostics {
			if d.Description().Summary == "Policy plugin crashed" {
				t.Errorf("n1: timeout result incorrectly labelled as 'Policy plugin crashed'")
			}
			if strings.Contains(d.Description().Summary, "timed out") {
				foundTimeout = true
			}
		}
		if !foundTimeout {
			t.Errorf("n1: no 'timed out' summary on timeout result; diags: %v", resp.Diagnostics)
		}
	}
}

// TestPolicyCrash_RaceDetector exercises concurrent crash responses to
// verify no data races occur in the production code paths.
// Run with: go test -race ./internal/terraform/...
func TestPolicyCrash_RaceDetector(t *testing.T) {
	sharedHook := &testHook{}
	ps := newPolicySubgraph()

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			mc := &policy.MockClient{
				EvaluateFn: crashEvaluateFn(t),
			}

			// Use executeTestCtx for a full provider/schema setup.
			fullCtx := executeTestCtx(t, mc, sharedHook)
			fullCtx.PolicyGraphValue = ps // share the subgraph

			n := makeTimeoutTestNode(idx)
			n.Execute(fullCtx, walkPlan) //nolint:errcheck
		}(i)
	}
	wg.Wait()

	if len(sharedHook.PolicyResults) != goroutines {
		t.Errorf("expected %d PolicyResult hook calls, got %d", goroutines, len(sharedHook.PolicyResults))
	}
	for addr, resp := range sharedHook.PolicyResults {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("%s: expected PolicyErrorResult for crash, got %v", addr, resp.Overall)
		}
	}
}
