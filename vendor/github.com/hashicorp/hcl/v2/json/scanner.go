package json

import (
	"fmt"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/hcl/v2"
)

//go:generate stringer -type tokenType scanner.go
type tokenType rune

const (
	tokenBraceO  tokenType = '{'
	tokenBraceC  tokenType = '}'
	tokenBrackO  tokenType = '['
	tokenBrackC  tokenType = ']'
	tokenComma   tokenType = ','
	tokenColon   tokenType = ':'
	tokenKeyword tokenType = 'K'
	tokenString  tokenType = 'S'
	tokenNumber  tokenType = 'N'
	tokenEOF     tokenType = '␄'
	tokenInvalid tokenType = 0
	tokenEquals  tokenType = '=' // used only for reminding the user of JSON syntax
)

type token struct {
	Type  tokenType
	Bytes []byte
	Range hcl.Range
}

// scan returns the primary tokens for the given JSON buffer in sequence.
//
// The responsibility of this pass is to just mark the slices of the buffer
// as being of various types. It is lax in how it interprets the multi-byte
// token types keyword, string and number, preferring to capture erroneous
// extra bytes that we presume the user intended to be part of the token
// so that we can generate more helpful diagnostics in the parser.
func scan(buf []byte, start pos) []token {
	var tokens []token
	p := start
	for {
		if len(buf) == 0 {
			tokens = append(tokens, token{
				Type:  tokenEOF,
				Bytes: nil,
				Range: posRange(p, p),
			})
			return tokens
		}

		buf, p = skipWhitespace(buf, p)

		if len(buf) == 0 {
			tokens = append(tokens, token{
				Type:  tokenEOF,
				Bytes: nil,
				Range: posRange(p, p),
			})
			return tokens
		}

		start = p

		first := buf[0]
		switch {
		case first == '{' || first == '}' || first == '[' || first == ']' || first == ',' || first == ':' || first == '=':
			p.Pos.Column++
			p.Pos.Byte++
			tokens = append(tokens, token{
				Type:  tokenType(first),
				Bytes: buf[0:1],
				Range: posRange(start, p),
			})
			buf = buf[1:]
		case first == '"':
			var tokBuf []byte
			tokBuf, buf, p = scanString(buf, p)
			tokens = append(tokens, token{
				Type:  tokenString,
				Bytes: tokBuf,
				Range: posRange(start, p),
			})
		case byteCanStartNumber(first):
			var tokBuf []byte
			tokBuf, buf, p = scanNumber(buf, p)
			tokens = append(tokens, token{
				Type:  tokenNumber,
				Bytes: tokBuf,
				Range: posRange(start, p),
			})
		case byteCanStartKeyword(first):
			var tokBuf []byte
			tokBuf, buf, p = scanKeyword(buf, p)
			tokens = append(tokens, token{
				Type:  tokenKeyword,
				Bytes: tokBuf,
				Range: posRange(start, p),
			})
		default:
			tokens = append(tokens, token{
				Type:  tokenInvalid,
				Bytes: buf[:1],
				Range: start.Range(1, 1),
			})
			// If we've encountered an invalid then we might as well stop
			// scanning since the parser won't proceed beyond this point.
			return tokens
		}
	}
}

func byteCanStartNumber(b byte) bool {
	switch b {
	// We are slightly more tolerant than JSON requires here since we
	// expect the parser will make a stricter interpretation of the
	// number bytes, but we specifically don't allow 'e' or 'E' here
	// since we want the scanner to treat that as the start of an
	// invalid keyword instead, to produce more intelligible error messages.
	case '-', '+', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	default:
		return false
	}
}

func scanNumber(buf []byte, start pos) ([]byte, []byte, pos) {
	// The scanner doesn't check that the sequence of digit-ish bytes is
	// in a valid order. The parser must do this when decoding a number
	// token.
	var i int
	p := start
Byte:
	for i = 0; i < len(buf); i++ {
		switch buf[i] {
		case '-', '+', '.', 'e', 'E', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			p.Pos.Byte++
			p.Pos.Column++
		default:
			break Byte
		}
	}
	return buf[:i], buf[i:], p
}

func byteCanStartKeyword(b byte) bool {
	switch {
	// We allow any sequence of alphabetical characters here, even though
	// JSON is more constrained, so that we can collect what we presume
	// the user intended to be a single keyword and then check its validity
	// in the parser, where we can generate better diagnostics.
	// So e.g. we want to be able to say:
	//   unrecognized keyword "True". Did you mean "true"?
	case isAlphabetical(b):
		return true
	default:
		return false
	}
}

func scanKeyword(buf []byte, start pos) ([]byte, []byte, pos) {
	var i int
	p := start
Byte:
	for i = 0; i < len(buf); i++ {
		b := buf[i]
		switch {
		case isAlphabetical(b) || b == '_':
			p.Pos.Byte++
			p.Pos.Column++
		default:
			break Byte
		}
	}
	return buf[:i], buf[i:], p
}

func scanString(buf []byte, start pos) ([]byte, []byte, pos) {
	// The scanner doesn't validate correct use of escapes, etc. It pays
	// attention to escapes only for the purpose of identifying the closing
	// quote character. It's the parser's responsibility to do proper
	// validation.
	//
	// The scanner also doesn't specifically detect unterminated string
	// literals, though they can be identified in the parser by checking if
	// the final byte in a string token is the double-quote character.

	// Skip the opening quote symbol
	i := 1
	p := start
	p.Pos.Byte++
	p.Pos.Column++
	escaping := false
Byte:
	for i < len(buf) {
		b := buf[i]

		switch {
		case b == '\\':
			escaping = !escaping
			p.Pos.Byte++
			p.Pos.Column++
			i++
		case b == '"':
			p.Pos.Byte++
			p.Pos.Column++
			i++
			if !escaping {
				break Byte
			}
			escaping = false
		case b < 32:
			break Byte
		default:
			// Advance by one grapheme cluster, so that we consider each
			// grapheme to be a "column".
			// Ignoring error because this scanner cannot produce errors.
			advance, _, _ := textseg.ScanGraphemeClusters(buf[i:], true)

			p.Pos.Byte += advance
			p.Pos.Column++
			i += advance

			escaping = false
		}
	}
	return buf[:i], buf[i:], p
}

func skipWhitespace(buf []byte, start pos) ([]byte, pos) {
	var i int
	p := start
Byte:
	for i = 0; i < len(buf); i++ {
		switch buf[i] {
		case ' ':
			p.Pos.Byte++
			p.Pos.Column++
		case '\n':
			p.Pos.Byte++
			p.Pos.Column = 1
			p.Pos.Line++
		case '\r':
			// For the purpose of line/column counting we consider a
			// carriage return to take up no space, assuming that it will
			// be paired up with a newline (on Windows, for example) that
			// will account for both of them.
			p.Pos.Byte++
		case '\t':
			// We arbitrarily count a tab as if it were two spaces, because
			// we need to choose _some_ number here. This means any system
			// that renders code on-screen with markers must itself treat
			// tabs as a pair of spaces for rendering purposes, or instead
			// use the byte offset and back into its own column position.
			p.Pos.Byte++
			p.Pos.Column += 2
		default:
			break Byte
		}
	}
	return buf[i:], p
}

type pos struct {
	Filename string
	Pos      hcl.Pos
}

func (p *pos) Range(byteLen, charLen int) hcl.Range {
	start := p.Pos
	end := p.Pos
	end.Byte += byteLen
	end.Column += charLen
	return hcl.Range{
		Filename: p.Filename,
		Start:    start,
		End:      end,
	}
}

func posRange(start, end pos) hcl.Range {
	return hcl.Range{
		Filename: start.Filename,
		Start:    start.Pos,
		End:      end.Pos,
	}
}

func (t token) GoString() string {
	return fmt.Sprintf("json.token{json.%s, []byte(%q), %#v}", t.Type, t.Bytes, t.Range)
}

func isAlphabetical(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
