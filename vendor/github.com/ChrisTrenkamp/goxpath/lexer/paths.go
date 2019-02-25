package lexer

import (
	"fmt"

	"github.com/ChrisTrenkamp/goxpath/xconst"
)

func absLocPathState(l *Lexer) stateFn {
	l.emit(XItemAbsLocPath)
	return stepState
}

func abbrAbsLocPathState(l *Lexer) stateFn {
	l.emit(XItemAbbrAbsLocPath)
	return stepState
}

func relLocPathState(l *Lexer) stateFn {
	l.emit(XItemRelLocPath)
	return stepState
}

func abbrRelLocPathState(l *Lexer) stateFn {
	l.emit(XItemAbbrRelLocPath)
	return stepState
}

func stepState(l *Lexer) stateFn {
	l.skipWS(true)
	r := l.next()

	for isElemChar(r) {
		r = l.next()
	}

	l.backup()
	tok := l.input[l.start:l.pos]

	state, err := parseSeparators(l, tok)
	if err != nil {
		return l.errorf(err.Error())
	}

	return getNextPathState(l, state)
}

func parseSeparators(l *Lexer, tok string) (XItemType, error) {
	l.skipWS(false)
	state := XItemType(XItemQName)
	r := l.peek()

	if string(r) == ":" && string(l.peekAt(2)) == ":" {
		var err error
		if state, err = getAxis(l, tok); err != nil {
			return state, fmt.Errorf(err.Error())
		}
	} else if string(r) == ":" {
		state = XItemNCName
		l.emitVal(state, tok)
		l.skip(1)
		l.skipWS(true)
	} else if string(r) == "@" {
		state = XItemAbbrAxis
		l.emitVal(state, tok)
		l.skip(1)
		l.skipWS(true)
	} else if string(r) == "(" {
		var err error
		if state, err = getNT(l, tok); err != nil {
			return state, fmt.Errorf(err.Error())
		}
	} else if len(tok) > 0 {
		l.emitVal(state, tok)
	}

	return state, nil
}

func getAxis(l *Lexer, tok string) (XItemType, error) {
	var state XItemType
	for i := range xconst.AxisNames {
		if tok == xconst.AxisNames[i] {
			state = XItemAxis
		}
	}
	if state != XItemAxis {
		return state, fmt.Errorf("Invalid Axis specifier, %s", tok)
	}
	l.emitVal(state, tok)
	l.skip(2)
	l.skipWS(true)
	return state, nil
}

func getNT(l *Lexer, tok string) (XItemType, error) {
	isNT := false
	for _, i := range xconst.NodeTypes {
		if tok == i {
			isNT = true
			break
		}
	}

	if isNT {
		return procNT(l, tok)
	}

	return XItemError, fmt.Errorf("Invalid node-type " + tok)
}

func procNT(l *Lexer, tok string) (XItemType, error) {
	state := XItemType(XItemNodeType)
	l.emitVal(state, tok)
	l.skip(1)
	l.skipWS(true)
	n := l.peek()
	if tok == xconst.NodeTypeProcInst && (string(n) == `"` || string(n) == `'`) {
		if err := getStrLit(l, XItemProcLit); err != nil {
			return state, fmt.Errorf(err.Error())
		}
		l.skipWS(true)
		n = l.next()
	}

	if string(n) != ")" {
		return state, fmt.Errorf("Missing ) at end of NodeType declaration.")
	}

	l.skip(1)
	return state, nil
}

func procFunc(l *Lexer, tok string) error {
	state := XItemType(XItemFunction)
	l.emitVal(state, tok)
	l.skip(1)
	l.skipWS(true)
	if string(l.peek()) != ")" {
		l.emit(XItemArgument)
		for {
			for state := startState; state != nil; {
				state = state(l)
			}
			l.skipWS(true)

			if string(l.peek()) == "," {
				l.emit(XItemArgument)
				l.skip(1)
			} else if string(l.peek()) == ")" {
				l.emit(XItemEndFunction)
				l.skip(1)
				break
			} else if l.peek() == eof {
				return fmt.Errorf("Missing ) at end of function declaration.")
			}
		}
	} else {
		l.emit(XItemEndFunction)
		l.skip(1)
	}

	return nil
}

func getNextPathState(l *Lexer, state XItemType) stateFn {
	isMultiPart := state == XItemAxis || state == XItemAbbrAxis || state == XItemNCName

	l.skipWS(true)

	for string(l.peek()) == "[" {
		if err := getPred(l); err != nil {
			return l.errorf(err.Error())
		}
	}

	if string(l.peek()) == "/" && !isMultiPart {
		l.skip(1)
		if string(l.peek()) == "/" {
			l.skip(1)
			return abbrRelLocPathState
		}
		l.skipWS(true)
		return relLocPathState
	} else if isMultiPart && isElemChar(l.peek()) {
		return stepState
	}

	if isMultiPart {
		return l.errorf("Step is not complete")
	}

	l.emit(XItemEndPath)
	return findOperatorState
}

func getPred(l *Lexer) error {
	l.emit(XItemPredicate)
	l.skip(1)
	l.skipWS(true)

	if string(l.peek()) == "]" {
		return fmt.Errorf("Missing content in predicate.")
	}

	for state := startState; state != nil; {
		state = state(l)
	}

	l.skipWS(true)
	if string(l.peek()) != "]" {
		return fmt.Errorf("Missing ] at end of predicate.")
	}
	l.skip(1)
	l.emit(XItemEndPredicate)
	l.skipWS(true)

	return nil
}
