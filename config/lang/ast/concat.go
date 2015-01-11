package ast

// Concat represents a node where the result of two or more expressions are
// concatenated. The result of all expressions must be a string.
type Concat struct {
	Exprs []Node
}
