// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import "testing"

func TestMoveOrderingPolicyFunc(t *testing.T) {
	a := testMoveStatement(t, "", "test_object.a", "test_object.b", nil)
	b := testMoveStatement(t, "", "test_object.b", "test_object.c", nil)

	called := false
	policy := MoveOrderingPolicyFunc(func(depender, dependee *MoveStatement) bool {
		called = true
		return depender == &a && dependee == &b
	})

	if !policy.DependsOn(&a, &b) {
		t.Fatal("expected policy func to report dependency")
	}
	if !called {
		t.Fatal("policy func was not called")
	}
}

func TestMoveOrderingPolicyOrDefault(t *testing.T) {
	depender := testMoveStatement(t, "", "test_object.b", "test_object.c", nil)
	dependee := testMoveStatement(t, "", "test_object.a", "test_object.b", nil)

	got := moveOrderingPolicyOrDefault(nil).DependsOn(&depender, &dependee)
	if !got {
		t.Fatal("expected default policy to detect chaining dependency")
	}

	unrelatedA := testMoveStatement(t, "", "test_object.x", "test_object.y", nil)
	unrelatedB := testMoveStatement(t, "", "test_object.a", "test_object.b", nil)
	if moveOrderingPolicyOrDefault(nil).DependsOn(&unrelatedA, &unrelatedB) {
		t.Fatal("unexpected dependency for unrelated statements")
	}
}

func TestOrderedMoveStatements(t *testing.T) {
	stmts := []MoveStatement{
		testMoveStatement(t, "", "test_object.b", "test_object.c", nil),
		testMoveStatement(t, "", "test_object.a", "test_object.b", nil),
	}

	ordered, diags := OrderedMoveStatements(stmts, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err())
	}
	if len(ordered) != 2 {
		t.Fatalf("wrong number of ordered statements: %d", len(ordered))
	}
	if got := ordered[0].Name(); got != stmts[1].Name() {
		t.Fatalf("wrong first statement: got %q want %q", got, stmts[1].Name())
	}
	if got := ordered[1].Name(); got != stmts[0].Name() {
		t.Fatalf("wrong second statement: got %q want %q", got, stmts[0].Name())
	}
}

func TestOrderedMoveStatementsCycle(t *testing.T) {
	stmts := []MoveStatement{
		testMoveStatement(t, "", "test_object.a", "test_object.b", nil),
		testMoveStatement(t, "", "test_object.b", "test_object.a", nil),
	}

	ordered, diags := OrderedMoveStatements(stmts, nil)
	if !diags.HasErrors() {
		t.Fatal("expected cycle diagnostics")
	}
	if len(ordered) != 0 {
		t.Fatalf("expected no ordered statements on error, got %d", len(ordered))
	}
}
