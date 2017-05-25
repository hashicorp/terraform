//line parse.y:4
package hcl

import __yyfmt__ "fmt"

//line parse.y:4
import (
	"fmt"
	"strconv"
)

//line parse.y:13
type hclSymType struct {
	yys     int
	b       bool
	num     int
	str     string
	obj     *Object
	objlist []*Object
}

const BOOL = 57346
const NUMBER = 57347
const COMMA = 57348
const IDENTIFIER = 57349
const EQUAL = 57350
const NEWLINE = 57351
const STRING = 57352
const MINUS = 57353
const LEFTBRACE = 57354
const RIGHTBRACE = 57355
const LEFTBRACKET = 57356
const RIGHTBRACKET = 57357
const PERIOD = 57358
const EPLUS = 57359
const EMINUS = 57360

var hclToknames = []string{
	"BOOL",
	"NUMBER",
	"COMMA",
	"IDENTIFIER",
	"EQUAL",
	"NEWLINE",
	"STRING",
	"MINUS",
	"LEFTBRACE",
	"RIGHTBRACE",
	"LEFTBRACKET",
	"RIGHTBRACKET",
	"PERIOD",
	"EPLUS",
	"EMINUS",
}
var hclStatenames = []string{}

const hclEofCode = 1
const hclErrCode = 2
const hclMaxDepth = 200

//line parse.y:228

//line yacctab:1
var hclExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const hclNprod = 30
const hclPrivate = 57344

var hclTokenNames []string
var hclStates []string

const hclLast = 56

var hclAct = []int{

	32, 3, 19, 41, 8, 27, 28, 29, 15, 22,
	10, 33, 40, 22, 16, 21, 12, 22, 20, 21,
	17, 14, 34, 21, 35, 8, 4, 31, 4, 7,
	13, 7, 36, 7, 24, 12, 2, 4, 22, 9,
	7, 5, 42, 34, 21, 39, 38, 37, 11, 23,
	1, 25, 26, 6, 30, 18,
}
var hclPact = []int{

	30, -1000, 30, -1000, 31, -1000, 23, -1000, -1000, 4,
	-1000, -1000, 21, -1000, -1000, -1000, -1000, -1000, -1000, -11,
	12, 8, -1000, 19, -1000, -1000, -1000, 42, 41, 40,
	-3, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, 33, -1000,
}
var hclPgo = []int{

	0, 2, 55, 54, 36, 41, 11, 10, 1, 0,
	53, 52, 51, 50,
}
var hclR1 = []int{

	0, 13, 4, 4, 7, 7, 8, 8, 8, 8,
	8, 8, 5, 5, 10, 10, 2, 2, 3, 3,
	9, 9, 6, 6, 6, 1, 1, 11, 11, 12,
}
var hclR2 = []int{

	0, 1, 1, 2, 3, 2, 3, 3, 3, 3,
	3, 1, 2, 2, 1, 1, 3, 2, 1, 3,
	1, 1, 1, 2, 2, 2, 1, 2, 2, 2,
}
var hclChk = []int{

	-1000, -13, -4, -8, 7, -5, -10, 10, -8, 8,
	-7, -5, 12, 7, -6, 4, 10, -7, -2, -1,
	14, 11, 5, -4, 13, -12, -11, 16, 17, 18,
	-3, 15, -9, -6, 10, -1, 13, 5, 5, 5,
	15, 6, -9,
}
var hclDef = []int{

	0, -2, 1, 2, 14, 11, 0, 15, 3, 0,
	12, 13, 0, 14, 6, 7, 8, 9, 10, 22,
	0, 0, 26, 0, 5, 23, 24, 0, 0, 0,
	0, 17, 18, 20, 21, 25, 4, 29, 27, 28,
	16, 0, 19,
}
var hclTok1 = []int{

	1,
}
var hclTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18,
}
var hclTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var hclDebug = 0

type hclLexer interface {
	Lex(lval *hclSymType) int
	Error(s string)
}

const hclFlag = -1000

func hclTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(hclToknames) {
		if hclToknames[c-4] != "" {
			return hclToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func hclStatname(s int) string {
	if s >= 0 && s < len(hclStatenames) {
		if hclStatenames[s] != "" {
			return hclStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func hcllex1(lex hclLexer, lval *hclSymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = hclTok1[0]
		goto out
	}
	if char < len(hclTok1) {
		c = hclTok1[char]
		goto out
	}
	if char >= hclPrivate {
		if char < hclPrivate+len(hclTok2) {
			c = hclTok2[char-hclPrivate]
			goto out
		}
	}
	for i := 0; i < len(hclTok3); i += 2 {
		c = hclTok3[i+0]
		if c == char {
			c = hclTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = hclTok2[1] /* unknown char */
	}
	if hclDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", hclTokname(c), uint(char))
	}
	return c
}

func hclParse(hcllex hclLexer) int {
	var hcln int
	var hcllval hclSymType
	var hclVAL hclSymType
	hclS := make([]hclSymType, hclMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	hclstate := 0
	hclchar := -1
	hclp := -1
	goto hclstack

ret0:
	return 0

ret1:
	return 1

hclstack:
	/* put a state and value onto the stack */
	if hclDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", hclTokname(hclchar), hclStatname(hclstate))
	}

	hclp++
	if hclp >= len(hclS) {
		nyys := make([]hclSymType, len(hclS)*2)
		copy(nyys, hclS)
		hclS = nyys
	}
	hclS[hclp] = hclVAL
	hclS[hclp].yys = hclstate

hclnewstate:
	hcln = hclPact[hclstate]
	if hcln <= hclFlag {
		goto hcldefault /* simple state */
	}
	if hclchar < 0 {
		hclchar = hcllex1(hcllex, &hcllval)
	}
	hcln += hclchar
	if hcln < 0 || hcln >= hclLast {
		goto hcldefault
	}
	hcln = hclAct[hcln]
	if hclChk[hcln] == hclchar { /* valid shift */
		hclchar = -1
		hclVAL = hcllval
		hclstate = hcln
		if Errflag > 0 {
			Errflag--
		}
		goto hclstack
	}

hcldefault:
	/* default state action */
	hcln = hclDef[hclstate]
	if hcln == -2 {
		if hclchar < 0 {
			hclchar = hcllex1(hcllex, &hcllval)
		}

		/* look through exception table */
		xi := 0
		for {
			if hclExca[xi+0] == -1 && hclExca[xi+1] == hclstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			hcln = hclExca[xi+0]
			if hcln < 0 || hcln == hclchar {
				break
			}
		}
		hcln = hclExca[xi+1]
		if hcln < 0 {
			goto ret0
		}
	}
	if hcln == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			hcllex.Error("syntax error")
			Nerrs++
			if hclDebug >= 1 {
				__yyfmt__.Printf("%s", hclStatname(hclstate))
				__yyfmt__.Printf(" saw %s\n", hclTokname(hclchar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for hclp >= 0 {
				hcln = hclPact[hclS[hclp].yys] + hclErrCode
				if hcln >= 0 && hcln < hclLast {
					hclstate = hclAct[hcln] /* simulate a shift of "error" */
					if hclChk[hclstate] == hclErrCode {
						goto hclstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if hclDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", hclS[hclp].yys)
				}
				hclp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if hclDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", hclTokname(hclchar))
			}
			if hclchar == hclEofCode {
				goto ret1
			}
			hclchar = -1
			goto hclnewstate /* try again in the same state */
		}
	}

	/* reduction by production hcln */
	if hclDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", hcln, hclStatname(hclstate))
	}

	hclnt := hcln
	hclpt := hclp
	_ = hclpt // guard against "declared and not used"

	hclp -= hclR2[hcln]
	hclVAL = hclS[hclp+1]

	/* consult goto table to find next state */
	hcln = hclR1[hcln]
	hclg := hclPgo[hcln]
	hclj := hclg + hclS[hclp].yys + 1

	if hclj >= hclLast {
		hclstate = hclAct[hclg]
	} else {
		hclstate = hclAct[hclj]
		if hclChk[hclstate] != -hcln {
			hclstate = hclAct[hclg]
		}
	}
	// dummy call; replaced with literal code
	switch hclnt {

	case 1:
		//line parse.y:37
		{
			hclResult = &Object{
				Type:  ValueTypeObject,
				Value: ObjectList(hclS[hclpt-0].objlist).Flat(),
			}
		}
	case 2:
		//line parse.y:46
		{
			hclVAL.objlist = []*Object{hclS[hclpt-0].obj}
		}
	case 3:
		//line parse.y:50
		{
			hclVAL.objlist = append(hclS[hclpt-1].objlist, hclS[hclpt-0].obj)
		}
	case 4:
		//line parse.y:56
		{
			hclVAL.obj = &Object{
				Type:  ValueTypeObject,
				Value: ObjectList(hclS[hclpt-1].objlist).Flat(),
			}
		}
	case 5:
		//line parse.y:63
		{
			hclVAL.obj = &Object{
				Type: ValueTypeObject,
			}
		}
	case 6:
		//line parse.y:71
		{
			hclVAL.obj = hclS[hclpt-0].obj
			hclVAL.obj.Key = hclS[hclpt-2].str
		}
	case 7:
		//line parse.y:76
		{
			hclVAL.obj = &Object{
				Key:   hclS[hclpt-2].str,
				Type:  ValueTypeBool,
				Value: hclS[hclpt-0].b,
			}
		}
	case 8:
		//line parse.y:84
		{
			hclVAL.obj = &Object{
				Key:   hclS[hclpt-2].str,
				Type:  ValueTypeString,
				Value: hclS[hclpt-0].str,
			}
		}
	case 9:
		//line parse.y:92
		{
			hclS[hclpt-0].obj.Key = hclS[hclpt-2].str
			hclVAL.obj = hclS[hclpt-0].obj
		}
	case 10:
		//line parse.y:97
		{
			hclVAL.obj = &Object{
				Key:   hclS[hclpt-2].str,
				Type:  ValueTypeList,
				Value: hclS[hclpt-0].objlist,
			}
		}
	case 11:
		//line parse.y:105
		{
			hclVAL.obj = hclS[hclpt-0].obj
		}
	case 12:
		//line parse.y:111
		{
			hclS[hclpt-0].obj.Key = hclS[hclpt-1].str
			hclVAL.obj = hclS[hclpt-0].obj
		}
	case 13:
		//line parse.y:116
		{
			hclVAL.obj = &Object{
				Key:   hclS[hclpt-1].str,
				Type:  ValueTypeObject,
				Value: []*Object{hclS[hclpt-0].obj},
			}
		}
	case 14:
		//line parse.y:126
		{
			hclVAL.str = hclS[hclpt-0].str
		}
	case 15:
		//line parse.y:130
		{
			hclVAL.str = hclS[hclpt-0].str
		}
	case 16:
		//line parse.y:136
		{
			hclVAL.objlist = hclS[hclpt-1].objlist
		}
	case 17:
		//line parse.y:140
		{
			hclVAL.objlist = nil
		}
	case 18:
		//line parse.y:146
		{
			hclVAL.objlist = []*Object{hclS[hclpt-0].obj}
		}
	case 19:
		//line parse.y:150
		{
			hclVAL.objlist = append(hclS[hclpt-2].objlist, hclS[hclpt-0].obj)
		}
	case 20:
		//line parse.y:156
		{
			hclVAL.obj = hclS[hclpt-0].obj
		}
	case 21:
		//line parse.y:160
		{
			hclVAL.obj = &Object{
				Type:  ValueTypeString,
				Value: hclS[hclpt-0].str,
			}
		}
	case 22:
		//line parse.y:169
		{
			hclVAL.obj = &Object{
				Type:  ValueTypeInt,
				Value: hclS[hclpt-0].num,
			}
		}
	case 23:
		//line parse.y:176
		{
			fs := fmt.Sprintf("%d.%s", hclS[hclpt-1].num, hclS[hclpt-0].str)
			f, err := strconv.ParseFloat(fs, 64)
			if err != nil {
				panic(err)
			}

			hclVAL.obj = &Object{
				Type:  ValueTypeFloat,
				Value: f,
			}
		}
	case 24:
		//line parse.y:189
		{
			fs := fmt.Sprintf("%d%s", hclS[hclpt-1].num, hclS[hclpt-0].str)
			f, err := strconv.ParseFloat(fs, 64)
			if err != nil {
				panic(err)
			}

			hclVAL.obj = &Object{
				Type:  ValueTypeFloat,
				Value: f,
			}
		}
	case 25:
		//line parse.y:204
		{
			hclVAL.num = hclS[hclpt-0].num * -1
		}
	case 26:
		//line parse.y:208
		{
			hclVAL.num = hclS[hclpt-0].num
		}
	case 27:
		//line parse.y:214
		{
			hclVAL.str = "e" + strconv.FormatInt(int64(hclS[hclpt-0].num), 10)
		}
	case 28:
		//line parse.y:218
		{
			hclVAL.str = "e-" + strconv.FormatInt(int64(hclS[hclpt-0].num), 10)
		}
	case 29:
		//line parse.y:224
		{
			hclVAL.str = strconv.FormatInt(int64(hclS[hclpt-0].num), 10)
		}
	}
	goto hclstack /* stack new state and value */
}
