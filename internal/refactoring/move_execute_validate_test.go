// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"strings"
	"testing"
)

func TestValidateMoveStatementsForExecutionCycle(t *testing.T) {
	stmts := []MoveStatement{
		testMoveStatement(t, "", "test_object.a", "test_object.b", nil),
		testMoveStatement(t, "", "test_object.b", "test_object.a", nil),
	}

	diags := ValidateMoveStatementsForExecution(stmts)
	if !diags.HasErrors() {
		t.Fatal("expected cycle diagnostics, got none")
	}

	got := diags.Err().Error()
	if !strings.Contains(got, "Cyclic dependency in move statements") {
		t.Fatalf("expected cycle diagnostic summary, got: %s", got)
	}
}
