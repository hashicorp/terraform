package parser

import (
	"fmt"

	"github.com/ChrisTrenkamp/goxpath/internal/lexer"
)

type stateType int

const (
	defState stateType = iota
	xpathState
	funcState
	paramState
	predState
	parenState
)

type parseStack struct {
	stack      []*Node
	stateTypes []stateType
	cur        *Node
}

func (p *parseStack) push(t stateType) {
	p.stack = append(p.stack, p.cur)
	p.stateTypes = append(p.stateTypes, t)
}

func (p *parseStack) pop() {
	stackPos := len(p.stack) - 1

	p.cur = p.stack[stackPos]
	p.stack = p.stack[:stackPos]
	p.stateTypes = p.stateTypes[:stackPos]
}

func (p *parseStack) curState() stateType {
	if len(p.stateTypes) == 0 {
		return defState
	}
	return p.stateTypes[len(p.stateTypes)-1]
}

type lexFn func(*parseStack, lexer.XItem)

var parseMap = map[lexer.XItemType]lexFn{
	lexer.XItemAbsLocPath:     xiXPath,
	lexer.XItemAbbrAbsLocPath: xiXPath,
	lexer.XItemAbbrRelLocPath: xiXPath,
	lexer.XItemRelLocPath:     xiXPath,
	lexer.XItemEndPath:        xiEndPath,
	lexer.XItemAxis:           xiXPath,
	lexer.XItemAbbrAxis:       xiXPath,
	lexer.XItemNCName:         xiXPath,
	lexer.XItemQName:          xiXPath,
	lexer.XItemNodeType:       xiXPath,
	lexer.XItemProcLit:        xiXPath,
	lexer.XItemFunction:       xiFunc,
	lexer.XItemArgument:       xiFuncArg,
	lexer.XItemEndFunction:    xiEndFunc,
	lexer.XItemPredicate:      xiPred,
	lexer.XItemEndPredicate:   xiEndPred,
	lexer.XItemStrLit:         xiValue,
	lexer.XItemNumLit:         xiValue,
	lexer.XItemOperator:       xiOp,
	lexer.XItemVariable:       xiValue,
}

var opPrecedence = map[string]int{
	"|":   1,
	"*":   2,
	"div": 2,
	"mod": 2,
	"+":   3,
	"-":   3,
	"=":   4,
	"!=":  4,
	"<":   4,
	"<=":  4,
	">":   4,
	">=":  4,
	"and": 5,
	"or":  6,
}

//Parse creates an AST tree for XPath expressions.
func Parse(xp string) (*Node, error) {
	var err error
	c := lexer.Lex(xp)
	n := &Node{}
	p := &parseStack{cur: n}

	for next := range c {
		if next.Typ != lexer.XItemError {
			parseMap[next.Typ](p, next)
		} else if err == nil {
			err = fmt.Errorf(next.Val)
		}
	}

	return n, err
}

func xiXPath(p *parseStack, i lexer.XItem) {
	if p.curState() == xpathState {
		p.cur.push(i)
		p.cur = p.cur.next
		return
	}

	if p.cur.Val.Typ == lexer.XItemFunction {
		p.cur.Right = &Node{Val: i, Parent: p.cur}
		p.cur.next = p.cur.Right
		p.push(xpathState)
		p.cur = p.cur.next
		return
	}

	p.cur.pushNotEmpty(i)
	p.push(xpathState)
	p.cur = p.cur.next
}

func xiEndPath(p *parseStack, i lexer.XItem) {
	p.pop()
}

func xiFunc(p *parseStack, i lexer.XItem) {
	p.cur.push(i)
	p.cur = p.cur.next
	p.push(funcState)
}

func xiFuncArg(p *parseStack, i lexer.XItem) {
	if p.curState() != funcState {
		p.pop()
	}

	p.cur.push(i)
	p.cur = p.cur.next
	p.push(paramState)
	p.cur.push(lexer.XItem{Typ: Empty, Val: ""})
	p.cur = p.cur.next
}

func xiEndFunc(p *parseStack, i lexer.XItem) {
	if p.curState() == paramState {
		p.pop()
	}
	p.pop()
}

func xiPred(p *parseStack, i lexer.XItem) {
	p.cur.push(i)
	p.cur = p.cur.next
	p.push(predState)
	p.cur.push(lexer.XItem{Typ: Empty, Val: ""})
	p.cur = p.cur.next
}

func xiEndPred(p *parseStack, i lexer.XItem) {
	p.pop()
}

func xiValue(p *parseStack, i lexer.XItem) {
	p.cur.add(i)
}

func xiOp(p *parseStack, i lexer.XItem) {
	if i.Val == "(" {
		p.cur.push(lexer.XItem{Typ: Empty, Val: ""})
		p.push(parenState)
		p.cur = p.cur.next
		return
	}

	if i.Val == ")" {
		p.pop()
		return
	}

	if p.cur.Val.Typ == lexer.XItemOperator {
		if opPrecedence[p.cur.Val.Val] <= opPrecedence[i.Val] {
			p.cur.add(i)
		} else {
			p.cur.push(i)
		}
	} else {
		p.cur.add(i)
	}
	p.cur = p.cur.next
}
