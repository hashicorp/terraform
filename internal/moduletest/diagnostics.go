package moduletest

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// CheckStatusDiagnosticExtra is an interface that should be implemented for the
// "extra info" on a diagnostic to signal when a diagnostic is directly
// reporting a check result and so should therefore not be included in a
// context where we're already showing the check results in a different way.
type CheckStatusDiagnosticExtra interface {
	// ExtraCheckStatus returns the address of the checkable object this
	// diagnostic is talking about and the check status the diagnostic
	// is reporting.
	//
	// The first return argument is nil if this object is not actually
	// reporting a check result after all. In that case, the second
	// return argument is meaningless.
	//
	// In practice there's no reason today for the status return value
	// to be anything other than checks.StatusFail, because we don't
	// signal anything other outcome directly as a diagnostic.
	// (For checks.StatusError, we consider any error messages to be
	// _indirectly_ reporting a problem, because the problem is with the
	// definitino of the check rather than with what it's checking.)
	ExtraCheckStatus() (addrs.Checkable, checks.Status)
}

// CheckStatusForDiagnostic determines whether the given diagnostic should
// be treated as a direct report of a check result and therefore ignored in
// any context where we're also returning the individual check results in
// detail.
//
// This can therefore avoid redundantly reporting the same check status as
// both a first-class check status and as a diagnostic.
//
// If the first return value is non-nil then it's the address of the object
// that the check relates to, and the second return value is the status that
// the diagnostic is directly describing. In practice the status can only
// possibly be checks.StatusFailure today, because that's the only status
// we report directly as a diagnostic.
//
// If the first return value is nil then this diagnostic is not describing
// a check result and the second return value is meaningless.
func CheckStatusForDiagnostic(diag tfdiags.Diagnostic) (addrs.Checkable, checks.Status) {
	if extra := tfdiags.ExtraInfo[CheckStatusDiagnosticExtra](diag); extra != nil {
		return extra.ExtraCheckStatus()
	}
	return nil, checks.StatusUnknown
}

// ConfigCheckableDiagnosticExtra is an interface that should be implemented
// for the "extra info" on a diagnostic whenever it is describing a problem
// related to a particular static checkable object in the configuration.
//
// The test harness will use this to associate the diagnostics with the
// TestCaseResult they relate to, instead of reporting them as top-level
// diagnostics.
type ConfigCheckableDiagnosticExtra interface {
	// If the associated diagnostic is about a particular static checkable
	// object in the configuration, ExtraConfigCheckable returns its address.
	//
	// Otherwise ExtraConfigCheckable returns nil, which is equivalent to
	// not implementing this interface at all.
	ExtraConfigCheckable() addrs.ConfigCheckable
}

// ConfigCheckableForDiagnostic determines whether the given diagnostic should
// be treated as part of the result for a particular static checkable object,
// returning its address if so and returning nil if not.
func ConfigCheckableForDiagnostic(diag tfdiags.Diagnostic) addrs.ConfigCheckable {
	if extra := tfdiags.ExtraInfo[ConfigCheckableDiagnosticExtra](diag); extra != nil {
		if addr := extra.ExtraConfigCheckable(); addr != nil {
			return addr
		}
	}

	// We'll also accept a CheckStatusDiagnosticExtra as a valid substitute,
	// because we can derive a single addrs.ConfigCheckable value from its
	// results.
	if extra := tfdiags.ExtraInfo[CheckStatusDiagnosticExtra](diag); extra != nil {
		if addr, _ := extra.ExtraCheckStatus(); addr != nil {
			return addr.ConfigCheckable()
		}
	}

	return nil
}
