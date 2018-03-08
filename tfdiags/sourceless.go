package tfdiags

// Sourceless returns a diagnostic that has a severity, summary and detail
// but does not carry source location information.
//
// This is primarily intended for reporting errors and warnings that are
// related to non-configuration issues, such as invalid command line arguments.
func Sourceless(severity Severity, summary string, detail string) Diagnostic {
	return sourcelessDiag{
		severity: severity,
		summary:  summary,
		detail:   detail,
	}
}

type sourcelessDiag struct {
	severity Severity
	summary  string
	detail   string
}

func (d sourcelessDiag) Severity() Severity {
	return Warning
}

func (d sourcelessDiag) Description() Description {
	return Description{
		Summary: d.summary,
		Detail:  d.detail,
	}
}

func (d sourcelessDiag) Source() Source {
	// No source information available for a native error
	return Source{}
}
