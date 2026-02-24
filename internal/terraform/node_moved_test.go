// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/refactoring"
)

func TestNodeExpandMovedReferencesForEach(t *testing.T) {
	n := &nodeExpandMoved{
		Stmt: &refactoring.MoveStatement{
			From: mustMoveEndpointInModuleForTest(t, "test_object.a"),
			To:   mustMoveEndpointInModuleForTest(t, "test_object.b"),
			ForEach: &hclsyntax.ScopeTraversalExpr{
				Traversal: hcl.Traversal{
					hcl.TraverseRoot{Name: "local"},
					hcl.TraverseAttr{Name: "moves"},
				},
			},
		},
	}

	got := n.References()
	if len(got) != 1 {
		t.Fatalf("wrong number of references: got %d, want 1", len(got))
	}
	if got[0] == nil {
		t.Fatal("reference is nil")
	}
	if got[0].DisplayString() != "local.moves" {
		t.Fatalf("wrong reference: got %q, want %q", got[0].DisplayString(), "local.moves")
	}
}

func TestNodeExpandMovedReferencesCount(t *testing.T) {
	n := &nodeExpandMoved{
		Stmt: &refactoring.MoveStatement{
			From: mustMoveEndpointInModuleForTest(t, "test_object.a"),
			To:   mustMoveEndpointInModuleForTest(t, "test_object.b"),
			Count: &hclsyntax.ScopeTraversalExpr{
				Traversal: hcl.Traversal{
					hcl.TraverseRoot{Name: "local"},
					hcl.TraverseAttr{Name: "move_count"},
				},
			},
		},
	}

	got := n.References()
	if len(got) != 1 {
		t.Fatalf("wrong number of references: got %d, want 1", len(got))
	}
	if got[0] == nil {
		t.Fatal("reference is nil")
	}
	if got[0].DisplayString() != "local.move_count" {
		t.Fatalf("wrong reference: got %q, want %q", got[0].DisplayString(), "local.move_count")
	}
}

func TestNodeExpandMovedForEachUnknownModuleInstancesDiag(t *testing.T) {
	n := &nodeExpandMoved{
		Stmt: &refactoring.MoveStatement{
			DeclModule: addrs.Module{"child"},
			From:       mustMoveEndpointInModuleForTest(t, "test_object.a"),
			To:         mustMoveEndpointInModuleForTest(t, "test_object.b"),
			ForEach: &hclsyntax.ScopeTraversalExpr{
				Traversal: hcl.Traversal{
					hcl.TraverseRoot{Name: "local"},
					hcl.TraverseAttr{Name: "moves"},
				},
			},
		},
	}

	exp := instances.NewExpander(nil)
	exp.SetModuleForEachUnknown(addrs.RootModuleInstance, addrs.ModuleCall{Name: "child"})

	ctx := &MockEvalContext{
		InstanceExpanderExpander: exp,
	}

	_, diags := n.expandStatements(ctx)
	if !diags.HasErrors() {
		t.Fatal("expected diagnostics, got none")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "cannot evaluate the `moved` block `for_each` expression") {
		t.Fatalf("unexpected error:\n%s", got)
	}
}

func TestNodeExpandMovedCountUnknownModuleInstancesDiag(t *testing.T) {
	n := &nodeExpandMoved{
		Stmt: &refactoring.MoveStatement{
			DeclModule: addrs.Module{"child"},
			From:       mustMoveEndpointInModuleForTest(t, "test_object.a"),
			To:         mustMoveEndpointInModuleForTest(t, "test_object.b"),
			Count: &hclsyntax.ScopeTraversalExpr{
				Traversal: hcl.Traversal{
					hcl.TraverseRoot{Name: "local"},
					hcl.TraverseAttr{Name: "move_count"},
				},
			},
		},
	}

	exp := instances.NewExpander(nil)
	exp.SetModuleCountUnknown(addrs.RootModuleInstance, addrs.ModuleCall{Name: "child"})

	ctx := &MockEvalContext{
		InstanceExpanderExpander: exp,
	}

	_, diags := n.expandStatements(ctx)
	if !diags.HasErrors() {
		t.Fatal("expected diagnostics, got none")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "cannot evaluate the `moved` block `count` expression") {
		t.Fatalf("unexpected error:\n%s", got)
	}
}

func mustMoveEndpointInModuleForTest(t *testing.T, expr string) *addrs.MoveEndpointInModule {
	t.Helper()

	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(expr), "test.tf", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid traversal %q: %s", expr, hclDiags.Error())
	}

	ep, diags := addrs.ParseMoveEndpoint(traversal)
	if diags.HasErrors() {
		t.Fatalf("invalid move endpoint %q: %s", expr, diags.Err())
	}

	from, _ := addrs.UnifyMoveEndpoints(addrs.RootModule, ep, ep)
	if from == nil {
		t.Fatalf("failed to unify move endpoint %q", expr)
	}
	return from
}
