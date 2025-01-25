// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import "github.com/google/go-cmp/cmp"

// DiagnosticComparer returns a cmp.Option that can be used with
// the package github.com/google/go-cmp/cmp.
//
// The comparer checks these match between the diagnostics:
// 1) Severity
// 2) Description
// 3) Attribute cty.Path, if present
//
// Example usage:
//
//	cmp.Diff(diag1, diag2, tfdiags.DiagnosticComparer())
var DiagnosticComparer cmp.Option = cmp.Comparer(diagnosticComparer)

func diagnosticComparer(l, r Diagnostic) bool {
	if l.Severity() != r.Severity() {
		return false
	}
	if l.Description() != r.Description() {
		return false
	}

	lp := GetAttribute(l)
	rp := GetAttribute(r)
	if len(lp) != len(rp) {
		return false
	}
	return lp.Equals(rp)
}
