// This is the yacc input for creating the parser for interpolation
// expressions in Go. To build it, just run `go generate` on this
// package, as the lexer has the go generate pragma within it.

%{
package lang

import (
    "github.com/hashicorp/terraform/config/lang/ast"
)

%}

%union {
    node     ast.Node
    nodeList []ast.Node
    str      string
    token    *parserToken
}

%token  <str> PROGRAM_BRACKET_LEFT PROGRAM_BRACKET_RIGHT
%token  <str> PROGRAM_STRING_START PROGRAM_STRING_END
%token  <str> PAREN_LEFT PAREN_RIGHT COMMA

%token <token> IDENTIFIER INTEGER FLOAT STRING

%type <node> expr interpolation literal literalModeTop literalModeValue
%type <nodeList> args

%%

top:
	literalModeTop
	{
        parserResult = $1
	}

literalModeTop:
    literalModeValue
    {
        $$ = $1
    }
|   literalModeTop literalModeValue
    {
        var result []ast.Node
        if c, ok := $1.(*ast.Concat); ok {
            result = append(c.Exprs, $2)
        } else {
            result = []ast.Node{$1, $2}
        }

        $$ = &ast.Concat{
            Exprs: result,
            Posx:  result[0].Pos(),
        }
    }

literalModeValue:
	literal
	{
        $$ = $1
	}
|   interpolation
    {
        $$ = $1
    }

interpolation:
    PROGRAM_BRACKET_LEFT expr PROGRAM_BRACKET_RIGHT
    {
        $$ = $2
    }

expr:
    literalModeTop
    {
        $$ = $1
    }
|   INTEGER
    {
        $$ = &ast.LiteralNode{
            Value: $1.Value.(int),
            Type:  ast.TypeInt,
            Posx:  $1.Pos,
        }
    }
|   FLOAT
    {
        $$ = &ast.LiteralNode{
            Value: $1.Value.(float64),
            Type:  ast.TypeFloat,
            Posx:  $1.Pos,
        }
    }
|   IDENTIFIER
    {
        $$ = &ast.VariableAccess{Name: $1.Value.(string), Posx: $1.Pos}
    }
|   IDENTIFIER PAREN_LEFT args PAREN_RIGHT
    {
        $$ = &ast.Call{Func: $1.Value.(string), Args: $3, Posx: $1.Pos}
    }

args:
	{
		$$ = nil
	}
|	args COMMA expr
	{
		$$ = append($1, $3)
	}
|	expr
	{
		$$ = append($$, $1)
	}

literal:
    STRING
    {
        $$ = &ast.LiteralNode{
            Value: $1.Value.(string),
            Type:  ast.TypeString,
            Posx:  $1.Pos,
        }
    }

%%
