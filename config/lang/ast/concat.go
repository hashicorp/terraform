package ast

import (
	"fmt"
)

// Concat represents a node where the result of two or more expressions are
// concatenated. The result of all expressions must be a string.
type Concat struct {
	Exprs []Node
}

func (n *Concat) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}
