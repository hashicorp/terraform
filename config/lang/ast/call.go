package ast

// Call represents a function call.
type Call struct {
	Func string
	Args []Node
	Posx Pos
}

func (n *Call) Accept(v Visitor) {
	for _, a := range n.Args {
		a.Accept(v)
	}

	v(n)
}

func (n *Call) Pos() Pos {
	return n.Posx
}
