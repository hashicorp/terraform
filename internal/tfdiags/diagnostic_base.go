// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

// diagnosticBase can be embedded in other diagnostic structs to get
// default implementations of Severity and Description. This type also
// has default implementations of Source and FromExpr that return no source
// location or expression-related information, so embedders should generally
// override those method to return more useful results where possible.
type diagnosticBase struct {
	severity Severity
	summary  string
	detail   string
	address  string
}

var _ Diagnostic = &diagnosticBase{}

// diagnosticBase doesn't implement ComparableDiagnostic because the lack of source data
// means separate diagnostics might be falsely identified as equal. This poses a user-facing
// risk if deduplication of diagnostics removes a diagnostic that's incorrectly been identified
// as a duplicate via comparison.

func (d diagnosticBase) Severity() Severity {
	return d.severity
}

func (d diagnosticBase) Description() Description {
	return Description{
		Summary: d.summary,
		Detail:  d.detail,
		Address: d.address,
	}
}

func (d diagnosticBase) Source() Source {
	return Source{}
}

func (d diagnosticBase) FromExpr() *FromExpr {
	return nil
}

func (d diagnosticBase) ExtraInfo() interface{} {
	return nil
}
