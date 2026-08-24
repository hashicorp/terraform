// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

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

// SourcelessWithExtra is like Sourceless but also attaches an arbitrary extra
// value that callers can retrieve with ExtraInfo.
func SourcelessWithExtra(severity Severity, summary, detail string, extra interface{}) Diagnostic {
	return diagnosticBase{
		severity: severity,
		summary:  summary,
		detail:   detail,
		extra:    extra,
	}
}

// ListBlockAddrExtra is the extra value attached to warning diagnostics that
// are produced for a specific list block. AddWarningDiags reads this field to
// route the diagnostic into the correct queryPolicyBlock without string parsing.
type ListBlockAddrExtra struct {
	ListBlockAddr string
}
