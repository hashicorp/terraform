// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

// Sourceless creates and returns a diagnostic with no source location
// information. This is generally used for operational-type errors that are
// caused by or relate to the environment where mnptu is running rather
// than to the provided configuration.
func Sourceless(severity Severity, summary, detail string) Diagnostic {
	return diagnosticBase{
		severity: severity,
		summary:  summary,
		detail:   detail,
	}
}
