package ast

// Call represents a function call.
type Call struct {
	Func string
	Args []Node
}
