package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	//XItemError is an error with the parser input
	XItemError XItemType = "Error"
	//XItemAbsLocPath is an absolute path
	XItemAbsLocPath = "Absolute path"
	//XItemAbbrAbsLocPath represents an abbreviated absolute path
	XItemAbbrAbsLocPath = "Abbreviated absolute path"
	//XItemAbbrRelLocPath marks the start of a path expression
	XItemAbbrRelLocPath = "Abbreviated relative path"
	//XItemRelLocPath represents a relative location path
	XItemRelLocPath = "Relative path"
	//XItemEndPath marks the end of a path
	XItemEndPath = "End path instruction"
	//XItemAxis marks an axis specifier of a path
	XItemAxis = "Axis"
	//XItemAbbrAxis marks an abbreviated axis specifier (just @ at this point)
	XItemAbbrAxis = "Abbreviated attribute axis"
	//XItemNCName marks a namespace name in a node test
	XItemNCName = "Namespace"
	//XItemQName marks the local name in an a node test
	XItemQName = "Local name"
	//XItemNodeType marks a node type in a node test
	XItemNodeType = "Node type"
	//XItemProcLit marks a processing-instruction literal
	XItemProcLit = "processing-instruction"
	//XItemFunction marks a function call
	XItemFunction = "function"
	//XItemArgument marks a function argument
	XItemArgument = "function argument"
	//XItemEndFunction marks the end of a function
	XItemEndFunction = "end of function"
	//XItemPredicate marks a predicate in an axis
	XItemPredicate = "predicate"
	//XItemEndPredicate marks a predicate in an axis
	XItemEndPredicate = "end of predicate"
	//XItemStrLit marks a string literal
	XItemStrLit = "string literal"
	//XItemNumLit marks a numeric literal
	XItemNumLit = "numeric literal"
	//XItemOperator marks an operator
	XItemOperator = "operator"
	//XItemVariable marks a variable reference
	XItemVariable = "variable"
)

const (
	eof = -(iota + 1)
)

//XItemType is the parser token types
type XItemType string

//XItem is the token emitted from the parser
type XItem struct {
	Typ XItemType
	Val string
}

type stateFn func(*Lexer) stateFn

//Lexer lexes out XPath expressions
type Lexer struct {
	input string
	start int
	pos   int
	width int
	items chan XItem
}

//Lex an XPath expresion on the io.Reader
func Lex(xpath string) chan XItem {
	l := &Lexer{
		input: xpath,
		items: make(chan XItem),
	}
	go l.run()
	return l.items
}

func (l *Lexer) run() {
	for state := startState; state != nil; {
		state = state(l)
	}

	if l.peek() != eof {
		l.errorf("Malformed XPath expression")
	}

	close(l.items)
}

func (l *Lexer) emit(t XItemType) {
	l.items <- XItem{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *Lexer) emitVal(t XItemType, val string) {
	l.items <- XItem{t, val}
	l.start = l.pos
}

func (l *Lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])

	l.pos += l.width

	return r
}

func (l *Lexer) ignore() {
	l.start = l.pos
}

func (l *Lexer) backup() {
	l.pos -= l.width
}

func (l *Lexer) peek() rune {
	r := l.next()

	l.backup()
	return r
}

func (l *Lexer) peekAt(n int) rune {
	if n <= 1 {
		return l.peek()
	}

	width := 0
	var ret rune

	for count := 0; count < n; count++ {
		r, s := utf8.DecodeRuneInString(l.input[l.pos+width:])
		width += s

		if l.pos+width > len(l.input) {
			return eof
		}

		ret = r
	}

	return ret
}

func (l *Lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}

	l.backup()
	return false
}

func (l *Lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *Lexer) skip(num int) {
	for i := 0; i < num; i++ {
		l.next()
	}
	l.ignore()
}

func (l *Lexer) skipWS(ig bool) {
	for {
		n := l.next()

		if n == eof || !unicode.IsSpace(n) {
			break
		}
	}

	l.backup()

	if ig {
		l.ignore()
	}
}

func (l *Lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- XItem{
		XItemError,
		fmt.Sprintf(format, args...),
	}

	return nil
}

func isElemChar(r rune) bool {
	return string(r) != ":" && string(r) != "/" &&
		(unicode.Is(first, r) || unicode.Is(second, r) || string(r) == "*") &&
		r != eof
}

func startState(l *Lexer) stateFn {
	l.skipWS(true)

	if string(l.peek()) == "/" {
		l.next()
		l.ignore()

		if string(l.next()) == "/" {
			l.ignore()
			return abbrAbsLocPathState
		}

		l.backup()
		return absLocPathState
	} else if string(l.peek()) == `'` || string(l.peek()) == `"` {
		if err := getStrLit(l, XItemStrLit); err != nil {
			return l.errorf(err.Error())
		}

		if l.peek() != eof {
			return startState
		}
	} else if getNumLit(l) {
		l.skipWS(true)
		if l.peek() != eof {
			return startState
		}
	} else if string(l.peek()) == "$" {
		l.next()
		l.ignore()
		r := l.peek()
		for unicode.Is(first, r) || unicode.Is(second, r) {
			l.next()
			r = l.peek()
		}
		tok := l.input[l.start:l.pos]
		if len(tok) == 0 {
			return l.errorf("Empty variable name")
		}
		l.emit(XItemVariable)
		l.skipWS(true)
		if l.peek() != eof {
			return startState
		}
	} else if st := findOperatorState(l); st != nil {
		return st
	} else {
		if isElemChar(l.peek()) {
			colons := 0

			for {
				if isElemChar(l.peek()) {
					l.next()
				} else if string(l.peek()) == ":" {
					l.next()
					colons++
				} else {
					break
				}
			}

			if string(l.peek()) == "(" && colons <= 1 {
				tok := l.input[l.start:l.pos]
				err := procFunc(l, tok)
				if err != nil {
					return l.errorf(err.Error())
				}

				l.skipWS(true)

				if string(l.peek()) == "/" {
					l.next()
					l.ignore()

					if string(l.next()) == "/" {
						l.ignore()
						return abbrRelLocPathState
					}

					l.backup()
					return relLocPathState
				}

				return startState
			}

			l.pos = l.start
			return relLocPathState
		} else if string(l.peek()) == "@" {
			return relLocPathState
		}
	}

	return nil
}

func strPeek(str string, l *Lexer) bool {
	for i := 0; i < len(str); i++ {
		if string(l.peekAt(i+1)) != string(str[i]) {
			return false
		}
	}
	return true
}

func findOperatorState(l *Lexer) stateFn {
	l.skipWS(true)

	switch string(l.peek()) {
	case ">", "<", "!":
		l.next()
		if string(l.peek()) == "=" {
			l.next()
		}
		l.emit(XItemOperator)
		return startState
	case "|", "+", "-", "*", "=":
		l.next()
		l.emit(XItemOperator)
		return startState
	case "(":
		l.next()
		l.emit(XItemOperator)
		for state := startState; state != nil; {
			state = state(l)
		}
		l.skipWS(true)
		if string(l.next()) != ")" {
			return l.errorf("Missing end )")
		}
		l.emit(XItemOperator)
		return startState
	}

	if strPeek("and", l) {
		l.next()
		l.next()
		l.next()
		l.emit(XItemOperator)
		return startState
	}

	if strPeek("or", l) {
		l.next()
		l.next()
		l.emit(XItemOperator)
		return startState
	}

	if strPeek("mod", l) {
		l.next()
		l.next()
		l.next()
		l.emit(XItemOperator)
		return startState
	}

	if strPeek("div", l) {
		l.next()
		l.next()
		l.next()
		l.emit(XItemOperator)
		return startState
	}

	return nil
}

func getStrLit(l *Lexer, tok XItemType) error {
	q := l.next()
	var r rune

	l.ignore()

	for r != q {
		r = l.next()
		if r == eof {
			return fmt.Errorf("Unexpected end of string literal.")
		}
	}

	l.backup()
	l.emit(tok)
	l.next()
	l.ignore()

	return nil
}

func getNumLit(l *Lexer) bool {
	const dig = "0123456789"
	l.accept("-")
	start := l.pos
	l.acceptRun(dig)

	if l.pos == start {
		return false
	}

	if l.accept(".") {
		l.acceptRun(dig)
	}

	l.emit(XItemNumLit)
	return true
}
