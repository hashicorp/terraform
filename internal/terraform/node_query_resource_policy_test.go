// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// insertQueryPolicyNodes test helper mirrors the insertion loop in listResourceExecute,
// exercising it in isolation without requiring the full listResourceExecute setup.
func insertQueryPolicyNodes(t *testing.T, inputs []listResourcePolicy, ctx *MockEvalContext, n *NodePlannableResourceInstance) {
	t.Helper()
	if ctx.PolicyGraph() == nil {
		return
	}
	for _, input := range inputs {
		ctx.PolicyGraph().AddQuery(&nodeQueryResourcePolicy{
			ResourceAddr:    input.SyntheticAddr,
			ProviderAddr:    n.ResolvedProvider,
			GeneratedConfig: input.GeneratedConfig,
			Identity:        input.Identity,
			ResourceConfig:  input.ResourceConfig,
			ListBlockAddr:   input.ListBlockAddr,
			Unknown:         input.Unknown,
		})
	}
}

// TestQueryPolicyNodeInsertion_CountMatchesResources verifies that a list block
// returning N non-unknown resources produces exactly N nodeQueryResourcePolicy
// nodes in the policy subgraph.
func TestQueryPolicyNodeInsertion_CountMatchesResources(t *testing.T) {
	const count = 3

	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr

	ctx := listPolicyTestContext(listBlockAddr, p, schema)
	ps := newPolicySubgraph()
	ctx.PolicyClientValue = policy.NewTestMockClient(t)
	ctx.PolicyGraphValue = ps

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")})

	elements := make([]cty.Value, count)
	for i := range elements {
		elements[i] = listPolicyTestElement(stateVal, identityVal)
	}
	data := cty.TupleVal(elements)

	inputs, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors generating policy data: %s", diags.Err())
	}

	insertQueryPolicyNodes(t, inputs, ctx, n)

	// Verify N nodes in policy subgraph
	nodes := dag.SelectSeq[*nodeQueryResourcePolicy](ps.graph.VerticesSeq()).Collect()
	if len(nodes) != count {
		t.Errorf("expected %d policy nodes, got %d", count, len(nodes))
	}
}

// TestQueryPolicyNodeInsertion_UnknownResourcesIncluded verifies that every
// discovered row produces a policy node, including pre-evaluation Unknown rows.
func TestQueryPolicyNodeInsertion_UnknownResourcesIncluded(t *testing.T) {
	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr

	ctx := listPolicyTestContext(listBlockAddr, p, schema)
	ps := newPolicySubgraph()
	ctx.PolicyClientValue = policy.NewTestMockClient(t)
	ctx.PolicyGraphValue = ps

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")})

	// Two elements with state (non-unknown) and one without (include_resource = false).
	data := cty.TupleVal([]cty.Value{
		listPolicyTestElement(stateVal, identityVal),
		listPolicyTestElementNoState(identityVal),
		listPolicyTestElement(stateVal, identityVal),
	})

	inputs, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors generating policy data: %s", diags.Err())
	}

	insertQueryPolicyNodes(t, inputs, ctx, n)

	nodes := dag.SelectSeq[*nodeQueryResourcePolicy](ps.graph.VerticesSeq()).Collect()

	if len(nodes) != 3 {
		t.Fatalf("expected 3 policy nodes, got %d", len(nodes))
	}
	var unknown *nodeQueryResourcePolicy
	for _, node := range nodes {
		if node.ResourceAddr.String() == "test_resource.mylist_1" {
			unknown = node
			break
		}
	}
	if unknown == nil {
		t.Fatal("expected a policy node for the Unknown row")
	}
	if !unknown.Unknown {
		t.Fatal("expected the Unknown flag to be propagated to the policy node")
	}
}

// TestQueryPolicyNodeInsertion_NodePayload verifies that the nodeQueryResourcePolicy
// added to the policy subgraph carries the correct field values from the
// listResourcePolicy input.
func TestQueryPolicyNodeInsertion_NodePayload(t *testing.T) {
	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr

	ctx := listPolicyTestContext(listBlockAddr, p, schema)
	ps := newPolicySubgraph()
	ctx.PolicyClientValue = policy.NewTestMockClient(t)
	ctx.PolicyGraphValue = ps

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("m5.large"),
		"ami":           cty.StringVal("ami-99999"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-abc")})
	data := cty.TupleVal([]cty.Value{listPolicyTestElement(stateVal, identityVal)})

	inputs, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors generating policy data: %s", diags.Err())
	}

	insertQueryPolicyNodes(t, inputs, ctx, n)

	nodes := dag.SelectSeq[*nodeQueryResourcePolicy](ps.graph.VerticesSeq()).Collect()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 policy node, got %d", len(nodes))
	}

	got := nodes[0]

	if got.ResourceAddr.String() != inputs[0].SyntheticAddr.String() {
		t.Errorf("ResourceAddr = %s, want %s", got.ResourceAddr, inputs[0].SyntheticAddr)
	}
	if got.ProviderAddr.String() != n.ResolvedProvider.String() {
		t.Errorf("ProviderAddr = %s, want %s", got.ProviderAddr, n.ResolvedProvider)
	}
	if got.GeneratedConfig == cty.NilVal {
		t.Error("expected non-nil GeneratedConfig")
	}
	if !got.Identity.RawEquals(identityVal) {
		t.Errorf("Identity = %#v, want %#v", got.Identity, identityVal)
	}
	if got.ResourceConfig != n.Config {
		t.Error("ResourceConfig should point to n.Config")
	}
	if got.ListBlockAddr.String() != listBlockAddr.String() {
		t.Errorf("ListBlockAddr = %s, want %s", got.ListBlockAddr, listBlockAddr)
	}
}

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

// TestQueryPolicyNodeInsertion_NilPolicyGraph verifies that insertQueryPolicyNodes
// does not panic when ctx.PolicyGraph() returns nil.
func TestQueryPolicyNodeInsertion_NilPolicyGraph(t *testing.T) {
	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr

	// PolicyGraphValue is intentionally left nil.
	ctx := listPolicyTestContext(listBlockAddr, p, schema)

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")})
	data := cty.TupleVal([]cty.Value{listPolicyTestElement(stateVal, identityVal)})

	inputs, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// Should not panic with nil PolicyGraph.
	insertQueryPolicyNodes(t, inputs, ctx, n)
}

// TestQueryPolicyNodeInsertion_NodeIndependence verifies that multiple
// nodeQueryResourcePolicy nodes added to the policy subgraph have no edges
// between them, ensuring they can be executed concurrently by the graph walker.
func TestQueryPolicyNodeInsertion_NodeIndependence(t *testing.T) {
	const count = 5

	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr

	ctx := listPolicyTestContext(listBlockAddr, p, schema)
	ps := newPolicySubgraph()
	ctx.PolicyClientValue = policy.NewTestMockClient(t)
	ctx.PolicyGraphValue = ps

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")})

	elements := make([]cty.Value, count)
	for i := range elements {
		elements[i] = listPolicyTestElement(stateVal, identityVal)
	}
	data := cty.TupleVal(elements)

	inputs, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors generating policy data: %s", diags.Err())
	}

	insertQueryPolicyNodes(t, inputs, ctx, n)

	// Collect all nodeQueryResourcePolicy nodes
	nodes := dag.SelectSeq[*nodeQueryResourcePolicy](ps.graph.VerticesSeq()).Collect()
	if len(nodes) != count {
		t.Fatalf("expected %d policy nodes, got %d", count, len(nodes))
	}

	// Verify no edges exist between any pair of nodeQueryResourcePolicy nodes
	for _, node := range nodes {
		for dep := range ps.graph.EdgesFrom(node).All() {
			if _, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("found edge from %s to %s; policy nodes must be independent with no inter-node edges",
					node.Name(), dep.Name())
			}
		}

		for dep := range ps.graph.EdgesTo(node).All() {
			if _, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("found edge from %s to %s; policy nodes must be independent with no inter-node edges",
					dep.Name(), node.Name())
			}
		}
	}

	// Verify the total number of edges in the subgraph is zero
	// (before finish node is added by DynamicExpand)
	edges := ps.graph.Edges()
	if len(edges) != 0 {
		t.Errorf("expected 0 edges between policy nodes in subgraph, got %d edges", len(edges))
		for _, edge := range edges {
			t.Logf("  edge: %s -> %s", edge.Source.Name(), edge.Target.Name())
		}
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

// TestNodeQueryResourcePolicy_Fields verifies that the node structure
// contains all required fields for policy evaluation and result correlation.
func TestNodeQueryResourcePolicy_Fields(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist[\"synthetic-0\"]")
	providerAddr := addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("test"),
		Module:   addrs.RootModule,
	}
	generatedConfig := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
	})
	identity := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-123"),
	})
	listBlockAddr := mustResourceInstanceAddr("test_resource.mylist")

	node := &nodeQueryResourcePolicy{
		ResourceAddr:    resourceAddr,
		ProviderAddr:    providerAddr,
		GeneratedConfig: generatedConfig,
		Identity:        identity,
		ListBlockAddr:   listBlockAddr,
		Unknown:         true,
	}

	// Verify all fields are accessible
	if node.ResourceAddr.String() != resourceAddr.String() {
		t.Error("ResourceAddr not set correctly")
	}
	if node.ProviderAddr.String() != providerAddr.String() {
		t.Error("ProviderAddr not set correctly")
	}
	if !node.GeneratedConfig.RawEquals(generatedConfig) {
		t.Error("GeneratedConfig not set correctly")
	}
	if !node.Identity.RawEquals(identity) {
		t.Error("Identity not set correctly")
	}
	if node.ListBlockAddr.String() != listBlockAddr.String() {
		t.Error("ListBlockAddr not set correctly")
	}
	if !node.Unknown {
		t.Error("Unknown not set correctly")
	}
}

// --- Execute() tests ---

type queryPolicySemaphoreHook struct {
	testHook
	sem      Semaphore
	occupied bool
}

func (h *queryPolicySemaphoreHook) PolicyResult(addr string, resp policy.EvaluationResponse) (HookAction, error) {
	h.occupied = len(h.sem) > 0
	return h.testHook.PolicyResult(addr, resp)
}

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

	// Passing results must still produce a hook call so downstream aggregators can record them.
	if _, ok := hook.PolicyResults[resourceAddr.String()]; !ok {
		t.Errorf("expected PolicyResult hook call for AllowResult (query node always emits), got none")
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

func TestNodeQueryResourcePolicy_Execute_PreEvaluationUnknown(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")
	mockClient := policy.NewTestMockClient(t)
	mockClient.EvaluateFn = func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		t.Fatal("policy client must not be called for a pre-evaluation Unknown")
		return policy.EvaluationResponse{}
	}

	sem := NewSemaphore(1)
	hook := &queryPolicySemaphoreHook{sem: sem}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.ProviderProvider = nil
	ctx.PolicySemaphoreValue = sem

	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-unknown"),
	}))
	n.Unknown = true

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected an Unknown PolicyResult hook call")
	}
	if resp.Overall != policy.UnknownResult {
		t.Fatalf("overall result = %s, want %s", resp.Overall, policy.UnknownResult)
	}
	if resp.Identity["id"] != "i-unknown" {
		t.Fatalf("identity = %#v, want id=i-unknown", resp.Identity)
	}
	if resp.ListBlockAddr != n.ListBlockAddr.String() {
		t.Fatalf("list block address = %q, want %q", resp.ListBlockAddr, n.ListBlockAddr)
	}
	if len(hook.Calls) != 1 {
		t.Fatalf("hook calls = %d, want 1", len(hook.Calls))
	}
	if hook.Calls[0].Action != "PolicyResult" {
		t.Fatalf("hook action = %q, want PolicyResult", hook.Calls[0].Action)
	}
	if hook.Calls[0].InstanceID != resourceAddr.String() {
		t.Fatalf("hook target = %q, want %q", hook.Calls[0].InstanceID, resourceAddr)
	}
	if len(hook.PolicyResults) != 1 {
		t.Fatalf("policy results = %d, want 1", len(hook.PolicyResults))
	}
	if len(sem) != 0 {
		t.Fatal("pre-evaluation Unknown consumed a policy semaphore slot")
	}
	if hook.occupied {
		t.Fatal("policy semaphore was occupied while emitting pre-evaluation Unknown")
	}
	if ctx.ProviderCalled || ctx.ProviderSchemaCalled {
		t.Fatal("pre-evaluation Unknown attempted provider preparation")
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

// TestNodeQueryResourcePolicy_Execute_ProviderError verifies that a provider
// preparation failure is emitted as a row-local policy Error.
func TestNodeQueryResourcePolicy_Execute_ProviderError(t *testing.T) {
	mockClient := policy.NewTestMockClient(t)
	mockClient.EvaluateFn = func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		t.Fatal("policy client must not be called after provider preparation fails")
		return policy.EvaluationResponse{}
	}
	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)

	// Override ProviderProvider to nil so that getProvider returns an error.
	ctx.ProviderProvider = nil

	sem := NewSemaphore(1)
	ctx.PolicySemaphoreValue = sem

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-err"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("provider preparation failure escaped as graph diagnostics: %s", diags.Err())
	}

	if len(sem) != 0 {
		t.Fatal("provider preparation failure consumed a policy semaphore slot")
	}
	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatal("expected a PolicyResult hook call after provider preparation failure")
	}
	if resp.Overall != policy.PolicyErrorResult || len(resp.Diagnostics) != 1 {
		t.Fatalf("unexpected policy response: %#v", resp)
	}
}

func TestNodeQueryResourcePolicy_Execute_ProviderSchemaError(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")
	schemaErr := errors.New("schema unavailable")
	mockClient := policy.NewTestMockClient(t)
	sem := NewSemaphore(1)
	hook := &queryPolicySemaphoreHook{sem: sem}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.ProviderSchemaError = schemaErr
	ctx.PolicySemaphoreValue = sem

	listRange := hcl.Range{
		Filename: "query.tfquery.hcl",
		Start:    hcl.Pos{Line: 3, Column: 1, Byte: 20},
		End:      hcl.Pos{Line: 5, Column: 2, Byte: 80},
	}
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-schema-error"),
	}))
	n.ResourceConfig = &configs.Resource{
		Mode:      addrs.ListResourceMode,
		Type:      "test_resource",
		Name:      "mylist",
		Config:    hcl.EmptyBody(),
		DeclRange: listRange,
	}

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("provider schema failure escaped as graph diagnostics: %s", diags.Err())
	}
	if mockClient.EvaluateCalled {
		t.Fatal("policy client was called after provider schema lookup failed")
	}
	if len(sem) != 0 {
		t.Fatal("provider schema failure consumed a policy semaphore slot")
	}
	if hook.occupied {
		t.Fatal("policy semaphore was occupied while emitting the provider schema Error")
	}
	if len(hook.Calls) != 1 || hook.Calls[0].Action != "PolicyResult" || hook.Calls[0].InstanceID != resourceAddr.String() {
		t.Fatalf("unexpected hook calls: %#v", hook.Calls)
	}
	if len(hook.PolicyResults) != 1 {
		t.Fatalf("policy results = %d, want 1", len(hook.PolicyResults))
	}
	resp := hook.PolicyResults[resourceAddr.String()]
	if resp.Overall != policy.PolicyErrorResult || len(resp.Diagnostics) != 1 {
		t.Fatalf("unexpected policy response: %#v", resp)
	}
	diag := resp.Diagnostics[0]
	if !strings.Contains(diag.Description().Detail, schemaErr.Error()) {
		t.Fatalf("diagnostic detail = %q, want schema error", diag.Description().Detail)
	}
	extra, ok := diag.ExtraInfo().(*policy.PolicyExtra)
	if !ok || extra.Result != policy.PolicyErrorResult {
		t.Fatalf("diagnostic result metadata = %#v, want %s", extra, policy.PolicyErrorResult)
	}
	wantRange := tfdiags.SourceRangeFromHCL(listRange)
	if got := diag.Source().Subject; got == nil || *got != wantRange {
		t.Fatalf("diagnostic local range = %#v, want %#v", got, wantRange)
	}
	if resp.Identity["id"] != "i-schema-error" {
		t.Fatalf("identity = %#v, want id=i-schema-error", resp.Identity)
	}
	if resp.ListBlockAddr != n.ListBlockAddr.String() {
		t.Fatalf("list block address = %q, want %q", resp.ListBlockAddr, n.ListBlockAddr)
	}
}

// TestNodeQueryResourcePolicy_Execute_NilClient verifies that Execute() is a no-op
// (no panic, no diagnostics) when no policy client is configured.
func TestNodeQueryResourcePolicy_Execute_NilClient(t *testing.T) {
	hook := &testHook{}
	ctx := executeTestCtx(t, nil, hook)

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.NilVal)
	n.Unknown = true
	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors with nil client: %s", diags.Err())
	}
	if len(hook.Calls) != 0 {
		t.Errorf("expected no hook calls with nil client, got %d", len(hook.Calls))
	}
}

// TestNodeQueryResourcePolicy_Execute_NilConfig verifies that missing
// configuration after a row exists is emitted as a row-local policy Error.
func TestNodeQueryResourcePolicy_Execute_NilConfig(t *testing.T) {
	mockClient := policy.NewTestMockClient(t)
	mockClient.EvaluateFn = func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		t.Fatal("policy client must not be called without configuration")
		return policy.EvaluationResponse{}
	}
	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.ConfigValue = nil // override to nil

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.NilVal)
	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors with nil config: %s", diags.Err())
	}
	resp, ok := hook.PolicyResults[n.ResourceAddr.String()]
	if !ok {
		t.Fatal("expected a PolicyResult hook call with nil config")
	}
	if resp.Overall != policy.PolicyErrorResult || len(resp.Diagnostics) != 1 {
		t.Fatalf("unexpected policy response: %#v", resp)
	}
}

// TestNodeQueryResourcePolicy_Execute_SemaphoreAcquired verifies that Execute()
// acquires and releases the policy semaphore exactly once per invocation.
func TestNodeQueryResourcePolicy_Execute_SemaphoreAcquired(t *testing.T) {
	sem := NewSemaphore(1)
	mockClient := policy.NewTestMockClient(t)
	mockClient.EvaluateFn = func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if len(sem) != 1 {
			t.Errorf("policy semaphore occupancy = %d, want 1 during evaluation", len(sem))
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}
	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.PolicySemaphoreValue = sem

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.NilVal)
	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(sem) != 0 {
		t.Error("semaphore was not released after Execute()")
	}
}

// TestNodeQueryResourcePolicy_Execute_NilPolicyGraph verifies that Execute() does not
// panic when ctx.PolicyGraph() returns nil (Finding #15 guard).
func TestNodeQueryResourcePolicy_Execute_NilPolicyGraph(t *testing.T) {
	mockClient := policy.NewTestMockClient(t)
	hook := &testHook{}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.PolicyGraphValue = nil // PolicyGraph() returns nil

	n := makeExecuteNode(mustResourceInstanceAddr("test_resource.mylist_0"), cty.NilVal)

	// Must not panic.
	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors with nil PolicyGraph: %s", diags.Err())
	}
}

// TestNodeQueryResourcePolicy_Execute_ResourceConfigSetsRange verifies that when
// a matching *configs.Resource exists in the config, its DeclRange is applied to the
// result exactly once (not twice) — guarding against the double WithLocalRange bug
// (Finding #4).
func TestNodeQueryResourcePolicy_Execute_ResourceConfigSetsRange(t *testing.T) {
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

	managedRange := hcl.Range{Filename: "main.tf", Start: hcl.Pos{Line: 10}, End: hcl.Pos{Line: 12}}
	managedResourceCfg := &configs.Resource{
		Mode:      addrs.ManagedResourceMode,
		Type:      "test_resource",
		Name:      "mylist_0",
		Config:    hcl.EmptyBody(),
		DeclRange: managedRange,
	}
	rootModule := &configs.Module{
		ManagedResources: map[string]*configs.Resource{
			"test_resource.mylist_0": managedResourceCfg,
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
	listRange := hcl.Range{Filename: "query.tfquery.hcl", Start: hcl.Pos{Line: 3}, End: hcl.Pos{Line: 5}}
	n.ResourceConfig = &configs.Resource{
		Mode:      addrs.ListResourceMode,
		Type:      "test_resource",
		Name:      "mylist",
		Config:    hcl.EmptyBody(),
		DeclRange: listRange,
	}

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Error("expected PolicyResult hook call, got none")
		return
	}
	if len(resp.Enforcements) != 1 || resp.Enforcements[0].LocalRange == nil {
		t.Fatalf("expected one enforcement with a local range, got %#v", resp.Enforcements)
	}
	if *resp.Enforcements[0].LocalRange != listRange {
		t.Fatalf("local range = %#v, want list block range %#v", *resp.Enforcements[0].LocalRange, listRange)
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
	ctx.InstanceExpanderExpander.SetResourceSingle(addrs.RootModuleInstance, resourceCfg.Addr())

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

// TestNodeQueryResourcePolicy_Execute_GetResources_Unknown verifies that a
// successful zero-resource query callback is reported as partial and can be
// classified as Unknown by the policy client.
func TestNodeQueryResourcePolicy_Execute_GetResources_Unknown(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")

	var callbackInvoked bool
	var gotPartial bool
	var gotResourceCount int
	mockClient := &policy.MockClient{}
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		if req.Callbacks.GetResources == nil {
			t.Error("GetResources callback was nil")
			return policy.EvaluationResponse{Overall: policy.UnknownResult}
		}

		resources, partial, err := req.Callbacks.GetResources(ctx, "test_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("GetResources returned unexpected error: %v", err)
		}
		gotResourceCount = len(resources)
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
	ctx := executeTestCtxWithState(t, mockClient, hook, nil, nil)
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
	if gotResourceCount != 0 {
		t.Fatalf("GetResources returned %d resources, want zero", gotResourceCount)
	}
	if !gotPartial {
		t.Error("expected partial=true for a successful zero-resource query callback")
	}

	resp, ok := hook.PolicyResults[resourceAddr.String()]
	if !ok {
		t.Fatal("expected PolicyResult hook call for GetResources UnknownResult, got none")
	}
	if resp.Overall != policy.UnknownResult {
		t.Errorf("expected UnknownResult, got %v", resp.Overall)
	}
	if resp.Identity["id"] != "i-unknown" {
		t.Fatalf("identity = %#v, want id=i-unknown", resp.Identity)
	}
	if resp.ListBlockAddr != n.ListBlockAddr.String() {
		t.Fatalf("list block address = %q, want %q", resp.ListBlockAddr, n.ListBlockAddr)
	}
	if len(hook.Calls) != 1 || hook.Calls[0].Action != "PolicyResult" || hook.Calls[0].InstanceID != resourceAddr.String() {
		t.Fatalf("unexpected hook calls: %#v", hook.Calls)
	}
	if len(hook.PolicyResults) != 1 {
		t.Fatalf("policy results = %d, want 1", len(hook.PolicyResults))
	}
}

func TestNodeQueryResourcePolicy_Execute_CallbackError(t *testing.T) {
	resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")
	var callbackInvoked bool
	var callbackErr error
	mockClient := policy.NewTestMockClient(t)
	mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		callbackInvoked = true
		_, partial, err := req.Callbacks.GetDataSource(ctx, "missing_data_source", cty.EmptyObjectVal)
		if partial {
			t.Error("GetDataSource returned partial=true with an error")
		}
		callbackErr = err
		if err == nil {
			err = errors.New("GetDataSource unexpectedly returned no error")
		}
		return policy.EvaluationResponse{
			Overall: policy.PolicyErrorResult,
			Diagnostics: policy.Diagnostics{
				policy.NewErrorDiagnostic("Failed to evaluate Terraform Policy", err.Error(), policy.PolicyErrorResult),
			},
		}
	}

	sem := NewSemaphore(1)
	hook := &queryPolicySemaphoreHook{sem: sem}
	ctx := executeTestCtx(t, mockClient, hook)
	ctx.PolicySemaphoreValue = sem
	n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("i-callback-error"),
	}))

	diags := n.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("callback error escaped as graph diagnostics: %s", diags.Err())
	}
	if !callbackInvoked {
		t.Fatal("policy client did not invoke GetDataSource")
	}
	if callbackErr == nil || !strings.Contains(callbackErr.Error(), "no data source found for missing_data_source") {
		t.Fatalf("callback error = %v, want missing data source error", callbackErr)
	}
	if len(sem) != 0 {
		t.Fatal("policy semaphore was not released after callback error")
	}
	if len(hook.Calls) != 1 || hook.Calls[0].Action != "PolicyResult" || hook.Calls[0].InstanceID != resourceAddr.String() {
		t.Fatalf("unexpected hook calls: %#v", hook.Calls)
	}
	if len(hook.PolicyResults) != 1 {
		t.Fatalf("policy results = %d, want 1", len(hook.PolicyResults))
	}
	resp := hook.PolicyResults[resourceAddr.String()]
	if resp.Overall != policy.PolicyErrorResult || resp.Overall == policy.UnknownResult {
		t.Fatalf("overall result = %s, want %s", resp.Overall, policy.PolicyErrorResult)
	}
	if len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Severity() != tfdiags.Error {
		t.Fatalf("diagnostics = %#v, want one Error", resp.Diagnostics)
	}
	extra, ok := resp.Diagnostics[0].ExtraInfo().(*policy.PolicyExtra)
	if !ok || extra.Result != policy.PolicyErrorResult {
		t.Fatalf("diagnostic result metadata = %#v, want %s", extra, policy.PolicyErrorResult)
	}
	if resp.Identity["id"] != "i-callback-error" {
		t.Fatalf("identity = %#v, want id=i-callback-error", resp.Identity)
	}
	if resp.ListBlockAddr != n.ListBlockAddr.String() {
		t.Fatalf("list block address = %q, want %q", resp.ListBlockAddr, n.ListBlockAddr)
	}
}

func TestNodeQueryResourcePolicy_Execute_PartialCallbackDoesNotRewriteResult(t *testing.T) {
	tests := []struct {
		name   string
		result policy.EvaluateResult
	}{
		{name: "allow", result: policy.AllowResult},
		{name: "deny", result: policy.DenyResult},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resourceAddr := mustResourceInstanceAddr("test_resource.mylist_0")
			var callbackInvoked bool
			var gotPartial bool
			mockClient := policy.NewTestMockClient(t)
			mockClient.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
				resources, partial, err := req.Callbacks.GetResources(ctx, "test_resource", cty.NullVal(cty.DynamicPseudoType))
				if err != nil {
					t.Errorf("GetResources returned unexpected error: %v", err)
				}
				if len(resources) != 0 {
					t.Errorf("GetResources returned %d resources, want zero", len(resources))
				}
				callbackInvoked = true
				gotPartial = partial
				return policy.EvaluationResponse{Overall: test.result}
			}

			hook := &testHook{}
			ctx := executeTestCtxWithState(t, mockClient, hook, nil, nil)
			ctx.StateState = states.NewState().SyncWrapper()
			defer ctx.StateState.Close()
			n := makeExecuteNode(resourceAddr, cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-authoritative"),
			}))

			diags := n.Execute(ctx, walkPlan)
			if diags.HasErrors() {
				t.Fatalf("unexpected errors: %s", diags.Err())
			}
			if !callbackInvoked || !gotPartial {
				t.Fatalf("callback invoked = %t, partial = %t; want true, true", callbackInvoked, gotPartial)
			}
			if len(hook.Calls) != 1 || hook.Calls[0].Action != "PolicyResult" || hook.Calls[0].InstanceID != resourceAddr.String() {
				t.Fatalf("unexpected hook calls: %#v", hook.Calls)
			}
			if len(hook.PolicyResults) != 1 {
				t.Fatalf("policy results = %d, want 1", len(hook.PolicyResults))
			}
			resp := hook.PolicyResults[resourceAddr.String()]
			if resp.Overall != test.result {
				t.Fatalf("overall result = %s, want authoritative client result %s", resp.Overall, test.result)
			}
			if resp.Identity["id"] != "i-authoritative" {
				t.Fatalf("identity = %#v, want id=i-authoritative", resp.Identity)
			}
			if resp.ListBlockAddr != n.ListBlockAddr.String() {
				t.Fatalf("list block address = %q, want %q", resp.ListBlockAddr, n.ListBlockAddr)
			}
		})
	}
}

func TestNodeQueryResourcePolicy_GetResourcesCallback(t *testing.T) {
	oneResource := []cty.Value{cty.StringVal("resource")}
	sentinelErr := errors.New("callback failed")

	tests := []struct {
		name        string
		resources   []cty.Value
		partial     bool
		err         error
		wantPartial bool
	}{
		{
			name:        "zero resources becomes partial",
			resources:   []cty.Value{},
			wantPartial: true,
		},
		{
			name:        "zero resources preserves partial",
			resources:   []cty.Value{},
			partial:     true,
			wantPartial: true,
		},
		{
			name:      "one resource remains complete",
			resources: oneResource,
		},
		{
			name:        "one resource preserves partial",
			resources:   oneResource,
			partial:     true,
			wantPartial: true,
		},
		{
			name:      "zero resources with error remains an error",
			resources: []cty.Value{},
			err:       sentinelErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callback := queryGetResourcesCallback(func(context.Context, string, cty.Value) ([]cty.Value, bool, error) {
				return test.resources, test.partial, test.err
			})
			resources, partial, err := callback(t.Context(), "test_resource", cty.NullVal(cty.DynamicPseudoType))
			if err != test.err {
				t.Fatalf("error = %v, want exact error %v", err, test.err)
			}
			if partial != test.wantPartial {
				t.Fatalf("partial = %t, want %t", partial, test.wantPartial)
			}
			if len(resources) != len(test.resources) {
				t.Fatalf("resources = %d, want %d", len(resources), len(test.resources))
			}
			for idx := range resources {
				if !resources[idx].RawEquals(test.resources[idx]) {
					t.Fatalf("resource %d = %#v, want %#v", idx, resources[idx], test.resources[idx])
				}
			}
		})
	}
}
