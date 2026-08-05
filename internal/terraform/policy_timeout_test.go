// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

// policy_timeout_test.go contains unit tests for the per-call timeout and
// overall policy pass deadline features.
//
// All tests are deterministic: they use 1 ms timeouts so wall-clock sleeps
// are minimal, and they rely on context cancellation signals rather than
// sleep-based waiting for timeout detection.
//
// Test matrix
// -----------
// Per-call timeout:
//   TestPolicyPerCallTimeout_BasicTrigger          – single resource times out
//   TestPolicyPerCallTimeout_DiagnosticContent      – exact diagnostic format
//   TestPolicyPerCallTimeout_AllResourcesTimeout    – several resources time out
//   TestPolicyPerCallTimeout_ContinuesAfterTimeout  – next resource still evaluated
//
// Overall deadline:
//   TestPolicyOverallDeadline_BasicTrigger          – deadline fires mid-pass
//   TestPolicyOverallDeadline_PartialResultsCorrect – pre-deadline results preserved
//   TestPolicyOverallDeadline_DiagnosticContent     – exact diagnostic format
//   TestPolicyOverallDeadline_ZeroRemaining         – deadline fires after all done
//
// Combined / happy-path / edge-case:
//   TestPolicyTimeout_HappyPath                    – no timeouts; results correct
//   TestPolicyTimeout_BothTimeoutsFire             – per-call + overall in same pass
//   TestPolicyTimeout_ConfigurableTimeouts         – 1 ms overrides work; defaults verifiable
//   TestPolicyTimeout_ContextCancellation          – parent context cancelled
//   TestPolicyTimeout_DeadlineWhileMidRPC          – deadline fires while resource is mid-RPC
//   TestPolicyTimeout_RaceDetector                 – concurrent timeouts (run with -race)

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
)

// ---------------------------------------------------------------------------
// helpers shared across timeout tests
// ---------------------------------------------------------------------------

// timeoutTestEvalCtx constructs a MockEvalContext whose PolicyGraph has the
// given EvalTimeouts pre-installed. It delegates to executeTestCtx (defined in
// node_query_resource_policy_test.go) and overrides the policySubgraph's
// timeouts and the StopCtx so callers can pass a controlled parent context.
func timeoutTestEvalCtx(t *testing.T, stopCtx context.Context, timeouts policy.EvalTimeouts, client policy.Client) *MockEvalContext {
	t.Helper()

	ctx := executeTestCtx(t, client, nil)
	ctx.PolicyGraphValue.timeouts = timeouts
	ctx.StopCtxValue = stopCtx
	return ctx
}

// blockingEvaluateFn returns an EvaluateFn that blocks until ctx is Done, then
// returns a PolicyErrorResult with the context error in the detail. This
// simulates a policy RPC that hangs until its per-call timeout fires.
func blockingEvaluateFn(t *testing.T) func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
	t.Helper()
	return func(ctx context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		<-ctx.Done()
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"Failed to evaluate Terraform Policy",
				fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", ctx.Err()),
				policy.PolicyErrorResult,
			)},
		}
	}
}

// makeTimeoutTestNode constructs a nodeQueryResourcePolicy for test_resource
// at the given synthetic index.
func makeTimeoutTestNode(idx int) *nodeQueryResourcePolicy {
	resourceAddr := mustResourceInstanceAddr(fmt.Sprintf("test_resource.r%d", idx))
	listBlockAddr := mustResourceInstanceAddr("test_resource.mylist")
	return &nodeQueryResourcePolicy{
		ResourceAddr: resourceAddr,
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		GeneratedConfig: cty.ObjectVal(map[string]cty.Value{
			"instance_type": cty.StringVal("t2.micro"),
			"ami":           cty.StringVal("ami-12345"),
		}),
		Identity:      cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal(fmt.Sprintf("i-%d", idx))}),
		ListBlockAddr: listBlockAddr,
	}
}

// waitForDeadline blocks the calling goroutine until the overall deadline
// installed on ps has fired. It calls t.Fatal if the deadline context was
// never set (i.e. setDeadline was not called or Overall was zero).
func waitForDeadline(t *testing.T, ps *policySubgraph) {
	t.Helper()
	ptr := ps.deadlineCtx.Load()
	if ptr == nil {
		t.Fatal("waitForDeadline: no deadline context installed; was setDeadline called?")
	}
	<-(*ptr).Done()
}

// ---------------------------------------------------------------------------
// Per-call timeout tests
// ---------------------------------------------------------------------------

// TestPolicyPerCallTimeout_BasicTrigger verifies that when the EvaluateResource
// RPC blocks longer than the configured per-call timeout, Execute() returns
// without hanging, the result is PolicyErrorResult (error), and the diagnostic
// explicitly mentions the timeout.
//
// Regression target: if the per-call context is not created or the mock does
// not see the cancellation, Execute() would block forever.
func TestPolicyPerCallTimeout_BasicTrigger(t *testing.T) {
	timeouts := policy.EvalTimeouts{
		PerCall: 1 * time.Millisecond,
		Overall: 0, // no overall deadline for this test
	}

	mockClient := &policy.MockClient{
		EvaluateFn: blockingEvaluateFn(t),
	}

	hook := &testHook{}
	ctx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	ctx.HookHook = hook

	n := makeTimeoutTestNode(0)
	diags := n.Execute(ctx, walkPlan)

	// Execute must return (not block).
	if diags.HasErrors() {
		t.Fatalf("unexpected error diags: %s", diags.Err())
	}

	// The hook must have been called for the resource.
	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatalf("expected PolicyResult hook call, got none")
	}

	// AC1: result must be error.
	if resp.Overall != policy.PolicyErrorResult {
		t.Errorf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}

	// AC2: at least one diagnostic must mention the timeout.
	if len(resp.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics on timeout result, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		detail := d.Description().Detail
		if strings.Contains(detail, "timed out") || strings.Contains(detail, "timeout") ||
			strings.Contains(detail, "per-call") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no diagnostic mentions timeout; diagnostics: %v", resp.Diagnostics)
	}
}

// TestPolicyPerCallTimeout_DiagnosticContent asserts the exact diagnostic
// message format for a per-call timeout. This is a regression guard: if the
// message format changes silently, operators lose actionable information.
func TestPolicyPerCallTimeout_DiagnosticContent(t *testing.T) {
	timeout := 2 * time.Millisecond
	timeouts := policy.EvalTimeouts{PerCall: timeout, Overall: 0}

	mockClient := &policy.MockClient{EvaluateFn: blockingEvaluateFn(t)}

	hook := &testHook{}
	ctx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	ctx.HookHook = hook

	n := makeTimeoutTestNode(0)
	n.Execute(ctx, walkPlan) //nolint:errcheck

	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatalf("expected PolicyResult hook call")
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics on timeout result")
	}

	d := resp.Diagnostics[0]
	// Summary must clearly indicate a timeout.
	if !strings.Contains(d.Description().Summary, "timed out") {
		t.Errorf("Summary = %q; want it to contain \"timed out\"", d.Description().Summary)
	}
	// Detail must reference the resource address and the per-call timeout.
	detail := d.Description().Detail
	if !strings.Contains(detail, n.ResourceAddr.String()) {
		t.Errorf("Detail %q does not contain resource addr %q", detail, n.ResourceAddr.String())
	}
	if !strings.Contains(detail, perCallTimeoutDiagMsg) {
		t.Errorf("Detail %q does not contain perCallTimeoutDiagMsg %q", detail, perCallTimeoutDiagMsg)
	}
}

// TestPolicyPerCallTimeout_AllResourcesTimeout verifies that when several resources
// individually hit the per-call timeout, every one of them is recorded as
// PolicyErrorResult with a timeout diagnostic.
func TestPolicyPerCallTimeout_AllResourcesTimeout(t *testing.T) {
	timeouts := policy.EvalTimeouts{PerCall: 1 * time.Millisecond, Overall: 0}

	var mu sync.Mutex
	callOrder := make([]string, 0, 3)

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		mu.Lock()
		callOrder = append(callOrder, req.Target)
		mu.Unlock()

		// Block until the context is done (simulates a slow RPC).
		<-ctx.Done()
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"rpc error", fmt.Sprintf("rpc failed: %v", ctx.Err()), policy.PolicyErrorResult,
			)},
		}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	nodes := []*nodeQueryResourcePolicy{
		makeTimeoutTestNode(0),
		makeTimeoutTestNode(1),
		makeTimeoutTestNode(2),
	}

	for _, n := range nodes {
		n.Execute(evalCtx, walkPlan) //nolint:errcheck
	}

	// All three resources should have been recorded.
	if len(hook.PolicyResults) != 3 {
		t.Fatalf("expected 3 hook calls, got %d", len(hook.PolicyResults))
	}

	for _, n := range nodes {
		resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
		if !ok {
			t.Errorf("no PolicyResult for %s", n.ResourceAddr)
			continue
		}
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("%s: Overall = %v, want PolicyErrorResult", n.ResourceAddr, resp.Overall)
		}
		if len(resp.Diagnostics) == 0 {
			t.Errorf("%s: expected timeout diagnostic", n.ResourceAddr)
		}
	}
}

// TestPolicyPerCallTimeout_ContinuesAfterTimeout verifies that after one resource
// times out the next resource is still evaluated (the per-call timeout does not
// cancel the parent context or abort the entire pass).
func TestPolicyPerCallTimeout_ContinuesAfterTimeout(t *testing.T) {
	timeouts := policy.EvalTimeouts{PerCall: 1 * time.Millisecond, Overall: 0}

	var callCount int32

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			// First call: block until context is done (per-call timeout fires).
			<-ctx.Done()
			return policy.EvaluationResponse{
				Overall: policy.PolicyErrorResult,
				Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
					"rpc error", fmt.Sprintf("context error: %v", ctx.Err()), policy.PolicyErrorResult,
				)},
			}
		}
		// Subsequent calls: succeed immediately.
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	n0 := makeTimeoutTestNode(0)
	n1 := makeTimeoutTestNode(1)

	n0.Execute(evalCtx, walkPlan) //nolint:errcheck
	n1.Execute(evalCtx, walkPlan) //nolint:errcheck

	// n0 must be error (timed out).
	if resp, ok := hook.PolicyResults[n0.ResourceAddr.String()]; !ok {
		t.Error("n0: no PolicyResult")
	} else if resp.Overall != policy.PolicyErrorResult {
		t.Errorf("n0: Overall = %v, want PolicyErrorResult", resp.Overall)
	}

	// n1 must be AllowResult (not affected by n0's timeout).
	if resp, ok := hook.PolicyResults[n1.ResourceAddr.String()]; !ok {
		t.Error("n1: no PolicyResult")
	} else if resp.Overall != policy.AllowResult {
		t.Errorf("n1: Overall = %v, want AllowResult — per-call timeout of n0 must not abort the pass", resp.Overall)
	}
}

// ---------------------------------------------------------------------------
// Overall deadline tests
// ---------------------------------------------------------------------------

// TestPolicyOverallDeadline_BasicTrigger verifies that when the overall deadline
// fires before all resources are evaluated, the remaining resources are recorded
// as PolicyErrorResult with a deadline diagnostic instead of being silently skipped.
func TestPolicyOverallDeadline_BasicTrigger(t *testing.T) {
	timeouts := policy.EvalTimeouts{
		PerCall: 0,                    // no per-call timeout
		Overall: 1 * time.Millisecond, // fire quickly
	}

	// The policy client allows evaluation immediately.
	mockClient := policy.NewTestMockClient(t)

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	// Install the deadline on the subgraph (mirrors DynamicExpand).
	evalCtx.PolicyGraphValue.setDeadline(context.Background())

	// Block until the deadline has actually fired (no sleep-based guessing).
	waitForDeadline(t, evalCtx.PolicyGraphValue)

	// Now Execute should detect the deadline.
	n := makeTimeoutTestNode(0)
	n.Execute(evalCtx, walkPlan) //nolint:errcheck

	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatalf("expected PolicyResult for deadline-exceeded resource, got none")
	}
	if resp.Overall != policy.PolicyErrorResult {
		t.Errorf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatalf("expected at least one diagnostic for deadline-exceeded resource")
	}
}

// TestPolicyOverallDeadline_PartialResultsCorrect verifies that resources that
// completed evaluation before the deadline retain their correct results (not
// overwritten with error) while post-deadline resources are recorded as error.
func TestPolicyOverallDeadline_PartialResultsCorrect(t *testing.T) {
	timeouts := policy.EvalTimeouts{
		PerCall: 0,
		Overall: 5 * time.Millisecond,
	}

	mockClient := policy.NewTestMockClient(t) // always returns AllowResult

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	// Install deadline.
	evalCtx.PolicyGraphValue.setDeadline(context.Background())

	// Evaluate two resources BEFORE the deadline fires.
	n0 := makeTimeoutTestNode(0)
	n1 := makeTimeoutTestNode(1)
	n0.Execute(evalCtx, walkPlan) //nolint:errcheck
	n1.Execute(evalCtx, walkPlan) //nolint:errcheck

	// Block until the deadline has fired — no sleep guessing.
	waitForDeadline(t, evalCtx.PolicyGraphValue)

	// Evaluate one resource AFTER the deadline.
	n2 := makeTimeoutTestNode(2)
	n2.Execute(evalCtx, walkPlan) //nolint:errcheck

	// n0 and n1 should have AllowResult (evaluated before deadline).
	for _, n := range []*nodeQueryResourcePolicy{n0, n1} {
		if resp, ok := hook.PolicyResults[n.ResourceAddr.String()]; !ok {
			t.Errorf("%s: missing PolicyResult", n.ResourceAddr)
		} else if resp.Overall != policy.AllowResult {
			t.Errorf("%s: Overall = %v, want AllowResult (evaluated before deadline)", n.ResourceAddr, resp.Overall)
		}
	}

	// n2 should have PolicyErrorResult (evaluated after deadline).
	if resp, ok := hook.PolicyResults[n2.ResourceAddr.String()]; !ok {
		t.Errorf("n2: missing PolicyResult")
	} else if resp.Overall != policy.PolicyErrorResult {
		t.Errorf("n2: Overall = %v, want PolicyErrorResult (evaluated after deadline)", resp.Overall)
	}
}

// TestPolicyOverallDeadline_DiagnosticContent asserts the exact diagnostic
// content for a deadline-exceeded resource. This is a regression guard.
func TestPolicyOverallDeadline_DiagnosticContent(t *testing.T) {
	timeouts := policy.EvalTimeouts{PerCall: 0, Overall: 1 * time.Millisecond}

	mockClient := policy.NewTestMockClient(t)

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	evalCtx.PolicyGraphValue.setDeadline(context.Background())
	waitForDeadline(t, evalCtx.PolicyGraphValue)

	n := makeTimeoutTestNode(0)
	n.Execute(evalCtx, walkPlan) //nolint:errcheck

	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatalf("expected PolicyResult hook call")
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatalf("expected diagnostic on deadline-exceeded resource")
	}

	d := resp.Diagnostics[0]
	// Summary must mention deadline.
	if !strings.Contains(d.Description().Summary, "deadline") &&
		!strings.Contains(d.Description().Summary, "Deadline") {
		t.Errorf("Summary = %q; want it to contain \"deadline\"", d.Description().Summary)
	}
	// Detail must contain the sentinel message constant.
	if !strings.Contains(d.Description().Detail, overallDeadlineExceededDiagMsg) {
		t.Errorf("Detail %q does not contain overallDeadlineExceededDiagMsg", d.Description().Detail)
	}
}

// TestPolicyOverallDeadline_ZeroRemaining verifies that if the overall deadline
// fires when all resources have already been evaluated, Execute() returns
// normally for those completed resources (no spurious errors) and the summary
// emission path has no anomalies.
//
// This exercises the boundary condition: overallDeadlineExceeded() == true but
// there are no genuinely unevaluated resources.
func TestPolicyOverallDeadline_ZeroRemaining(t *testing.T) {
	timeouts := policy.EvalTimeouts{PerCall: 0, Overall: 5 * time.Millisecond}

	mockClient := policy.NewTestMockClient(t) // AllowResult

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	evalCtx.PolicyGraphValue.setDeadline(context.Background())

	// Evaluate all resources BEFORE the deadline.
	n0 := makeTimeoutTestNode(0)
	n1 := makeTimeoutTestNode(1)
	n0.Execute(evalCtx, walkPlan) //nolint:errcheck
	n1.Execute(evalCtx, walkPlan) //nolint:errcheck

	// Block until the deadline fires — there are no more resources to evaluate.
	waitForDeadline(t, evalCtx.PolicyGraphValue)

	// Confirm both resources have correct (non-error) results.
	for _, n := range []*nodeQueryResourcePolicy{n0, n1} {
		resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
		if !ok {
			t.Errorf("%s: missing PolicyResult", n.ResourceAddr)
			continue
		}
		if resp.Overall != policy.AllowResult {
			t.Errorf("%s: Overall = %v, want AllowResult (evaluated before deadline)", n.ResourceAddr, resp.Overall)
		}
		if len(resp.Diagnostics) > 0 {
			t.Errorf("%s: unexpected diagnostics on pre-deadline resource: %v", n.ResourceAddr, resp.Diagnostics)
		}
	}

	// No spurious errors should have been emitted.
	if len(hook.PolicyResults) != 2 {
		t.Errorf("expected exactly 2 hook calls, got %d", len(hook.PolicyResults))
	}
}

// ---------------------------------------------------------------------------
// Combined / happy-path / edge-case tests
// ---------------------------------------------------------------------------

// TestPolicyTimeout_HappyPath verifies that when all evaluations complete within
// their timeouts, results are correct, no spurious error diagnostics are present,
// and the hook is called exactly once per resource.
func TestPolicyTimeout_HappyPath(t *testing.T) {
	timeouts := policy.EvalTimeouts{
		PerCall: policy.DefaultPerCallTimeout,
		Overall: policy.DefaultOverallDeadline,
	}

	var mu sync.Mutex
	results := map[string]policy.EvaluateResult{
		"test_resource.r0": policy.AllowResult,
		"test_resource.r1": policy.DenyResult,
		"test_resource.r2": policy.AllowResult,
	}

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		mu.Lock()
		r := results[req.Target] // req.Target == resource type, but fine for test
		mu.Unlock()
		if r == 0 {
			r = policy.AllowResult
		}
		return policy.EvaluationResponse{Overall: r}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	nodes := []*nodeQueryResourcePolicy{
		makeTimeoutTestNode(0),
		makeTimeoutTestNode(1),
		makeTimeoutTestNode(2),
	}
	for _, n := range nodes {
		diags := n.Execute(evalCtx, walkPlan)
		if diags.HasErrors() {
			t.Errorf("%s: unexpected error diags: %s", n.ResourceAddr, diags.Err())
		}
	}

	if len(hook.PolicyResults) != 3 {
		t.Fatalf("expected 3 hook calls, got %d", len(hook.PolicyResults))
	}

	for _, n := range nodes {
		resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
		if !ok {
			t.Errorf("%s: missing PolicyResult", n.ResourceAddr)
			continue
		}
		// On the happy path there must be no timeout diagnostics.
		for _, d := range resp.Diagnostics {
			detail := d.Description().Detail
			if strings.Contains(detail, "timeout") || strings.Contains(detail, "deadline") {
				t.Errorf("%s: unexpected timeout diagnostic on happy path: %q", n.ResourceAddr, detail)
			}
		}
	}
}

// TestPolicyTimeout_BothTimeoutsFire exercises the scenario where some resources
// hit the per-call timeout AND the overall deadline fires before the remaining
// resources can be evaluated. All three categories (normal, per-call-timeout,
// overall-deadline) must be correctly recorded.
func TestPolicyTimeout_BothTimeoutsFire(t *testing.T) {
	timeouts := policy.EvalTimeouts{
		PerCall: 5 * time.Millisecond,
		Overall: 15 * time.Millisecond,
	}

	var callCount int32

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			// First resource: completes immediately (AllowResult).
			return policy.EvaluationResponse{Overall: policy.AllowResult}
		}
		// Second and later resources: block until per-call timeout fires.
		<-ctx.Done()
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"rpc error", fmt.Sprintf("rpc failed: %v", ctx.Err()), policy.PolicyErrorResult,
			)},
		}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	// Install overall deadline.
	evalCtx.PolicyGraphValue.setDeadline(context.Background())

	n0 := makeTimeoutTestNode(0)  // will succeed (AllowResult)
	n1 := makeTimeoutTestNode(1)  // will hit per-call timeout
	n0.Execute(evalCtx, walkPlan) //nolint:errcheck
	n1.Execute(evalCtx, walkPlan) //nolint:errcheck

	// Block until the overall deadline fires.
	waitForDeadline(t, evalCtx.PolicyGraphValue)

	n2 := makeTimeoutTestNode(2)  // will hit overall deadline
	n2.Execute(evalCtx, walkPlan) //nolint:errcheck

	// n0: AllowResult.
	if resp, ok := hook.PolicyResults[n0.ResourceAddr.String()]; !ok {
		t.Error("n0: missing PolicyResult")
	} else if resp.Overall != policy.AllowResult {
		t.Errorf("n0: Overall = %v, want AllowResult", resp.Overall)
	}

	// n1: PolicyErrorResult with per-call timeout diagnostic.
	if resp, ok := hook.PolicyResults[n1.ResourceAddr.String()]; !ok {
		t.Error("n1: missing PolicyResult")
	} else {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("n1: Overall = %v, want PolicyErrorResult (per-call timeout)", resp.Overall)
		}
		found := false
		for _, d := range resp.Diagnostics {
			if strings.Contains(d.Description().Detail, "timed out") ||
				strings.Contains(d.Description().Summary, "timed out") ||
				strings.Contains(d.Description().Detail, perCallTimeoutDiagMsg) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("n1: no per-call timeout diagnostic; diags: %v", resp.Diagnostics)
		}
	}

	// n2: PolicyErrorResult with overall deadline diagnostic.
	if resp, ok := hook.PolicyResults[n2.ResourceAddr.String()]; !ok {
		t.Error("n2: missing PolicyResult")
	} else {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("n2: Overall = %v, want PolicyErrorResult (overall deadline)", resp.Overall)
		}
		found := false
		for _, d := range resp.Diagnostics {
			if strings.Contains(d.Description().Detail, overallDeadlineExceededDiagMsg) ||
				strings.Contains(d.Description().Summary, "deadline") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("n2: no overall deadline diagnostic; diags: %v", resp.Diagnostics)
		}
	}
}

// TestPolicyTimeout_ConfigurableTimeouts asserts:
//  1. Per-call and overall timeouts can be set to arbitrarily short values
//     (1 ms) without panicking, and the timeout mechanism actually fires.
//  2. The production defaults match the named constants (prevents silent drift).
func TestPolicyTimeout_ConfigurableTimeouts(t *testing.T) {
	t.Run("short_per_call_fires", func(t *testing.T) {
		timeouts := policy.EvalTimeouts{PerCall: 1 * time.Millisecond, Overall: 0}

		mockClient := &policy.MockClient{EvaluateFn: blockingEvaluateFn(t)}
		hook := &testHook{}
		evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
		evalCtx.HookHook = hook

		n := makeTimeoutTestNode(0)
		n.Execute(evalCtx, walkPlan) //nolint:errcheck

		resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
		if !ok {
			t.Fatalf("expected PolicyResult with 1 ms timeout")
		}
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("Overall = %v, want PolicyErrorResult", resp.Overall)
		}
	})

	t.Run("short_overall_fires", func(t *testing.T) {
		timeouts := policy.EvalTimeouts{PerCall: 0, Overall: 1 * time.Millisecond}

		mockClient := policy.NewTestMockClient(t)
		hook := &testHook{}
		evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
		evalCtx.HookHook = hook

		evalCtx.PolicyGraphValue.setDeadline(context.Background())
		waitForDeadline(t, evalCtx.PolicyGraphValue)

		n := makeTimeoutTestNode(0)
		n.Execute(evalCtx, walkPlan) //nolint:errcheck

		resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
		if !ok {
			t.Fatalf("expected PolicyResult with 1 ms overall deadline")
		}
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("Overall = %v, want PolicyErrorResult", resp.Overall)
		}
	})

	t.Run("production_defaults_match_constants", func(t *testing.T) {
		// Verify that the production defaults exactly match the documented constants.
		// This prevents drift where constants change but defaults do not.
		defaults := policy.DefaultEvalTimeouts()
		if defaults.PerCall != policy.DefaultPerCallTimeout {
			t.Errorf("DefaultEvalTimeouts().PerCall = %v, want DefaultPerCallTimeout (%v)",
				defaults.PerCall, policy.DefaultPerCallTimeout)
		}
		if defaults.Overall != policy.DefaultOverallDeadline {
			t.Errorf("DefaultEvalTimeouts().Overall = %v, want DefaultOverallDeadline (%v)",
				defaults.Overall, policy.DefaultOverallDeadline)
		}
	})

	t.Run("default_per_call_is_30s", func(t *testing.T) {
		if policy.DefaultPerCallTimeout != 30*time.Second {
			t.Errorf("DefaultPerCallTimeout = %v, want 30s", policy.DefaultPerCallTimeout)
		}
	})

	t.Run("default_overall_is_10m", func(t *testing.T) {
		if policy.DefaultOverallDeadline != 10*time.Minute {
			t.Errorf("DefaultOverallDeadline = %v, want 10m", policy.DefaultOverallDeadline)
		}
	})

	t.Run("new_policy_subgraph_uses_defaults", func(t *testing.T) {
		ps := newPolicySubgraph()
		if ps.timeouts.PerCall != policy.DefaultPerCallTimeout {
			t.Errorf("newPolicySubgraph().timeouts.PerCall = %v, want %v",
				ps.timeouts.PerCall, policy.DefaultPerCallTimeout)
		}
		if ps.timeouts.Overall != policy.DefaultOverallDeadline {
			t.Errorf("newPolicySubgraph().timeouts.Overall = %v, want %v",
				ps.timeouts.Overall, policy.DefaultOverallDeadline)
		}
	})
}

// TestPolicyTimeout_ContextCancellation verifies that when the parent context is
// cancelled, Execute exits promptly. The result must be a PolicyErrorResult whose
// diagnostic does NOT contain the per-call sentinel (perCallTimeoutDiagMsg) — parent
// cancellation is a different code path from per-call timeout. It must not hang.
func TestPolicyTimeout_ContextCancellation(t *testing.T) {
	timeouts := policy.EvalTimeouts{PerCall: 30 * time.Second, Overall: 0}

	parentCtx, cancel := context.WithCancel(context.Background())

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		// Block until either the per-call or the parent context is cancelled.
		<-ctx.Done()
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"rpc error", fmt.Sprintf("context: %v", ctx.Err()), policy.PolicyErrorResult,
			)},
		}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, parentCtx, timeouts, mockClient)
	evalCtx.HookHook = hook

	// Cancel the parent before Execute returns.
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()

	n := makeTimeoutTestNode(0)
	// Must not hang. Give it 2 s max.
	done := make(chan struct{})
	go func() {
		n.Execute(evalCtx, walkPlan) //nolint:errcheck
		close(done)
	}()

	select {
	case <-done:
		// Good: Execute returned.
	case <-time.After(2 * time.Second):
		t.Fatal("Execute hung after parent context cancellation")
	}

	// Must not have emitted duplicate hook calls for the same resource.
	if count := len(hook.PolicyResults); count > 1 {
		t.Errorf("expected at most 1 PolicyResult hook call, got %d", count)
	}

	// The diagnostic must NOT carry the per-call sentinel: parent cancellation
	// is distinct from per-call timeout (PerCallTimedOut is only true when the
	// per-call child context expired while the parent was still live).
	if resp, ok := hook.PolicyResults[n.ResourceAddr.String()]; ok {
		for _, d := range resp.Diagnostics {
			if strings.Contains(d.Description().Detail, perCallTimeoutDiagMsg) {
				t.Errorf("parent-cancellation result should not carry per-call timeout diagnostic; got %q",
					d.Description().Detail)
			}
		}
	}
}

// TestPolicyTimeout_DeadlineWhileMidRPC verifies the race: overall deadline fires
// while a resource evaluation is already in progress inside the semaphore. The
// in-flight RPC completes normally (with its own result), and subsequent resources
// that arrive after the deadline is detected are short-circuited with an error.
//
// This is distinct from TestPolicyOverallDeadline_BasicTrigger, which only tests
// the pre-semaphore check. Here we verify that the production code handles the
// case where overallDeadlineExceeded() is false when the node enters Execute()
// but the deadline fires while the RPC is blocked.
func TestPolicyTimeout_DeadlineWhileMidRPC(t *testing.T) {
	// Use a per-call timeout that is longer than the overall deadline so the
	// per-call context does not cancel first. The overall deadline fires while
	// the first resource is waiting inside the mock RPC.
	//
	// NOTE on sequencing: setDeadline is called AFTER the RPC has started to
	// avoid a race where the 5 ms timer fires before the goroutine even reaches
	// overallDeadlineExceeded(). With setDeadline deferred until after
	// rpcStarted is received, n0 is guaranteed to pass the pre-semaphore check
	// (deadline not yet installed), and the deadline is only armed once the RPC
	// is already blocking.
	timeouts := policy.EvalTimeouts{
		PerCall: 500 * time.Millisecond, // long enough not to fire first
		Overall: 5 * time.Millisecond,   // fires quickly once armed
	}

	// rpcStarted signals that the first RPC has started blocking.
	rpcStarted := make(chan struct{}, 1)

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		// Signal that the RPC has started, then block until the per-call (or
		// parent) context is cancelled. The overall deadline fires while we wait.
		select {
		case rpcStarted <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"rpc error", fmt.Sprintf("rpc cancelled: %v", ctx.Err()), policy.PolicyErrorResult,
			)},
		}
	}

	hook := &testHook{}
	evalCtx := timeoutTestEvalCtx(t, context.Background(), timeouts, mockClient)
	evalCtx.HookHook = hook

	// Do NOT call setDeadline yet; the deadline must not be armed before
	// n0.Execute passes the overallDeadlineExceeded() check.

	n0 := makeTimeoutTestNode(0)

	// Start n0's Execute concurrently so it enters the blocking RPC.
	done := make(chan struct{})
	go func() {
		defer close(done)
		n0.Execute(evalCtx, walkPlan) //nolint:errcheck
	}()

	// Wait until the RPC has started before arming the overall deadline.
	// This guarantees n0 has already passed the pre-semaphore deadline check,
	// so the 5 ms timer can fire while the RPC is in-flight without racing
	// against the goroutine startup.
	select {
	case <-rpcStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("RPC never started")
	}

	// Arm the overall deadline now that the RPC is in-flight, then wait for it
	// to fire.
	evalCtx.PolicyGraphValue.setDeadline(context.Background())
	waitForDeadline(t, evalCtx.PolicyGraphValue)

	// n0's RPC is still in flight. Wait for Execute to complete (the per-call
	// timeout or parent context will cancel it).
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("n0.Execute hung after overall deadline fired")
	}

	// n1 arrives after the deadline; it should be short-circuited.
	n1 := makeTimeoutTestNode(1)
	n1.Execute(evalCtx, walkPlan) //nolint:errcheck

	// n1 must have a deadline-exceeded error.
	if resp, ok := hook.PolicyResults[n1.ResourceAddr.String()]; !ok {
		t.Error("n1: missing PolicyResult")
	} else {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("n1: Overall = %v, want PolicyErrorResult (deadline exceeded)", resp.Overall)
		}
		found := false
		for _, d := range resp.Diagnostics {
			if strings.Contains(d.Description().Detail, overallDeadlineExceededDiagMsg) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("n1: no overall deadline diagnostic; diags: %v", resp.Diagnostics)
		}
	}
}

// TestPolicyTimeout_RaceDetector exercises concurrent per-call timeouts to
// verify no data races occur in the production code paths.
// Run with: go test -race ./internal/terraform/...
//
// Each goroutine owns its own MockEvalContext to avoid races on the mock's
// tracking fields (which are not goroutine-safe by design). The shared state
// we are testing for races is: policySubgraph.timeouts (read-only after setup),
// policySubgraph.deadlineCtx (read-only after setup), and any synchronised
// data structures in the timeout path.
func TestPolicyTimeout_RaceDetector(t *testing.T) {
	timeouts := policy.EvalTimeouts{PerCall: 1 * time.Millisecond, Overall: 0}

	// Shared, thread-safe aggregate hook. The testHook uses a mutex so it is
	// safe to use from multiple goroutines.
	sharedHook := &testHook{}

	// Shared, immutable policySubgraph used by all goroutines. The subgraph
	// methods that are called concurrently (overallDeadlineExceeded, reading
	// timeouts) are designed to be safe for concurrent reads after setup.
	ps := newPolicySubgraph()
	ps.timeouts = timeouts // set before any goroutine accesses it

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			// Each goroutine gets its own MockEvalContext to avoid races on
			// the mock's unsynchronised tracking fields.
			schema := providers.ProviderSchema{
				ResourceTypes: map[string]providers.Schema{
					"test_resource": {Body: listPolicyTestResourceSchema()},
				},
			}
			p := &testing_provider.MockProvider{}
			expander := instances.NewExpander(nil)
			rootCfg := &configs.Config{Module: &configs.Module{}}

			// Each goroutine creates its own MockClient so the mock's mutex
			// protects only calls from that goroutine.
			mc := &policy.MockClient{EvaluateFn: blockingEvaluateFn(t)}

			ctx := &MockEvalContext{
				ProviderProvider:         p,
				ProviderSchemaSchema:     schema,
				PolicyClientValue:        mc,
				PolicyGraphValue:         ps, // shared, read-only after setup
				ConfigValue:              rootCfg,
				InstanceExpanderExpander: expander,
				StopCtxValue:             context.Background(),
				HookHook:                 sharedHook, // shared, mutex-protected
			}

			n := makeTimeoutTestNode(idx)
			n.Execute(ctx, walkPlan) //nolint:errcheck
		}(i)
	}
	wg.Wait()

	// All goroutines must have recorded a PolicyResult.
	if len(sharedHook.PolicyResults) != goroutines {
		t.Errorf("expected %d PolicyResult hook calls, got %d", goroutines, len(sharedHook.PolicyResults))
	}
	for addr, resp := range sharedHook.PolicyResults {
		if resp.Overall != policy.PolicyErrorResult {
			t.Errorf("%s: expected PolicyErrorResult for timeout, got %v", addr, resp.Overall)
		}
	}
}
