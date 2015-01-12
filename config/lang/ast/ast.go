package ast

import (
	"fmt"
)

// Node is the interface that all AST nodes must implement.
type Node interface {
	// Accept is called to dispatch to the visitors.
	Accept(Visitor)

	// Pos returns the position of this node in some source.
	Pos() Pos
}

// Pos is the starting position of an AST node
type Pos struct {
	Column, Line int // Column/Line number, starting at 1
}

func (p *Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Visitors are just implementations of this function.
//
// Note that this isn't a true implementation of the visitor pattern, which
// generally requires proper type dispatch on the function. However,
// implementing this basic visitor pattern style is still very useful even
// if you have to type switch.
type Visitor func(Node)

//go:generate stringer -type=Type

// Type is the type of a literal.
type Type uint

const (
	TypeInvalid Type = 0
	TypeString       = 1 << iota
	TypeInt
)
