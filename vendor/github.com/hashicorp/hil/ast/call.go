package ast

import (
	"fmt"
	"strings"
)

// Call represents a function call.
//
// The type checker replaces Call nodes with CallTyped nodes in order to retain
// the type information for use in the evaluation phase.
type Call struct {
	Func string
	Args []Node
	Posx Pos
}

func (n *Call) Accept(v Visitor) Node {
	for i, a := range n.Args {
		n.Args[i] = a.Accept(v)
	}

	return v(n)
}

func (n *Call) Pos() Pos {
	return n.Posx
}

func (n *Call) String() string {
	args := make([]string, len(n.Args))
	for i, arg := range n.Args {
		args[i] = fmt.Sprintf("%s", arg)
	}

	return fmt.Sprintf("Call(%s, %s)", n.Func, strings.Join(args, ", "))
}

func (n *Call) Type(s Scope) (Type, error) {
	f, ok := s.LookupFunc(n.Func)
	if !ok {
		return TypeInvalid, fmt.Errorf("unknown function: %s", n.Func)
	}

	return f.ReturnType, nil
}

func (n *Call) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}

// CallTyped represents a function call *after* type checking.
//
// The type check phase replaces any Call node with a CallTyped node in order to
// capture the type information that was determined so that it can be used during
// a subsequent evaluation.
type CallTyped struct {
	// CallTyped embeds the Call it was created from.
	Call

	// ReturnType is the return type determined for the function during type checking.
	// A well-behaved function implementation is bound by the interface contract to return
	// a value that conforms to this type.
	ReturnType Type
}

func (n *CallTyped) Accept(v Visitor) Node {
	// Accept must be re-implemented on CallTyped to make sure we pass the full CallTyped
	// value, rather than the embedded Call value that would result were we to inherit
	// the implementation from Call.
	for i, a := range n.Args {
		n.Args[i] = a.Accept(v)
	}

	return v(n)
}

func (n *CallTyped) Type(s Scope) (Type, error) {
	return n.ReturnType, nil
}
