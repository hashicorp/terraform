package tfdiags

type Severity rune

const Warning Severity = 'W'

type Diagnostic interface{}

type Diagnostics []Diagnostic

func (diags Diagnostics) Append(new ...interface{}) Diagnostics {
	return diags
}

func Sourceless(Severity, string, string) Diagnostic {
	return nil
}
