// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This file contains some package-local helpers for working with diagnostics.
// For the main diagnostics API, see the separate "tfdiags" package.

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
