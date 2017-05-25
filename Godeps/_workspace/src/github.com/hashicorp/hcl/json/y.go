//line parse.y:3
package json

import __yyfmt__ "fmt"

//line parse.y:5
import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/hcl"
)

//line parse.y:15
type jsonSymType struct {
	yys     int
	num     int
	str     string
	obj     *hcl.Object
	objlist []*hcl.Object
}

const NUMBER = 57346
const COLON = 57347
const COMMA = 57348
const IDENTIFIER = 57349
const EQUAL = 57350
const NEWLINE = 57351
const STRING = 57352
const LEFTBRACE = 57353
const RIGHTBRACE = 57354
const LEFTBRACKET = 57355
const RIGHTBRACKET = 57356
const TRUE = 57357
const FALSE = 57358
const NULL = 57359
const MINUS = 57360
const PERIOD = 57361
const EPLUS = 57362
const EMINUS = 57363

var jsonToknames = []string{
	"NUMBER",
	"COLON",
	"COMMA",
	"IDENTIFIER",
	"EQUAL",
	"NEWLINE",
	"STRING",
	"LEFTBRACE",
	"RIGHTBRACE",
	"LEFTBRACKET",
	"RIGHTBRACKET",
	"TRUE",
	"FALSE",
	"NULL",
	"MINUS",
	"PERIOD",
	"EPLUS",
	"EMINUS",
}
var jsonStatenames = []string{}

const jsonEofCode = 1
const jsonErrCode = 2
const jsonMaxDepth = 200

//line parse.y:196

//line yacctab:1
var jsonExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const jsonNprod = 26
const jsonPrivate = 57344

var jsonTokenNames []string
var jsonStates []string

const jsonLast = 50

var jsonAct = []int{

	12, 23, 26, 27, 28, 23, 3, 13, 3, 20,
	21, 29, 17, 18, 19, 22, 7, 23, 5, 22,
	7, 9, 31, 13, 3, 37, 21, 8, 17, 18,
	19, 22, 32, 36, 6, 10, 35, 34, 38, 33,
	15, 2, 1, 24, 11, 25, 4, 30, 16, 14,
}
var jsonPact = []int{

	-5, -1000, -1000, 6, 15, -1000, -1000, 30, -1000, 10,
	13, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-17, -3, 1, -1000, -1000, -1000, 35, 33, 32, -1000,
	19, -1000, -1000, -1000, -1000, -1000, -1000, 13, -1000,
}
var jsonPgo = []int{

	0, 9, 49, 40, 34, 0, 48, 47, 46, 45,
	43, 42,
}
var jsonR1 = []int{

	0, 11, 3, 3, 8, 8, 4, 5, 5, 5,
	5, 5, 5, 5, 6, 6, 7, 7, 2, 2,
	2, 1, 1, 9, 9, 10,
}
var jsonR2 = []int{

	0, 1, 3, 2, 1, 3, 3, 1, 1, 1,
	1, 1, 1, 1, 2, 3, 1, 3, 1, 2,
	2, 2, 1, 2, 2, 2,
}
var jsonChk = []int{

	-1000, -11, -3, 11, -8, 12, -4, 10, 12, 6,
	5, -4, -5, 10, -2, -3, -6, 15, 16, 17,
	-1, 13, 18, 4, -10, -9, 19, 20, 21, 14,
	-7, -5, -1, 4, 4, 4, 14, 6, -5,
}
var jsonDef = []int{

	0, -2, 1, 0, 0, 3, 4, 0, 2, 0,
	0, 5, 6, 7, 8, 9, 10, 11, 12, 13,
	18, 0, 0, 22, 19, 20, 0, 0, 0, 14,
	0, 16, 21, 25, 23, 24, 15, 0, 17,
}
var jsonTok1 = []int{

	1,
}
var jsonTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
}
var jsonTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var jsonDebug = 0

type jsonLexer interface {
	Lex(lval *jsonSymType) int
	Error(s string)
}

const jsonFlag = -1000

func jsonTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(jsonToknames) {
		if jsonToknames[c-4] != "" {
			return jsonToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func jsonStatname(s int) string {
	if s >= 0 && s < len(jsonStatenames) {
		if jsonStatenames[s] != "" {
			return jsonStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func jsonlex1(lex jsonLexer, lval *jsonSymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = jsonTok1[0]
		goto out
	}
	if char < len(jsonTok1) {
		c = jsonTok1[char]
		goto out
	}
	if char >= jsonPrivate {
		if char < jsonPrivate+len(jsonTok2) {
			c = jsonTok2[char-jsonPrivate]
			goto out
		}
	}
	for i := 0; i < len(jsonTok3); i += 2 {
		c = jsonTok3[i+0]
		if c == char {
			c = jsonTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = jsonTok2[1] /* unknown char */
	}
	if jsonDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", jsonTokname(c), uint(char))
	}
	return c
}

func jsonParse(jsonlex jsonLexer) int {
	var jsonn int
	var jsonlval jsonSymType
	var jsonVAL jsonSymType
	jsonS := make([]jsonSymType, jsonMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	jsonstate := 0
	jsonchar := -1
	jsonp := -1
	goto jsonstack

ret0:
	return 0

ret1:
	return 1

jsonstack:
	/* put a state and value onto the stack */
	if jsonDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", jsonTokname(jsonchar), jsonStatname(jsonstate))
	}

	jsonp++
	if jsonp >= len(jsonS) {
		nyys := make([]jsonSymType, len(jsonS)*2)
		copy(nyys, jsonS)
		jsonS = nyys
	}
	jsonS[jsonp] = jsonVAL
	jsonS[jsonp].yys = jsonstate

jsonnewstate:
	jsonn = jsonPact[jsonstate]
	if jsonn <= jsonFlag {
		goto jsondefault /* simple state */
	}
	if jsonchar < 0 {
		jsonchar = jsonlex1(jsonlex, &jsonlval)
	}
	jsonn += jsonchar
	if jsonn < 0 || jsonn >= jsonLast {
		goto jsondefault
	}
	jsonn = jsonAct[jsonn]
	if jsonChk[jsonn] == jsonchar { /* valid shift */
		jsonchar = -1
		jsonVAL = jsonlval
		jsonstate = jsonn
		if Errflag > 0 {
			Errflag--
		}
		goto jsonstack
	}

jsondefault:
	/* default state action */
	jsonn = jsonDef[jsonstate]
	if jsonn == -2 {
		if jsonchar < 0 {
			jsonchar = jsonlex1(jsonlex, &jsonlval)
		}

		/* look through exception table */
		xi := 0
		for {
			if jsonExca[xi+0] == -1 && jsonExca[xi+1] == jsonstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			jsonn = jsonExca[xi+0]
			if jsonn < 0 || jsonn == jsonchar {
				break
			}
		}
		jsonn = jsonExca[xi+1]
		if jsonn < 0 {
			goto ret0
		}
	}
	if jsonn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			jsonlex.Error("syntax error")
			Nerrs++
			if jsonDebug >= 1 {
				__yyfmt__.Printf("%s", jsonStatname(jsonstate))
				__yyfmt__.Printf(" saw %s\n", jsonTokname(jsonchar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for jsonp >= 0 {
				jsonn = jsonPact[jsonS[jsonp].yys] + jsonErrCode
				if jsonn >= 0 && jsonn < jsonLast {
					jsonstate = jsonAct[jsonn] /* simulate a shift of "error" */
					if jsonChk[jsonstate] == jsonErrCode {
						goto jsonstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if jsonDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", jsonS[jsonp].yys)
				}
				jsonp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if jsonDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", jsonTokname(jsonchar))
			}
			if jsonchar == jsonEofCode {
				goto ret1
			}
			jsonchar = -1
			goto jsonnewstate /* try again in the same state */
		}
	}

	/* reduction by production jsonn */
	if jsonDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", jsonn, jsonStatname(jsonstate))
	}

	jsonnt := jsonn
	jsonpt := jsonp
	_ = jsonpt // guard against "declared and not used"

	jsonp -= jsonR2[jsonn]
	jsonVAL = jsonS[jsonp+1]

	/* consult goto table to find next state */
	jsonn = jsonR1[jsonn]
	jsong := jsonPgo[jsonn]
	jsonj := jsong + jsonS[jsonp].yys + 1

	if jsonj >= jsonLast {
		jsonstate = jsonAct[jsong]
	} else {
		jsonstate = jsonAct[jsonj]
		if jsonChk[jsonstate] != -jsonn {
			jsonstate = jsonAct[jsong]
		}
	}
	// dummy call; replaced with literal code
	switch jsonnt {

	case 1:
		//line parse.y:36
		{
			jsonResult = jsonS[jsonpt-0].obj
		}
	case 2:
		//line parse.y:42
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeObject,
				Value: hcl.ObjectList(jsonS[jsonpt-1].objlist).Flat(),
			}
		}
	case 3:
		//line parse.y:49
		{
			jsonVAL.obj = &hcl.Object{Type: hcl.ValueTypeObject}
		}
	case 4:
		//line parse.y:55
		{
			jsonVAL.objlist = []*hcl.Object{jsonS[jsonpt-0].obj}
		}
	case 5:
		//line parse.y:59
		{
			jsonVAL.objlist = append(jsonS[jsonpt-2].objlist, jsonS[jsonpt-0].obj)
		}
	case 6:
		//line parse.y:65
		{
			jsonS[jsonpt-0].obj.Key = jsonS[jsonpt-2].str
			jsonVAL.obj = jsonS[jsonpt-0].obj
		}
	case 7:
		//line parse.y:72
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeString,
				Value: jsonS[jsonpt-0].str,
			}
		}
	case 8:
		//line parse.y:79
		{
			jsonVAL.obj = jsonS[jsonpt-0].obj
		}
	case 9:
		//line parse.y:83
		{
			jsonVAL.obj = jsonS[jsonpt-0].obj
		}
	case 10:
		//line parse.y:87
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeList,
				Value: jsonS[jsonpt-0].objlist,
			}
		}
	case 11:
		//line parse.y:94
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeBool,
				Value: true,
			}
		}
	case 12:
		//line parse.y:101
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeBool,
				Value: false,
			}
		}
	case 13:
		//line parse.y:108
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeNil,
				Value: nil,
			}
		}
	case 14:
		//line parse.y:117
		{
			jsonVAL.objlist = nil
		}
	case 15:
		//line parse.y:121
		{
			jsonVAL.objlist = jsonS[jsonpt-1].objlist
		}
	case 16:
		//line parse.y:127
		{
			jsonVAL.objlist = []*hcl.Object{jsonS[jsonpt-0].obj}
		}
	case 17:
		//line parse.y:131
		{
			jsonVAL.objlist = append(jsonS[jsonpt-2].objlist, jsonS[jsonpt-0].obj)
		}
	case 18:
		//line parse.y:137
		{
			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeInt,
				Value: jsonS[jsonpt-0].num,
			}
		}
	case 19:
		//line parse.y:144
		{
			fs := fmt.Sprintf("%d.%s", jsonS[jsonpt-1].num, jsonS[jsonpt-0].str)
			f, err := strconv.ParseFloat(fs, 64)
			if err != nil {
				panic(err)
			}

			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeFloat,
				Value: f,
			}
		}
	case 20:
		//line parse.y:157
		{
			fs := fmt.Sprintf("%d%s", jsonS[jsonpt-1].num, jsonS[jsonpt-0].str)
			f, err := strconv.ParseFloat(fs, 64)
			if err != nil {
				panic(err)
			}

			jsonVAL.obj = &hcl.Object{
				Type:  hcl.ValueTypeFloat,
				Value: f,
			}
		}
	case 21:
		//line parse.y:172
		{
			jsonVAL.num = jsonS[jsonpt-0].num * -1
		}
	case 22:
		//line parse.y:176
		{
			jsonVAL.num = jsonS[jsonpt-0].num
		}
	case 23:
		//line parse.y:182
		{
			jsonVAL.str = "e" + strconv.FormatInt(int64(jsonS[jsonpt-0].num), 10)
		}
	case 24:
		//line parse.y:186
		{
			jsonVAL.str = "e-" + strconv.FormatInt(int64(jsonS[jsonpt-0].num), 10)
		}
	case 25:
		//line parse.y:192
		{
			jsonVAL.str = strconv.FormatInt(int64(jsonS[jsonpt-0].num), 10)
		}
	}
	goto jsonstack /* stack new state and value */
}
