package ast

import (
	"fmt"
)

// UnaryArithmetic represents a node where the result is arithmetic of
// one operands
type UnaryArithmetic struct {
	Op   ArithmeticOp
	Expr Node
	Posx Pos
}

func (n *UnaryArithmetic) Accept(v Visitor) Node {
	n.Expr = n.Expr.Accept(v)

	return v(n)
}

func (n *UnaryArithmetic) Pos() Pos {
	return n.Posx
}

func (n *UnaryArithmetic) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}

func (n *UnaryArithmetic) String() string {
	var sign rune
	switch n.Op {
	case ArithmeticOpAdd:
		sign = '+'
	case ArithmeticOpSub:
		sign = '-'
	}
	return fmt.Sprintf("%c%s", sign, n.Expr)
}

func (n *UnaryArithmetic) Type(Scope) (Type, error) {
	return TypeInt, nil
}
