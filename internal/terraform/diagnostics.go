package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This file contains some package-local helpers for working with diagnostics.
// For the main diagnostics API, see the separate "tfdiags" package.

// diagnosticCausedByUnknown is an implementation of
// tfdiags.DiagnosticExtraBecauseUnknown which we can use in the "Extra" field
// of a diagnostic to indicate that the problem was caused by unknown values
// being involved in an expression evaluation.
//
// When using this, set the Extra to diagnosticCausedByUnknown(true) and also
// populate the EvalContext and Expression fields of the diagnostic so that
// the diagnostic renderer can use all of that information together to assist
// the user in understanding what was unknown.
type diagnosticCausedByUnknown bool

var _ tfdiags.DiagnosticExtraBecauseUnknown = diagnosticCausedByUnknown(true)

func (e diagnosticCausedByUnknown) DiagnosticCausedByUnknown() bool {
	return bool(e)
}

// diagnosticCausedBySensitive is an implementation of
// tfdiags.DiagnosticExtraBecauseSensitive which we can use in the "Extra" field
// of a diagnostic to indicate that the problem was caused by sensitive values
// being involved in an expression evaluation.
//
// When using this, set the Extra to diagnosticCausedBySensitive(true) and also
// populate the EvalContext and Expression fields of the diagnostic so that
// the diagnostic renderer can use all of that information together to assist
// the user in understanding what was sensitive.
type diagnosticCausedBySensitive bool

var _ tfdiags.DiagnosticExtraBecauseSensitive = diagnosticCausedBySensitive(true)

func (e diagnosticCausedBySensitive) DiagnosticCausedBySensitive() bool {
	return bool(e)
}

// diagnosticAboutConfigCheckable is an implementation of
// moduletest.ConfigCheckableDiagnosticExtra which can be used to annotate
// a diagnostic as being related to a particular static checkable object.
type diagnosticAboutConfigCheckable struct {
	addr    addrs.ConfigCheckable
	wrapped interface{}
}

var _ tfdiags.DiagnosticExtraUnwrapper = diagnosticAboutConfigCheckable{}

func diagnosticExtraAboutConfigCheckable(addr addrs.ConfigCheckable, wrapping interface{}) diagnosticAboutConfigCheckable {
	return diagnosticAboutConfigCheckable{addr, wrapping}
}

func (e diagnosticAboutConfigCheckable) ExtraConfigCheckable() addrs.ConfigCheckable {
	return e.addr
}

func (e diagnosticAboutConfigCheckable) UnwrapDiagnosticExtra() interface{} {
	return e.wrapped
}

// diagnosticAboutCheckFailure is an implementation of
// moduletest.CheckStatusDiagnosticExtra which can be used to annotate a
// diagnostic as being a direct report of a check failure for a particular
// dynamic checkable object.
type diagnosticAboutCheckFailure struct {
	addr addrs.Checkable
}

func diagnosticExtraForCheckFailure(addr addrs.Checkable) diagnosticAboutCheckFailure {
	return diagnosticAboutCheckFailure{addr}
}

func (e diagnosticAboutCheckFailure) ExtraCheckStatus() (addrs.Checkable, checks.Status) {
	return e.addr, checks.StatusFail
}
