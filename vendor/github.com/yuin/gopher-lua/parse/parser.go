//line parser.go.y:2
package parse

import __yyfmt__ "fmt"

//line parser.go.y:2
import (
	"github.com/yuin/gopher-lua/ast"
)

//line parser.go.y:34
type yySymType struct {
	yys   int
	token ast.Token

	stmts []ast.Stmt
	stmt  ast.Stmt

	funcname *ast.FuncName
	funcexpr *ast.FunctionExpr

	exprlist []ast.Expr
	expr     ast.Expr

	fieldlist []*ast.Field
	field     *ast.Field
	fieldsep  string

	namelist []string
	parlist  *ast.ParList
}

const TAnd = 57346
const TBreak = 57347
const TDo = 57348
const TElse = 57349
const TElseIf = 57350
const TEnd = 57351
const TFalse = 57352
const TFor = 57353
const TFunction = 57354
const TIf = 57355
const TIn = 57356
const TLocal = 57357
const TNil = 57358
const TNot = 57359
const TOr = 57360
const TReturn = 57361
const TRepeat = 57362
const TThen = 57363
const TTrue = 57364
const TUntil = 57365
const TWhile = 57366
const TEqeq = 57367
const TNeq = 57368
const TLte = 57369
const TGte = 57370
const T2Comma = 57371
const T3Comma = 57372
const TIdent = 57373
const TNumber = 57374
const TString = 57375
const UNARY = 57376

var yyToknames = []string{
	"TAnd",
	"TBreak",
	"TDo",
	"TElse",
	"TElseIf",
	"TEnd",
	"TFalse",
	"TFor",
	"TFunction",
	"TIf",
	"TIn",
	"TLocal",
	"TNil",
	"TNot",
	"TOr",
	"TReturn",
	"TRepeat",
	"TThen",
	"TTrue",
	"TUntil",
	"TWhile",
	"TEqeq",
	"TNeq",
	"TLte",
	"TGte",
	"T2Comma",
	"T3Comma",
	"TIdent",
	"TNumber",
	"TString",
	" {",
	" (",
	" >",
	" <",
	" +",
	" -",
	" *",
	" /",
	" %",
	"UNARY",
	" ^",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line parser.go.y:514
func TokenName(c int) string {
	if c >= TAnd && c-TAnd < len(yyToknames) {
		if yyToknames[c-TAnd] != "" {
			return yyToknames[c-TAnd]
		}
	}
	return string([]byte{byte(c)})
}

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 17,
	46, 31,
	47, 31,
	-2, 68,
	-1, 93,
	46, 32,
	47, 32,
	-2, 68,
}

const yyNprod = 95
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 579

var yyAct = []int{

	24, 88, 50, 23, 45, 84, 56, 65, 137, 153,
	136, 113, 52, 142, 54, 53, 33, 134, 65, 132,
	62, 63, 32, 61, 108, 109, 48, 111, 106, 41,
	42, 105, 49, 155, 166, 81, 82, 83, 138, 104,
	22, 91, 131, 80, 95, 92, 162, 74, 48, 85,
	150, 99, 165, 148, 49, 149, 75, 76, 77, 78,
	79, 67, 80, 107, 106, 148, 114, 115, 116, 117,
	118, 119, 120, 121, 122, 123, 124, 125, 126, 127,
	128, 129, 72, 73, 71, 70, 74, 65, 39, 40,
	47, 139, 133, 68, 69, 75, 76, 77, 78, 79,
	60, 80, 141, 144, 143, 146, 145, 31, 67, 147,
	9, 48, 110, 97, 48, 152, 151, 49, 38, 62,
	49, 17, 66, 77, 78, 79, 96, 80, 59, 72,
	73, 71, 70, 74, 154, 102, 91, 156, 55, 157,
	68, 69, 75, 76, 77, 78, 79, 21, 80, 187,
	94, 20, 26, 184, 37, 179, 163, 112, 25, 35,
	178, 93, 170, 172, 27, 171, 164, 173, 19, 159,
	175, 174, 29, 89, 28, 39, 40, 20, 182, 181,
	100, 34, 135, 183, 67, 39, 40, 47, 186, 64,
	51, 1, 90, 87, 36, 130, 86, 30, 66, 18,
	46, 44, 43, 8, 58, 72, 73, 71, 70, 74,
	57, 67, 168, 169, 167, 3, 68, 69, 75, 76,
	77, 78, 79, 160, 80, 66, 4, 2, 0, 0,
	0, 158, 72, 73, 71, 70, 74, 0, 0, 0,
	0, 0, 0, 68, 69, 75, 76, 77, 78, 79,
	26, 80, 37, 0, 0, 0, 25, 35, 140, 0,
	0, 0, 27, 0, 0, 0, 0, 0, 0, 0,
	29, 21, 28, 39, 40, 20, 26, 0, 37, 34,
	0, 0, 25, 35, 0, 0, 0, 0, 27, 0,
	0, 0, 36, 98, 0, 0, 29, 89, 28, 39,
	40, 20, 26, 0, 37, 34, 0, 0, 25, 35,
	0, 0, 0, 0, 27, 67, 90, 176, 36, 0,
	0, 0, 29, 21, 28, 39, 40, 20, 0, 66,
	0, 34, 0, 0, 0, 0, 72, 73, 71, 70,
	74, 0, 67, 0, 36, 0, 0, 68, 69, 75,
	76, 77, 78, 79, 0, 80, 66, 0, 177, 0,
	0, 0, 0, 72, 73, 71, 70, 74, 0, 67,
	0, 185, 0, 0, 68, 69, 75, 76, 77, 78,
	79, 0, 80, 66, 0, 161, 0, 0, 0, 0,
	72, 73, 71, 70, 74, 0, 67, 0, 0, 0,
	0, 68, 69, 75, 76, 77, 78, 79, 0, 80,
	66, 0, 0, 180, 0, 0, 0, 72, 73, 71,
	70, 74, 0, 67, 0, 0, 0, 0, 68, 69,
	75, 76, 77, 78, 79, 0, 80, 66, 0, 0,
	103, 0, 0, 0, 72, 73, 71, 70, 74, 0,
	67, 0, 101, 0, 0, 68, 69, 75, 76, 77,
	78, 79, 0, 80, 66, 0, 0, 0, 0, 0,
	0, 72, 73, 71, 70, 74, 0, 67, 0, 0,
	0, 0, 68, 69, 75, 76, 77, 78, 79, 0,
	80, 66, 0, 0, 0, 0, 0, 0, 72, 73,
	71, 70, 74, 0, 0, 0, 0, 0, 0, 68,
	69, 75, 76, 77, 78, 79, 0, 80, 72, 73,
	71, 70, 74, 0, 0, 0, 0, 0, 0, 68,
	69, 75, 76, 77, 78, 79, 0, 80, 7, 10,
	0, 0, 0, 0, 14, 15, 13, 0, 16, 0,
	0, 0, 6, 12, 0, 0, 0, 11, 0, 0,
	0, 0, 0, 0, 21, 0, 0, 0, 20, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 5,
}
var yyPact = []int{

	-1000, -1000, 533, -5, -1000, -1000, 292, -1000, -17, 152,
	-1000, 292, -1000, 292, 107, 97, 88, -1000, -1000, -1000,
	292, -1000, -1000, -29, 473, -1000, -1000, -1000, -1000, -1000,
	-1000, 152, -1000, -1000, 292, 292, 292, 14, -1000, -1000,
	142, 292, 116, 292, 95, -1000, 82, 240, -1000, -1000,
	171, -1000, 446, 112, 419, -7, 17, 14, -24, -1000,
	81, -19, -1000, 104, -42, 292, 292, 292, 292, 292,
	292, 292, 292, 292, 292, 292, 292, 292, 292, 292,
	292, -1, -1, -1, -1000, -11, -1000, -37, -1000, -8,
	292, 473, -29, -1000, 152, 207, -1000, 55, -1000, -40,
	-1000, -1000, 292, -1000, 292, 292, 34, -1000, 24, 19,
	14, 292, -1000, -1000, 473, 57, 493, 18, 18, 18,
	18, 18, 18, 18, 83, 83, -1, -1, -1, -1,
	-44, -1000, -1000, -14, -1000, 266, -1000, -1000, 292, 180,
	-1000, -1000, -1000, 160, 473, -1000, 338, 40, -1000, -1000,
	-1000, -1000, -29, -1000, 157, 22, -1000, 473, -12, -1000,
	205, 292, -1000, 154, -1000, -1000, 292, -1000, -1000, 292,
	311, 151, -1000, 473, 146, 392, -1000, 292, -1000, -1000,
	-1000, 144, 365, -1000, -1000, -1000, 140, -1000,
}
var yyPgo = []int{

	0, 190, 227, 2, 226, 223, 215, 210, 204, 203,
	118, 6, 3, 0, 22, 107, 168, 199, 4, 197,
	5, 195, 16, 193, 1, 182,
}
var yyR1 = []int{

	0, 1, 1, 1, 2, 2, 2, 3, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 5, 5, 6, 6, 6, 7, 7, 8,
	8, 9, 9, 10, 10, 10, 11, 11, 12, 12,
	13, 13, 13, 13, 13, 13, 13, 13, 13, 13,
	13, 13, 13, 13, 13, 13, 13, 13, 13, 13,
	13, 13, 13, 13, 13, 13, 13, 14, 15, 15,
	15, 15, 17, 16, 16, 18, 18, 18, 18, 19,
	20, 20, 21, 21, 21, 22, 22, 23, 23, 23,
	24, 24, 24, 25, 25,
}
var yyR2 = []int{

	0, 1, 2, 3, 0, 2, 2, 1, 3, 1,
	3, 5, 4, 6, 8, 9, 11, 7, 3, 4,
	4, 2, 0, 5, 1, 2, 1, 1, 3, 1,
	3, 1, 3, 1, 4, 3, 1, 3, 1, 3,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 2, 2, 2, 1, 1, 1,
	1, 3, 3, 2, 4, 2, 3, 1, 1, 2,
	5, 4, 1, 1, 3, 2, 3, 1, 3, 2,
	3, 5, 1, 1, 1,
}
var yyChk = []int{

	-1000, -1, -2, -6, -4, 45, 19, 5, -9, -15,
	6, 24, 20, 13, 11, 12, 15, -10, -17, -16,
	35, 31, 45, -12, -13, 16, 10, 22, 32, 30,
	-19, -15, -14, -22, 39, 17, 52, 12, -10, 33,
	34, 46, 47, 50, 49, -18, 48, 35, -22, -14,
	-3, -1, -13, -3, -13, 31, -11, -7, -8, 31,
	12, -11, 31, -13, -16, 47, 18, 4, 36, 37,
	28, 27, 25, 26, 29, 38, 39, 40, 41, 42,
	44, -13, -13, -13, -20, 35, 54, -23, -24, 31,
	50, -13, -12, -10, -15, -13, 31, 31, 53, -12,
	9, 6, 23, 21, 46, 14, 47, -20, 48, 49,
	31, 46, 53, 53, -13, -13, -13, -13, -13, -13,
	-13, -13, -13, -13, -13, -13, -13, -13, -13, -13,
	-21, 53, 30, -11, 54, -25, 47, 45, 46, -13,
	51, -18, 53, -3, -13, -3, -13, -12, 31, 31,
	31, -20, -12, 53, -3, 47, -24, -13, 51, 9,
	-5, 47, 6, -3, 9, 30, 46, 9, 7, 8,
	-13, -3, 9, -13, -3, -13, 6, 47, 9, 9,
	21, -3, -13, -3, 9, 6, -3, 9,
}
var yyDef = []int{

	4, -2, 1, 2, 5, 6, 24, 26, 0, 9,
	4, 0, 4, 0, 0, 0, 0, -2, 69, 70,
	0, 33, 3, 25, 38, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 0, 0, 0, 0, 68, 67,
	0, 0, 0, 0, 0, 73, 0, 0, 77, 78,
	0, 7, 0, 0, 0, 36, 0, 0, 27, 29,
	0, 21, 36, 0, 70, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 64, 65, 66, 79, 0, 85, 0, 87, 33,
	0, 92, 8, -2, 0, 0, 35, 0, 75, 0,
	10, 4, 0, 4, 0, 0, 0, 18, 0, 0,
	0, 0, 71, 72, 39, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	0, 4, 82, 83, 86, 89, 93, 94, 0, 0,
	34, 74, 76, 0, 12, 22, 0, 0, 37, 28,
	30, 19, 20, 4, 0, 0, 88, 90, 0, 11,
	0, 0, 4, 0, 81, 84, 0, 13, 4, 0,
	0, 0, 80, 91, 0, 0, 4, 0, 17, 14,
	4, 0, 0, 23, 15, 4, 0, 16,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 52, 3, 42, 3, 3,
	35, 53, 40, 38, 47, 39, 49, 41, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 48, 45,
	37, 46, 36, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 50, 3, 51, 44, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 34, 3, 54,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 43,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(yyToknames) {
		if yyToknames[c-4] != "" {
			return yyToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(c), uint(char))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		//line parser.go.y:73
		{
			yyVAL.stmts = yyS[yypt-0].stmts
			if l, ok := yylex.(*Lexer); ok {
				l.Stmts = yyVAL.stmts
			}
		}
	case 2:
		//line parser.go.y:79
		{
			yyVAL.stmts = append(yyS[yypt-1].stmts, yyS[yypt-0].stmt)
			if l, ok := yylex.(*Lexer); ok {
				l.Stmts = yyVAL.stmts
			}
		}
	case 3:
		//line parser.go.y:85
		{
			yyVAL.stmts = append(yyS[yypt-2].stmts, yyS[yypt-1].stmt)
			if l, ok := yylex.(*Lexer); ok {
				l.Stmts = yyVAL.stmts
			}
		}
	case 4:
		//line parser.go.y:93
		{
			yyVAL.stmts = []ast.Stmt{}
		}
	case 5:
		//line parser.go.y:96
		{
			yyVAL.stmts = append(yyS[yypt-1].stmts, yyS[yypt-0].stmt)
		}
	case 6:
		//line parser.go.y:99
		{
			yyVAL.stmts = yyS[yypt-1].stmts
		}
	case 7:
		//line parser.go.y:104
		{
			yyVAL.stmts = yyS[yypt-0].stmts
		}
	case 8:
		//line parser.go.y:109
		{
			yyVAL.stmt = &ast.AssignStmt{Lhs: yyS[yypt-2].exprlist, Rhs: yyS[yypt-0].exprlist}
			yyVAL.stmt.SetLine(yyS[yypt-2].exprlist[0].Line())
		}
	case 9:
		//line parser.go.y:114
		{
			if _, ok := yyS[yypt-0].expr.(*ast.FuncCallExpr); !ok {
				yylex.(*Lexer).Error("parse error")
			} else {
				yyVAL.stmt = &ast.FuncCallStmt{Expr: yyS[yypt-0].expr}
				yyVAL.stmt.SetLine(yyS[yypt-0].expr.Line())
			}
		}
	case 10:
		//line parser.go.y:122
		{
			yyVAL.stmt = &ast.DoBlockStmt{Stmts: yyS[yypt-1].stmts}
			yyVAL.stmt.SetLine(yyS[yypt-2].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 11:
		//line parser.go.y:127
		{
			yyVAL.stmt = &ast.WhileStmt{Condition: yyS[yypt-3].expr, Stmts: yyS[yypt-1].stmts}
			yyVAL.stmt.SetLine(yyS[yypt-4].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 12:
		//line parser.go.y:132
		{
			yyVAL.stmt = &ast.RepeatStmt{Condition: yyS[yypt-0].expr, Stmts: yyS[yypt-2].stmts}
			yyVAL.stmt.SetLine(yyS[yypt-3].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].expr.Line())
		}
	case 13:
		//line parser.go.y:137
		{
			yyVAL.stmt = &ast.IfStmt{Condition: yyS[yypt-4].expr, Then: yyS[yypt-2].stmts}
			cur := yyVAL.stmt
			for _, elseif := range yyS[yypt-1].stmts {
				cur.(*ast.IfStmt).Else = []ast.Stmt{elseif}
				cur = elseif
			}
			yyVAL.stmt.SetLine(yyS[yypt-5].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 14:
		//line parser.go.y:147
		{
			yyVAL.stmt = &ast.IfStmt{Condition: yyS[yypt-6].expr, Then: yyS[yypt-4].stmts}
			cur := yyVAL.stmt
			for _, elseif := range yyS[yypt-3].stmts {
				cur.(*ast.IfStmt).Else = []ast.Stmt{elseif}
				cur = elseif
			}
			cur.(*ast.IfStmt).Else = yyS[yypt-1].stmts
			yyVAL.stmt.SetLine(yyS[yypt-7].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 15:
		//line parser.go.y:158
		{
			yyVAL.stmt = &ast.NumberForStmt{Name: yyS[yypt-7].token.Str, Init: yyS[yypt-5].expr, Limit: yyS[yypt-3].expr, Stmts: yyS[yypt-1].stmts}
			yyVAL.stmt.SetLine(yyS[yypt-8].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 16:
		//line parser.go.y:163
		{
			yyVAL.stmt = &ast.NumberForStmt{Name: yyS[yypt-9].token.Str, Init: yyS[yypt-7].expr, Limit: yyS[yypt-5].expr, Step: yyS[yypt-3].expr, Stmts: yyS[yypt-1].stmts}
			yyVAL.stmt.SetLine(yyS[yypt-10].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 17:
		//line parser.go.y:168
		{
			yyVAL.stmt = &ast.GenericForStmt{Names: yyS[yypt-5].namelist, Exprs: yyS[yypt-3].exprlist, Stmts: yyS[yypt-1].stmts}
			yyVAL.stmt.SetLine(yyS[yypt-6].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 18:
		//line parser.go.y:173
		{
			yyVAL.stmt = &ast.FuncDefStmt{Name: yyS[yypt-1].funcname, Func: yyS[yypt-0].funcexpr}
			yyVAL.stmt.SetLine(yyS[yypt-2].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].funcexpr.LastLine())
		}
	case 19:
		//line parser.go.y:178
		{
			yyVAL.stmt = &ast.LocalAssignStmt{Names: []string{yyS[yypt-1].token.Str}, Exprs: []ast.Expr{yyS[yypt-0].funcexpr}}
			yyVAL.stmt.SetLine(yyS[yypt-3].token.Pos.Line)
			yyVAL.stmt.SetLastLine(yyS[yypt-0].funcexpr.LastLine())
		}
	case 20:
		//line parser.go.y:183
		{
			yyVAL.stmt = &ast.LocalAssignStmt{Names: yyS[yypt-2].namelist, Exprs: yyS[yypt-0].exprlist}
			yyVAL.stmt.SetLine(yyS[yypt-3].token.Pos.Line)
		}
	case 21:
		//line parser.go.y:187
		{
			yyVAL.stmt = &ast.LocalAssignStmt{Names: yyS[yypt-0].namelist, Exprs: []ast.Expr{}}
			yyVAL.stmt.SetLine(yyS[yypt-1].token.Pos.Line)
		}
	case 22:
		//line parser.go.y:193
		{
			yyVAL.stmts = []ast.Stmt{}
		}
	case 23:
		//line parser.go.y:196
		{
			yyVAL.stmts = append(yyS[yypt-4].stmts, &ast.IfStmt{Condition: yyS[yypt-2].expr, Then: yyS[yypt-0].stmts})
			yyVAL.stmts[len(yyVAL.stmts)-1].SetLine(yyS[yypt-3].token.Pos.Line)
		}
	case 24:
		//line parser.go.y:202
		{
			yyVAL.stmt = &ast.ReturnStmt{Exprs: nil}
			yyVAL.stmt.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 25:
		//line parser.go.y:206
		{
			yyVAL.stmt = &ast.ReturnStmt{Exprs: yyS[yypt-0].exprlist}
			yyVAL.stmt.SetLine(yyS[yypt-1].token.Pos.Line)
		}
	case 26:
		//line parser.go.y:210
		{
			yyVAL.stmt = &ast.BreakStmt{}
			yyVAL.stmt.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 27:
		//line parser.go.y:216
		{
			yyVAL.funcname = yyS[yypt-0].funcname
		}
	case 28:
		//line parser.go.y:219
		{
			yyVAL.funcname = &ast.FuncName{Func: nil, Receiver: yyS[yypt-2].funcname.Func, Method: yyS[yypt-0].token.Str}
		}
	case 29:
		//line parser.go.y:224
		{
			yyVAL.funcname = &ast.FuncName{Func: &ast.IdentExpr{Value: yyS[yypt-0].token.Str}}
			yyVAL.funcname.Func.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 30:
		//line parser.go.y:228
		{
			key := &ast.StringExpr{Value: yyS[yypt-0].token.Str}
			key.SetLine(yyS[yypt-0].token.Pos.Line)
			fn := &ast.AttrGetExpr{Object: yyS[yypt-2].funcname.Func, Key: key}
			fn.SetLine(yyS[yypt-0].token.Pos.Line)
			yyVAL.funcname = &ast.FuncName{Func: fn}
		}
	case 31:
		//line parser.go.y:237
		{
			yyVAL.exprlist = []ast.Expr{yyS[yypt-0].expr}
		}
	case 32:
		//line parser.go.y:240
		{
			yyVAL.exprlist = append(yyS[yypt-2].exprlist, yyS[yypt-0].expr)
		}
	case 33:
		//line parser.go.y:245
		{
			yyVAL.expr = &ast.IdentExpr{Value: yyS[yypt-0].token.Str}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 34:
		//line parser.go.y:249
		{
			yyVAL.expr = &ast.AttrGetExpr{Object: yyS[yypt-3].expr, Key: yyS[yypt-1].expr}
			yyVAL.expr.SetLine(yyS[yypt-3].expr.Line())
		}
	case 35:
		//line parser.go.y:253
		{
			key := &ast.StringExpr{Value: yyS[yypt-0].token.Str}
			key.SetLine(yyS[yypt-0].token.Pos.Line)
			yyVAL.expr = &ast.AttrGetExpr{Object: yyS[yypt-2].expr, Key: key}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 36:
		//line parser.go.y:261
		{
			yyVAL.namelist = []string{yyS[yypt-0].token.Str}
		}
	case 37:
		//line parser.go.y:264
		{
			yyVAL.namelist = append(yyS[yypt-2].namelist, yyS[yypt-0].token.Str)
		}
	case 38:
		//line parser.go.y:269
		{
			yyVAL.exprlist = []ast.Expr{yyS[yypt-0].expr}
		}
	case 39:
		//line parser.go.y:272
		{
			yyVAL.exprlist = append(yyS[yypt-2].exprlist, yyS[yypt-0].expr)
		}
	case 40:
		//line parser.go.y:277
		{
			yyVAL.expr = &ast.NilExpr{}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 41:
		//line parser.go.y:281
		{
			yyVAL.expr = &ast.FalseExpr{}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 42:
		//line parser.go.y:285
		{
			yyVAL.expr = &ast.TrueExpr{}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 43:
		//line parser.go.y:289
		{
			yyVAL.expr = &ast.NumberExpr{Value: yyS[yypt-0].token.Str}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 44:
		//line parser.go.y:293
		{
			yyVAL.expr = &ast.Comma3Expr{}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 45:
		//line parser.go.y:297
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 46:
		//line parser.go.y:300
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 47:
		//line parser.go.y:303
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 48:
		//line parser.go.y:306
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 49:
		//line parser.go.y:309
		{
			yyVAL.expr = &ast.LogicalOpExpr{Lhs: yyS[yypt-2].expr, Operator: "or", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 50:
		//line parser.go.y:313
		{
			yyVAL.expr = &ast.LogicalOpExpr{Lhs: yyS[yypt-2].expr, Operator: "and", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 51:
		//line parser.go.y:317
		{
			yyVAL.expr = &ast.RelationalOpExpr{Lhs: yyS[yypt-2].expr, Operator: ">", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 52:
		//line parser.go.y:321
		{
			yyVAL.expr = &ast.RelationalOpExpr{Lhs: yyS[yypt-2].expr, Operator: "<", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 53:
		//line parser.go.y:325
		{
			yyVAL.expr = &ast.RelationalOpExpr{Lhs: yyS[yypt-2].expr, Operator: ">=", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 54:
		//line parser.go.y:329
		{
			yyVAL.expr = &ast.RelationalOpExpr{Lhs: yyS[yypt-2].expr, Operator: "<=", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 55:
		//line parser.go.y:333
		{
			yyVAL.expr = &ast.RelationalOpExpr{Lhs: yyS[yypt-2].expr, Operator: "==", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 56:
		//line parser.go.y:337
		{
			yyVAL.expr = &ast.RelationalOpExpr{Lhs: yyS[yypt-2].expr, Operator: "~=", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 57:
		//line parser.go.y:341
		{
			yyVAL.expr = &ast.StringConcatOpExpr{Lhs: yyS[yypt-2].expr, Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 58:
		//line parser.go.y:345
		{
			yyVAL.expr = &ast.ArithmeticOpExpr{Lhs: yyS[yypt-2].expr, Operator: "+", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 59:
		//line parser.go.y:349
		{
			yyVAL.expr = &ast.ArithmeticOpExpr{Lhs: yyS[yypt-2].expr, Operator: "-", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 60:
		//line parser.go.y:353
		{
			yyVAL.expr = &ast.ArithmeticOpExpr{Lhs: yyS[yypt-2].expr, Operator: "*", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 61:
		//line parser.go.y:357
		{
			yyVAL.expr = &ast.ArithmeticOpExpr{Lhs: yyS[yypt-2].expr, Operator: "/", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 62:
		//line parser.go.y:361
		{
			yyVAL.expr = &ast.ArithmeticOpExpr{Lhs: yyS[yypt-2].expr, Operator: "%", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 63:
		//line parser.go.y:365
		{
			yyVAL.expr = &ast.ArithmeticOpExpr{Lhs: yyS[yypt-2].expr, Operator: "^", Rhs: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-2].expr.Line())
		}
	case 64:
		//line parser.go.y:369
		{
			yyVAL.expr = &ast.UnaryMinusOpExpr{Expr: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-0].expr.Line())
		}
	case 65:
		//line parser.go.y:373
		{
			yyVAL.expr = &ast.UnaryNotOpExpr{Expr: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-0].expr.Line())
		}
	case 66:
		//line parser.go.y:377
		{
			yyVAL.expr = &ast.UnaryLenOpExpr{Expr: yyS[yypt-0].expr}
			yyVAL.expr.SetLine(yyS[yypt-0].expr.Line())
		}
	case 67:
		//line parser.go.y:383
		{
			yyVAL.expr = &ast.StringExpr{Value: yyS[yypt-0].token.Str}
			yyVAL.expr.SetLine(yyS[yypt-0].token.Pos.Line)
		}
	case 68:
		//line parser.go.y:389
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 69:
		//line parser.go.y:392
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line parser.go.y:395
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 71:
		//line parser.go.y:398
		{
			yyVAL.expr = yyS[yypt-1].expr
			yyVAL.expr.SetLine(yyS[yypt-2].token.Pos.Line)
		}
	case 72:
		//line parser.go.y:404
		{
			yyS[yypt-1].expr.(*ast.FuncCallExpr).AdjustRet = true
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 73:
		//line parser.go.y:410
		{
			yyVAL.expr = &ast.FuncCallExpr{Func: yyS[yypt-1].expr, Args: yyS[yypt-0].exprlist}
			yyVAL.expr.SetLine(yyS[yypt-1].expr.Line())
		}
	case 74:
		//line parser.go.y:414
		{
			yyVAL.expr = &ast.FuncCallExpr{Method: yyS[yypt-1].token.Str, Receiver: yyS[yypt-3].expr, Args: yyS[yypt-0].exprlist}
			yyVAL.expr.SetLine(yyS[yypt-3].expr.Line())
		}
	case 75:
		//line parser.go.y:420
		{
			if yylex.(*Lexer).PNewLine {
				yylex.(*Lexer).TokenError(yyS[yypt-1].token, "ambiguous syntax (function call x new statement)")
			}
			yyVAL.exprlist = []ast.Expr{}
		}
	case 76:
		//line parser.go.y:426
		{
			if yylex.(*Lexer).PNewLine {
				yylex.(*Lexer).TokenError(yyS[yypt-2].token, "ambiguous syntax (function call x new statement)")
			}
			yyVAL.exprlist = yyS[yypt-1].exprlist
		}
	case 77:
		//line parser.go.y:432
		{
			yyVAL.exprlist = []ast.Expr{yyS[yypt-0].expr}
		}
	case 78:
		//line parser.go.y:435
		{
			yyVAL.exprlist = []ast.Expr{yyS[yypt-0].expr}
		}
	case 79:
		//line parser.go.y:440
		{
			yyVAL.expr = &ast.FunctionExpr{ParList: yyS[yypt-0].funcexpr.ParList, Stmts: yyS[yypt-0].funcexpr.Stmts}
			yyVAL.expr.SetLine(yyS[yypt-1].token.Pos.Line)
			yyVAL.expr.SetLastLine(yyS[yypt-0].funcexpr.LastLine())
		}
	case 80:
		//line parser.go.y:447
		{
			yyVAL.funcexpr = &ast.FunctionExpr{ParList: yyS[yypt-3].parlist, Stmts: yyS[yypt-1].stmts}
			yyVAL.funcexpr.SetLine(yyS[yypt-4].token.Pos.Line)
			yyVAL.funcexpr.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 81:
		//line parser.go.y:452
		{
			yyVAL.funcexpr = &ast.FunctionExpr{ParList: &ast.ParList{HasVargs: false, Names: []string{}}, Stmts: yyS[yypt-1].stmts}
			yyVAL.funcexpr.SetLine(yyS[yypt-3].token.Pos.Line)
			yyVAL.funcexpr.SetLastLine(yyS[yypt-0].token.Pos.Line)
		}
	case 82:
		//line parser.go.y:459
		{
			yyVAL.parlist = &ast.ParList{HasVargs: true, Names: []string{}}
		}
	case 83:
		//line parser.go.y:462
		{
			yyVAL.parlist = &ast.ParList{HasVargs: false, Names: []string{}}
			yyVAL.parlist.Names = append(yyVAL.parlist.Names, yyS[yypt-0].namelist...)
		}
	case 84:
		//line parser.go.y:466
		{
			yyVAL.parlist = &ast.ParList{HasVargs: true, Names: []string{}}
			yyVAL.parlist.Names = append(yyVAL.parlist.Names, yyS[yypt-2].namelist...)
		}
	case 85:
		//line parser.go.y:473
		{
			yyVAL.expr = &ast.TableExpr{Fields: []*ast.Field{}}
			yyVAL.expr.SetLine(yyS[yypt-1].token.Pos.Line)
		}
	case 86:
		//line parser.go.y:477
		{
			yyVAL.expr = &ast.TableExpr{Fields: yyS[yypt-1].fieldlist}
			yyVAL.expr.SetLine(yyS[yypt-2].token.Pos.Line)
		}
	case 87:
		//line parser.go.y:484
		{
			yyVAL.fieldlist = []*ast.Field{yyS[yypt-0].field}
		}
	case 88:
		//line parser.go.y:487
		{
			yyVAL.fieldlist = append(yyS[yypt-2].fieldlist, yyS[yypt-0].field)
		}
	case 89:
		//line parser.go.y:490
		{
			yyVAL.fieldlist = yyS[yypt-1].fieldlist
		}
	case 90:
		//line parser.go.y:495
		{
			yyVAL.field = &ast.Field{Key: &ast.StringExpr{Value: yyS[yypt-2].token.Str}, Value: yyS[yypt-0].expr}
			yyVAL.field.Key.SetLine(yyS[yypt-2].token.Pos.Line)
		}
	case 91:
		//line parser.go.y:499
		{
			yyVAL.field = &ast.Field{Key: yyS[yypt-3].expr, Value: yyS[yypt-0].expr}
		}
	case 92:
		//line parser.go.y:502
		{
			yyVAL.field = &ast.Field{Value: yyS[yypt-0].expr}
		}
	case 93:
		//line parser.go.y:507
		{
			yyVAL.fieldsep = ","
		}
	case 94:
		//line parser.go.y:510
		{
			yyVAL.fieldsep = ";"
		}
	}
	goto yystack /* stack new state and value */
}
