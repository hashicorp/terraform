// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

// Sourceless creates and returns a diagnostic with no source location
// information. This is generally used for operational-type errors that are
// caused by or relate to the environment where Terraform is running rather
// than to the provided configuration.
func Sourceless(severity Severity, summary, detail string) Diagnostic {
	return diagnosticBase{
		severity: severity,
		summary:  summary,
		detail:   detail,
	}
}
