package scanner

// Peeker is a utility that wraps a token channel returned by Scan and
// provides an interface that allows a caller (e.g. the parser) to
// work with the token stream in a mode that allows one token of lookahead,
// and provides utilities for more convenient processing of the stream.
type Peeker struct {
	ch     <-chan *Token
	peeked *Token
}

func NewPeeker(ch <-chan *Token) *Peeker {
	return &Peeker{
		ch: ch,
	}
}

// Peek returns the next token in the stream without consuming it. A
// subsequent call to Read will return the same token.
func (p *Peeker) Peek() *Token {
	if p.peeked == nil {
		p.peeked = <-p.ch
	}
	return p.peeked
}

// Read consumes the next token in the stream and returns it.
func (p *Peeker) Read() *Token {
	token := p.Peek()

	// As a special case, we will produce the EOF token forever once
	// it is reached.
	if token.Type != EOF {
		p.peeked = nil
	}

	return token
}

// Close ensures that the token stream has been exhausted, to prevent
// the goroutine in the underlying scanner from leaking.
//
// It's not necessary to call this if the caller reads the token stream
// to EOF, since that implicitly closes the scanner.
func (p *Peeker) Close() {
	for _ = range p.ch {
		// discard
	}
	// Install a synthetic EOF token in 'peeked' in case someone
	// erroneously calls Peek() or Read() after we've closed.
	p.peeked = &Token{
		Type:    EOF,
		Content: "",
	}
}
