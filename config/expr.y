// This is the yacc input for creating the parser for interpolation
// expressions in Go.

// To build it:
//
// go tool yacc -p "expr" expr.y (produces y.go)
//

%{
package config

import (
	"fmt"
)

%}

%union {
	expr Interpolation
    str string
	variable InterpolatedVariable
	args []Interpolation
}

%type	<args> args
%type   <expr> expr
%type   <str> string
%type   <variable> variable

%token  <str> STRING IDENTIFIER
%token	<str> COMMA LEFTPAREN RIGHTPAREN

%%

top:
	expr
	{
		exprResult = $1
	}

expr:
	string
	{
		$$ = &LiteralInterpolation{Literal: $1}
	}
|	variable
	{
		$$ = &VariableInterpolation{Variable: $1}
	}
|	IDENTIFIER LEFTPAREN args RIGHTPAREN
	{
		f, ok := Funcs[$1]
		if !ok {
			exprErrors = append(exprErrors, fmt.Errorf(
				"Unknown function: %s", $1))
		}

		$$ = &FunctionInterpolation{Func: f, Args: $3}
	}

args:
	{
		$$ = nil
	}
|	expr COMMA expr
	{
		$$ = append($$, $1, $3)
	}
|	expr
	{
		$$ = append($$, $1)
	}

string:
	STRING
	{
		$$ = $1
	}

variable:
	IDENTIFIER
	{
		var err error
		$$, err = NewInterpolatedVariable($1)
		if err != nil {
			panic(err)
		}
	}

%%
