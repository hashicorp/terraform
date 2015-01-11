package ast

// LiteralNode represents a single literal value, such as "foo" or
// 42 or 3.14159. Based on the Type, the Value can be safely cast.
type LiteralNode struct {
	Value interface{}
	Type  Type
}
