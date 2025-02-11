// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// AssertDiagnosticsMatch fails the test in progress (using t.Fatal) if the
// two sets of diagnostics don't match after being normalized using the
// "ForRPC" processing step, which eliminates the specific type information
// and HCL expression information of each diagnostic.
//
// AssertDiagnosticsMatch sorts the two sets of diagnostics in the usual way
// before comparing them, though diagnostics only have a partial order so that
// will not totally normalize the ordering of all diagnostics sets.
func AssertDiagnosticsMatch(t *testing.T, got, want Diagnostics) {
	t.Helper()

	if diff := assertDiagnosticsMatch(got, want); diff != "" {
		t.Fatalf("unexpected diagnostics difference:\n%s", diff)
	}
}

func assertDiagnosticsMatch(got, want Diagnostics) string {
	got = got.ForRPC()
	want = want.ForRPC()

	got.Sort()
	want.Sort()

	return cmp.Diff(want, got, DiagnosticComparer)
}

// AssertDiagnosticMatch fails the test in progress (using t.Fatal) if the
// two (singular) diagnostics don't match after being normalized to an
// "RPC-friendly" diagnostic, which eliminates the specific type information
// and HCL expression information of each diagnostic.
func AssertDiagnosticMatch(t *testing.T, got, want Diagnostic) {
	t.Helper()

	if diff := assertDiagnosticMatch(want, got); diff != "" {
		t.Fatalf("unexpected diagnostics difference:\n%s", diff)
	}
}

func assertDiagnosticMatch(got, want Diagnostic) string {

	got = makeRPCFriendlyDiag(got)
	want = makeRPCFriendlyDiag(want)

	return cmp.Diff(want, got, DiagnosticComparer)
}

// AssertNoDiagnostics will fail a test if any diagnostics are present.
// If diagnostics are present, they will each be logged.
func AssertNoDiagnostics(t *testing.T, diags Diagnostics) {
	t.Helper()
	AssertDiagnosticCount(t, diags, 0)
}

// AssertDiagnosticCount will fail a test if the number of diagnostics present
// doesn't match the expected number.
// If an incorrect number of diagnostics are present, they will each be logged.
func AssertDiagnosticCount(t *testing.T, diags Diagnostics, want int) {
	t.Helper()
	if len(diags) != want {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %#v", diag)
		}
		t.FailNow()
	}
}
