package parser

import (
	"fmt"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/hil/scanner"
)

type ParseError struct {
	Message string
	Pos     ast.Pos
}

func Errorf(pos ast.Pos, format string, args ...interface{}) error {
	return &ParseError{
		Message: fmt.Sprintf(format, args...),
		Pos:     pos,
	}
}

// TokenErrorf is a convenient wrapper around Errorf that uses the
// position of the given token.
func TokenErrorf(token *scanner.Token, format string, args ...interface{}) error {
	return Errorf(token.Pos, format, args...)
}

func ExpectationError(wanted string, got *scanner.Token) error {
	return TokenErrorf(got, "expected %s but found %s", wanted, got)
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %s: %s", e.Pos, e.Message)
}

func (e *ParseError) String() string {
	return e.Error()
}
