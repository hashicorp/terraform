package lang

import (
	"strconv"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func registerBuiltins(scope *Scope) {
	if scope.FuncMap == nil {
		scope.FuncMap = make(map[string]Function)
	}
	scope.FuncMap["__builtin_IntToString"] = builtinIntToString()
}

func builtinIntToString() Function {
	return Function{
		ArgTypes:   []ast.Type{ast.TypeInt},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return strconv.FormatInt(int64(args[0].(int)), 10), nil
		},
	}
}
