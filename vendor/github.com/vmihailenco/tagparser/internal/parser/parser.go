package parser

import (
	"bytes"

	"github.com/vmihailenco/tagparser/internal"
)

type Parser struct {
	b []byte
	i int
}

func New(b []byte) *Parser {
	return &Parser{
		b: b,
	}
}

func NewString(s string) *Parser {
	return New(internal.StringToBytes(s))
}

func (p *Parser) Bytes() []byte {
	return p.b[p.i:]
}

func (p *Parser) Valid() bool {
	return p.i < len(p.b)
}

func (p *Parser) Read() byte {
	if p.Valid() {
		c := p.b[p.i]
		p.Advance()
		return c
	}
	return 0
}

func (p *Parser) Peek() byte {
	if p.Valid() {
		return p.b[p.i]
	}
	return 0
}

func (p *Parser) Advance() {
	p.i++
}

func (p *Parser) Skip(skip byte) bool {
	if p.Peek() == skip {
		p.Advance()
		return true
	}
	return false
}

func (p *Parser) SkipBytes(skip []byte) bool {
	if len(skip) > len(p.b[p.i:]) {
		return false
	}
	if !bytes.Equal(p.b[p.i:p.i+len(skip)], skip) {
		return false
	}
	p.i += len(skip)
	return true
}

func (p *Parser) ReadSep(sep byte) ([]byte, bool) {
	ind := bytes.IndexByte(p.b[p.i:], sep)
	if ind == -1 {
		b := p.b[p.i:]
		p.i = len(p.b)
		return b, false
	}

	b := p.b[p.i : p.i+ind]
	p.i += ind + 1
	return b, true
}
