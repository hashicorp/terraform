// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import "github.com/hashicorp/terraform/internal/tfdiags"

// DiagnosticCausedByTestFailure implements multiple interfaces that enables it to
// be used in the "Extra" field of a diagnostic. This type should only be used as
// the Extra for diagnostics reporting assertions that fail in a run block during
// `terraform test`.
//
// DiagnosticCausedByTestFailure implements the [DiagnosticExtraCausedByTestFailure]
// interface. This allows downstream logic to identify diagnostics that are specifically
// due to assertion failures.
//
// DiagnosticCausedByTestFailure also implements the [DiagnosticExtraBecauseEphemeral],
// [DiagnosticExtraBecauseSensitive], and [DiagnosticExtraBecauseUnknown] interfaces.
// These interfaces allow the diagnostic renderer to include ephemeral, sensitive or
// unknown data if it's present. This is enabled because if a test fails then the user
// will want to know what values contributed to the failing assertion.
//
// When using this, set the Extra to DiagnosticCausedByTestFailure(true) and also
// populate the EvalContext and Expression fields of the diagnostic.

type DiagnosticCausedByTestFailure struct {
	Verbose bool
}

var _ tfdiags.DiagnosticExtraCausedByTestFailure = DiagnosticCausedByTestFailure{false}
var _ tfdiags.DiagnosticExtraBecauseEphemeral = DiagnosticCausedByTestFailure{false}
var _ tfdiags.DiagnosticExtraBecauseSensitive = DiagnosticCausedByTestFailure{false}
var _ tfdiags.DiagnosticExtraBecauseUnknown = DiagnosticCausedByTestFailure{false}

func (e DiagnosticCausedByTestFailure) DiagnosticCausedByTestFailure() bool {
	return true
}

func (e DiagnosticCausedByTestFailure) IsTestVerboseMode() bool {
	return e.Verbose
}

func (e DiagnosticCausedByTestFailure) DiagnosticCausedByEphemeral() bool {
	return true
}

func (e DiagnosticCausedByTestFailure) DiagnosticCausedBySensitive() bool {
	return true
}

func (e DiagnosticCausedByTestFailure) DiagnosticCausedByUnknown() bool {
	return true
}
