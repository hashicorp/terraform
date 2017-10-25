package tfdiags

type simpleWarning string

var _ Diagnostic = simpleWarning("")

// SimpleWarning constructs a simple (summary-only) warning diagnostic.
func SimpleWarning(msg string) Diagnostic {
	return simpleWarning(msg)
}

func (e simpleWarning) Severity() Severity {
	return Warning
}

func (e simpleWarning) Description() Description {
	return Description{
		Summary: string(e),
	}
}

func (e simpleWarning) Source() Source {
	// No source information available for a native error
	return Source{}
}
