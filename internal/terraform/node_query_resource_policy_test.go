// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/policy"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
)

// insertQueryPolicyNodes test helper mirrors the insertion loop in listResourceExecute,
// exercising it in isolation without requiring the full listResourceExecute setup.
func insertQueryPolicyNodes(t *testing.T, inputs []listResourcePolicy, ctx *MockEvalContext, n *NodePlannableResourceInstance) {
	t.Helper()
	if ctx.PolicyGraph() == nil {
		return
	}
	for _, input := range inputs {
		if input.Unknown {
			continue
		}
		ctx.PolicyGraph().AddQuery(&nodeQueryResourcePolicy{
			ResourceAddr:    input.SyntheticAddr,
			ProviderAddr:    n.ResolvedProvider,
			GeneratedConfig: input.GeneratedConfig,
			Identity:        input.Identity,
			ResourceConfig:  input.ResourceConfig,
			ListBlockAddr:   input.ListBlockAddr,
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

// TestQueryPolicyNodeInsertion_UnknownResourcesSkipped verifies that policy
// inputs with Unknown == true do not produce a nodeQueryResourcePolicy in the
// policy subgraph.
func TestQueryPolicyNodeInsertion_UnknownResourcesSkipped(t *testing.T) {
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

	// Verify N nodes in policy subgraph, accounting for skipped unknowns
	if len(nodes) != 2 {
		t.Errorf("expected 2 policy nodes (unknown skipped), got %d", len(nodes))
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
		downEdges := ps.graph.DownEdges(node)
		for _, dep := range downEdges {
			if _, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("found edge from %s to %s; policy nodes must be independent with no inter-node edges",
					node.Name(), dag.VertexName(dep))
			}
		}

		upEdges := ps.graph.UpEdges(node)
		for _, dep := range upEdges {
			if _, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("found edge from %s to %s; policy nodes must be independent with no inter-node edges",
					dag.VertexName(dep), node.Name())
			}
		}
	}

	// Verify the total number of edges in the subgraph is zero
	// (before finish node is added by DynamicExpand)
	edges := ps.graph.Edges()
	if len(edges) != 0 {
		t.Errorf("expected 0 edges between policy nodes in subgraph, got %d edges", len(edges))
		for _, edge := range edges {
			t.Logf("  edge: %s -> %s", dag.VertexName(edge.Source()), dag.VertexName(edge.Target()))
		}
	}
}
