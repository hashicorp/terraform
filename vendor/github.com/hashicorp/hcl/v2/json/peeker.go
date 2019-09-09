package json

type peeker struct {
	tokens []token
	pos    int
}

func newPeeker(tokens []token) *peeker {
	return &peeker{
		tokens: tokens,
		pos:    0,
	}
}

func (p *peeker) Peek() token {
	return p.tokens[p.pos]
}

func (p *peeker) Read() token {
	ret := p.tokens[p.pos]
	if ret.Type != tokenEOF {
		p.pos++
	}
	return ret
}
