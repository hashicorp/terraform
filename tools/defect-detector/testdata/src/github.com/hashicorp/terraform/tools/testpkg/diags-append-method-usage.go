// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package testpkg

import "github.com/hashicorp/terraform/internal/tfdiags"

func goodAssignment() {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.Sourceless(tfdiags.Warning, "summary", "detail"))
}

func goodReturn(diags tfdiags.Diagnostics) tfdiags.Diagnostics {
	return diags.Append(tfdiags.Sourceless(tfdiags.Warning, "summary", "detail"))
}

func badLocal() {
	var diags tfdiags.Diagnostics
	diags.Append(tfdiags.Sourceless(tfdiags.Warning, "summary", "detail")) // want "ignored return value from tfdiags.Diagnostics.Append"
}

func badField(resp *withDiagnostics) {
	resp.Diagnostics.Append(tfdiags.Sourceless(tfdiags.Warning, "summary", "detail")) // want "ignored return value from tfdiags.Diagnostics.Append"
}

// Something else with an Append method should not get flagged like tfdiags.Diagnostics does.
func ignoreOtherAppendType() {
	var other customAppender
	other.Append("not tfdiags")
}

type customAppender struct{}

func (customAppender) Append(...interface{}) {}

type withDiagnostics struct {
	Diagnostics tfdiags.Diagnostics
}
