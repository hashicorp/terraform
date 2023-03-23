package tfdiags

type diagForceWarningSeverity struct {
	wrapped Diagnostic
}

func WithErrorsAsWarnings(diags Diagnostics) Diagnostics {
	if len(diags) == 0 {
		return nil
	}

	ret := make(Diagnostics, len(diags))
	for i, diag := range diags {
		if diag.Severity() == Error {
			ret[i] = diagForceWarningSeverity{diag}
		} else {
			ret[i] = diag
		}
	}
	return ret
}

func (diag diagForceWarningSeverity) Severity() Severity {
	return Warning
}

func (diag diagForceWarningSeverity) Description() Description {
	return diag.wrapped.Description()
}

func (diag diagForceWarningSeverity) Source() Source {
	return diag.wrapped.Source()
}

func (diag diagForceWarningSeverity) FromExpr() *FromExpr {
	return diag.wrapped.FromExpr()
}

func (diag diagForceWarningSeverity) ExtraInfo() any {
	return diag.wrapped.ExtraInfo()
}
