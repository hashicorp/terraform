package s3

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// diagnosticComparer is a Comparer function for use with cmp.Diff to compare two tfdiags.Diagnostic values
func diagnosticComparer(l, r tfdiags.Diagnostic) bool {
	if l.Severity() != r.Severity() {
		return false
	}
	if l.Description() != r.Description() {
		return false
	}

	lp := tfdiags.GetAttribute(l)
	rp := tfdiags.GetAttribute(r)
	if len(lp) != len(rp) {
		return false
	}
	if !lp.Equals(rp) {
		return false
	}

	return true
}

// diagnosticSummaryComparer is a Comparer function for use with cmp.Diff to compare
// the Severity and Summary fields two tfdiags.Diagnostic values
func diagnosticSummaryComparer(l, r tfdiags.Diagnostic) bool {
	if l.Severity() != r.Severity() {
		return false
	}

	ld := l.Description()
	rd := r.Description()
	if ld.Summary != rd.Summary {
		return false
	}

	return true
}
