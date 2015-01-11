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
}

%token  <str> STRING IDENTIFIER PROGRAM_BRACKET_LEFT PROGRAM_BRACKET_RIGHT
%token  <str> PAREN_LEFT PAREN_RIGHT COMMA

%type <node> expr interpolation literal
%type <nodeList> args

%%

top:
	literal
	{
        parserResult = $1
	}
|   interpolation
    {
        parserResult = $1
    }
|   literal interpolation
    {
        parserResult = &ast.Concat{
            Exprs: []ast.Node{$1, $2},
        }
    }

interpolation:
    PROGRAM_BRACKET_LEFT expr PROGRAM_BRACKET_RIGHT
    {
        $$ = $2
    }

expr:
    literal
    {
        $$ = $1
    }
|   IDENTIFIER
    {
        $$ = &ast.VariableAccess{Name: $1}
    }
|   IDENTIFIER PAREN_LEFT args PAREN_RIGHT
    {
        $$ = &ast.Call{Func: $1, Args: $3}
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
        $$ = &ast.LiteralNode{Value: $1, Type: ast.TypeString}
    }

%%
