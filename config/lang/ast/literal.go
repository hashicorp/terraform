package ast

import (
	"fmt"
)

// LiteralNode represents a single literal value, such as "foo" or
// 42 or 3.14159. Based on the Type, the Value can be safely cast.
type LiteralNode struct {
	Value interface{}
	Type  Type
}

func (n *LiteralNode) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}
