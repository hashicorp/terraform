package xpath

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

// A XPath expression token type.
type itemType int

const (
	itemComma      itemType = iota // ','
	itemSlash                      // '/'
	itemAt                         // '@'
	itemDot                        // '.'
	itemLParens                    // '('
	itemRParens                    // ')'
	itemLBracket                   // '['
	itemRBracket                   // ']'
	itemStar                       // '*'
	itemPlus                       // '+'
	itemMinus                      // '-'
	itemEq                         // '='
	itemLt                         // '<'
	itemGt                         // '>'
	itemBang                       // '!'
	itemDollar                     // '$'
	itemApos                       // '\''
	itemQuote                      // '"'
	itemUnion                      // '|'
	itemNe                         // '!='
	itemLe                         // '<='
	itemGe                         // '>='
	itemAnd                        // '&&'
	itemOr                         // '||'
	itemDotDot                     // '..'
	itemSlashSlash                 // '//'
	itemName                       // XML Name
	itemString                     // Quoted string constant
	itemNumber                     // Number constant
	itemAxe                        // Axe (like child::)
	itemEOF                        // END
)

// A node is an XPath node in the parse tree.
type node interface {
	Type() nodeType
}

// nodeType identifies the type of a parse tree node.
type nodeType int

func (t nodeType) Type() nodeType {
	return t
}

const (
	nodeRoot nodeType = iota
	nodeAxis
	nodeFilter
	nodeFunction
	nodeOperator
	nodeVariable
	nodeConstantOperand
)

type parser struct {
	r *scanner
	d int
}

// newOperatorNode returns new operator node OperatorNode.
func newOperatorNode(op string, left, right node) node {
	return &operatorNode{nodeType: nodeOperator, Op: op, Left: left, Right: right}
}

// newOperand returns new constant operand node OperandNode.
func newOperandNode(v interface{}) node {
	return &operandNode{nodeType: nodeConstantOperand, Val: v}
}

// newAxisNode returns new axis node AxisNode.
func newAxisNode(axeTyp, localName, prefix, prop string, n node) node {
	return &axisNode{
		nodeType:  nodeAxis,
		LocalName: localName,
		Prefix:    prefix,
		AxeType:   axeTyp,
		Prop:      prop,
		Input:     n,
	}
}

// newVariableNode returns new variable node VariableNode.
func newVariableNode(prefix, name string) node {
	return &variableNode{nodeType: nodeVariable, Name: name, Prefix: prefix}
}

// newFilterNode returns a new filter node FilterNode.
func newFilterNode(n, m node) node {
	return &filterNode{nodeType: nodeFilter, Input: n, Condition: m}
}

// newRootNode returns a root node.
func newRootNode(s string) node {
	return &rootNode{nodeType: nodeRoot, slash: s}
}

// newFunctionNode returns function call node.
func newFunctionNode(name, prefix string, args []node) node {
	return &functionNode{nodeType: nodeFunction, Prefix: prefix, FuncName: name, Args: args}
}

// testOp reports whether current item name is an operand op.
func testOp(r *scanner, op string) bool {
	return r.typ == itemName && r.prefix == "" && r.name == op
}

func isPrimaryExpr(r *scanner) bool {
	switch r.typ {
	case itemString, itemNumber, itemDollar, itemLParens:
		return true
	case itemName:
		return r.canBeFunc && !isNodeType(r)
	}
	return false
}

func isNodeType(r *scanner) bool {
	switch r.name {
	case "node", "text", "processing-instruction", "comment":
		return r.prefix == ""
	}
	return false
}

func isStep(item itemType) bool {
	switch item {
	case itemDot, itemDotDot, itemAt, itemAxe, itemStar, itemName:
		return true
	}
	return false
}

func checkItem(r *scanner, typ itemType) {
	if r.typ != typ {
		panic(fmt.Sprintf("%s has an invalid token", r.text))
	}
}

// parseExpression parsing the expression with input node n.
func (p *parser) parseExpression(n node) node {
	if p.d = p.d + 1; p.d > 200 {
		panic("the xpath query is too complex(depth > 200)")
	}
	n = p.parseOrExpr(n)
	p.d--
	return n
}

// next scanning next item on forward.
func (p *parser) next() bool {
	return p.r.nextItem()
}

func (p *parser) skipItem(typ itemType) {
	checkItem(p.r, typ)
	p.next()
}

// OrExpr ::= AndExpr | OrExpr 'or' AndExpr
func (p *parser) parseOrExpr(n node) node {
	opnd := p.parseAndExpr(n)
	for {
		if !testOp(p.r, "or") {
			break
		}
		p.next()
		opnd = newOperatorNode("or", opnd, p.parseAndExpr(n))
	}
	return opnd
}

// AndExpr ::= EqualityExpr	| AndExpr 'and' EqualityExpr
func (p *parser) parseAndExpr(n node) node {
	opnd := p.parseEqualityExpr(n)
	for {
		if !testOp(p.r, "and") {
			break
		}
		p.next()
		opnd = newOperatorNode("and", opnd, p.parseEqualityExpr(n))
	}
	return opnd
}

// EqualityExpr ::= RelationalExpr | EqualityExpr '=' RelationalExpr | EqualityExpr '!=' RelationalExpr
func (p *parser) parseEqualityExpr(n node) node {
	opnd := p.parseRelationalExpr(n)
Loop:
	for {
		var op string
		switch p.r.typ {
		case itemEq:
			op = "="
		case itemNe:
			op = "!="
		default:
			break Loop
		}
		p.next()
		opnd = newOperatorNode(op, opnd, p.parseRelationalExpr(n))
	}
	return opnd
}

// RelationalExpr ::= AdditiveExpr	| RelationalExpr '<' AdditiveExpr | RelationalExpr '>' AdditiveExpr
//					| RelationalExpr '<=' AdditiveExpr
//					| RelationalExpr '>=' AdditiveExpr
func (p *parser) parseRelationalExpr(n node) node {
	opnd := p.parseAdditiveExpr(n)
Loop:
	for {
		var op string
		switch p.r.typ {
		case itemLt:
			op = "<"
		case itemGt:
			op = ">"
		case itemLe:
			op = "<="
		case itemGe:
			op = ">="
		default:
			break Loop
		}
		p.next()
		opnd = newOperatorNode(op, opnd, p.parseAdditiveExpr(n))
	}
	return opnd
}

// AdditiveExpr	::= MultiplicativeExpr	| AdditiveExpr '+' MultiplicativeExpr | AdditiveExpr '-' MultiplicativeExpr
func (p *parser) parseAdditiveExpr(n node) node {
	opnd := p.parseMultiplicativeExpr(n)
Loop:
	for {
		var op string
		switch p.r.typ {
		case itemPlus:
			op = "+"
		case itemMinus:
			op = "-"
		default:
			break Loop
		}
		p.next()
		opnd = newOperatorNode(op, opnd, p.parseMultiplicativeExpr(n))
	}
	return opnd
}

// MultiplicativeExpr ::= UnaryExpr	| MultiplicativeExpr MultiplyOperator(*) UnaryExpr
//						| MultiplicativeExpr 'div' UnaryExpr | MultiplicativeExpr 'mod' UnaryExpr
func (p *parser) parseMultiplicativeExpr(n node) node {
	opnd := p.parseUnaryExpr(n)
Loop:
	for {
		var op string
		if p.r.typ == itemStar {
			op = "*"
		} else if testOp(p.r, "div") || testOp(p.r, "mod") {
			op = p.r.name
		} else {
			break Loop
		}
		p.next()
		opnd = newOperatorNode(op, opnd, p.parseUnaryExpr(n))
	}
	return opnd
}

// UnaryExpr ::= UnionExpr | '-' UnaryExpr
func (p *parser) parseUnaryExpr(n node) node {
	minus := false
	// ignore '-' sequence
	for p.r.typ == itemMinus {
		p.next()
		minus = !minus
	}
	opnd := p.parseUnionExpr(n)
	if minus {
		opnd = newOperatorNode("*", opnd, newOperandNode(float64(-1)))
	}
	return opnd
}

// 	UnionExpr ::= PathExpr | UnionExpr '|' PathExpr
func (p *parser) parseUnionExpr(n node) node {
	opnd := p.parsePathExpr(n)
Loop:
	for {
		if p.r.typ != itemUnion {
			break Loop
		}
		p.next()
		opnd2 := p.parsePathExpr(n)
		// Checking the node type that must be is node set type?
		opnd = newOperatorNode("|", opnd, opnd2)
	}
	return opnd
}

// PathExpr ::= LocationPath | FilterExpr | FilterExpr '/' RelativeLocationPath	| FilterExpr '//' RelativeLocationPath
func (p *parser) parsePathExpr(n node) node {
	var opnd node
	if isPrimaryExpr(p.r) {
		opnd = p.parseFilterExpr(n)
		switch p.r.typ {
		case itemSlash:
			p.next()
			opnd = p.parseRelativeLocationPath(opnd)
		case itemSlashSlash:
			p.next()
			opnd = p.parseRelativeLocationPath(newAxisNode("descendant-or-self", "", "", "", opnd))
		}
	} else {
		opnd = p.parseLocationPath(nil)
	}
	return opnd
}

// FilterExpr ::= PrimaryExpr | FilterExpr Predicate
func (p *parser) parseFilterExpr(n node) node {
	opnd := p.parsePrimaryExpr(n)
	if p.r.typ == itemLBracket {
		opnd = newFilterNode(opnd, p.parsePredicate(opnd))
	}
	return opnd
}

// 	Predicate ::=  '[' PredicateExpr ']'
func (p *parser) parsePredicate(n node) node {
	p.skipItem(itemLBracket)
	opnd := p.parseExpression(n)
	p.skipItem(itemRBracket)
	return opnd
}

// LocationPath ::= RelativeLocationPath | AbsoluteLocationPath
func (p *parser) parseLocationPath(n node) (opnd node) {
	switch p.r.typ {
	case itemSlash:
		p.next()
		opnd = newRootNode("/")
		if isStep(p.r.typ) {
			opnd = p.parseRelativeLocationPath(opnd) // ?? child:: or self ??
		}
	case itemSlashSlash:
		p.next()
		opnd = newRootNode("//")
		opnd = p.parseRelativeLocationPath(newAxisNode("descendant-or-self", "", "", "", opnd))
	default:
		opnd = p.parseRelativeLocationPath(n)
	}
	return opnd
}

// RelativeLocationPath	 ::= Step | RelativeLocationPath '/' Step | AbbreviatedRelativeLocationPath
func (p *parser) parseRelativeLocationPath(n node) node {
	opnd := n
Loop:
	for {
		opnd = p.parseStep(opnd)
		switch p.r.typ {
		case itemSlashSlash:
			p.next()
			opnd = newAxisNode("descendant-or-self", "", "", "", opnd)
		case itemSlash:
			p.next()
		default:
			break Loop
		}
	}
	return opnd
}

// Step	::= AxisSpecifier NodeTest Predicate* | AbbreviatedStep
func (p *parser) parseStep(n node) (opnd node) {
	axeTyp := "child" // default axes value.
	if p.r.typ == itemDot || p.r.typ == itemDotDot {
		if p.r.typ == itemDot {
			axeTyp = "self"
		} else {
			axeTyp = "parent"
		}
		p.next()
		opnd = newAxisNode(axeTyp, "", "", "", n)
		if p.r.typ != itemLBracket {
			return opnd
		}
	} else {
		switch p.r.typ {
		case itemAt:
			p.next()
			axeTyp = "attribute"
		case itemAxe:
			axeTyp = p.r.name
			p.next()
		case itemLParens:
			return p.parseSequence(n)
		}
		opnd = p.parseNodeTest(n, axeTyp)
	}
	for p.r.typ == itemLBracket {
		opnd = newFilterNode(opnd, p.parsePredicate(opnd))
	}
	return opnd
}

// Expr ::= '(' Step ("," Step)* ')'
func (p *parser) parseSequence(n node) (opnd node) {
	p.skipItem(itemLParens)
	opnd = p.parseStep(n)
	for {
		if p.r.typ != itemComma {
			break
		}
		p.next()
		opnd2 := p.parseStep(n)
		opnd = newOperatorNode("|", opnd, opnd2)
	}
	p.skipItem(itemRParens)
	return opnd
}

// 	NodeTest ::= NameTest | nodeType '(' ')' | 'processing-instruction' '(' Literal ')'
func (p *parser) parseNodeTest(n node, axeTyp string) (opnd node) {
	switch p.r.typ {
	case itemName:
		if p.r.canBeFunc && isNodeType(p.r) {
			var prop string
			switch p.r.name {
			case "comment", "text", "processing-instruction", "node":
				prop = p.r.name
			}
			var name string
			p.next()
			p.skipItem(itemLParens)
			if prop == "processing-instruction" && p.r.typ != itemRParens {
				checkItem(p.r, itemString)
				name = p.r.strval
				p.next()
			}
			p.skipItem(itemRParens)
			opnd = newAxisNode(axeTyp, name, "", prop, n)
		} else {
			prefix := p.r.prefix
			name := p.r.name
			p.next()
			if p.r.name == "*" {
				name = ""
			}
			opnd = newAxisNode(axeTyp, name, prefix, "", n)
		}
	case itemStar:
		opnd = newAxisNode(axeTyp, "", "", "", n)
		p.next()
	default:
		panic("expression must evaluate to a node-set")
	}
	return opnd
}

// PrimaryExpr ::= VariableReference | '(' Expr ')'	| Literal | Number | FunctionCall
func (p *parser) parsePrimaryExpr(n node) (opnd node) {
	switch p.r.typ {
	case itemString:
		opnd = newOperandNode(p.r.strval)
		p.next()
	case itemNumber:
		opnd = newOperandNode(p.r.numval)
		p.next()
	case itemDollar:
		p.next()
		checkItem(p.r, itemName)
		opnd = newVariableNode(p.r.prefix, p.r.name)
		p.next()
	case itemLParens:
		p.next()
		opnd = p.parseExpression(n)
		p.skipItem(itemRParens)
	case itemName:
		if p.r.canBeFunc && !isNodeType(p.r) {
			opnd = p.parseMethod(nil)
		}
	}
	return opnd
}

// FunctionCall	 ::=  FunctionName '(' ( Argument ( ',' Argument )* )? ')'
func (p *parser) parseMethod(n node) node {
	var args []node
	name := p.r.name
	prefix := p.r.prefix

	p.skipItem(itemName)
	p.skipItem(itemLParens)
	if p.r.typ != itemRParens {
		for {
			args = append(args, p.parseExpression(n))
			if p.r.typ == itemRParens {
				break
			}
			p.skipItem(itemComma)
		}
	}
	p.skipItem(itemRParens)
	return newFunctionNode(name, prefix, args)
}

// Parse parsing the XPath express string expr and returns a tree node.
func parse(expr string) node {
	r := &scanner{text: expr}
	r.nextChar()
	r.nextItem()
	p := &parser{r: r}
	return p.parseExpression(nil)
}

// rootNode holds a top-level node of tree.
type rootNode struct {
	nodeType
	slash string
}

func (r *rootNode) String() string {
	return r.slash
}

// operatorNode holds two Nodes operator.
type operatorNode struct {
	nodeType
	Op          string
	Left, Right node
}

func (o *operatorNode) String() string {
	return fmt.Sprintf("%v%s%v", o.Left, o.Op, o.Right)
}

// axisNode holds a location step.
type axisNode struct {
	nodeType
	Input     node
	Prop      string // node-test name.[comment|text|processing-instruction|node]
	AxeType   string // name of the axes.[attribute|ancestor|child|....]
	LocalName string // local part name of node.
	Prefix    string // prefix name of node.
}

func (a *axisNode) String() string {
	var b bytes.Buffer
	if a.AxeType != "" {
		b.Write([]byte(a.AxeType + "::"))
	}
	if a.Prefix != "" {
		b.Write([]byte(a.Prefix + ":"))
	}
	b.Write([]byte(a.LocalName))
	if a.Prop != "" {
		b.Write([]byte("/" + a.Prop + "()"))
	}
	return b.String()
}

// operandNode holds a constant operand.
type operandNode struct {
	nodeType
	Val interface{}
}

func (o *operandNode) String() string {
	return fmt.Sprintf("%v", o.Val)
}

// filterNode holds a condition filter.
type filterNode struct {
	nodeType
	Input, Condition node
}

func (f *filterNode) String() string {
	return fmt.Sprintf("%s[%s]", f.Input, f.Condition)
}

// variableNode holds a variable.
type variableNode struct {
	nodeType
	Name, Prefix string
}

func (v *variableNode) String() string {
	if v.Prefix == "" {
		return v.Name
	}
	return fmt.Sprintf("%s:%s", v.Prefix, v.Name)
}

// functionNode holds a function call.
type functionNode struct {
	nodeType
	Args     []node
	Prefix   string
	FuncName string // function name
}

func (f *functionNode) String() string {
	var b bytes.Buffer
	// fun(arg1, ..., argn)
	b.Write([]byte(f.FuncName))
	b.Write([]byte("("))
	for i, arg := range f.Args {
		if i > 0 {
			b.Write([]byte(","))
		}
		b.Write([]byte(fmt.Sprintf("%s", arg)))
	}
	b.Write([]byte(")"))
	return b.String()
}

type scanner struct {
	text, name, prefix string

	pos       int
	curr      rune
	typ       itemType
	strval    string  // text value at current pos
	numval    float64 // number value at current pos
	canBeFunc bool
}

func (s *scanner) nextChar() bool {
	if s.pos >= len(s.text) {
		s.curr = rune(0)
		return false
	}
	s.curr = rune(s.text[s.pos])
	s.pos++
	return true
}

func (s *scanner) nextItem() bool {
	s.skipSpace()
	switch s.curr {
	case 0:
		s.typ = itemEOF
		return false
	case ',', '@', '(', ')', '|', '*', '[', ']', '+', '-', '=', '#', '$':
		s.typ = asItemType(s.curr)
		s.nextChar()
	case '<':
		s.typ = itemLt
		s.nextChar()
		if s.curr == '=' {
			s.typ = itemLe
			s.nextChar()
		}
	case '>':
		s.typ = itemGt
		s.nextChar()
		if s.curr == '=' {
			s.typ = itemGe
			s.nextChar()
		}
	case '!':
		s.typ = itemBang
		s.nextChar()
		if s.curr == '=' {
			s.typ = itemNe
			s.nextChar()
		}
	case '.':
		s.typ = itemDot
		s.nextChar()
		if s.curr == '.' {
			s.typ = itemDotDot
			s.nextChar()
		} else if isDigit(s.curr) {
			s.typ = itemNumber
			s.numval = s.scanFraction()
		}
	case '/':
		s.typ = itemSlash
		s.nextChar()
		if s.curr == '/' {
			s.typ = itemSlashSlash
			s.nextChar()
		}
	case '"', '\'':
		s.typ = itemString
		s.strval = s.scanString()
	default:
		if isDigit(s.curr) {
			s.typ = itemNumber
			s.numval = s.scanNumber()
		} else if isName(s.curr) {
			s.typ = itemName
			s.name = s.scanName()
			s.prefix = ""
			// "foo:bar" is one itemem not three because it doesn't allow spaces in between
			// We should distinct it from "foo::" and need process "foo ::" as well
			if s.curr == ':' {
				s.nextChar()
				// can be "foo:bar" or "foo::"
				if s.curr == ':' {
					// "foo::"
					s.nextChar()
					s.typ = itemAxe
				} else { // "foo:*", "foo:bar" or "foo: "
					s.prefix = s.name
					if s.curr == '*' {
						s.nextChar()
						s.name = "*"
					} else if isName(s.curr) {
						s.name = s.scanName()
					} else {
						panic(fmt.Sprintf("%s has an invalid qualified name.", s.text))
					}
				}
			} else {
				s.skipSpace()
				if s.curr == ':' {
					s.nextChar()
					// it can be "foo ::" or just "foo :"
					if s.curr == ':' {
						s.nextChar()
						s.typ = itemAxe
					} else {
						panic(fmt.Sprintf("%s has an invalid qualified name.", s.text))
					}
				}
			}
			s.skipSpace()
			s.canBeFunc = s.curr == '('
		} else {
			panic(fmt.Sprintf("%s has an invalid token.", s.text))
		}
	}
	return true
}

func (s *scanner) skipSpace() {
Loop:
	for {
		if !unicode.IsSpace(s.curr) || !s.nextChar() {
			break Loop
		}
	}
}

func (s *scanner) scanFraction() float64 {
	var (
		i = s.pos - 2
		c = 1 // '.'
	)
	for isDigit(s.curr) {
		s.nextChar()
		c++
	}
	v, err := strconv.ParseFloat(s.text[i:i+c], 64)
	if err != nil {
		panic(fmt.Errorf("xpath: scanFraction parse float got error: %v", err))
	}
	return v
}

func (s *scanner) scanNumber() float64 {
	var (
		c int
		i = s.pos - 1
	)
	for isDigit(s.curr) {
		s.nextChar()
		c++
	}
	if s.curr == '.' {
		s.nextChar()
		c++
		for isDigit(s.curr) {
			s.nextChar()
			c++
		}
	}
	v, err := strconv.ParseFloat(s.text[i:i+c], 64)
	if err != nil {
		panic(fmt.Errorf("xpath: scanNumber parse float got error: %v", err))
	}
	return v
}

func (s *scanner) scanString() string {
	var (
		c   = 0
		end = s.curr
	)
	s.nextChar()
	i := s.pos - 1
	for s.curr != end {
		if !s.nextChar() {
			panic(errors.New("xpath: scanString got unclosed string"))
		}
		c++
	}
	s.nextChar()
	return s.text[i : i+c]
}

func (s *scanner) scanName() string {
	var (
		c int
		i = s.pos - 1
	)
	for isName(s.curr) {
		c++
		if !s.nextChar() {
			break
		}
	}
	return s.text[i : i+c]
}

func isName(r rune) bool {
	return string(r) != ":" && string(r) != "/" &&
		(unicode.Is(first, r) || unicode.Is(second, r) || string(r) == "*")
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

func asItemType(r rune) itemType {
	switch r {
	case ',':
		return itemComma
	case '@':
		return itemAt
	case '(':
		return itemLParens
	case ')':
		return itemRParens
	case '|':
		return itemUnion
	case '*':
		return itemStar
	case '[':
		return itemLBracket
	case ']':
		return itemRBracket
	case '+':
		return itemPlus
	case '-':
		return itemMinus
	case '=':
		return itemEq
	case '$':
		return itemDollar
	}
	panic(fmt.Errorf("unknown item: %v", r))
}

var first = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x003A, 0x003A, 1},
		{0x0041, 0x005A, 1},
		{0x005F, 0x005F, 1},
		{0x0061, 0x007A, 1},
		{0x00C0, 0x00D6, 1},
		{0x00D8, 0x00F6, 1},
		{0x00F8, 0x00FF, 1},
		{0x0100, 0x0131, 1},
		{0x0134, 0x013E, 1},
		{0x0141, 0x0148, 1},
		{0x014A, 0x017E, 1},
		{0x0180, 0x01C3, 1},
		{0x01CD, 0x01F0, 1},
		{0x01F4, 0x01F5, 1},
		{0x01FA, 0x0217, 1},
		{0x0250, 0x02A8, 1},
		{0x02BB, 0x02C1, 1},
		{0x0386, 0x0386, 1},
		{0x0388, 0x038A, 1},
		{0x038C, 0x038C, 1},
		{0x038E, 0x03A1, 1},
		{0x03A3, 0x03CE, 1},
		{0x03D0, 0x03D6, 1},
		{0x03DA, 0x03E0, 2},
		{0x03E2, 0x03F3, 1},
		{0x0401, 0x040C, 1},
		{0x040E, 0x044F, 1},
		{0x0451, 0x045C, 1},
		{0x045E, 0x0481, 1},
		{0x0490, 0x04C4, 1},
		{0x04C7, 0x04C8, 1},
		{0x04CB, 0x04CC, 1},
		{0x04D0, 0x04EB, 1},
		{0x04EE, 0x04F5, 1},
		{0x04F8, 0x04F9, 1},
		{0x0531, 0x0556, 1},
		{0x0559, 0x0559, 1},
		{0x0561, 0x0586, 1},
		{0x05D0, 0x05EA, 1},
		{0x05F0, 0x05F2, 1},
		{0x0621, 0x063A, 1},
		{0x0641, 0x064A, 1},
		{0x0671, 0x06B7, 1},
		{0x06BA, 0x06BE, 1},
		{0x06C0, 0x06CE, 1},
		{0x06D0, 0x06D3, 1},
		{0x06D5, 0x06D5, 1},
		{0x06E5, 0x06E6, 1},
		{0x0905, 0x0939, 1},
		{0x093D, 0x093D, 1},
		{0x0958, 0x0961, 1},
		{0x0985, 0x098C, 1},
		{0x098F, 0x0990, 1},
		{0x0993, 0x09A8, 1},
		{0x09AA, 0x09B0, 1},
		{0x09B2, 0x09B2, 1},
		{0x09B6, 0x09B9, 1},
		{0x09DC, 0x09DD, 1},
		{0x09DF, 0x09E1, 1},
		{0x09F0, 0x09F1, 1},
		{0x0A05, 0x0A0A, 1},
		{0x0A0F, 0x0A10, 1},
		{0x0A13, 0x0A28, 1},
		{0x0A2A, 0x0A30, 1},
		{0x0A32, 0x0A33, 1},
		{0x0A35, 0x0A36, 1},
		{0x0A38, 0x0A39, 1},
		{0x0A59, 0x0A5C, 1},
		{0x0A5E, 0x0A5E, 1},
		{0x0A72, 0x0A74, 1},
		{0x0A85, 0x0A8B, 1},
		{0x0A8D, 0x0A8D, 1},
		{0x0A8F, 0x0A91, 1},
		{0x0A93, 0x0AA8, 1},
		{0x0AAA, 0x0AB0, 1},
		{0x0AB2, 0x0AB3, 1},
		{0x0AB5, 0x0AB9, 1},
		{0x0ABD, 0x0AE0, 0x23},
		{0x0B05, 0x0B0C, 1},
		{0x0B0F, 0x0B10, 1},
		{0x0B13, 0x0B28, 1},
		{0x0B2A, 0x0B30, 1},
		{0x0B32, 0x0B33, 1},
		{0x0B36, 0x0B39, 1},
		{0x0B3D, 0x0B3D, 1},
		{0x0B5C, 0x0B5D, 1},
		{0x0B5F, 0x0B61, 1},
		{0x0B85, 0x0B8A, 1},
		{0x0B8E, 0x0B90, 1},
		{0x0B92, 0x0B95, 1},
		{0x0B99, 0x0B9A, 1},
		{0x0B9C, 0x0B9C, 1},
		{0x0B9E, 0x0B9F, 1},
		{0x0BA3, 0x0BA4, 1},
		{0x0BA8, 0x0BAA, 1},
		{0x0BAE, 0x0BB5, 1},
		{0x0BB7, 0x0BB9, 1},
		{0x0C05, 0x0C0C, 1},
		{0x0C0E, 0x0C10, 1},
		{0x0C12, 0x0C28, 1},
		{0x0C2A, 0x0C33, 1},
		{0x0C35, 0x0C39, 1},
		{0x0C60, 0x0C61, 1},
		{0x0C85, 0x0C8C, 1},
		{0x0C8E, 0x0C90, 1},
		{0x0C92, 0x0CA8, 1},
		{0x0CAA, 0x0CB3, 1},
		{0x0CB5, 0x0CB9, 1},
		{0x0CDE, 0x0CDE, 1},
		{0x0CE0, 0x0CE1, 1},
		{0x0D05, 0x0D0C, 1},
		{0x0D0E, 0x0D10, 1},
		{0x0D12, 0x0D28, 1},
		{0x0D2A, 0x0D39, 1},
		{0x0D60, 0x0D61, 1},
		{0x0E01, 0x0E2E, 1},
		{0x0E30, 0x0E30, 1},
		{0x0E32, 0x0E33, 1},
		{0x0E40, 0x0E45, 1},
		{0x0E81, 0x0E82, 1},
		{0x0E84, 0x0E84, 1},
		{0x0E87, 0x0E88, 1},
		{0x0E8A, 0x0E8D, 3},
		{0x0E94, 0x0E97, 1},
		{0x0E99, 0x0E9F, 1},
		{0x0EA1, 0x0EA3, 1},
		{0x0EA5, 0x0EA7, 2},
		{0x0EAA, 0x0EAB, 1},
		{0x0EAD, 0x0EAE, 1},
		{0x0EB0, 0x0EB0, 1},
		{0x0EB2, 0x0EB3, 1},
		{0x0EBD, 0x0EBD, 1},
		{0x0EC0, 0x0EC4, 1},
		{0x0F40, 0x0F47, 1},
		{0x0F49, 0x0F69, 1},
		{0x10A0, 0x10C5, 1},
		{0x10D0, 0x10F6, 1},
		{0x1100, 0x1100, 1},
		{0x1102, 0x1103, 1},
		{0x1105, 0x1107, 1},
		{0x1109, 0x1109, 1},
		{0x110B, 0x110C, 1},
		{0x110E, 0x1112, 1},
		{0x113C, 0x1140, 2},
		{0x114C, 0x1150, 2},
		{0x1154, 0x1155, 1},
		{0x1159, 0x1159, 1},
		{0x115F, 0x1161, 1},
		{0x1163, 0x1169, 2},
		{0x116D, 0x116E, 1},
		{0x1172, 0x1173, 1},
		{0x1175, 0x119E, 0x119E - 0x1175},
		{0x11A8, 0x11AB, 0x11AB - 0x11A8},
		{0x11AE, 0x11AF, 1},
		{0x11B7, 0x11B8, 1},
		{0x11BA, 0x11BA, 1},
		{0x11BC, 0x11C2, 1},
		{0x11EB, 0x11F0, 0x11F0 - 0x11EB},
		{0x11F9, 0x11F9, 1},
		{0x1E00, 0x1E9B, 1},
		{0x1EA0, 0x1EF9, 1},
		{0x1F00, 0x1F15, 1},
		{0x1F18, 0x1F1D, 1},
		{0x1F20, 0x1F45, 1},
		{0x1F48, 0x1F4D, 1},
		{0x1F50, 0x1F57, 1},
		{0x1F59, 0x1F5B, 0x1F5B - 0x1F59},
		{0x1F5D, 0x1F5D, 1},
		{0x1F5F, 0x1F7D, 1},
		{0x1F80, 0x1FB4, 1},
		{0x1FB6, 0x1FBC, 1},
		{0x1FBE, 0x1FBE, 1},
		{0x1FC2, 0x1FC4, 1},
		{0x1FC6, 0x1FCC, 1},
		{0x1FD0, 0x1FD3, 1},
		{0x1FD6, 0x1FDB, 1},
		{0x1FE0, 0x1FEC, 1},
		{0x1FF2, 0x1FF4, 1},
		{0x1FF6, 0x1FFC, 1},
		{0x2126, 0x2126, 1},
		{0x212A, 0x212B, 1},
		{0x212E, 0x212E, 1},
		{0x2180, 0x2182, 1},
		{0x3007, 0x3007, 1},
		{0x3021, 0x3029, 1},
		{0x3041, 0x3094, 1},
		{0x30A1, 0x30FA, 1},
		{0x3105, 0x312C, 1},
		{0x4E00, 0x9FA5, 1},
		{0xAC00, 0xD7A3, 1},
	},
}

var second = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x002D, 0x002E, 1},
		{0x0030, 0x0039, 1},
		{0x00B7, 0x00B7, 1},
		{0x02D0, 0x02D1, 1},
		{0x0300, 0x0345, 1},
		{0x0360, 0x0361, 1},
		{0x0387, 0x0387, 1},
		{0x0483, 0x0486, 1},
		{0x0591, 0x05A1, 1},
		{0x05A3, 0x05B9, 1},
		{0x05BB, 0x05BD, 1},
		{0x05BF, 0x05BF, 1},
		{0x05C1, 0x05C2, 1},
		{0x05C4, 0x0640, 0x0640 - 0x05C4},
		{0x064B, 0x0652, 1},
		{0x0660, 0x0669, 1},
		{0x0670, 0x0670, 1},
		{0x06D6, 0x06DC, 1},
		{0x06DD, 0x06DF, 1},
		{0x06E0, 0x06E4, 1},
		{0x06E7, 0x06E8, 1},
		{0x06EA, 0x06ED, 1},
		{0x06F0, 0x06F9, 1},
		{0x0901, 0x0903, 1},
		{0x093C, 0x093C, 1},
		{0x093E, 0x094C, 1},
		{0x094D, 0x094D, 1},
		{0x0951, 0x0954, 1},
		{0x0962, 0x0963, 1},
		{0x0966, 0x096F, 1},
		{0x0981, 0x0983, 1},
		{0x09BC, 0x09BC, 1},
		{0x09BE, 0x09BF, 1},
		{0x09C0, 0x09C4, 1},
		{0x09C7, 0x09C8, 1},
		{0x09CB, 0x09CD, 1},
		{0x09D7, 0x09D7, 1},
		{0x09E2, 0x09E3, 1},
		{0x09E6, 0x09EF, 1},
		{0x0A02, 0x0A3C, 0x3A},
		{0x0A3E, 0x0A3F, 1},
		{0x0A40, 0x0A42, 1},
		{0x0A47, 0x0A48, 1},
		{0x0A4B, 0x0A4D, 1},
		{0x0A66, 0x0A6F, 1},
		{0x0A70, 0x0A71, 1},
		{0x0A81, 0x0A83, 1},
		{0x0ABC, 0x0ABC, 1},
		{0x0ABE, 0x0AC5, 1},
		{0x0AC7, 0x0AC9, 1},
		{0x0ACB, 0x0ACD, 1},
		{0x0AE6, 0x0AEF, 1},
		{0x0B01, 0x0B03, 1},
		{0x0B3C, 0x0B3C, 1},
		{0x0B3E, 0x0B43, 1},
		{0x0B47, 0x0B48, 1},
		{0x0B4B, 0x0B4D, 1},
		{0x0B56, 0x0B57, 1},
		{0x0B66, 0x0B6F, 1},
		{0x0B82, 0x0B83, 1},
		{0x0BBE, 0x0BC2, 1},
		{0x0BC6, 0x0BC8, 1},
		{0x0BCA, 0x0BCD, 1},
		{0x0BD7, 0x0BD7, 1},
		{0x0BE7, 0x0BEF, 1},
		{0x0C01, 0x0C03, 1},
		{0x0C3E, 0x0C44, 1},
		{0x0C46, 0x0C48, 1},
		{0x0C4A, 0x0C4D, 1},
		{0x0C55, 0x0C56, 1},
		{0x0C66, 0x0C6F, 1},
		{0x0C82, 0x0C83, 1},
		{0x0CBE, 0x0CC4, 1},
		{0x0CC6, 0x0CC8, 1},
		{0x0CCA, 0x0CCD, 1},
		{0x0CD5, 0x0CD6, 1},
		{0x0CE6, 0x0CEF, 1},
		{0x0D02, 0x0D03, 1},
		{0x0D3E, 0x0D43, 1},
		{0x0D46, 0x0D48, 1},
		{0x0D4A, 0x0D4D, 1},
		{0x0D57, 0x0D57, 1},
		{0x0D66, 0x0D6F, 1},
		{0x0E31, 0x0E31, 1},
		{0x0E34, 0x0E3A, 1},
		{0x0E46, 0x0E46, 1},
		{0x0E47, 0x0E4E, 1},
		{0x0E50, 0x0E59, 1},
		{0x0EB1, 0x0EB1, 1},
		{0x0EB4, 0x0EB9, 1},
		{0x0EBB, 0x0EBC, 1},
		{0x0EC6, 0x0EC6, 1},
		{0x0EC8, 0x0ECD, 1},
		{0x0ED0, 0x0ED9, 1},
		{0x0F18, 0x0F19, 1},
		{0x0F20, 0x0F29, 1},
		{0x0F35, 0x0F39, 2},
		{0x0F3E, 0x0F3F, 1},
		{0x0F71, 0x0F84, 1},
		{0x0F86, 0x0F8B, 1},
		{0x0F90, 0x0F95, 1},
		{0x0F97, 0x0F97, 1},
		{0x0F99, 0x0FAD, 1},
		{0x0FB1, 0x0FB7, 1},
		{0x0FB9, 0x0FB9, 1},
		{0x20D0, 0x20DC, 1},
		{0x20E1, 0x3005, 0x3005 - 0x20E1},
		{0x302A, 0x302F, 1},
		{0x3031, 0x3035, 1},
		{0x3099, 0x309A, 1},
		{0x309D, 0x309E, 1},
		{0x30FC, 0x30FE, 1},
	},
}
