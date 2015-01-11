package lang

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

//go:generate go tool yacc -p parser lang.y

// The parser expects the lexer to return 0 on EOF.
const lexEOF = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type parserLex struct {
	Err   error
	Input string

	interpolationDepth int
	pos                int
	width              int
}

// The parser calls this method to get each new token.
func (x *parserLex) Lex(yylval *parserSymType) int {
	for {
		c := x.next()
		if c == lexEOF {
			return lexEOF
		}

		// Are we starting an interpolation?
		if c == '$' && x.peek() == '{' {
			x.next()
			x.interpolationDepth++
			return PROGRAM_BRACKET_LEFT
		}

		if x.interpolationDepth == 0 {
			// We're just a normal string that isn't part of any
			// interpolation yet.
			x.backup()
			return x.lexString(yylval, false)
		}

		// Ignore all whitespace
		if unicode.IsSpace(c) {
			continue
		}

		// If we see a double quote and we're in an interpolation, then
		// we are lexing a string.
		if c == '"' {
			return x.lexString(yylval, true)
		}

		switch c {
		case '}':
			x.interpolationDepth--
			return PROGRAM_BRACKET_RIGHT
		case '(':
			return PAREN_LEFT
		case ')':
			return PAREN_RIGHT
		case ',':
			return COMMA
		default:
			x.backup()
			return x.lexId(yylval)
		}
	}
}

func (x *parserLex) lexId(yylval *parserSymType) int {
	var b bytes.Buffer
	for {
		c := x.next()
		if c == lexEOF {
			break
		}

		// If this isn't a character we want in an ID, return out.
		// One day we should make this a regexp.
		if c != '_' &&
			c != '-' &&
			c != '.' &&
			c != '*' &&
			!unicode.IsLetter(c) &&
			!unicode.IsNumber(c) {
			x.backup()
			break
		}

		if _, err := b.WriteRune(c); err != nil {
			x.Error(err.Error())
			return lexEOF
		}
	}

	yylval.str = b.String()
	return IDENTIFIER
}

func (x *parserLex) lexString(yylval *parserSymType, quoted bool) int {
	var b bytes.Buffer
	for {
		c := x.next()
		if c == lexEOF {
			break
		}

		// Behavior is a bit different if we're lexing within a quoted string.
		if quoted {
			// If its a double quote, we've reached the end of the string
			if c == '"' {
				break
			}

			// Let's check to see if we're escaping anything.
			if c == '\\' {
				switch n := x.next(); n {
				case '\\':
					fallthrough
				case '"':
					c = n
				case 'n':
					c = '\n'
				default:
					x.backup()
				}
			}
		}

		// If we hit a '}' and we're in a program, then end it.
		if c == '}' && x.interpolationDepth > 0 {
			x.backup()
			break
		}

		// If we hit a dollar sign, then check if we're starting
		// another interpolation. If so, then we're done.
		if c == '$' && x.peek() == '{' {
			x.backup()
			break
		}

		if _, err := b.WriteRune(c); err != nil {
			x.Error(err.Error())
			return lexEOF
		}
	}

	yylval.str = b.String()
	return STRING
}

// Return the next rune for the lexer.
func (x *parserLex) next() rune {
	if int(x.pos) >= len(x.Input) {
		x.width = 0
		return lexEOF
	}

	r, w := utf8.DecodeRuneInString(x.Input[x.pos:])
	x.width = w
	x.pos += x.width
	return r
}

// peek returns but does not consume the next rune in the input
func (x *parserLex) peek() rune {
	r := x.next()
	x.backup()
	return r
}

// backup steps back one rune. Can only be called once per next.
func (x *parserLex) backup() {
	x.pos -= x.width
}

// The parser calls this method on a parse error.
func (x *parserLex) Error(s string) {
	x.Err = fmt.Errorf("parse error: %s", s)
}
