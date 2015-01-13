package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config/lang"
	"github.com/hashicorp/terraform/config/lang/ast"
)

// Funcs is the mapping of built-in functions for configuration.
var Funcs map[string]lang.Function

func init() {
	Funcs = map[string]lang.Function{
		"concat":  interpolationFuncConcat(),
		"file":    interpolationFuncFile(),
		"join":    interpolationFuncJoin(),
		"element": interpolationFuncElement(),
	}
}

// interpolationFuncConcat implements the "concat" function that
// concatenates multiple strings. This isn't actually necessary anymore
// since our language supports string concat natively, but for backwards
// compat we do this.
func interpolationFuncConcat() lang.Function {
	return lang.Function{
		ArgTypes:     []ast.Type{ast.TypeString},
		ReturnType:   ast.TypeString,
		Variadic:     true,
		VariadicType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			var b bytes.Buffer
			for _, v := range args {
				b.WriteString(v.(string))
			}

			return b.String(), nil
		},
	}
}

// interpolationFuncFile implements the "file" function that allows
// loading contents from a file.
func interpolationFuncFile() lang.Function {
	return lang.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			data, err := ioutil.ReadFile(args[0].(string))
			if err != nil {
				return "", err
			}

			return string(data), nil
		},
	}
}

// interpolationFuncJoin implements the "join" function that allows
// multi-variable values to be joined by some character.
func interpolationFuncJoin() lang.Function {
	return lang.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			var list []string
			for _, arg := range args[1:] {
				parts := strings.Split(arg.(string), InterpSplitDelim)
				list = append(list, parts...)
			}

			return strings.Join(list, args[0].(string)), nil
		},
	}
}

// interpolationFuncLookup implements the "lookup" function that allows
// dynamic lookups of map types within a Terraform configuration.
func interpolationFuncLookup(vs map[string]string) lang.Function {
	return lang.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			k := fmt.Sprintf("var.%s.%s", args[0].(string), args[1].(string))
			v, ok := vs[k]
			if !ok {
				return "", fmt.Errorf(
					"lookup in '%s' failed to find '%s'",
					args[0].(string), args[1].(string))
			}

			return v, nil
		},
	}
}

// interpolationFuncElement implements the "element" function that allows
// a specific index to be looked up in a multi-variable value. Note that this will
// wrap if the index is larger than the number of elements in the multi-variable value.
func interpolationFuncElement() lang.Function {
	return lang.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			list := strings.Split(args[0].(string), InterpSplitDelim)

			index, err := strconv.Atoi(args[1].(string))
			if err != nil {
				return "", fmt.Errorf(
					"invalid number for index, got %s", args[1])
			}

			v := list[index%len(list)]
			return v, nil
		},
	}
}
