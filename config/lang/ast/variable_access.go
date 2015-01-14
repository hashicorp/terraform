package ast

import (
	"fmt"
)

// VariableAccess represents a variable access.
type VariableAccess struct {
	Name string
	Posx Pos
}

func (n *VariableAccess) Accept(v Visitor) {
	v(n)
}

func (n *VariableAccess) Pos() Pos {
	return n.Posx
}

func (n *VariableAccess) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}

func (n *VariableAccess) String() string {
	return fmt.Sprintf("Variable(%s)", n.Name)
}
