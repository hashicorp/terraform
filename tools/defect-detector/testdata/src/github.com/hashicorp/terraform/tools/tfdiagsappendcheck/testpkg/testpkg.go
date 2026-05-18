package testpkg

import "github.com/hashicorp/terraform/internal/tfdiags"

type customAppender struct{}

func (customAppender) Append(...interface{}) {}

type withDiagnostics struct {
	Diagnostics tfdiags.Diagnostics
}

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

func ignoreOtherAppendType() {
	var other customAppender
	other.Append("not tfdiags")
}
