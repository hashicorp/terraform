package dataurl

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type item struct {
	t   itemType
	val string
}

func (i item) String() string {
	switch i.t {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	if len(i.val) > 10 {
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

type itemType int

const (
	itemError itemType = iota
	itemEOF

	itemDataPrefix

	itemMediaType
	itemMediaSep
	itemMediaSubType
	itemParamSemicolon
	itemParamAttr
	itemParamEqual
	itemLeftStringQuote
	itemRightStringQuote
	itemParamVal

	itemBase64Enc

	itemDataComma
	itemData
)

const eof rune = -1

func isTokenRune(r rune) bool {
	return r <= unicode.MaxASCII &&
		!unicode.IsControl(r) &&
		!unicode.IsSpace(r) &&
		!isTSpecialRune(r)
}

func isTSpecialRune(r rune) bool {
	return r == '(' ||
		r == ')' ||
		r == '<' ||
		r == '>' ||
		r == '@' ||
		r == ',' ||
		r == ';' ||
		r == ':' ||
		r == '\\' ||
		r == '"' ||
		r == '/' ||
		r == '[' ||
		r == ']' ||
		r == '?' ||
		r == '='
}

// See http://tools.ietf.org/html/rfc2045
// This doesn't include extension-token case
// as it's handled separatly
func isDiscreteType(s string) bool {
	if strings.HasPrefix(s, "text") ||
		strings.HasPrefix(s, "image") ||
		strings.HasPrefix(s, "audio") ||
		strings.HasPrefix(s, "video") ||
		strings.HasPrefix(s, "application") {
		return true
	}
	return false
}

// See http://tools.ietf.org/html/rfc2045
// This doesn't include extension-token case
// as it's handled separatly
func isCompositeType(s string) bool {
	if strings.HasPrefix(s, "message") ||
		strings.HasPrefix(s, "multipart") {
		return true
	}
	return false
}

func isURLCharRune(r rune) bool {
	// We're a bit permissive here,
	// by not including '%' in delims
	// This is okay, since url unescaping will validate
	// that later in the parser.
	return r <= unicode.MaxASCII &&
		!(r >= 0x00 && r <= 0x1F) && r != 0x7F && /* control */
		// delims
		r != ' ' &&
		r != '<' &&
		r != '>' &&
		r != '#' &&
		r != '"' &&
		// unwise
		r != '{' &&
		r != '}' &&
		r != '|' &&
		r != '\\' &&
		r != '^' &&
		r != '[' &&
		r != ']' &&
		r != '`'
}

func isBase64Rune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '+' ||
		r == '/' ||
		r == '=' ||
		r == '\n'
}

type stateFn func(*lexer) stateFn

// lexer lexes the data URL scheme input string.
// The implementation is from the text/template/parser package.
type lexer struct {
	input          string
	start          int
	pos            int
	width          int
	seenBase64Item bool
	items          chan item
}

func (l *lexer) run() {
	for state := lexBeforeDataPrefix; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, fmt.Sprintf(format, args...)}
	return nil
}

func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrently run state machine.
	return l
}

const (
	dataPrefix     = "data:"
	mediaSep       = '/'
	paramSemicolon = ';'
	paramEqual     = '='
	dataComma      = ','
)

// start lexing by detecting data prefix
func lexBeforeDataPrefix(l *lexer) stateFn {
	if strings.HasPrefix(l.input[l.pos:], dataPrefix) {
		return lexDataPrefix
	}
	return l.errorf("missing data prefix")
}

// lex data prefix
func lexDataPrefix(l *lexer) stateFn {
	l.pos += len(dataPrefix)
	l.emit(itemDataPrefix)
	return lexAfterDataPrefix
}

// lex what's after data prefix.
// it can be the media type/subtype separator,
// the base64 encoding, or the comma preceding the data
func lexAfterDataPrefix(l *lexer) stateFn {
	switch r := l.next(); {
	case r == paramSemicolon:
		l.backup()
		return lexParamSemicolon
	case r == dataComma:
		l.backup()
		return lexDataComma
	case r == eof:
		return l.errorf("missing comma before data")
	case r == 'x' || r == 'X':
		if l.next() == '-' {
			return lexXTokenMediaType
		}
		return lexInDiscreteMediaType
	case isTokenRune(r):
		return lexInDiscreteMediaType
	default:
		return l.errorf("invalid character after data prefix")
	}
}

func lexXTokenMediaType(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == mediaSep:
			l.backup()
			return lexMediaType
		case r == eof:
			return l.errorf("missing media type slash")
		case isTokenRune(r):
		default:
			return l.errorf("invalid character for media type")
		}
	}
}

func lexInDiscreteMediaType(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == mediaSep:
			l.backup()
			// check it's valid discrete type
			if !isDiscreteType(l.input[l.start:l.pos]) &&
				!isCompositeType(l.input[l.start:l.pos]) {
				return l.errorf("invalid media type")
			}
			return lexMediaType
		case r == eof:
			return l.errorf("missing media type slash")
		case isTokenRune(r):
		default:
			return l.errorf("invalid character for media type")
		}
	}
}

func lexMediaType(l *lexer) stateFn {
	if l.pos > l.start {
		l.emit(itemMediaType)
	}
	return lexMediaSep
}

func lexMediaSep(l *lexer) stateFn {
	l.next()
	l.emit(itemMediaSep)
	return lexAfterMediaSep
}

func lexAfterMediaSep(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == paramSemicolon || r == dataComma:
			l.backup()
			return lexMediaSubType
		case r == eof:
			return l.errorf("incomplete media type")
		case isTokenRune(r):
		default:
			return l.errorf("invalid character for media subtype")
		}
	}
}

func lexMediaSubType(l *lexer) stateFn {
	if l.pos > l.start {
		l.emit(itemMediaSubType)
	}
	return lexAfterMediaSubType
}

func lexAfterMediaSubType(l *lexer) stateFn {
	switch r := l.next(); {
	case r == paramSemicolon:
		l.backup()
		return lexParamSemicolon
	case r == dataComma:
		l.backup()
		return lexDataComma
	case r == eof:
		return l.errorf("missing comma before data")
	default:
		return l.errorf("expected semicolon or comma")
	}
}

func lexParamSemicolon(l *lexer) stateFn {
	l.next()
	l.emit(itemParamSemicolon)
	return lexAfterParamSemicolon
}

func lexAfterParamSemicolon(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		return l.errorf("unterminated parameter sequence")
	case r == paramEqual || r == dataComma:
		return l.errorf("unterminated parameter sequence")
	case isTokenRune(r):
		l.backup()
		return lexInParamAttr
	default:
		return l.errorf("invalid character for parameter attribute")
	}
}

func lexBase64Enc(l *lexer) stateFn {
	if l.pos > l.start {
		if v := l.input[l.start:l.pos]; v != "base64" {
			return l.errorf("expected base64, got %s", v)
		}
		l.seenBase64Item = true
		l.emit(itemBase64Enc)
	}
	return lexDataComma
}

func lexInParamAttr(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == paramEqual:
			l.backup()
			return lexParamAttr
		case r == dataComma:
			l.backup()
			return lexBase64Enc
		case r == eof:
			return l.errorf("unterminated parameter sequence")
		case isTokenRune(r):
		default:
			return l.errorf("invalid character for parameter attribute")
		}
	}
}

func lexParamAttr(l *lexer) stateFn {
	if l.pos > l.start {
		l.emit(itemParamAttr)
	}
	return lexParamEqual
}

func lexParamEqual(l *lexer) stateFn {
	l.next()
	l.emit(itemParamEqual)
	return lexAfterParamEqual
}

func lexAfterParamEqual(l *lexer) stateFn {
	switch r := l.next(); {
	case r == '"':
		l.emit(itemLeftStringQuote)
		return lexInQuotedStringParamVal
	case r == eof:
		return l.errorf("missing comma before data")
	case isTokenRune(r):
		return lexInParamVal
	default:
		return l.errorf("invalid character for parameter value")
	}
}

func lexInQuotedStringParamVal(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unclosed quoted string")
		case r == '\\':
			return lexEscapedChar
		case r == '"':
			l.backup()
			return lexQuotedStringParamVal
		case r <= unicode.MaxASCII:
		default:
			return l.errorf("invalid character for parameter value")
		}
	}
}

func lexEscapedChar(l *lexer) stateFn {
	switch r := l.next(); {
	case r <= unicode.MaxASCII:
		return lexInQuotedStringParamVal
	case r == eof:
		return l.errorf("unexpected eof")
	default:
		return l.errorf("invalid escaped character")
	}
}

func lexInParamVal(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == paramSemicolon || r == dataComma:
			l.backup()
			return lexParamVal
		case r == eof:
			return l.errorf("missing comma before data")
		case isTokenRune(r):
		default:
			return l.errorf("invalid character for parameter value")
		}
	}
}

func lexQuotedStringParamVal(l *lexer) stateFn {
	if l.pos > l.start {
		l.emit(itemParamVal)
	}
	l.next()
	l.emit(itemRightStringQuote)
	return lexAfterParamVal
}

func lexParamVal(l *lexer) stateFn {
	if l.pos > l.start {
		l.emit(itemParamVal)
	}
	return lexAfterParamVal
}

func lexAfterParamVal(l *lexer) stateFn {
	switch r := l.next(); {
	case r == paramSemicolon:
		l.backup()
		return lexParamSemicolon
	case r == dataComma:
		l.backup()
		return lexDataComma
	case r == eof:
		return l.errorf("missing comma before data")
	default:
		return l.errorf("expected semicolon or comma")
	}
}

func lexDataComma(l *lexer) stateFn {
	l.next()
	l.emit(itemDataComma)
	if l.seenBase64Item {
		return lexBase64Data
	}
	return lexData
}

func lexData(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case r == eof:
			break Loop
		case isURLCharRune(r):
		default:
			return l.errorf("invalid data character")
		}
	}
	if l.pos > l.start {
		l.emit(itemData)
	}
	l.emit(itemEOF)
	return nil
}

func lexBase64Data(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case r == eof:
			break Loop
		case isBase64Rune(r):
		default:
			return l.errorf("invalid data character")
		}
	}
	if l.pos > l.start {
		l.emit(itemData)
	}
	l.emit(itemEOF)
	return nil
}
