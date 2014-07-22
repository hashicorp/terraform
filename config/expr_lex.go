package config

import (
	"bytes"
	"log"
	"unicode"
	"unicode/utf8"
)

// The parser expects the lexer to return 0 on EOF.
const lexEOF = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type exprLex struct {
	input string
	pos   int
	width int
}

// The parser calls this method to get each new token.
func (x *exprLex) Lex(yylval *exprSymType) int {
	for {
		c := x.next()
		if c == lexEOF {
			return lexEOF
		}

		// Ignore all whitespace
		if unicode.IsSpace(c) {
			continue
		}

		switch c {
		case '"':
			return x.lexString(yylval)
		case ',':
			return COMMA
		case '(':
			return LEFTPAREN
		case ')':
			return RIGHTPAREN
		default:
			x.backup()
			return x.lexId(yylval)
		}
	}
}

func (x *exprLex) lexId(yylval *exprSymType) int {
	var b bytes.Buffer
	for {
		c := x.next()
		if c == lexEOF {
			break
		}

		// If this isn't a character we want in an ID, return out.
		// One day we should make this a regexp.
		if c != '_' &&
			c != '.' &&
			c != '*' &&
			!unicode.IsLetter(c) &&
			!unicode.IsNumber(c) {
			x.backup()
			break
		}

		if _, err := b.WriteRune(c); err != nil {
			log.Printf("ERR: %s", err)
			return lexEOF
		}
	}

	yylval.str = b.String()
	return IDENTIFIER
}

func (x *exprLex) lexString(yylval *exprSymType) int {
	var b bytes.Buffer
	for {
		c := x.next()
		if c == lexEOF {
			break
		}

		// String end
		if c == '"' {
			break
		}

		if _, err := b.WriteRune(c); err != nil {
			log.Printf("ERR: %s", err)
			return lexEOF
		}
	}

	yylval.str = b.String()
	return STRING
}

// Return the next rune for the lexer.
func (x *exprLex) next() rune {
	if int(x.pos) >= len(x.input) {
		x.width = 0
		return lexEOF
	}

	r, w := utf8.DecodeRuneInString(x.input[x.pos:])
	x.width = w
	x.pos += x.width
	return r
}

// peek returns but does not consume the next rune in the input
func (x *exprLex) peek() rune {
	r := x.next()
	x.backup()
	return r
}

// backup steps back one rune. Can only be called once per next.
func (x *exprLex) backup() {
	x.pos -= x.width
}

// The parser calls this method on a parse error.
func (x *exprLex) Error(s string) {
	log.Printf("parse error: %s", s)
}
