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

	mode               parserMode
	interpolationDepth int
	pos                int
	width              int
}

// parserMode keeps track of what mode we're in for the parser. We have
// two modes: literal and interpolation. Literal mode is when strings
// don't have to be quoted, and interpolations are defined as ${foo}.
// Interpolation mode means that strings have to be quoted and unquoted
// things are identifiers, such as foo("bar").
type parserMode uint8

const (
	parserModeInvalid parserMode = 0
	parserModeLiteral            = 1 << iota
	parserModeInterpolation
)

// The parser calls this method to get each new token.
func (x *parserLex) Lex(yylval *parserSymType) int {
	if x.mode == parserModeInvalid {
		x.mode = parserModeLiteral
	}

	switch x.mode {
	case parserModeLiteral:
		return x.lexModeLiteral(yylval)
	case parserModeInterpolation:
		return x.lexModeInterpolation(yylval)
	default:
		x.Error(fmt.Sprintf("Unknown parse mode: %s", x.mode))
		return lexEOF
	}
}

func (x *parserLex) lexModeLiteral(yylval *parserSymType) int {
	for {
		c := x.next()
		if c == lexEOF {
			return lexEOF
		}

		// Are we starting an interpolation?
		if c == '$' && x.peek() == '{' {
			x.next()
			x.interpolationDepth++
			x.mode = parserModeInterpolation
			return PROGRAM_BRACKET_LEFT
		}

		// We're just a normal string that isn't part of any interpolation yet.
		x.backup()
		result, terminated := x.lexString(yylval, x.interpolationDepth > 0)

		// If the string terminated and we're within an interpolation already
		// then that means that we finished a nested string, so pop
		// back out to interpolation mode.
		if terminated && x.interpolationDepth > 0 {
			x.mode = parserModeInterpolation

			// If the string is empty, just skip it. We're still in
			// an interpolation so we do this to avoid empty nodes.
			if yylval.str == "" {
				return x.Lex(yylval)
			}
		}

		return result
	}
}

func (x *parserLex) lexModeInterpolation(yylval *parserSymType) int {
	for {
		c := x.next()
		if c == lexEOF {
			return lexEOF
		}

		// Ignore all whitespace
		if unicode.IsSpace(c) {
			continue
		}

		// If we see a double quote and we're in an interpolation, then
		// we are lexing a string.
		if c == '"' {
			result, terminated := x.lexString(yylval, true)
			if !terminated {
				// The string didn't end, which means that we're in the
				// middle of starting another interpolation.
				x.mode = parserModeLiteral

				// If the string is empty and we're starting an interpolation,
				// then just skip it to avoid empty string AST nodes
				if yylval.str == "" {
					return x.Lex(yylval)
				}
			}

			return result
		}

		switch c {
		case '}':
			x.interpolationDepth--
			x.mode = parserModeLiteral
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

func (x *parserLex) lexString(yylval *parserSymType, quoted bool) (int, bool) {
	var b bytes.Buffer
	terminated := false
	for {
		c := x.next()
		if c == lexEOF {
			if quoted {
				x.Error("unterminated string")
			}

			break
		}

		// Behavior is a bit different if we're lexing within a quoted string.
		if quoted {
			// If its a double quote, we've reached the end of the string
			if c == '"' {
				terminated = true
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
			return lexEOF, false
		}
	}

	yylval.str = b.String()
	return STRING, terminated
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
