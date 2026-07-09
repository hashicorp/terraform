// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/policy"
)

// stubConfigResource is a minimal GraphNodeConfigResource used to exercise
// policyEvalTransformer in isolation without constructing a full
// NodePlannableResourceInstance.
type stubConfigResource struct {
	addr addrs.ConfigResource
}

func (s *stubConfigResource) Name() string                       { return s.addr.String() }
func (s *stubConfigResource) ResourceAddr() addrs.ConfigResource { return s.addr }

// stubCloseProvider is a minimal GraphNodeCloseProvider used to verify that
// policyEvalTransformer wires provider-close nodes to run after policy evaluation.
type stubCloseProvider struct{}

func (s *stubCloseProvider) Name() string {
	return `provider["registry.terraform.io/hashicorp/test"] (close)`
}
func (s *stubCloseProvider) ModulePath() addrs.Module { return addrs.RootModule }
func (s *stubCloseProvider) CloseProviderAddr() addrs.AbsProviderConfig {
	return addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("test"),
		Module:   addrs.RootModule,
	}
}

// TestPolicyEvalTransformer_NilClient verifies that Transform is a no-op when
// PolicyClient is nil: no nodePolicyEval node is added to the graph.
func TestPolicyEvalTransformer_NilClient(t *testing.T) {
	var g Graph

	tr := &policyEvalTransformer{PolicyClient: nil}
	if err := tr.Transform(&g); err != nil {
		t.Fatalf("Transform returned error: %s", err)
	}

	nodes := dag.SelectSeq[*nodePolicyEval](g.VerticesSeq()).Collect()
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodePolicyEval nodes with nil client, got %d", len(nodes))
	}
}

// TestPolicyEvalTransformer_NonQueryMode verifies that when QueryPlan == false,
// policyEvalTransformer wires nodePolicyEval to every non-provider-close vertex
// and wires every provider-close vertex to run after nodePolicyEval.
func TestPolicyEvalTransformer_NonQueryMode(t *testing.T) {
	var g Graph

	managedRes := &stubConfigResource{
		addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		}.InModule(addrs.RootModule),
	}
	closeProvider := &stubCloseProvider{}
	g.Add(managedRes)
	g.Add(closeProvider)

	tr := &policyEvalTransformer{
		PolicyClient: policy.NewTestMockClient(t),
		QueryPlan:    false,
	}
	if err := tr.Transform(&g); err != nil {
		t.Fatalf("Transform returned error: %s", err)
	}

	nodes := dag.SelectSeq[*nodePolicyEval](g.VerticesSeq()).Collect()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 nodePolicyEval node, got %d", len(nodes))
	}

	// All non-provider-close vertices run before policy evaluation.
	testGraphHappensBefore(t, &g, managedRes.Name(), "(evaluate policies)")
	// Provider-close runs after policy evaluation.
	testGraphHappensBefore(t, &g, "(evaluate policies)", closeProvider.Name())
}

// TestPolicyEvalTransformer_QueryMode verifies that when QueryPlan == true,
// policyEvalTransformer wires nodePolicyEval only to GraphNodeConfigResource
// vertices whose mode is addrs.ListResourceMode. Non-list managed resources
// are not wired as upstream dependencies of the policy node.
func TestPolicyEvalTransformer_QueryMode(t *testing.T) {
	var g Graph

	listRes := &stubConfigResource{
		addr: addrs.Resource{
			Mode: addrs.ListResourceMode,
			Type: "test_resource",
			Name: "mylist",
		}.InModule(addrs.RootModule),
	}
	managedRes := &stubConfigResource{
		addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		}.InModule(addrs.RootModule),
	}
	g.Add(listRes)
	g.Add(managedRes)

	tr := &policyEvalTransformer{
		PolicyClient: policy.NewTestMockClient(t),
		QueryPlan:    true,
	}
	if err := tr.Transform(&g); err != nil {
		t.Fatalf("Transform returned error: %s", err)
	}

	nodes := dag.SelectSeq[*nodePolicyEval](g.VerticesSeq()).Collect()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 nodePolicyEval node, got %d", len(nodes))
	}

	// List block runs before policy.
	testGraphHappensBefore(t, &g, listRes.Name(), "(evaluate policies)")

	// Managed resource must NOT be a (transitive) ancestor of the policy node.
	policyNode := nodes[0]
	for _, anc := range g.Ancestors(policyNode) {
		if dag.VertexName(anc) == managedRes.Name() {
			t.Errorf("policy node should not depend on managed resource in query mode, but %q is an ancestor", managedRes.Name())
		}
	}
}
