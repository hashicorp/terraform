package ast

import (
	"fmt"
)

type Conditional struct {
	CondExpr  Node
	TrueExpr  Node
	FalseExpr Node
	Posx      Pos
}

// Accept passes the given visitor to the child nodes in this order:
// CondExpr, TrueExpr, FalseExpr. It then finally passes itself to the visitor.
func (n *Conditional) Accept(v Visitor) Node {
	n.CondExpr = n.CondExpr.Accept(v)
	n.TrueExpr = n.TrueExpr.Accept(v)
	n.FalseExpr = n.FalseExpr.Accept(v)

	return v(n)
}

func (n *Conditional) Pos() Pos {
	return n.Posx
}

func (n *Conditional) Type(Scope) (Type, error) {
	// This is not actually a useful value; the type checker ignores
	// this function when analyzing conditionals, just as with Arithmetic.
	return TypeInt, nil
}

func (n *Conditional) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}
