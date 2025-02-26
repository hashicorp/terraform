// Copyright (c) HashiCorp, Inc.
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

func (d diagnosticBase) Equals(otherDiag ComparableDiagnostic) bool {
	od, ok := otherDiag.(diagnosticBase)
	if !ok {
		return false
	}
	if d.severity != od.severity {
		return false
	}
	if d.summary != od.summary {
		return false
	}
	if d.detail != od.detail {
		return false
	}
	if d.address != od.address {
		return false
	}
	return true
}
