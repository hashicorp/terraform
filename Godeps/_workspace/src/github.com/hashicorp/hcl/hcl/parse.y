// This is the yacc input for creating the parser for HCL.

%{
package hcl

import (
	"fmt"
	"strconv"
)

%}

%union {
	b        bool
	num      int
	str      string
	obj      *Object
	objlist  []*Object
}

%type   <num> int
%type   <objlist> list listitems objectlist
%type   <obj> block number object objectitem
%type   <obj> listitem
%type   <str> blockId exp frac

%token  <b> BOOL
%token  <num> NUMBER
%token  <str> COMMA IDENTIFIER EQUAL NEWLINE STRING MINUS
%token  <str> LEFTBRACE RIGHTBRACE LEFTBRACKET RIGHTBRACKET PERIOD
%token  <str> EPLUS EMINUS

%%

top:
	objectlist
	{
		hclResult = &Object{
			Type:  ValueTypeObject,
			Value: ObjectList($1).Flat(),
		}
	}

objectlist:
	objectitem
	{
		$$ = []*Object{$1}
	}
|	objectlist objectitem
	{
		$$ = append($1, $2)
	}

object:
	LEFTBRACE objectlist RIGHTBRACE
	{
		$$ = &Object{
			Type:  ValueTypeObject,
			Value: ObjectList($2).Flat(),
		}
	}
|	LEFTBRACE RIGHTBRACE
	{
		$$ = &Object{
			Type: ValueTypeObject,
		}
	}

objectitem:
	IDENTIFIER EQUAL number
	{
		$$ = $3
		$$.Key = $1
	}
|	IDENTIFIER EQUAL BOOL
	{
		$$ = &Object{
			Key:   $1,
			Type:  ValueTypeBool,
			Value: $3,
		}
	}
|	IDENTIFIER EQUAL STRING
	{
		$$ = &Object{
			Key:   $1,
			Type:  ValueTypeString,
			Value: $3,
		}
	}
|	IDENTIFIER EQUAL object
	{
		$3.Key = $1
		$$ = $3
	}
|	IDENTIFIER EQUAL list
	{
		$$ = &Object{
			Key:   $1,
			Type:  ValueTypeList,
			Value: $3,
		}
	}
|	block
	{
		$$ = $1
	}

block:
	blockId object
	{
		$2.Key = $1
		$$ = $2
	}
|	blockId block
	{
		$$ = &Object{
			Key:   $1,
			Type:  ValueTypeObject,
			Value: []*Object{$2},
		}
	}

blockId:
	IDENTIFIER
	{
		$$ = $1
	}
|	STRING
	{
		$$ = $1
	}

list:
	LEFTBRACKET listitems RIGHTBRACKET
	{
		$$ = $2
	}
|	LEFTBRACKET RIGHTBRACKET
	{
		$$ = nil
	}

listitems:
	listitem
	{
		$$ = []*Object{$1}
	}
|	listitems COMMA listitem
	{
		$$ = append($1, $3)
	}

listitem:
	number
	{
		$$ = $1
	}
|	STRING
	{
		$$ = &Object{
			Type:  ValueTypeString,
			Value: $1,
		}
	}

number:
	int
	{
		$$ = &Object{
			Type:  ValueTypeInt,
			Value: $1,
		}
	}
|	int frac
	{
		fs := fmt.Sprintf("%d.%s", $1, $2)
		f, err := strconv.ParseFloat(fs, 64)
		if err != nil {
			panic(err)
		}

		$$ = &Object{
			Type:  ValueTypeFloat,
			Value: f,
		}
	}
|   int exp
    {
		fs := fmt.Sprintf("%d%s", $1, $2)
		f, err := strconv.ParseFloat(fs, 64)
		if err != nil {
			panic(err)
		}

		$$ = &Object{
			Type:  ValueTypeFloat,
			Value: f,
		}
    }

int:
	MINUS int
	{
		$$ = $2 * -1
	}
|	NUMBER
	{
		$$ = $1
	}

exp:
    EPLUS NUMBER
    {
        $$ = "e" + strconv.FormatInt(int64($2), 10)
    }
|   EMINUS NUMBER
    {
        $$ = "e-" + strconv.FormatInt(int64($2), 10)
    }

frac:
	PERIOD NUMBER
	{
		$$ = strconv.FormatInt(int64($2), 10)
	}

%%
