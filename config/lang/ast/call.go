package ast

// Call represents a function call.
type Call struct {
	Func string
	Args []Node
}

func (n *Call) Accept(v Visitor) {
	for _, a := range n.Args {
		a.Accept(v)
	}

	v(n)
}
