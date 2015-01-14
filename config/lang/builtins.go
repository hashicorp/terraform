package lang

import (
	"strconv"

	"github.com/hashicorp/terraform/config/lang/ast"
)

// NOTE: All builtins are tested in engine_test.go

func registerBuiltins(scope *Scope) {
	if scope.FuncMap == nil {
		scope.FuncMap = make(map[string]Function)
	}
	scope.FuncMap["__builtin_IntToString"] = builtinIntToString()
	scope.FuncMap["__builtin_StringToInt"] = builtinStringToInt()
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

func builtinStringToInt() Function {
	return Function{
		ArgTypes:   []ast.Type{ast.TypeInt},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			v, err := strconv.ParseInt(args[0].(string), 0, 0)
			if err != nil {
				return nil, err
			}

			return int(v), nil
		},
	}
}
