// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	return lp.Equals(rp)
}
