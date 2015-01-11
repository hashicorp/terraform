package ast

// Node is the interface that all AST nodes must implement.
type Node interface{}

//go:generate stringer -type=Type

// Type is the type of a literal.
type Type uint

const (
	TypeInvalid Type = 1 << iota
	TypeString
)
