package parser

import (
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/hil/scanner"
)

var binaryOps []map[scanner.TokenType]ast.ArithmeticOp

func init() {
	// This operation table maps from the operator's scanner token type
	// to the AST arithmetic operation. All expressions produced from
	// binary operators are *ast.Arithmetic nodes.
	//
	// Binary operator groups are listed in order of precedence, with
	// the *lowest* precedence first. Operators within the same group
	// have left-to-right associativity.
	binaryOps = []map[scanner.TokenType]ast.ArithmeticOp{
		{
			scanner.PLUS:  ast.ArithmeticOpAdd,
			scanner.MINUS: ast.ArithmeticOpSub,
		},
		{
			scanner.STAR:    ast.ArithmeticOpMul,
			scanner.SLASH:   ast.ArithmeticOpDiv,
			scanner.PERCENT: ast.ArithmeticOpMod,
		},
	}
}
