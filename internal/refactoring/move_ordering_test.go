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

