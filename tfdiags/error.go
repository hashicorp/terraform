package tfdiags

// nativeError is a Diagnostic implementation that wraps a normal Go error
type nativeError struct {
	err error
}

var _ Diagnostic = nativeError{}

func (e nativeError) Severity() Severity {
	return Error
}

func (e nativeError) Description() Description {
	return Description{
		Summary: FormatError(e.err),
	}
}

func (e nativeError) Source() Source {
	// No source information available for a native error
	return Source{}
}
