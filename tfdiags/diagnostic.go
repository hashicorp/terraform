package tfdiags

type Diagnostic interface {
	Severity() Severity
	Description() Description
	Source() Source
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
