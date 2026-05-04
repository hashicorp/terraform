// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

// Sourceless creates and returns a diagnostic with no source location
// information. This is generally used for operational-type errors that are
// caused by or relate to the environment where Terraform is running rather
// than to the provided configuration.
func Sourceless(severity Severity, summary, detail string) Diagnostic {
	return SourcelessWithExtra(severity, summary, detail, nil)
}

func SourcelessWithExtra(severity Severity, summary, detail string, extra any) Diagnostic {
	return diagnosticBase{
		severity: severity,
		summary:  summary,
		detail:   detail,
		extra:    extra,
	}
}
