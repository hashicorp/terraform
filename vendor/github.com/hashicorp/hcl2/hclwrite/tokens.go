package hclwrite

import (
	"bytes"
	"io"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
)

// TokenGen is an abstract type that can append tokens to a list. It is the
// low-level foundation underlying the zclwrite AST; the AST provides a
// convenient abstraction over raw token sequences to facilitate common tasks,
// but it's also possible to directly manipulate the tree of token generators
// to make changes that the AST API doesn't directly allow.
type TokenGen interface {
	EachToken(TokenCallback)
}

// TokenCallback is used with TokenGen implementations to specify the action
// that is to be taken for each token in the flattened token sequence.
type TokenCallback func(*Token)

// Token is a single sequence of bytes annotated with a type. It is similar
// in purpose to zclsyntax.Token, but discards the source position information
// since that is not useful in code generation.
type Token struct {
	Type  hclsyntax.TokenType
	Bytes []byte

	// We record the number of spaces before each token so that we can
	// reproduce the exact layout of the original file when we're making
	// surgical changes in-place. When _new_ code is created it will always
	// be in the canonical style, but we preserve layout of existing code.
	SpacesBefore int
}

// Tokens is a flat list of tokens.
type Tokens []*Token

func (ts Tokens) WriteTo(wr io.Writer) (int, error) {
	seq := &TokenSeq{ts}
	return seq.WriteTo(wr)
}

func (ts Tokens) Bytes() []byte {
	buf := &bytes.Buffer{}
	ts.WriteTo(buf)
	return buf.Bytes()
}

// Columns returns the number of columns (grapheme clusters) the token sequence
// occupies. The result is not meaningful if there are newline or single-line
// comment tokens in the sequence.
func (ts Tokens) Columns() int {
	ret := 0
	for _, token := range ts {
		ret += token.SpacesBefore // spaces are always worth one column each
		ct, _ := textseg.TokenCount(token.Bytes, textseg.ScanGraphemeClusters)
		ret += ct
	}
	return ret
}

// TokenSeq combines zero or more TokenGens together to produce a flat sequence
// of tokens from a tree of TokenGens.
type TokenSeq []TokenGen

func (t *Token) EachToken(cb TokenCallback) {
	cb(t)
}

func (ts Tokens) EachToken(cb TokenCallback) {
	for _, t := range ts {
		cb(t)
	}
}

func (ts *TokenSeq) EachToken(cb TokenCallback) {
	if ts == nil {
		return
	}
	for _, gen := range *ts {
		gen.EachToken(cb)
	}
}

// Tokens returns the flat list of tokens represented by the receiving
// token sequence.
func (ts *TokenSeq) Tokens() Tokens {
	var tokens Tokens
	ts.EachToken(func(token *Token) {
		tokens = append(tokens, token)
	})
	return tokens
}

// WriteTo takes an io.Writer and writes the bytes for each token to it,
// along with the spacing that separates each token. In other words, this
// allows serializing the tokens to a file or other such byte stream.
func (ts *TokenSeq) WriteTo(wr io.Writer) (int, error) {
	// We know we're going to be writing a lot of small chunks of repeated
	// space characters, so we'll prepare a buffer of these that we can
	// easily pass to wr.Write without any further allocation.
	spaces := make([]byte, 40)
	for i := range spaces {
		spaces[i] = ' '
	}

	var n int
	var err error
	ts.EachToken(func(token *Token) {
		if err != nil {
			return
		}

		for spacesBefore := token.SpacesBefore; spacesBefore > 0; spacesBefore -= len(spaces) {
			thisChunk := spacesBefore
			if thisChunk > len(spaces) {
				thisChunk = len(spaces)
			}
			var thisN int
			thisN, err = wr.Write(spaces[:thisChunk])
			n += thisN
			if err != nil {
				return
			}
		}

		var thisN int
		thisN, err = wr.Write(token.Bytes)
		n += thisN
	})

	return n, err
}

// SoloToken returns the single token represented by the receiving sequence,
// or nil if the sequence does not represent exactly one token.
func (ts *TokenSeq) SoloToken() *Token {
	var ret *Token
	found := false
	ts.EachToken(func(tok *Token) {
		if ret == nil && !found {
			ret = tok
			found = true
		} else if ret != nil && found {
			ret = nil
		}
	})
	return ret
}

// IsIdent returns true if and only if the token sequence represents a single
// ident token whose name matches the given string.
func (ts *TokenSeq) IsIdent(name []byte) bool {
	tok := ts.SoloToken()
	if tok == nil {
		return false
	}
	if tok.Type != hclsyntax.TokenIdent {
		return false
	}
	return bytes.Equal(tok.Bytes, name)
}

// TokenSeqEmpty is a TokenSeq that contains no tokens. It can be used anywhere,
// but its primary purpose is to be assigned as a replacement for a non-empty
// TokenSeq when eliminating a section of an input file.
var TokenSeqEmpty = TokenSeq([]TokenGen(nil))
