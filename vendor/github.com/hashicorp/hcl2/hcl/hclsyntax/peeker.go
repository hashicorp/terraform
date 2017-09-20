package hclsyntax

import (
	"github.com/hashicorp/hcl2/hcl"
)

type peeker struct {
	Tokens    Tokens
	NextIndex int

	IncludeComments      bool
	IncludeNewlinesStack []bool
}

func newPeeker(tokens Tokens, includeComments bool) *peeker {
	return &peeker{
		Tokens:          tokens,
		IncludeComments: includeComments,

		IncludeNewlinesStack: []bool{true},
	}
}

func (p *peeker) Peek() Token {
	ret, _ := p.nextToken()
	return ret
}

func (p *peeker) Read() Token {
	ret, nextIdx := p.nextToken()
	p.NextIndex = nextIdx
	return ret
}

func (p *peeker) NextRange() hcl.Range {
	return p.Peek().Range
}

func (p *peeker) PrevRange() hcl.Range {
	if p.NextIndex == 0 {
		return p.NextRange()
	}

	return p.Tokens[p.NextIndex-1].Range
}

func (p *peeker) nextToken() (Token, int) {
	for i := p.NextIndex; i < len(p.Tokens); i++ {
		tok := p.Tokens[i]
		switch tok.Type {
		case TokenComment:
			if !p.IncludeComments {
				// Single-line comment tokens, starting with # or //, absorb
				// the trailing newline that terminates them as part of their
				// bytes. When we're filtering out comments, we must as a
				// special case transform these to newline tokens in order
				// to properly parse newline-terminated block items.

				if p.includingNewlines() {
					if len(tok.Bytes) > 0 && tok.Bytes[len(tok.Bytes)-1] == '\n' {
						fakeNewline := Token{
							Type:  TokenNewline,
							Bytes: tok.Bytes[len(tok.Bytes)-1 : len(tok.Bytes)],

							// We use the whole token range as the newline
							// range, even though that's a little... weird,
							// because otherwise we'd need to go count
							// characters again in order to figure out the
							// column of the newline, and that complexity
							// isn't justified when ranges of newlines are
							// so rarely printed anyway.
							Range: tok.Range,
						}
						return fakeNewline, i + 1
					}
				}

				continue
			}
		case TokenNewline:
			if !p.includingNewlines() {
				continue
			}
		}

		return tok, i + 1
	}

	// if we fall out here then we'll return the EOF token, and leave
	// our index pointed off the end of the array so we'll keep
	// returning EOF in future too.
	return p.Tokens[len(p.Tokens)-1], len(p.Tokens)
}

func (p *peeker) includingNewlines() bool {
	return p.IncludeNewlinesStack[len(p.IncludeNewlinesStack)-1]
}

func (p *peeker) PushIncludeNewlines(include bool) {
	p.IncludeNewlinesStack = append(p.IncludeNewlinesStack, include)
}

func (p *peeker) PopIncludeNewlines() bool {
	stack := p.IncludeNewlinesStack
	remain, ret := stack[:len(stack)-1], stack[len(stack)-1]
	p.IncludeNewlinesStack = remain
	return ret
}
