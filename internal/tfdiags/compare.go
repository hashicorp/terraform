// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import "github.com/google/go-cmp/cmp"

// DiagnosticComparer returns a cmp.Option that can be used with
// the package github.com/google/go-cmp/cmp.
//
// The comparer relies on the underlying Diagnostic implementing
// [ComparableDiagnostic].
//
// Example usage:
//
//	cmp.Diff(diag1, diag2, tfdiags.DiagnosticComparer)
var DiagnosticComparer cmp.Option = cmp.Comparer(diagnosticComparerSimple)

// diagnosticComparerSimple returns false when a difference is identified between
// the two Diagnostic arguments.
func diagnosticComparerSimple(l, r Diagnostic) bool {
	ld, ok := l.(ComparableDiagnostic)
	if !ok {
		return false
	}

	rd, ok := r.(ComparableDiagnostic)
	if !ok {
		return false
	}

	return ld.Equals(rd)
}
