// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"github.com/google/go-cmp/cmp"
)

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

// DiagnosticComparerWithExtras returns a cmp.Option that can be used with
// the package github.com/google/go-cmp/cmp.
//
// Unlike DiagnosticComparer, this comparer also checks the ExtraInfo() field
// of diagnostics. This is useful for tests that need to verify that extra
// diagnostic information (such as deprecation origin descriptions) is correct.
//
// Example usage:
//
//	cmp.Diff(diag1, diag2, tfdiags.DiagnosticComparerWithExtras)
var DiagnosticComparerWithExtras cmp.Option = cmp.Options{
	// Transform Diagnostics to a comparable representation
	cmp.Transformer("DiagnosticWithExtras", func(d Diagnostic) diagnosticWithExtrasForComparison {
		if d == nil {
			return diagnosticWithExtrasForComparison{}
		}
		desc := d.Description()
		src := d.Source()
		return diagnosticWithExtrasForComparison{
			Severity: d.Severity(),
			Summary:  desc.Summary,
			Detail:   desc.Detail,
			Subject:  src.Subject,
			Context:  src.Context,
			Extra:    d.ExtraInfo(),
		}
	}),
	cmp.Comparer(func(l, r diagnosticWithExtrasForComparison) bool {
		if l.Severity != r.Severity {
			return false
		}
		if l.Summary != r.Summary {
			return false
		}
		if l.Detail != r.Detail {
			return false
		}
		if !sourceRangeEquals(l.Subject, r.Subject) {
			return false
		}
		if !sourceRangeEquals(l.Context, r.Context) {
			return false
		}
		return extrasMatch(l.Extra, r.Extra)
	}),
}

// diagnosticWithExtrasForComparison is a flattened representation of a Diagnostic
// that can be compared using cmp.Equal
type diagnosticWithExtrasForComparison struct {
	Severity Severity
	Summary  string
	Detail   string
	Subject  *SourceRange
	Context  *SourceRange
	Extra    interface{}
}

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

// extrasMatch compares two extra info values by comparing the results of
// their interface methods. This allows for generic comparison of different
// extra types without needing to know their concrete types.
func extrasMatch(l, r interface{}) bool {
	if l == nil && r == nil {
		return true
	}

	if l == nil || r == nil {
		return false
	}

	// Compare DiagnosticExtraDeprecationOrigin
	lDeprecation, lHasDeprecation := l.(DiagnosticExtraDeprecationOrigin)
	rDeprecation, rHasDeprecation := r.(DiagnosticExtraDeprecationOrigin)
	if lHasDeprecation != rHasDeprecation {
		return false
	}
	if lHasDeprecation && lDeprecation.DeprecatedOriginDescription() != rDeprecation.DeprecatedOriginDescription() {
		return false
	}

	// Compare DiagnosticExtraBecauseUnknown
	lUnknown, lHasUnknown := l.(DiagnosticExtraBecauseUnknown)
	rUnknown, rHasUnknown := r.(DiagnosticExtraBecauseUnknown)
	if lHasUnknown != rHasUnknown {
		return false
	}
	if lHasUnknown && lUnknown.DiagnosticCausedByUnknown() != rUnknown.DiagnosticCausedByUnknown() {
		return false
	}

	// Compare DiagnosticExtraBecauseSensitive
	lSensitive, lHasSensitive := l.(DiagnosticExtraBecauseSensitive)
	rSensitive, rHasSensitive := r.(DiagnosticExtraBecauseSensitive)
	if lHasSensitive != rHasSensitive {
		return false
	}
	if lHasSensitive && lSensitive.DiagnosticCausedBySensitive() != rSensitive.DiagnosticCausedBySensitive() {
		return false
	}

	// Compare DiagnosticExtraBecauseEphemeral
	lEphemeral, lHasEphemeral := l.(DiagnosticExtraBecauseEphemeral)
	rEphemeral, rHasEphemeral := r.(DiagnosticExtraBecauseEphemeral)
	if lHasEphemeral != rHasEphemeral {
		return false
	}
	if lHasEphemeral && lEphemeral.DiagnosticCausedByEphemeral() != rEphemeral.DiagnosticCausedByEphemeral() {
		return false
	}

	// Compare DiagnosticExtraDoNotConsolidate
	lNoConsolidate, lHasNoConsolidate := l.(DiagnosticExtraDoNotConsolidate)
	rNoConsolidate, rHasNoConsolidate := r.(DiagnosticExtraDoNotConsolidate)
	if lHasNoConsolidate != rHasNoConsolidate {
		return false
	}
	if lHasNoConsolidate && lNoConsolidate.DoNotConsolidateDiagnostic() != rNoConsolidate.DoNotConsolidateDiagnostic() {
		return false
	}

	// Compare DiagnosticExtraCausedByTestFailure
	lTestFailure, lHasTestFailure := l.(DiagnosticExtraCausedByTestFailure)
	rTestFailure, rHasTestFailure := r.(DiagnosticExtraCausedByTestFailure)
	if lHasTestFailure != rHasTestFailure {
		return false
	}
	if lHasTestFailure {
		if lTestFailure.DiagnosticCausedByTestFailure() != rTestFailure.DiagnosticCausedByTestFailure() {
			return false
		}
		if lTestFailure.IsTestVerboseMode() != rTestFailure.IsTestVerboseMode() {
			return false
		}
	}

	// Recursively compare wrapped extras
	lUnwrapper, lCanUnwrap := l.(DiagnosticExtraUnwrapper)
	rUnwrapper, rCanUnwrap := r.(DiagnosticExtraUnwrapper)
	if lCanUnwrap != rCanUnwrap {
		return false
	}
	if lCanUnwrap {
		return extrasMatch(lUnwrapper.UnwrapDiagnosticExtra(), rUnwrapper.UnwrapDiagnosticExtra())
	}

	return true
}
