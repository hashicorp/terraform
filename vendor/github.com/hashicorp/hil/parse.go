package hil

import (
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/hil/parser"
	"github.com/hashicorp/hil/scanner"
)

// Parse parses the given program and returns an executable AST tree.
//
// Syntax errors are returned with error having the dynamic type
// *parser.ParseError, which gives the caller access to the source position
// where the error was found, which allows (for example) combining it with
// a known source filename to add context to the error message.
func Parse(v string) (ast.Node, error) {
	return ParseWithPosition(v, ast.Pos{Line: 1, Column: 1})
}

// ParseWithPosition is like Parse except that it overrides the source
// row and column position of the first character in the string, which should
// be 1-based.
//
// This can be used when HIL is embedded in another language and the outer
// parser knows the row and column where the HIL expression started within
// the overall source file.
func ParseWithPosition(v string, pos ast.Pos) (ast.Node, error) {
	ch := scanner.Scan(v, pos)
	return parser.Parse(ch)
}
