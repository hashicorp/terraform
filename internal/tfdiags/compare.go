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
//	cmp.Diff(diag1, diag2, tfdiags.DiagnosticComparer)
var DiagnosticComparer cmp.Option = cmp.Comparer(diagnosticComparerSimple)

// diagnosticComparerSimple returns false when a difference is identified between
// the two Diagnostic arguments.
func diagnosticComparerSimple(l, r Diagnostic) bool {
	if l.Severity() != r.Severity() {
		return false
	}
	if l.Description() != r.Description() {
		return false
	}

	// Do the diagnostics originate from the same attribute name, if any?
	lp := GetAttribute(l)
	rp := GetAttribute(r)
	if len(lp) != len(rp) {
		return false
	}
	return lp.Equals(rp)
}

// DiagnosticComparerWithSource returns a cmp.Option that can be used with
// the package github.com/google/go-cmp/cmp.
//
// The comparer expands on the DiagnosticComparer by additionally
// comparing based on:
// 1) Source.Subject
// 2) Source.Context
//
// Example usage:
//
//	cmp.Diff(diag1, diag2, tfdiags.DiagnosticComparerWithSource)
var DiagnosticComparerWithSource cmp.Option = cmp.Comparer(diagnosticComparerWithSource)

// diagnosticComparerWithSource returns false when a difference is identified between
// the two Diagnostic arguments.
func diagnosticComparerWithSource(l, r Diagnostic) bool {
	// Compare diagnostics using high-level fields
	same := diagnosticComparerSimple(l, r)
	if !same {
		return false
	}

	// Is there a Subject mismatch?
	lHasSource := l.Source().Subject != nil
	rHasSource := r.Source().Subject != nil
	if lHasSource && !rHasSource {
		return false
	}
	if !lHasSource && rHasSource {
		return false
	}
	if lHasSource && rHasSource {
		if l.Source().Subject.Filename != r.Source().Subject.Filename {
			return false
		}
		if l.Source().Subject.Start != r.Source().Subject.Start {
			return false
		}
		if l.Source().Subject.End != r.Source().Subject.End {
			return false
		}
	}

	// Is there a Context mismatch?
	lHasContext := l.Source().Context != nil
	rHasContext := r.Source().Context != nil
	if lHasContext && !rHasContext {
		return false
	}
	if !lHasContext && rHasContext {
		return false
	}
	if lHasContext && rHasContext {
		if l.Source().Context.Filename != r.Source().Context.Filename {
			return false
		}
		if l.Source().Context.Start != r.Source().Context.Start {
			return false
		}
		if l.Source().Context.End != r.Source().Context.End {
			return false
		}
	}

	return true
}
