// Package dump contains some helper functions for printing and formatting
// values.
package dump

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

// Value produces a formatted string representation of the given value.
//
// The result is a pretty-printed version of the result of the fmt.GoStringer
// implementation for the given value. The pretty-printing expects the
// GoString result to be a valid Go expression; if it is not then the result
// may be sub-optimal but this function will still make a best effort.
//
// This function is intended primarily as a helper for writing unit tests, and
// so it is not optimized for performance in any way.
func Value(v interface{}) string {
	return prettyPrint(fmt.Sprintf("%#v", v))
}

func prettyPrint(s string) string {
	var buf bytes.Buffer
	fs := token.NewFileSet()
	f := fs.AddFile("", -1, len(s))
	sc := &scanner.Scanner{}
	sc.Init(f, []byte(s), nil, scanner.ScanComments)

	type Token struct {
		Type token.Token
		Str  string
	}

	var tokens []Token
	for {
		_, tok, lit := sc.Scan()

		switch tok {
		case token.IDENT, token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING, token.COMMENT, token.SEMICOLON, token.ILLEGAL:
			// "lit" is already populated
		default:
			lit = tok.String()
		}

		tokens = append(tokens, Token{
			Type: tok,
			Str:  lit,
		})

		if tok == token.EOF {
			break
		}
	}

	indent := 0
	otherBrackets := 0
	emptyBraces := false
	for i, tok := range tokens {
		if tok.Type == token.EOF {
			break
		}
		nextTok := tokens[i+1]

		switch {
		case tok.Type == token.LBRACE:
			if otherBrackets > 0 {
				buf.WriteString("{")
				continue
			}
			if nextTok.Type == token.RBRACE {
				buf.WriteString("{")
				emptyBraces = true
				continue
			}
			indent++
			buf.WriteString("{\n" + strings.Repeat("  ", indent))
		case tok.Type == token.RBRACE:
			if otherBrackets > 0 {
				buf.WriteString("}")
				continue
			}
			if emptyBraces {
				buf.WriteString("}")
				emptyBraces = false
				continue
			}
			indent--
			sp := strings.Repeat("  ", indent)
			if nextTok.Type == token.SEMICOLON || nextTok.Type == token.COMMA || nextTok.Type == token.RBRACE {
				buf.WriteString(fmt.Sprintf("\n%s}", sp))
			} else {
				buf.WriteString(fmt.Sprintf("\n%s}\n%s", sp, sp))
			}
		case tok.Type == token.LBRACK || tok.Type == token.LPAREN:
			buf.WriteString(tok.Str)
			otherBrackets++
		case tok.Type == token.RBRACK || tok.Type == token.RPAREN:
			buf.WriteString(tok.Str)
			otherBrackets--
		case tok.Type == token.COMMA:
			if otherBrackets > 0 {
				buf.WriteString(", ")
				continue
			}
			buf.WriteString(",\n" + strings.Repeat("  ", indent))
		case tok.Type == token.SEMICOLON:
			if otherBrackets > 0 {
				buf.WriteString("; ")
				continue
			}
			buf.WriteString("\n" + strings.Repeat("  ", indent))
		case tok.Type == token.COLON:
			buf.WriteString(": ")
		default:
			buf.WriteString(tok.Str)
		}
	}

	return buf.String()
}
