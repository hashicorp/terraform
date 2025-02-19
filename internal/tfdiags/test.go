// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

// FailedRunDiagnosticInstance is a special instance of Diagnostic that can be
// used to indicate that a diagnostic was caused by a failed run. This is
// intended to be used in cases where the diagnostic is not directly related to
// a specific configuration element, but rather to the overall evaluation of
// the test run.
// The showEphemeral is set to true, so that the diagnostic will include
// information about the ephemeral values that were involved in the evaluation.
var FailedRunDiagnosticInstance = &failedRunDiagnostic{showEphemeral: true}
var _ DiagnosticExtraBecauseEphemeral = (*failedRunDiagnostic)(nil)

type failedRunDiagnostic struct{ showEphemeral bool }

func (f *failedRunDiagnostic) DiagnosticCausedByEphemeral() bool {
	return f.showEphemeral
}

func IsFailedRunDiagnostic(diag Diagnostic) bool {
	maybeFailedRunDiagnostic := diag.ExtraInfo()
	if maybeFailedRunDiagnostic == nil {
		return false
	}
	_, ok := maybeFailedRunDiagnostic.(*failedRunDiagnostic)
	return ok
}
