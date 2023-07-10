package tfdiags

// overriddenDiagnostic implements the Diagnostic interface by wrapping another
// Diagnostic while overriding the severity of the original Diagnostic.
type overriddenDiagnostic struct {
	original Diagnostic
	severity Severity
	extra    interface{}
}

var _ Diagnostic = overriddenDiagnostic{}

// OverrideAll accepts a set of Diagnostics and wraps them with a new severity
// and, optionally, a new ExtraInfo.
func OverrideAll(originals Diagnostics, severity Severity, createExtra func() DiagnosticExtraWrapper) Diagnostics {
	var diags Diagnostics
	for _, diag := range originals {
		diags = diags.Append(Override(diag, severity, createExtra))
	}
	return diags
}

// Override matches OverrideAll except it operates over a single Diagnostic
// rather than multiple Diagnostics.
func Override(original Diagnostic, severity Severity, createExtra func() DiagnosticExtraWrapper) Diagnostic {
	extra := original.ExtraInfo()
	if createExtra != nil {
		nw := createExtra()
		nw.WrapDiagnosticExtra(extra)
		extra = nw
	}

	return overriddenDiagnostic{
		original: original,
		severity: severity,
		extra:    extra,
	}
}

// UndoOverride will return the original diagnostic that was overridden within
// the OverrideAll function.
//
// If the provided Diagnostic was never overridden then it is simply returned
// unchanged.
func UndoOverride(diag Diagnostic) Diagnostic {
	if override, ok := diag.(overriddenDiagnostic); ok {
		return override.original
	}

	// Then it wasn't overridden, so we'll just return the diag unchanged.
	return diag
}

func (o overriddenDiagnostic) Severity() Severity {
	return o.severity
}

func (o overriddenDiagnostic) Description() Description {
	return o.original.Description()
}

func (o overriddenDiagnostic) Source() Source {
	return o.original.Source()
}

func (o overriddenDiagnostic) FromExpr() *FromExpr {
	return o.original.FromExpr()
}

func (o overriddenDiagnostic) ExtraInfo() interface{} {
	return o.extra
}
