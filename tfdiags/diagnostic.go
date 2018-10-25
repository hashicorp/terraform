package tfdiags

import (
	"github.com/hashicorp/hcl2/hcl"
)

type Diagnostic interface {
	Severity() Severity
	Description() Description
	Source() Source

	// FromExpr returns the expression-related context for the diagnostic, if
	// available. Returns nil if the diagnostic is not related to an
	// expression evaluation.
	FromExpr() *FromExpr
}

type Severity rune

//go:generate stringer -type=Severity

const (
	Error   Severity = 'E'
	Warning Severity = 'W'
)

type Description struct {
	Summary string
	Detail  string
}

type Source struct {
	Subject *SourceRange
	Context *SourceRange
}

type FromExpr struct {
	Expression  hcl.Expression
	EvalContext *hcl.EvalContext
}
