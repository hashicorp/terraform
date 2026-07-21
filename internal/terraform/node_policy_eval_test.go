// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
)

// TestNodePolicyEvalFinish_AllowUpstreamFailure verifies that only
// nodeResourcePolicy and nodeQueryResourcePolicy return true; all other vertex
// types return false.
func TestNodePolicyEvalFinish_AllowUpstreamFailure(t *testing.T) {
	finish := &nodePolicyEvalFinish{span: trace.SpanFromContext(context.Background())}

	cases := []struct {
		name string
		dep  dag.Vertex
		want bool
	}{
		{
			name: "nodeResourcePolicy",
			dep: &nodeResourcePolicy{
				ResourceAddr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			},
			want: true,
		},
		{
			name: "nodeQueryResourcePolicy",
			dep: &nodeQueryResourcePolicy{
				ResourceAddr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "bar",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			},
			want: true,
		},
		{
			name: "nodePolicyEvalFinish",
			dep:  &nodePolicyEvalFinish{},
			want: false,
		},
		{
			name: "nodePolicyEval",
			dep:  &nodePolicyEval{},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := finish.AllowUpstreamFailure(tc.dep)
			if got != tc.want {
				t.Errorf("AllowUpstreamFailure(%T) = %v, want %v", tc.dep, got, tc.want)
			}
		})
	}
}

// TestNodePolicyEval_DynamicExpand_FinishWiring verifies that DynamicExpand
// appends a nodePolicyEvalFinish node to the policy subgraph and wires it to
// depend on every nodeResourcePolicy and nodeQueryResourcePolicy node already
// present in the subgraph.
func TestNodePolicyEval_DynamicExpand_FinishWiring(t *testing.T) {
	rp := &nodeResourcePolicy{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
	}
	qrp := &nodeQueryResourcePolicy{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "bar",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
	}

	ps := newPolicySubgraph()
	ps.Add(rp)
	ps.AddQuery(qrp)

	ctx := &MockEvalContext{
		PolicyGraphValue: ps,
		ChangesChanges:   plans.NewChanges().SyncWrapper(),
		StateState:       states.NewState().SyncWrapper(),
		StopCtxValue:     context.Background(),
	}

	n := &nodePolicyEval{}
	g, diags := n.DynamicExpand(ctx)
	if diags.HasErrors() {
		t.Fatalf("DynamicExpand returned errors: %s", diags.Err())
	}
	if g == nil {
		t.Fatal("DynamicExpand returned nil graph")
	}

	// nodePolicyEvalFinish must be present in the expanded graph.
	finishNodes := dag.SelectSeq[*nodePolicyEvalFinish](g.VerticesSeq()).Collect()
	if len(finishNodes) != 1 {
		t.Fatalf("expected 1 nodePolicyEvalFinish node, got %d", len(finishNodes))
	}

	// Both policy node types must be scheduled before the finish node.
	testGraphHappensBefore(t, g, rp.Name(), "(policy evaluation complete)")
	testGraphHappensBefore(t, g, qrp.Name(), "(policy evaluation complete)")

	// Dynamic expansion returns a walk graph, but must not mutate the accumulated
	// policy subgraph while doing finish/root wiring.
	testGraphNotContains(t, &ps.graph, "(policy evaluation complete)")
	testGraphNotContains(t, &ps.graph, rootNodeName)
}

// TestNodePolicyEval_DynamicExpand_ConcurrentExecution verifies that when
// DynamicExpand returns a graph with multiple nodeQueryResourcePolicy nodes,
// those nodes have no dependency edges between them, allowing the graph walker
// to execute them concurrently.
func TestNodePolicyEval_DynamicExpand_ConcurrentExecution(t *testing.T) {
	const count = 10

	ps := newPolicySubgraph()

	// Add multiple nodeQueryResourcePolicy nodes to simulate a list block
	// returning multiple resources
	for i := 0; i < count; i++ {
		qrp := &nodeQueryResourcePolicy{
			ResourceAddr: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "discovered",
			}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance),
		}
		ps.AddQuery(qrp)
	}

	ctx := &MockEvalContext{
		PolicyGraphValue: ps,
		ChangesChanges:   plans.NewChanges().SyncWrapper(),
		StateState:       states.NewState().SyncWrapper(),
		StopCtxValue:     context.Background(),
	}

	n := &nodePolicyEval{}
	g, diags := n.DynamicExpand(ctx)
	if diags.HasErrors() {
		t.Fatalf("DynamicExpand returned errors: %s", diags.Err())
	}
	if g == nil {
		t.Fatal("DynamicExpand returned nil graph")
	}

	// Collect all nodeQueryResourcePolicy nodes in the expanded graph
	policyNodes := dag.SelectSeq[*nodeQueryResourcePolicy](g.VerticesSeq()).Collect()
	if len(policyNodes) != count {
		t.Fatalf("expected %d policy nodes in expanded graph, got %d", count, len(policyNodes))
	}

	// The critical requirement for concurrent execution: verify no edges exist
	// between any pair of nodeQueryResourcePolicy nodes. This allows the graph
	// walker to execute all policy nodes in parallel.
	for _, node := range policyNodes {
		// Check upstream edges (what this node depends on)
		upEdges := g.UpEdges(node)
		for _, dep := range upEdges {
			if otherPolicy, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("policy node %s has upstream dependency on another policy node %s; nodes must be independent for concurrent execution",
					node.Name(), otherPolicy.Name())
			}
		}

		// Check downstream edges (what depends on this node)
		downEdges := g.DownEdges(node)
		for _, dep := range downEdges {
			if otherPolicy, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("policy node %s has downstream dependency on another policy node %s; nodes must be independent for concurrent execution",
					node.Name(), otherPolicy.Name())
			}
		}
	}

	// Verify finish node exists (wiring is tested in TestNodePolicyEval_DynamicExpand_FinishWiring)
	finishNodes := dag.SelectSeq[*nodePolicyEvalFinish](g.VerticesSeq()).Collect()
	if len(finishNodes) != 1 {
		t.Errorf("expected 1 finish node in expanded graph, got %d", len(finishNodes))
	}
}
