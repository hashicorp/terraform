// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/refactoring"
)

func TestMovedBlockEdgeTransformerAddsChainingEdges(t *testing.T) {
	stmts := []refactoring.MoveStatement{
		testMovedGraphStatement(t, "test_object.a", "test_object.b"),
		testMovedGraphStatement(t, "test_object.b", "test_object.c"),
		testMovedGraphStatement(t, "test_object.x", "test_object.y"),
	}

	g := Graph{Path: addrs.RootModuleInstance}

	if err := (&MovedBlockTransformer{
		Statements: stmts,
		Runtime:    &movedExecutionRuntime{},
	}).Transform(&g); err != nil {
		t.Fatalf("unexpected error adding moved nodes: %s", err)
	}
	if err := (&MovedBlockEdgeTransformer{}).Transform(&g); err != nil {
		t.Fatalf("unexpected error adding moved edges: %s", err)
	}

	nodes := map[int]*nodeExpandMoved{}
	for _, v := range g.Vertices() {
		if node, ok := v.(*nodeExpandMoved); ok {
			nodes[node.Index] = node
		}
	}

	if len(nodes) != len(stmts) {
		t.Fatalf("wrong number of moved nodes: got %d, want %d", len(nodes), len(stmts))
	}

	if !g.HasEdge(dag.BasicEdge(nodes[1], nodes[0])) {
		t.Fatalf("expected chaining edge moved[1] -> moved[0]")
	}
	if g.HasEdge(dag.BasicEdge(nodes[0], nodes[1])) {
		t.Fatalf("unexpected reverse chaining edge moved[0] -> moved[1]")
	}
	if g.HasEdge(dag.BasicEdge(nodes[2], nodes[0])) || g.HasEdge(dag.BasicEdge(nodes[2], nodes[1])) {
		t.Fatalf("unexpected dependency edges from unrelated move statement")
	}
}

func TestMovedBlockEdgeTransformerUsesInjectedPolicy(t *testing.T) {
	stmts := []refactoring.MoveStatement{
		testMovedGraphStatement(t, "test_object.a", "test_object.b"),
		testMovedGraphStatement(t, "test_object.b", "test_object.c"),
	}

	g := Graph{Path: addrs.RootModuleInstance}
	if err := (&MovedBlockTransformer{
		Statements: stmts,
		Runtime:    &movedExecutionRuntime{},
	}).Transform(&g); err != nil {
		t.Fatalf("unexpected error adding moved nodes: %s", err)
	}

	if err := (&MovedBlockEdgeTransformer{
		Policy: refactoring.MoveOrderingPolicyFunc(func(depender, dependee *refactoring.MoveStatement) bool {
			return depender == &stmts[0] && dependee == &stmts[1]
		}),
	}).Transform(&g); err != nil {
		t.Fatalf("unexpected error adding moved edges: %s", err)
	}

	nodes := map[int]*nodeExpandMoved{}
	for _, v := range g.Vertices() {
		if node, ok := v.(*nodeExpandMoved); ok {
			nodes[node.Index] = node
		}
	}

	if !g.HasEdge(dag.BasicEdge(nodes[0], nodes[1])) {
		t.Fatalf("expected injected policy edge moved[0] -> moved[1]")
	}
	if g.HasEdge(dag.BasicEdge(nodes[1], nodes[0])) {
		t.Fatalf("unexpected default-policy edge moved[1] -> moved[0]")
	}
}

func testMovedGraphStatement(t *testing.T, from, to string) refactoring.MoveStatement {
	t.Helper()

	fromTraversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(from), "from", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid 'from' traversal %q: %s", from, hclDiags.Error())
	}
	toTraversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(to), "to", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid 'to' traversal %q: %s", to, hclDiags.Error())
	}

	fromEP, diags := addrs.ParseMoveEndpoint(fromTraversal)
	if diags.HasErrors() {
		t.Fatalf("invalid 'from' endpoint %q: %s", from, diags.Err())
	}
	toEP, diags := addrs.ParseMoveEndpoint(toTraversal)
	if diags.HasErrors() {
		t.Fatalf("invalid 'to' endpoint %q: %s", to, diags.Err())
	}

	fromAbs, toAbs := addrs.UnifyMoveEndpoints(addrs.RootModule, fromEP, toEP)
	if fromAbs == nil || toAbs == nil {
		t.Fatalf("incompatible move endpoints: %q -> %q", from, to)
	}

	return refactoring.MoveStatement{
		From: fromAbs,
		To:   toAbs,
	}
}
