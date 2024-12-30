// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
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

// DiagnosticCausedByEphemeral is an implementation of
// tfdiags.DiagnosticExtraBecauseEphemeral which we can use in the "Extra" field
// of a diagnostic to indicate that the problem was caused by ephemeral values
// being involved in an expression evaluation.
//
// When using this, set the Extra to DiagnosticCausedByEphemeral(true) and also
// populate the EvalContext and Expression fields of the diagnostic so that
// the diagnostic renderer can use all of that information together to assist
// the user in understanding what was ephemeral.
type DiagnosticCausedByEphemeral bool

var _ tfdiags.DiagnosticExtraBecauseEphemeral = DiagnosticCausedByEphemeral(true)

func (e DiagnosticCausedByEphemeral) DiagnosticCausedByEphemeral() bool {
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
