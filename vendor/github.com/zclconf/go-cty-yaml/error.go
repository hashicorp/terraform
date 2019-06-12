package yaml

import (
	"errors"
	"fmt"
)

// Error is an error implementation used to report errors that correspond to
// a particular position in an input buffer.
type Error struct {
	cause        error
	Line, Column int
}

func (e Error) Error() string {
	return fmt.Sprintf("on line %d, column %d: %s", e.Line, e.Column, e.cause.Error())
}

// Cause is an implementation of the interface used by
// github.com/pkg/errors.Cause, returning the underlying error without the
// position information.
func (e Error) Cause() error {
	return e.cause
}

// WrappedErrors is an implementation of github.com/hashicorp/errwrap.Wrapper
// returning the underlying error without the position information.
func (e Error) WrappedErrors() []error {
	return []error{e.cause}
}

func parserError(p *yaml_parser_t) error {
	var cause error
	if len(p.problem) > 0 {
		cause = errors.New(p.problem)
	} else {
		cause = errors.New("invalid YAML syntax") // useless generic error, then
	}

	return parserErrorWrap(p, cause)
}

func parserErrorWrap(p *yaml_parser_t, cause error) error {
	switch {
	case p.problem_mark.line != 0:
		line := p.problem_mark.line
		column := p.problem_mark.column
		// Scanner errors don't iterate line before returning error
		if p.error == yaml_SCANNER_ERROR {
			line++
			column = 0
		}
		return Error{
			cause:  cause,
			Line:   line,
			Column: column + 1,
		}
	case p.context_mark.line != 0:
		return Error{
			cause:  cause,
			Line:   p.context_mark.line,
			Column: p.context_mark.column + 1,
		}
	default:
		return cause
	}
}

func parserErrorf(p *yaml_parser_t, f string, vals ...interface{}) error {
	return parserErrorWrap(p, fmt.Errorf(f, vals...))
}

func parseEventErrorWrap(evt *yaml_event_t, cause error) error {
	if evt.start_mark.line == 0 {
		// Event does not have a start mark, so we won't wrap the error at all
		return cause
	}
	return Error{
		cause:  cause,
		Line:   evt.start_mark.line,
		Column: evt.start_mark.column + 1,
	}
}

func parseEventErrorf(evt *yaml_event_t, f string, vals ...interface{}) error {
	return parseEventErrorWrap(evt, fmt.Errorf(f, vals...))
}

func emitterError(e *yaml_emitter_t) error {
	var cause error
	if len(e.problem) > 0 {
		cause = errors.New(e.problem)
	} else {
		cause = errors.New("failed to write YAML token") // useless generic error, then
	}
	return cause
}
