package scanner

import (
	"unicode"
	"unicode/utf8"

	"github.com/hashicorp/hil/ast"
)

// Scan returns a channel that recieves Tokens from the given input string.
//
// The scanner's job is just to partition the string into meaningful parts.
// It doesn't do any transformation of the raw input string, so the caller
// must deal with any further interpretation required, such as parsing INTEGER
// tokens into real ints, or dealing with escape sequences in LITERAL or
// STRING tokens.
//
// Strings in the returned tokens are slices from the original string.
//
// startPos should be set to ast.InitPos unless the caller knows that
// this interpolation string is part of a larger file and knows the position
// of the first character in that larger file.
func Scan(s string, startPos ast.Pos) <-chan *Token {
	ch := make(chan *Token)
	go scan(s, ch, startPos)
	return ch
}

func scan(s string, ch chan<- *Token, pos ast.Pos) {
	// 'remain' starts off as the whole string but we gradually
	// slice of the front of it as we work our way through.
	remain := s

	// nesting keeps track of how many ${ .. } sequences we are
	// inside, so we can recognize the minor differences in syntax
	// between outer string literals (LITERAL tokens) and quoted
	// string literals (STRING tokens).
	nesting := 0

	// We're going to flip back and forth between parsing literals/strings
	// and parsing interpolation sequences ${ .. } until we reach EOF or
	// some INVALID token.
All:
	for {
		startPos := pos
		// Literal string processing first, since the beginning of
		// a string is always outside of an interpolation sequence.
		literalVal, terminator := scanLiteral(remain, pos, nesting > 0)

		if len(literalVal) > 0 {
			litType := LITERAL
			if nesting > 0 {
				litType = STRING
			}
			ch <- &Token{
				Type:    litType,
				Content: literalVal,
				Pos:     startPos,
			}
			remain = remain[len(literalVal):]
		}

		ch <- terminator
		remain = remain[len(terminator.Content):]
		pos = terminator.Pos
		// Safe to use len() here because none of the terminator tokens
		// can contain UTF-8 sequences.
		pos.Column = pos.Column + len(terminator.Content)

		switch terminator.Type {
		case INVALID:
			// Synthetic EOF after invalid token, since further scanning
			// is likely to just produce more garbage.
			ch <- &Token{
				Type:    EOF,
				Content: "",
				Pos:     pos,
			}
			break All
		case EOF:
			// All done!
			break All
		case BEGIN:
			nesting++
		case CQUOTE:
			// nothing special to do
		default:
			// Should never happen
			panic("invalid string/literal terminator")
		}

		// Now we do the processing of the insides of ${ .. } sequences.
		// This loop terminates when we encounter either a closing } or
		// an opening ", which will cause us to return to literal processing.
	Interpolation:
		for {

			token, size, newPos := scanInterpolationToken(remain, pos)
			ch <- token
			remain = remain[size:]
			pos = newPos

			switch token.Type {
			case INVALID:
				// Synthetic EOF after invalid token, since further scanning
				// is likely to just produce more garbage.
				ch <- &Token{
					Type:    EOF,
					Content: "",
					Pos:     pos,
				}
				break All
			case EOF:
				// All done
				// (though a syntax error that we'll catch in the parser)
				break All
			case END:
				nesting--
				if nesting < 0 {
					// Can happen if there are unbalanced ${ and } sequences
					// in the input, which we'll catch in the parser.
					nesting = 0
				}
				break Interpolation
			case OQUOTE:
				// Beginning of nested quoted string
				break Interpolation
			}
		}
	}

	close(ch)
}

// Returns the token found at the start of the given string, followed by
// the number of bytes that were consumed from the string and the adjusted
// source position.
//
// Note that the number of bytes consumed can be more than the length of
// the returned token contents if the string begins with whitespace, since
// it will be silently consumed before reading the token.
func scanInterpolationToken(s string, startPos ast.Pos) (*Token, int, ast.Pos) {
	pos := startPos
	size := 0

	// Consume whitespace, if any
	for len(s) > 0 && byteIsSpace(s[0]) {
		if s[0] == '\n' {
			pos.Column = 1
			pos.Line++
		} else {
			pos.Column++
		}
		size++
		s = s[1:]
	}

	// Unexpected EOF during sequence
	if len(s) == 0 {
		return &Token{
			Type:    EOF,
			Content: "",
			Pos:     pos,
		}, size, pos
	}

	next := s[0]
	var token *Token

	switch next {
	case '(', ')', '[', ']', ',', '.', '+', '-', '*', '/', '%', '?', ':':
		// Easy punctuation symbols that don't have any special meaning
		// during scanning, and that stand for themselves in the
		// TokenType enumeration.
		token = &Token{
			Type:    TokenType(next),
			Content: s[:1],
			Pos:     pos,
		}
	case '}':
		token = &Token{
			Type:    END,
			Content: s[:1],
			Pos:     pos,
		}
	case '"':
		token = &Token{
			Type:    OQUOTE,
			Content: s[:1],
			Pos:     pos,
		}
	case '!':
		if len(s) >= 2 && s[:2] == "!=" {
			token = &Token{
				Type:    NOTEQUAL,
				Content: s[:2],
				Pos:     pos,
			}
		} else {
			token = &Token{
				Type:    BANG,
				Content: s[:1],
				Pos:     pos,
			}
		}
	case '<':
		if len(s) >= 2 && s[:2] == "<=" {
			token = &Token{
				Type:    LTE,
				Content: s[:2],
				Pos:     pos,
			}
		} else {
			token = &Token{
				Type:    LT,
				Content: s[:1],
				Pos:     pos,
			}
		}
	case '>':
		if len(s) >= 2 && s[:2] == ">=" {
			token = &Token{
				Type:    GTE,
				Content: s[:2],
				Pos:     pos,
			}
		} else {
			token = &Token{
				Type:    GT,
				Content: s[:1],
				Pos:     pos,
			}
		}
	case '=':
		if len(s) >= 2 && s[:2] == "==" {
			token = &Token{
				Type:    EQUAL,
				Content: s[:2],
				Pos:     pos,
			}
		} else {
			// A single equals is not a valid operator
			token = &Token{
				Type:    INVALID,
				Content: s[:1],
				Pos:     pos,
			}
		}
	case '&':
		if len(s) >= 2 && s[:2] == "&&" {
			token = &Token{
				Type:    AND,
				Content: s[:2],
				Pos:     pos,
			}
		} else {
			token = &Token{
				Type:    INVALID,
				Content: s[:1],
				Pos:     pos,
			}
		}
	case '|':
		if len(s) >= 2 && s[:2] == "||" {
			token = &Token{
				Type:    OR,
				Content: s[:2],
				Pos:     pos,
			}
		} else {
			token = &Token{
				Type:    INVALID,
				Content: s[:1],
				Pos:     pos,
			}
		}
	default:
		if next >= '0' && next <= '9' {
			num, numType := scanNumber(s)
			token = &Token{
				Type:    numType,
				Content: num,
				Pos:     pos,
			}
		} else if stringStartsWithIdentifier(s) {
			ident, runeLen := scanIdentifier(s)
			tokenType := IDENTIFIER
			if ident == "true" || ident == "false" {
				tokenType = BOOL
			}
			token = &Token{
				Type:    tokenType,
				Content: ident,
				Pos:     pos,
			}
			// Skip usual token handling because it doesn't
			// know how to deal with UTF-8 sequences.
			pos.Column = pos.Column + runeLen
			return token, size + len(ident), pos
		} else {
			_, byteLen := utf8.DecodeRuneInString(s)
			token = &Token{
				Type:    INVALID,
				Content: s[:byteLen],
				Pos:     pos,
			}
			// Skip usual token handling because it doesn't
			// know how to deal with UTF-8 sequences.
			pos.Column = pos.Column + 1
			return token, size + byteLen, pos
		}
	}

	// Here we assume that the token content contains no UTF-8 sequences,
	// because we dealt with UTF-8 characters as a special case where
	// necessary above.
	size = size + len(token.Content)
	pos.Column = pos.Column + len(token.Content)

	return token, size, pos
}

// Returns the (possibly-empty) prefix of the given string that represents
// a literal, followed by the token that marks the end of the literal.
func scanLiteral(s string, startPos ast.Pos, nested bool) (string, *Token) {
	litLen := 0
	pos := startPos
	var terminator *Token
	for {

		if litLen >= len(s) {
			if nested {
				// We've ended in the middle of a quoted string,
				// which means this token is actually invalid.
				return "", &Token{
					Type:    INVALID,
					Content: s,
					Pos:     startPos,
				}
			}
			terminator = &Token{
				Type:    EOF,
				Content: "",
				Pos:     pos,
			}
			break
		}

		next := s[litLen]

		if next == '$' && len(s) > litLen+1 {
			follow := s[litLen+1]

			if follow == '{' {
				terminator = &Token{
					Type:    BEGIN,
					Content: s[litLen : litLen+2],
					Pos:     pos,
				}
				pos.Column = pos.Column + 2
				break
			} else if follow == '$' {
				// Double-$ escapes the special processing of $,
				// so we will consume both characters here.
				pos.Column = pos.Column + 2
				litLen = litLen + 2
				continue
			}
		}

		// special handling that applies only to quoted strings
		if nested {
			if next == '"' {
				terminator = &Token{
					Type:    CQUOTE,
					Content: s[litLen : litLen+1],
					Pos:     pos,
				}
				pos.Column = pos.Column + 1
				break
			}

			// Escaped quote marks do not terminate the string.
			//
			// All we do here in the scanner is avoid terminating a string
			// due to an escaped quote. The parser is responsible for the
			// full handling of escape sequences, since it's able to produce
			// better error messages than we can produce in here.
			if next == '\\' && len(s) > litLen+1 {
				follow := s[litLen+1]

				if follow == '"' {
					// \" escapes the special processing of ",
					// so we will consume both characters here.
					pos.Column = pos.Column + 2
					litLen = litLen + 2
					continue
				} else if follow == '\\' {
					// \\ escapes \
					// so we will consume both characters here.
					pos.Column = pos.Column + 2
					litLen = litLen + 2
					continue
				}
			}
		}

		if next == '\n' {
			pos.Column = 1
			pos.Line++
			litLen++
		} else {
			pos.Column++

			// "Column" measures runes, so we need to actually consume
			// a valid UTF-8 character here.
			_, size := utf8.DecodeRuneInString(s[litLen:])
			litLen = litLen + size
		}

	}

	return s[:litLen], terminator
}

// scanNumber returns the extent of the prefix of the string that represents
// a valid number, along with what type of number it represents: INT or FLOAT.
//
// scanNumber does only basic character analysis: numbers consist of digits
// and periods, with at least one period signalling a FLOAT. It's the parser's
// responsibility to validate the form and range of the number, such as ensuring
// that a FLOAT actually contains only one period, etc.
func scanNumber(s string) (string, TokenType) {
	period := -1
	byteLen := 0
	numType := INTEGER
	for {
		if byteLen >= len(s) {
			break
		}

		next := s[byteLen]
		if next != '.' && (next < '0' || next > '9') {
			// If our last value was a period, then we're not a float,
			// we're just an integer that ends in a period.
			if period == byteLen-1 {
				byteLen--
				numType = INTEGER
			}

			break
		}

		if next == '.' {
			// If we've already seen a period, break out
			if period >= 0 {
				break
			}

			period = byteLen
			numType = FLOAT
		}

		byteLen++
	}

	return s[:byteLen], numType
}

// scanIdentifier returns the extent of the prefix of the string that
// represents a valid identifier, along with the length of that prefix
// in runes.
//
// Identifiers may contain utf8-encoded non-Latin letters, which will
// cause the returned "rune length" to be shorter than the byte length
// of the returned string.
func scanIdentifier(s string) (string, int) {
	byteLen := 0
	runeLen := 0
	for {
		if byteLen >= len(s) {
			break
		}

		nextRune, size := utf8.DecodeRuneInString(s[byteLen:])
		if !(nextRune == '_' ||
			nextRune == '-' ||
			nextRune == '.' ||
			nextRune == '*' ||
			unicode.IsNumber(nextRune) ||
			unicode.IsLetter(nextRune) ||
			unicode.IsMark(nextRune)) {
			break
		}

		// If we reach a star, it must be between periods to be part
		// of the same identifier.
		if nextRune == '*' && s[byteLen-1] != '.' {
			break
		}

		// If our previous character was a star, then the current must
		// be period. Otherwise, undo that and exit.
		if byteLen > 0 && s[byteLen-1] == '*' && nextRune != '.' {
			byteLen--
			if s[byteLen-1] == '.' {
				byteLen--
			}

			break
		}

		byteLen = byteLen + size
		runeLen = runeLen + 1
	}

	return s[:byteLen], runeLen
}

// byteIsSpace implements a restrictive interpretation of spaces that includes
// only what's valid inside interpolation sequences: spaces, tabs, newlines.
func byteIsSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

// stringStartsWithIdentifier returns true if the given string begins with
// a character that is a legal start of an identifier: an underscore or
// any character that Unicode considers to be a letter.
func stringStartsWithIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	first := s[0]

	// Easy ASCII cases first
	if (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_' {
		return true
	}

	// If our first byte begins a UTF-8 sequence then the sequence might
	// be a unicode letter.
	if utf8.RuneStart(first) {
		firstRune, _ := utf8.DecodeRuneInString(s)
		if unicode.IsLetter(firstRune) {
			return true
		}
	}

	return false
}
