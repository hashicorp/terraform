package ast

import (
	"fmt"
	"strings"
)

// Call represents a function call.
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
