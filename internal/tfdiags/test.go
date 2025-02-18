// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

var FailedRunDiagnosticInstance = &failedRunDiagnostic{showEphemeral: true}
var _ DiagnosticExtraBecauseEphemeral = &failedRunDiagnostic{}

type failedRunDiagnostic struct {
	// when true, the diagnostic will include information about the ephemeral
	// resources that were involved in the evaluation.
	showEphemeral bool
}

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
