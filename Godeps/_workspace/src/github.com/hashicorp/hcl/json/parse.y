// This is the yacc input for creating the parser for HCL JSON.

%{
package json

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/hcl"
)

%}

%union {
	num      int
	str      string
	obj      *hcl.Object
	objlist  []*hcl.Object
}

%type	<num> int
%type	<obj> number object pair value
%type	<objlist> array elements members
%type	<str> exp frac

%token  <num> NUMBER
%token  <str> COLON COMMA IDENTIFIER EQUAL NEWLINE STRING
%token  <str> LEFTBRACE RIGHTBRACE LEFTBRACKET RIGHTBRACKET
%token  <str> TRUE FALSE NULL MINUS PERIOD EPLUS EMINUS

%%

top:
	object
	{
		jsonResult = $1
	}

object:
	LEFTBRACE members RIGHTBRACE
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeObject,
			Value: hcl.ObjectList($2).Flat(),
		}
	}
|	LEFTBRACE RIGHTBRACE
	{
		$$ = &hcl.Object{Type: hcl.ValueTypeObject}
	}

members:
	pair
	{
		$$ = []*hcl.Object{$1}
	}
|	members COMMA pair
	{
		$$ = append($1, $3)
	}

pair:
	STRING COLON value
	{
		$3.Key = $1
		$$ = $3
	}

value:
	STRING
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeString,
			Value: $1,
		}
	}
|	number
	{
		$$ = $1
	}
|	object
	{
		$$ = $1
	}
|	array
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeList,
			Value: $1,
		}
	}
|	TRUE
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeBool,
			Value: true,
		}
	}
|	FALSE
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeBool,
			Value: false,
		}
	}
|	NULL
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeNil,
			Value: nil,
		}
	}

array:
	LEFTBRACKET RIGHTBRACKET
	{
		$$ = nil
	}
|	LEFTBRACKET elements RIGHTBRACKET
	{
		$$ = $2
	}

elements:
	value
	{
		$$ = []*hcl.Object{$1}
	}
|	elements COMMA value
	{
		$$ = append($1, $3)
	}

number:
	int
	{
		$$ = &hcl.Object{
			Type:  hcl.ValueTypeInt,
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

		$$ = &hcl.Object{
			Type:  hcl.ValueTypeFloat,
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

		$$ = &hcl.Object{
			Type:  hcl.ValueTypeFloat,
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
