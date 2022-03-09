package tfdiags

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
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

//go:generate go run golang.org/x/tools/cmd/stringer -type=Severity

const (
	Error   Severity = 'E'
	Warning Severity = 'W'
)

// ToHCL converts a Severity to the equivalent HCL diagnostic severity.
func (s Severity) ToHCL() hcl.DiagnosticSeverity {
	switch s {
	case Warning:
		return hcl.DiagWarning
	case Error:
		return hcl.DiagError
	default:
		// The above should always be exhaustive for all of the valid
		// Severity values in this package.
		panic(fmt.Sprintf("unknown diagnostic severity %s", s))
	}
}

type Description struct {
	Address string
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
