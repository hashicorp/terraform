package tfdiags

// diagnosticBase can be embedded in other diagnostic structs to get
// default implementations of Severity and Description. This type also
// has a default implementation of Source that returns no source location
// information, so embedders should generally override that method to
// return more useful results.
type diagnosticBase struct {
	severity Severity
	summary  string
	detail   string
}

func (d diagnosticBase) Severity() Severity {
	return d.severity
}

func (d diagnosticBase) Description() Description {
	return Description{
		Summary: d.summary,
		Detail:  d.detail,
	}
}

func (d diagnosticBase) Source() Source {
	return Source{}
}
