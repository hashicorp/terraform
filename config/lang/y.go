//line lang.y:6
package lang

import __yyfmt__ "fmt"

//line lang.y:6
import (
	"github.com/hashicorp/terraform/config/lang/ast"
)

//line lang.y:14
type parserSymType struct {
	yys      int
	node     ast.Node
	nodeList []ast.Node
	str      string
	token    *parserToken
}

const PROGRAM_BRACKET_LEFT = 57346
const PROGRAM_BRACKET_RIGHT = 57347
const PROGRAM_STRING_START = 57348
const PROGRAM_STRING_END = 57349
const PAREN_LEFT = 57350
const PAREN_RIGHT = 57351
const COMMA = 57352
const ARITH_OP = 57353
const IDENTIFIER = 57354
const INTEGER = 57355
const FLOAT = 57356
const STRING = 57357

var parserToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"PROGRAM_BRACKET_LEFT",
	"PROGRAM_BRACKET_RIGHT",
	"PROGRAM_STRING_START",
	"PROGRAM_STRING_END",
	"PAREN_LEFT",
	"PAREN_RIGHT",
	"COMMA",
	"ARITH_OP",
	"IDENTIFIER",
	"INTEGER",
	"FLOAT",
	"STRING",
}
var parserStatenames = [...]string{}

const parserEofCode = 1
const parserErrCode = 2
const parserMaxDepth = 200

//line lang.y:173

//line yacctab:1
var parserExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const parserNprod = 20
const parserPrivate = 57344

var parserTokenNames []string
var parserStates []string

const parserLast = 34

var parserAct = [...]int{

	9, 7, 3, 16, 22, 8, 17, 17, 20, 17,
	1, 18, 6, 23, 8, 19, 25, 26, 21, 11,
	2, 24, 7, 4, 5, 0, 10, 27, 0, 14,
	15, 12, 13, 6,
}
var parserPact = [...]int{

	-3, -1000, -3, -1000, -1000, -1000, -1000, 18, -1000, -2,
	18, -3, -1000, -1000, 18, 0, -1000, 18, -5, -1000,
	18, -1000, -1000, 7, -4, -1000, 18, -4,
}
var parserPgo = [...]int{

	0, 0, 24, 23, 19, 2, 13, 10,
}
var parserR1 = [...]int{

	0, 7, 7, 4, 4, 5, 5, 2, 1, 1,
	1, 1, 1, 1, 1, 1, 6, 6, 6, 3,
}
var parserR2 = [...]int{

	0, 0, 1, 1, 2, 1, 1, 3, 3, 1,
	1, 1, 3, 2, 1, 4, 0, 3, 1, 1,
}
var parserChk = [...]int{

	-1000, -7, -4, -5, -3, -2, 15, 4, -5, -1,
	8, -4, 13, 14, 11, 12, 5, 11, -1, -1,
	8, -1, 9, -6, -1, 9, 10, -1,
}
var parserDef = [...]int{

	1, -2, 2, 3, 5, 6, 19, 0, 4, 0,
	0, 9, 10, 11, 0, 14, 7, 0, 0, 13,
	16, 12, 8, 0, 18, 15, 0, 17,
}
var parserTok1 = [...]int{

	1,
}
var parserTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15,
}
var parserTok3 = [...]int{
	0,
}

var parserErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	parserDebug        = 0
	parserErrorVerbose = false
)

type parserLexer interface {
	Lex(lval *parserSymType) int
	Error(s string)
}

type parserParser interface {
	Parse(parserLexer) int
	Lookahead() int
}

type parserParserImpl struct {
	lookahead func() int
}

func (p *parserParserImpl) Lookahead() int {
	return p.lookahead()
}

func parserNewParser() parserParser {
	p := &parserParserImpl{
		lookahead: func() int { return -1 },
	}
	return p
}

const parserFlag = -1000

func parserTokname(c int) string {
	if c >= 1 && c-1 < len(parserToknames) {
		if parserToknames[c-1] != "" {
			return parserToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func parserStatname(s int) string {
	if s >= 0 && s < len(parserStatenames) {
		if parserStatenames[s] != "" {
			return parserStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func parserErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !parserErrorVerbose {
		return "syntax error"
	}

	for _, e := range parserErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + parserTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := parserPact[state]
	for tok := TOKSTART; tok-1 < len(parserToknames); tok++ {
		if n := base + tok; n >= 0 && n < parserLast && parserChk[parserAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if parserDef[state] == -2 {
		i := 0
		for parserExca[i] != -1 || parserExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; parserExca[i] >= 0; i += 2 {
			tok := parserExca[i]
			if tok < TOKSTART || parserExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if parserExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += parserTokname(tok)
	}
	return res
}

func parserlex1(lex parserLexer, lval *parserSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = parserTok1[0]
		goto out
	}
	if char < len(parserTok1) {
		token = parserTok1[char]
		goto out
	}
	if char >= parserPrivate {
		if char < parserPrivate+len(parserTok2) {
			token = parserTok2[char-parserPrivate]
			goto out
		}
	}
	for i := 0; i < len(parserTok3); i += 2 {
		token = parserTok3[i+0]
		if token == char {
			token = parserTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = parserTok2[1] /* unknown char */
	}
	if parserDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", parserTokname(token), uint(char))
	}
	return char, token
}

func parserParse(parserlex parserLexer) int {
	return parserNewParser().Parse(parserlex)
}

func (parserrcvr *parserParserImpl) Parse(parserlex parserLexer) int {
	var parsern int
	var parserlval parserSymType
	var parserVAL parserSymType
	var parserDollar []parserSymType
	_ = parserDollar // silence set and not used
	parserS := make([]parserSymType, parserMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	parserstate := 0
	parserchar := -1
	parsertoken := -1 // parserchar translated into internal numbering
	parserrcvr.lookahead = func() int { return parserchar }
	defer func() {
		// Make sure we report no lookahead when not parsing.
		parserstate = -1
		parserchar = -1
		parsertoken = -1
	}()
	parserp := -1
	goto parserstack

ret0:
	return 0

ret1:
	return 1

parserstack:
	/* put a state and value onto the stack */
	if parserDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", parserTokname(parsertoken), parserStatname(parserstate))
	}

	parserp++
	if parserp >= len(parserS) {
		nyys := make([]parserSymType, len(parserS)*2)
		copy(nyys, parserS)
		parserS = nyys
	}
	parserS[parserp] = parserVAL
	parserS[parserp].yys = parserstate

parsernewstate:
	parsern = parserPact[parserstate]
	if parsern <= parserFlag {
		goto parserdefault /* simple state */
	}
	if parserchar < 0 {
		parserchar, parsertoken = parserlex1(parserlex, &parserlval)
	}
	parsern += parsertoken
	if parsern < 0 || parsern >= parserLast {
		goto parserdefault
	}
	parsern = parserAct[parsern]
	if parserChk[parsern] == parsertoken { /* valid shift */
		parserchar = -1
		parsertoken = -1
		parserVAL = parserlval
		parserstate = parsern
		if Errflag > 0 {
			Errflag--
		}
		goto parserstack
	}

parserdefault:
	/* default state action */
	parsern = parserDef[parserstate]
	if parsern == -2 {
		if parserchar < 0 {
			parserchar, parsertoken = parserlex1(parserlex, &parserlval)
		}

		/* look through exception table */
		xi := 0
		for {
			if parserExca[xi+0] == -1 && parserExca[xi+1] == parserstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			parsern = parserExca[xi+0]
			if parsern < 0 || parsern == parsertoken {
				break
			}
		}
		parsern = parserExca[xi+1]
		if parsern < 0 {
			goto ret0
		}
	}
	if parsern == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			parserlex.Error(parserErrorMessage(parserstate, parsertoken))
			Nerrs++
			if parserDebug >= 1 {
				__yyfmt__.Printf("%s", parserStatname(parserstate))
				__yyfmt__.Printf(" saw %s\n", parserTokname(parsertoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for parserp >= 0 {
				parsern = parserPact[parserS[parserp].yys] + parserErrCode
				if parsern >= 0 && parsern < parserLast {
					parserstate = parserAct[parsern] /* simulate a shift of "error" */
					if parserChk[parserstate] == parserErrCode {
						goto parserstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if parserDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", parserS[parserp].yys)
				}
				parserp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if parserDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", parserTokname(parsertoken))
			}
			if parsertoken == parserEofCode {
				goto ret1
			}
			parserchar = -1
			parsertoken = -1
			goto parsernewstate /* try again in the same state */
		}
	}

	/* reduction by production parsern */
	if parserDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", parsern, parserStatname(parserstate))
	}

	parsernt := parsern
	parserpt := parserp
	_ = parserpt // guard against "declared and not used"

	parserp -= parserR2[parsern]
	// parserp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if parserp+1 >= len(parserS) {
		nyys := make([]parserSymType, len(parserS)*2)
		copy(nyys, parserS)
		parserS = nyys
	}
	parserVAL = parserS[parserp+1]

	/* consult goto table to find next state */
	parsern = parserR1[parsern]
	parserg := parserPgo[parsern]
	parserj := parserg + parserS[parserp].yys + 1

	if parserj >= parserLast {
		parserstate = parserAct[parserg]
	} else {
		parserstate = parserAct[parserj]
		if parserChk[parserstate] != -parsern {
			parserstate = parserAct[parserg]
		}
	}
	// dummy call; replaced with literal code
	switch parsernt {

	case 1:
		parserDollar = parserS[parserpt-0 : parserpt+1]
		//line lang.y:35
		{
			parserResult = &ast.LiteralNode{
				Value: "",
				Typex: ast.TypeString,
				Posx:  ast.Pos{Column: 1, Line: 1},
			}
		}
	case 2:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:43
		{
			parserResult = parserDollar[1].node

			// We want to make sure that the top value is always a Concat
			// so that the return value is always a string type from an
			// interpolation.
			//
			// The logic for checking for a LiteralNode is a little annoying
			// because functionally the AST is the same, but we do that because
			// it makes for an easy literal check later (to check if a string
			// has any interpolations).
			if _, ok := parserDollar[1].node.(*ast.Concat); !ok {
				if n, ok := parserDollar[1].node.(*ast.LiteralNode); !ok || n.Typex != ast.TypeString {
					parserResult = &ast.Concat{
						Exprs: []ast.Node{parserDollar[1].node},
						Posx:  parserDollar[1].node.Pos(),
					}
				}
			}
		}
	case 3:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:66
		{
			parserVAL.node = parserDollar[1].node
		}
	case 4:
		parserDollar = parserS[parserpt-2 : parserpt+1]
		//line lang.y:70
		{
			var result []ast.Node
			if c, ok := parserDollar[1].node.(*ast.Concat); ok {
				result = append(c.Exprs, parserDollar[2].node)
			} else {
				result = []ast.Node{parserDollar[1].node, parserDollar[2].node}
			}

			parserVAL.node = &ast.Concat{
				Exprs: result,
				Posx:  result[0].Pos(),
			}
		}
	case 5:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:86
		{
			parserVAL.node = parserDollar[1].node
		}
	case 6:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:90
		{
			parserVAL.node = parserDollar[1].node
		}
	case 7:
		parserDollar = parserS[parserpt-3 : parserpt+1]
		//line lang.y:96
		{
			parserVAL.node = parserDollar[2].node
		}
	case 8:
		parserDollar = parserS[parserpt-3 : parserpt+1]
		//line lang.y:102
		{
			parserVAL.node = parserDollar[2].node
		}
	case 9:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:106
		{
			parserVAL.node = parserDollar[1].node
		}
	case 10:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:110
		{
			parserVAL.node = &ast.LiteralNode{
				Value: parserDollar[1].token.Value.(int),
				Typex: ast.TypeInt,
				Posx:  parserDollar[1].token.Pos,
			}
		}
	case 11:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:118
		{
			parserVAL.node = &ast.LiteralNode{
				Value: parserDollar[1].token.Value.(float64),
				Typex: ast.TypeFloat,
				Posx:  parserDollar[1].token.Pos,
			}
		}
	case 12:
		parserDollar = parserS[parserpt-3 : parserpt+1]
		//line lang.y:126
		{
			parserVAL.node = &ast.Arithmetic{
				Op:    parserDollar[2].token.Value.(ast.ArithmeticOp),
				Exprs: []ast.Node{parserDollar[1].node, parserDollar[3].node},
				Posx:  parserDollar[1].node.Pos(),
			}
		}
	case 13:
		parserDollar = parserS[parserpt-2 : parserpt+1]
		//line lang.y:134
		{
			parserVAL.node = &ast.UnaryArithmetic{
				Op:   parserDollar[1].token.Value.(ast.ArithmeticOp),
				Expr: parserDollar[2].node,
				Posx: parserDollar[1].token.Pos,
			}
		}
	case 14:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:142
		{
			parserVAL.node = &ast.VariableAccess{Name: parserDollar[1].token.Value.(string), Posx: parserDollar[1].token.Pos}
		}
	case 15:
		parserDollar = parserS[parserpt-4 : parserpt+1]
		//line lang.y:146
		{
			parserVAL.node = &ast.Call{Func: parserDollar[1].token.Value.(string), Args: parserDollar[3].nodeList, Posx: parserDollar[1].token.Pos}
		}
	case 16:
		parserDollar = parserS[parserpt-0 : parserpt+1]
		//line lang.y:151
		{
			parserVAL.nodeList = nil
		}
	case 17:
		parserDollar = parserS[parserpt-3 : parserpt+1]
		//line lang.y:155
		{
			parserVAL.nodeList = append(parserDollar[1].nodeList, parserDollar[3].node)
		}
	case 18:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:159
		{
			parserVAL.nodeList = append(parserVAL.nodeList, parserDollar[1].node)
		}
	case 19:
		parserDollar = parserS[parserpt-1 : parserpt+1]
		//line lang.y:165
		{
			parserVAL.node = &ast.LiteralNode{
				Value: parserDollar[1].token.Value.(string),
				Typex: ast.TypeString,
				Posx:  parserDollar[1].token.Pos,
			}
		}
	}
	goto parserstack /* stack new state and value */
}
