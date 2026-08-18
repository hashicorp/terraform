// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"runtime"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
)

// TestNodeQueryResourcePolicy_Name verifies that Name() returns the resource
// address followed by the "(query policy evaluation)" suffix.
func TestNodeQueryResourcePolicy_Name(t *testing.T) {
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	n := &nodeQueryResourcePolicy{ResourceAddr: addr}
	want := addr.String() + " (query policy evaluation)"
	if got := n.Name(); got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

// TestGetPolicyParallelism_Default verifies that getPolicyParallelism returns
// GOMAXPROCS when TF_POLICY_PARALLELISM is not set.
func TestGetPolicyParallelism_Default(t *testing.T) {
	// Ensure env var is not set; t.Setenv restores the previous value on cleanup.
	t.Setenv("TF_POLICY_PARALLELISM", "")

	capacity := getPolicyParallelism()
	expected := runtime.GOMAXPROCS(0)

	if capacity != expected {
		t.Errorf("Expected capacity %d (GOMAXPROCS), got %d", expected, capacity)
	}
}

// TestGetPolicyParallelism_EnvOverride verifies that TF_POLICY_PARALLELISM
// overrides the default GOMAXPROCS value.
func TestGetPolicyParallelism_EnvOverride(t *testing.T) {
	t.Setenv("TF_POLICY_PARALLELISM", "5")

	capacity := getPolicyParallelism()

	if capacity != 5 {
		t.Errorf("Expected capacity 5 from TF_POLICY_PARALLELISM, got %d", capacity)
	}
}

// TestGetPolicyParallelism_InvalidEnvFallback verifies that invalid
// TF_POLICY_PARALLELISM values fall back to GOMAXPROCS.
func TestGetPolicyParallelism_InvalidEnvFallback(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{"non-numeric", "invalid"},
		{"negative", "-1"},
		{"zero", "0"},
		{"empty", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TF_POLICY_PARALLELISM", tc.value)

			capacity := getPolicyParallelism()
			expected := runtime.GOMAXPROCS(0)

			if capacity != expected {
				t.Errorf("Expected fallback to GOMAXPROCS=%d for invalid value %q, got %d",
					expected, tc.value, capacity)
			}
		})
	}
}

// TestPolicySemaphore_Initialization verifies that the policy semaphore
// is initialized eagerly in NewContext with the correct capacity.
func TestPolicySemaphore_Initialization(t *testing.T) {
	t.Setenv("TF_POLICY_PARALLELISM", "3")

	ctx, diags := NewContext(&ContextOpts{
		Parallelism: 10,
	})
	if diags.HasErrors() {
		t.Fatalf("NewContext failed: %s", diags.Err())
	}

	sem := ctx.policySemaphore()

	// Verify we can acquire up to the capacity
	for i := 0; i < 3; i++ {
		if !sem.TryAcquire() {
			t.Errorf("Failed to acquire semaphore slot %d of 3", i+1)
		}
	}

	// Verify we cannot acquire beyond capacity
	if sem.TryAcquire() {
		t.Error("Should not be able to acquire beyond capacity of 3")
		sem.Release()
	}

	// Release all
	for i := 0; i < 3; i++ {
		sem.Release()
	}
}

// TestPolicySemaphore_SeparateFromProvider verifies that the policy semaphore
// is separate from the provider parallelism semaphore.
func TestPolicySemaphore_SeparateFromProvider(t *testing.T) {
	t.Setenv("TF_POLICY_PARALLELISM", "2")

	ctx, diags := NewContext(&ContextOpts{
		Parallelism: 5, // Different from policy parallelism
	})
	if diags.HasErrors() {
		t.Fatalf("NewContext failed: %s", diags.Err())
	}

	policySem := ctx.policySemaphore()
	providerSem := ctx.parallelSem

	// Verify they have different capacities
	// Policy semaphore should have capacity 2
	acquired1 := policySem.TryAcquire()
	acquired2 := policySem.TryAcquire()
	if !acquired1 || !acquired2 {
		t.Error("Failed to acquire 2 policy semaphore slots")
	}
	if policySem.TryAcquire() {
		t.Error("Should not be able to acquire 3rd policy semaphore slot")
		policySem.Release()
	}

	// Provider semaphore should have capacity 5
	for i := 0; i < 5; i++ {
		if !providerSem.TryAcquire() {
			t.Errorf("Failed to acquire provider semaphore slot %d of 5", i+1)
		}
	}
	if providerSem.TryAcquire() {
		t.Error("Should not be able to acquire 6th provider semaphore slot")
		providerSem.Release()
	}

	// Clean up
	policySem.Release()
	policySem.Release()
	for i := 0; i < 5; i++ {
		providerSem.Release()
	}
}

// --- Execute() tests ---

// executeTestCtx builds a minimal MockEvalContext suitable for nodeQueryResourcePolicy.Execute()
// tests. It wires up a mock provider, schema, policy client, policy graph, and a
// minimal root configs.Config so that config.DescendantForInstance succeeds.
func executeTestCtx(t *testing.T, policyClient policy.Client, hook Hook) *MockEvalContext {
	t.Helper()

	schema := providers.ProviderSchema{
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {Body: listPolicyTestResourceSchema()},
		},
	}

	p := &testing_provider.MockProvider{}

	expander := instances.NewExpander(nil)

	rootCfg := &configs.Config{
		Module: &configs.Module{},
	}

	ps := newPolicySubgraph()

	ctx := &MockEvalContext{
		ProviderProvider:         p,
		ProviderSchemaSchema:     schema,
		PolicyClientValue:        policyClient,
		PolicyGraphValue:         ps,
		PolicySemaphoreValue:     NewSemaphore(1),
		ConfigValue:              rootCfg,
		InstanceExpanderExpander: expander,
		StopCtxValue:             context.Background(),
	}

	if hook != nil {
		ctx.HookHook = hook
	}

	return ctx
}

// makeExecuteNode builds a nodeQueryResourcePolicy with the given resource address
// and identity value, using the shared provider/list block addresses from tests.
func makeExecuteNode(resourceAddr addrs.AbsResourceInstance, identity cty.Value) *nodeQueryResourcePolicy {
	listBlockAddr := mustResourceInstanceAddr("test_resource.mylist")
	providerAddr := addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("test"),
		Module:   addrs.RootModule,
	}
	generatedConfig := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	return &nodeQueryResourcePolicy{
		ResourceAddr:    resourceAddr,
		ProviderAddr:    providerAddr,
		GeneratedConfig: generatedConfig,
		ResourceConfig:  &configs.Resource{},
		Identity:        identity,
		ListBlockAddr:   listBlockAddr,
	}
}

// TestNodeQueryResourcePolicy_Execute_AllowResult verifies that a passing (AllowResult)
// evaluation still emits a PolicyResult hook call. Unlike nodeResourcePolicy (plan/apply),
// query policy nodes always emit so that downstream aggregators can include passing
// resources in summary records. The !result.Empty() gate must NOT be applied here.
func TestNodeQueryResourcePolicy_Execute_AllowResult(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	mockClient := policy.NewTestMockClient(t)
	// Default EvaluateFn: returns AllowResult with no enforcements — result.Empty() == true.
	// For query policy nodes this result must still be emitted through the hook.

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-allow"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// Passing results must still produce a hook call so downstream aggregators can distinguish
	// an unmatched resource from one whose policy evaluation never ran.
	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call for AllowResult (query node always emits), got none")
	}
	if resp.Overall != policy.AllowResult || len(resp.Policies) != 0 {
		t.Fatalf("unexpected unmatched policy response: %#v", resp)
	}
	if got, want := resp.ListBlockAddr, n.ListBlockAddr.String(); got != want {
		t.Fatalf("list block address = %q, want %q", got, want)
	}
	if got, want := resp.Identity["id"], "i-allow"; got != want {
		t.Fatalf("identity id = %q, want %q", got, want)
	}
}

// TestNodeQueryResourcePolicy_Execute_MultiplePolicies verifies that an EvaluationResponse
// with multiple Policy entries and multiple EnforcementResults is propagated correctly.
func TestNodeQueryResourcePolicy_Execute_MultiplePolicies(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	pol1 := &policy.Policy{Result: policy.DenyResult, Address: "policy.deny_type"}
	pol2 := &policy.Policy{Result: policy.AllowResult, Address: "policy.allow_tag"}

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall:  policy.DenyResult,
			Policies: []*policy.Policy{pol1, pol2},
			Enforcements: []policy.EnforcementResult{
				{Result: policy.DenyResult, Message: "instance type not allowed", Policy: pol1},
				{Result: policy.AllowResult, Message: "tag check passed", Policy: pol2},
			},
		}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-multi"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call for multi-policy response, got none")
	}
	if len(resp.Policies) != 2 {
		t.Errorf("expected 2 policies in response, got %d", len(resp.Policies))
	}
	if len(resp.Enforcements) != 2 {
		t.Errorf("expected 2 enforcement results in response, got %d", len(resp.Enforcements))
	}
	if resp.Overall != policy.DenyResult {
		t.Errorf("expected Overall = DenyResult, got %v", resp.Overall)
	}
}

// TestNodeQueryResourcePolicy_Execute_DenyResult verifies that a DenyResult (non-empty)
// causes a PolicyResult hook call with the correct address.
func TestNodeQueryResourcePolicy_Execute_DenyResult(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Enforcements: []policy.EnforcementResult{
				{Result: policy.DenyResult, Message: "denied"},
			},
		}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-deny"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(hook.PolicyResults) == 0 {
		t.Fatal("expected PolicyResult hook call for DenyResult, got none")
	}
	if _, ok := hook.PolicyResults[resourceAddr.String()]; !ok {
		t.Errorf("expected PolicyResult for addr %s, got keys: %v", resourceAddr.String(), hook.PolicyResults)
	}
}

// TestNodeQueryResourcePolicy_Execute_UnknownResult verifies that an UnknownResult
// (non-empty) triggers a hook call.
func TestNodeQueryResourcePolicy_Execute_UnknownResult(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall: policy.UnknownResult,
			Enforcements: []policy.EnforcementResult{
				{Result: policy.UnknownResult},
			},
		}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-unknown"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if _, ok := hook.PolicyResults[resourceAddr.String()]; !ok {
		t.Errorf("expected PolicyResult hook call for UnknownResult, got results: %v", hook.PolicyResults)
	}
}

// TestNodeQueryResourcePolicy_Execute_DenyNoEnforcements verifies that a DenyResult
// with no enforcement details and no diagnostics is NOT dropped by the gate.
// Previously the gate used len(Diagnostics)>0 || len(Enforcements)>0, which would
// silently drop a bare DenyResult. The fix uses !result.Empty().
func TestNodeQueryResourcePolicy_Execute_DenyNoEnforcements(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		// DenyResult with no enforcement details and no diagnostics:
		// Empty() returns false (Overall != AllowResult), but the old
		// len-based gate would have treated it as empty and dropped it.
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
		}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	n := makeExecuteNode(resourceAddr, cty.NilVal)

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if _, ok := hook.PolicyResults[resourceAddr.String()]; !ok {
		t.Errorf("DenyResult with no enforcement details was silently dropped; expected PolicyResult hook call")
	}
}

// TestNodeQueryResourcePolicy_Execute_IdentityAnnotation verifies that the
// EvaluationResponse delivered to the hook is annotated with the structured
// identity map derived from the node's Identity field.
func TestNodeQueryResourcePolicy_Execute_IdentityAnnotation(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Enforcements: []policy.EnforcementResult{
				{Result: policy.DenyResult, Message: "denied"},
			},
		}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id":     cty.StringVal("i-abc"),
		"region": cty.StringVal("us-east-1"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call, got none")
	}

	if resp.Identity == nil {
		t.Fatal("EvaluationResponse.Identity is nil; expected annotated identity map")
	}
	if resp.Identity["id"] != "i-abc" {
		t.Errorf("Identity[\"id\"] = %q, want %q", resp.Identity["id"], "i-abc")
	}
	if resp.Identity["region"] != "us-east-1" {
		t.Errorf("Identity[\"region\"] = %q, want %q", resp.Identity["region"], "us-east-1")
	}
}

// TestNodeQueryResourcePolicy_Execute_ListBlockAddrAnnotation verifies that the
// EvaluationResponse delivered to the hook carries the originating list block address.
func TestNodeQueryResourcePolicy_Execute_ListBlockAddrAnnotation(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")
	expectedListBlockAddr := mustResourceInstanceAddr("test_resource.mylist")

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Enforcements: []policy.EnforcementResult{
				{Result: policy.DenyResult},
			},
		}
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-abc"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call, got none")
	}

	if resp.ListBlockAddr != expectedListBlockAddr.String() {
		t.Errorf("EvaluationResponse.ListBlockAddr = %q, want %q", resp.ListBlockAddr, expectedListBlockAddr.String())
	}
}

// TestNodeQueryResourcePolicy_Execute_ProviderError verifies that a getProvider error
// is returned as a diagnostic rather than a panic.
func TestNodeQueryResourcePolicy_Execute_ProviderError(t *testing.T) {
	mockClient := policy.NewTestMockClient(t)
	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	// Override ProviderProvider to nil so that getProvider returns an error.
	ctx.ProviderProvider = nil

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-err"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if !diags.HasErrors() {
		t.Fatal("expected an error diagnostic when provider is nil, got none")
	}

	// No hook call should have been made.
	if len(hook.Calls) != 0 {
		t.Errorf("expected no hook calls after provider error, got %d", len(hook.Calls))
	}
}

// TestNodeQueryResourcePolicy_Execute_SemaphoreAcquired verifies that Execute()
// holds the policy semaphore during evaluation and releases it afterward.
func TestNodeQueryResourcePolicy_Execute_SemaphoreAcquired(t *testing.T) {
	sem := NewSemaphore(1)
	var occupiedDuringEvaluate bool
	mockClient := &policy.MockClient{
		EvaluateFn: func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
			if sem.TryAcquire() {
				sem.Release()
			} else {
				occupiedDuringEvaluate = true
			}
			return policy.EvaluationResponse{Overall: policy.AllowResult}
		},
	}
	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.PolicySemaphoreValue = sem

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.NilVal)
	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	if !occupiedDuringEvaluate {
		t.Fatal("policy semaphore was not occupied during EvaluateResource")
	}

	if !sem.TryAcquire() {
		t.Fatal("policy semaphore was not released after Execute")
	}
	sem.Release()
}

// TestNodeQueryResourcePolicy_Execute_ResourceConfigSetsRange verifies that the
// originating list block supplies source ranges for query policy results.
func TestNodeQueryResourcePolicy_Execute_ResourceConfigSetsRange(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(_ context.Context, _ policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Diagnostics: policy.Diagnostics{policy.NewErrorDiagnostic(
				"denied",
				"policy denied this resource",
				policy.DenyResult,
			)},
			Enforcements: []policy.EnforcementResult{
				{Result: policy.DenyResult, Message: "denied"},
			},
		}
	}

	declRange := hcl.Range{
		Filename: "query.tfquery.hcl",
		Start:    hcl.Pos{Line: 10, Column: 3, Byte: 120},
		End:      hcl.Pos{Line: 12, Column: 5, Byte: 180},
	}
	resourceCfg := &configs.Resource{
		Mode:      addrs.ListResourceMode,
		Type:      "test_resource",
		Name:      "mylist",
		Config:    hcl.EmptyBody(),
		DeclRange: declRange,
	}
	wrongRange := hcl.Range{
		Filename: "main.tf",
		Start:    hcl.Pos{Line: 20, Column: 7, Byte: 220},
		End:      hcl.Pos{Line: 22, Column: 9, Byte: 280},
	}
	rootModule := &configs.Module{
		ManagedResources: map[string]*configs.Resource{
			"test_resource.mylist_0": {
				Mode:      addrs.ManagedResourceMode,
				Type:      "test_resource",
				Name:      "mylist_0",
				Config:    hcl.EmptyBody(),
				DeclRange: wrongRange,
			},
		},
	}
	rootCfg := &configs.Config{
		Module: rootModule,
	}

	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.ConfigValue = rootCfg

	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-abc"),
	}))
	n.ResourceConfig = resourceCfg

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	result, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call, got none")
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(result.Diagnostics))
	}
	subject := result.Diagnostics[0].Source().Subject
	if subject == nil || subject.ToHCL() != declRange {
		t.Fatalf("diagnostic source = %#v, want range from list block %#v", subject, declRange)
	}
	if len(result.Enforcements) != 1 || result.Enforcements[0].LocalRange == nil {
		t.Fatalf("enforcement local range missing: %#v", result.Enforcements)
	}
	if got := result.Enforcements[0].LocalRange; *got != declRange {
		t.Fatalf("enforcement local range = %#v, want %#v", got, declRange)
	}
}

// TestCtyIdentityToStringMap verifies edge cases for the ctyIdentityToStringMap helper.
func TestCtyIdentityToStringMap(t *testing.T) {
	t.Run("nil_val", func(t *testing.T) {
		if got := ctyIdentityToStringMap(cty.NilVal); got != nil {
			t.Errorf("expected nil for cty.NilVal, got %v", got)
		}
	})
	t.Run("null_val", func(t *testing.T) {
		if got := ctyIdentityToStringMap(cty.NullVal(cty.String)); got != nil {
			t.Errorf("expected nil for null value, got %v", got)
		}
	})
	t.Run("non_object", func(t *testing.T) {
		if got := ctyIdentityToStringMap(cty.StringVal("hello")); got != nil {
			t.Errorf("expected nil for non-object, got %v", got)
		}
	})
	t.Run("empty_object", func(t *testing.T) {
		if got := ctyIdentityToStringMap(cty.EmptyObjectVal); got != nil {
			t.Errorf("expected nil for empty object, got %v", got)
		}
	})
	t.Run("string_attrs", func(t *testing.T) {
		val := cty.ObjectVal(map[string]cty.Value{
			"id":     cty.StringVal("i-abc"),
			"region": cty.StringVal("us-east-1"),
		})
		got := ctyIdentityToStringMap(val)
		if got == nil {
			t.Fatal("expected non-nil map")
		}
		if got["id"] != "i-abc" {
			t.Errorf("id = %q, want %q", got["id"], "i-abc")
		}
		if got["region"] != "us-east-1" {
			t.Errorf("region = %q, want %q", got["region"], "us-east-1")
		}
	})
	t.Run("skips_non_string_attrs", func(t *testing.T) {
		val := cty.ObjectVal(map[string]cty.Value{
			"id":    cty.StringVal("i-abc"),
			"count": cty.NumberIntVal(3),
		})
		got := ctyIdentityToStringMap(val)
		if got == nil {
			t.Fatal("expected non-nil map")
		}
		if _, ok := got["count"]; ok {
			t.Error("expected non-string attr 'count' to be omitted")
		}
		if got["id"] != "i-abc" {
			t.Errorf("id = %q, want %q", got["id"], "i-abc")
		}
	})
	t.Run("unknown_attr_skipped", func(t *testing.T) {
		val := cty.ObjectVal(map[string]cty.Value{
			"id": cty.UnknownVal(cty.String),
		})
		got := ctyIdentityToStringMap(val)
		if got != nil {
			t.Errorf("expected nil when all attrs unknown, got %v", got)
		}
	})
}

// --- GetResources callback tests ---

// executeTestCtxWithState builds a MockEvalContext like executeTestCtx but also wires up
// ChangesChanges, DeferralsState, and a root configs.Config with the given ManagedResources
// so that getResourcesForPolicyCallback can be invoked without panicking.
//
// The ChangesSync is populated with the provided changes and then closed (required by
// ReadInstancesForConfigResource). The configs.Config contains a single managed resource
// declaration for "test_resource" (type) / "target" (name) in the root module, matching
// the synthetic resource instances appended to the change set.
func executeTestCtxWithState(t *testing.T, policyClient policy.Client, hook Hook, changes []*plans.ResourceInstanceChange, deferred *deferring.Deferred) *MockEvalContext {
	t.Helper()

	ctx := executeTestCtx(t, policyClient, hook)

	// Build and close ChangesSync so ReadInstancesForConfigResource can iterate it.
	changesSync := plans.NewChanges().SyncWrapper()
	for _, ch := range changes {
		changesSync.AppendResourceInstanceChange(ch)
	}
	changesSync.Close()
	ctx.ChangesChanges = changesSync

	if deferred == nil {
		deferred = deferring.NewDeferred(false)
	}
	ctx.DeferralsState = deferred

	// Populate a configs.Config whose Module.ManagedResources contains a resource entry
	// for "test_resource.target" so that config.DeepEach finds it during callback iteration.
	resourceCfg := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_resource",
		Name:   "target",
		Config: hcl.EmptyBody(),
	}
	ctx.ConfigValue = &configs.Config{
		Module: &configs.Module{
			ManagedResources: map[string]*configs.Resource{
				"test_resource.target": resourceCfg,
			},
		},
	}

	return ctx
}

// makeResourceInstanceChange builds a minimal ResourceInstanceChange for
// "test_resource.target" with the given After value. walkPlan uses the
// Changes (not State), so we record it as a Create change.
func makeResourceInstanceChange(afterVal cty.Value) *plans.ResourceInstanceChange {
	addr := mustResourceInstanceAddr("test_resource.target")
	return &plans.ResourceInstanceChange{
		Addr:        addr,
		PrevRunAddr: addr,
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		Change: plans.Change{
			Action: plans.Create,
			Before: cty.NullVal(afterVal.Type()),
			After:  afterVal,
		},
	}
}

// TestNodeQueryResourcePolicy_Execute_GetResources_Pass verifies that when the policy
// plugin calls req.Callbacks.GetResources(...) and the callback returns the target
// resource, the overall result is AllowResult and a hook event is emitted.
//
// This exercises the GetResources callback path within a query policy node's Execute().
func TestNodeQueryResourcePolicy_Execute_GetResources_Pass(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	// The resource that will be found by the callback.
	targetVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	change := makeResourceInstanceChange(targetVal)

	var callbackInvoked bool
	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if req.Callbacks.GetResources == nil {
			t.Error("GetResources callback was nil")
			return policy.EvaluationResponse{Overall: policy.AllowResult}
		}

		// Call GetResources and verify at least one resource is returned.
		resources, partial, err := req.Callbacks.GetResources(ctx, "test_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources returned unexpected error: %v", err)
		}
		if partial {
			t.Error("expected partial=false for non-deferred resources")
		}
		if len(resources) == 0 {
			t.Error("expected at least one resource from GetResources, got none")
		}
		callbackInvoked = true
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	hook := &testHook{}
	ctx := executeTestCtxWithState(t, mockClient, hook, []*plans.ResourceInstanceChange{change}, nil)
	ctx.StateState = states.NewState().SyncWrapper()
	defer ctx.StateState.Close()

	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-pass")}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !callbackInvoked {
		t.Error("EvaluateFn never invoked GetResources callback")
	}

	// AllowResult must still emit a hook event (query nodes always emit).
	if _, ok := hook.PolicyResults[resourceAddr.String()]; !ok {
		t.Error("expected PolicyResult hook call for GetResources AllowResult, got none")
	}
}

// TestNodeQueryResourcePolicy_Execute_GetResources_Fail verifies that when the policy
// plugin calls req.Callbacks.GetResources(...), the callback returns the target resource,
// but the plugin decides to deny — the result is DenyResult and a hook event is emitted.
//
// This exercises the GetResources callback path within a query policy node's Execute().
func TestNodeQueryResourcePolicy_Execute_GetResources_Fail(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	targetVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.xlarge"), // forbidden type
		"ami":           cty.StringVal("ami-12345"),
	})
	change := makeResourceInstanceChange(targetVal)

	var callbackInvoked bool
	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if req.Callbacks.GetResources == nil {
			t.Error("GetResources callback was nil")
			return policy.EvaluationResponse{Overall: policy.DenyResult}
		}

		resources, _, err := req.Callbacks.GetResources(ctx, "test_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources returned unexpected error: %v", err)
		}
		callbackInvoked = true

		// Simulate a policy that checks for a forbidden instance type.
		for _, r := range resources {
			if r.GetAttr("instance_type").AsString() == "t2.xlarge" {
				return policy.EvaluationResponse{
					Overall: policy.DenyResult,
					Enforcements: []policy.EnforcementResult{
						{Result: policy.DenyResult, Message: "instance type t2.xlarge is not allowed"},
					},
				}
			}
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	hook := &testHook{}
	ctx := executeTestCtxWithState(t, mockClient, hook, []*plans.ResourceInstanceChange{change}, nil)
	ctx.StateState = states.NewState().SyncWrapper()
	defer ctx.StateState.Close()

	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-fail")}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !callbackInvoked {
		t.Error("EvaluateFn never invoked GetResources callback")
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call for GetResources DenyResult, got none")
	}
	if resp.Overall != policy.DenyResult {
		t.Errorf("expected DenyResult, got %v", resp.Overall)
	}
}

// TestNodeQueryResourcePolicy_Execute_GetResources_Unknown verifies that when the policy
// plugin calls req.Callbacks.GetResources(...) and the callback returns partial=true
// (because the resource address has a deferral), the plugin maps this to UnknownResult
// and a hook event is emitted.
//
// This exercises the GetResources callback path within a query policy node's Execute()
// under a deferral — the callback receives an empty-or-partial result set and the plugin
// signals UnknownResult.
func TestNodeQueryResourcePolicy_Execute_GetResources_Unknown(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	// Set up a Deferred with deferral enabled so DependenciesDeferred returns true
	// for our target resource, causing the callback to set isPartialResult=true.
	deferred := deferring.NewDeferred(true)
	targetAddr := mustResourceInstanceAddr("test_resource.target")
	deferredChange := &plans.ResourceInstanceChange{
		Addr:        targetAddr,
		PrevRunAddr: targetAddr,
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		Change: plans.Change{
			Action: plans.Create,
			Before: cty.NullVal(cty.DynamicPseudoType),
			After:  cty.DynamicVal,
		},
	}
	deferred.ReportResourceInstanceDeferred(targetAddr, providers.DeferredReasonProviderConfigUnknown, deferredChange)

	var callbackInvoked bool
	var gotPartial bool
	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if req.Callbacks.GetResources == nil {
			t.Error("GetResources callback was nil")
			return policy.EvaluationResponse{Overall: policy.UnknownResult}
		}

		_, partial, err := req.Callbacks.GetResources(ctx, "test_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources returned unexpected error: %v", err)
		}
		gotPartial = partial
		callbackInvoked = true

		// A real policy plugin would return UnknownResult when partial=true because
		// it cannot determine compliance with incomplete data.
		if partial {
			return policy.EvaluationResponse{
				Overall: policy.UnknownResult,
				Enforcements: []policy.EnforcementResult{
					{Result: policy.UnknownResult},
				},
			}
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}

	hook := &testHook{}
	ctx := executeTestCtxWithState(t, mockClient, hook, nil, deferred)
	ctx.StateState = states.NewState().SyncWrapper()
	defer ctx.StateState.Close()

	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-unknown")}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !callbackInvoked {
		t.Error("EvaluateFn never invoked GetResources callback")
	}
	if !gotPartial {
		t.Error("expected partial=true from GetResources when resource is deferred, got false")
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call for GetResources UnknownResult, got none")
	}
	if resp.Overall != policy.UnknownResult {
		t.Errorf("expected UnknownResult, got %v", resp.Overall)
	}
}
