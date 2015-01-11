package ast

import (
	"fmt"
)

// VariableAccess represents a variable access.
type VariableAccess struct {
	Name string
}

func (n *VariableAccess) Accept(v Visitor) {
	v(n)
}

func (n *VariableAccess) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}
