package tfdiags

type diagForceErrorSeverity struct {
	wrapped Diagnostic
}

func WithErrorsAsWarnings(diags Diagnostics) Diagnostics {
	if len(diags) == 0 {
		return nil
	}

	ret := make(Diagnostics, len(diags))
	for i, diag := range diags {
		if diag.Severity() == Error {
			ret[i] = diagForceErrorSeverity{diag}
		} else {
			ret[i] = diag
		}
	}
	return ret
}

func (diag diagForceErrorSeverity) Severity() Severity {
	return Error
}

func (diag diagForceErrorSeverity) Description() Description {
	return diag.wrapped.Description()
}

func (diag diagForceErrorSeverity) Source() Source {
	return diag.wrapped.Source()
}

func (diag diagForceErrorSeverity) FromExpr() *FromExpr {
	return diag.wrapped.FromExpr()
}

func (diag diagForceErrorSeverity) ExtraInfo() any {
	return diag.wrapped.ExtraInfo()
}
