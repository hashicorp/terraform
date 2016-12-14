package hil

import (
	"errors"
	"strconv"

	"github.com/hashicorp/hil/ast"
)

// NOTE: All builtins are tested in engine_test.go

func registerBuiltins(scope *ast.BasicScope) *ast.BasicScope {
	if scope == nil {
		scope = new(ast.BasicScope)
	}
	if scope.FuncMap == nil {
		scope.FuncMap = make(map[string]ast.Function)
	}

	// Implicit conversions
	scope.FuncMap["__builtin_BoolToString"] = builtinBoolToString()
	scope.FuncMap["__builtin_FloatToInt"] = builtinFloatToInt()
	scope.FuncMap["__builtin_FloatToString"] = builtinFloatToString()
	scope.FuncMap["__builtin_IntToFloat"] = builtinIntToFloat()
	scope.FuncMap["__builtin_IntToString"] = builtinIntToString()
	scope.FuncMap["__builtin_StringToInt"] = builtinStringToInt()
	scope.FuncMap["__builtin_StringToFloat"] = builtinStringToFloat()
	scope.FuncMap["__builtin_StringToBool"] = builtinStringToBool()

	// Math operations
	scope.FuncMap["__builtin_IntMath"] = builtinIntMath()
	scope.FuncMap["__builtin_FloatMath"] = builtinFloatMath()
	scope.FuncMap["__builtin_BoolCompare"] = builtinBoolCompare()
	scope.FuncMap["__builtin_FloatCompare"] = builtinFloatCompare()
	scope.FuncMap["__builtin_IntCompare"] = builtinIntCompare()
	scope.FuncMap["__builtin_StringCompare"] = builtinStringCompare()
	scope.FuncMap["__builtin_Logical"] = builtinLogical()
	return scope
}

func builtinFloatMath() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeInt},
		Variadic:     true,
		VariadicType: ast.TypeFloat,
		ReturnType:   ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			result := args[1].(float64)
			for _, raw := range args[2:] {
				arg := raw.(float64)
				switch op {
				case ast.ArithmeticOpAdd:
					result += arg
				case ast.ArithmeticOpSub:
					result -= arg
				case ast.ArithmeticOpMul:
					result *= arg
				case ast.ArithmeticOpDiv:
					result /= arg
				}
			}

			return result, nil
		},
	}
}

func builtinIntMath() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeInt},
		Variadic:     true,
		VariadicType: ast.TypeInt,
		ReturnType:   ast.TypeInt,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			result := args[1].(int)
			for _, raw := range args[2:] {
				arg := raw.(int)
				switch op {
				case ast.ArithmeticOpAdd:
					result += arg
				case ast.ArithmeticOpSub:
					result -= arg
				case ast.ArithmeticOpMul:
					result *= arg
				case ast.ArithmeticOpDiv:
					if arg == 0 {
						return nil, errors.New("divide by zero")
					}

					result /= arg
				case ast.ArithmeticOpMod:
					if arg == 0 {
						return nil, errors.New("divide by zero")
					}

					result = result % arg
				}
			}

			return result, nil
		},
	}
}

func builtinBoolCompare() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt, ast.TypeBool, ast.TypeBool},
		Variadic:   false,
		ReturnType: ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			lhs := args[1].(bool)
			rhs := args[2].(bool)

			switch op {
			case ast.ArithmeticOpEqual:
				return lhs == rhs, nil
			case ast.ArithmeticOpNotEqual:
				return lhs != rhs, nil
			default:
				return nil, errors.New("invalid comparison operation")
			}
		},
	}
}

func builtinFloatCompare() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt, ast.TypeFloat, ast.TypeFloat},
		Variadic:   false,
		ReturnType: ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			lhs := args[1].(float64)
			rhs := args[2].(float64)

			switch op {
			case ast.ArithmeticOpEqual:
				return lhs == rhs, nil
			case ast.ArithmeticOpNotEqual:
				return lhs != rhs, nil
			case ast.ArithmeticOpLessThan:
				return lhs < rhs, nil
			case ast.ArithmeticOpLessThanOrEqual:
				return lhs <= rhs, nil
			case ast.ArithmeticOpGreaterThan:
				return lhs > rhs, nil
			case ast.ArithmeticOpGreaterThanOrEqual:
				return lhs >= rhs, nil
			default:
				return nil, errors.New("invalid comparison operation")
			}
		},
	}
}

func builtinIntCompare() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt, ast.TypeInt, ast.TypeInt},
		Variadic:   false,
		ReturnType: ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			lhs := args[1].(int)
			rhs := args[2].(int)

			switch op {
			case ast.ArithmeticOpEqual:
				return lhs == rhs, nil
			case ast.ArithmeticOpNotEqual:
				return lhs != rhs, nil
			case ast.ArithmeticOpLessThan:
				return lhs < rhs, nil
			case ast.ArithmeticOpLessThanOrEqual:
				return lhs <= rhs, nil
			case ast.ArithmeticOpGreaterThan:
				return lhs > rhs, nil
			case ast.ArithmeticOpGreaterThanOrEqual:
				return lhs >= rhs, nil
			default:
				return nil, errors.New("invalid comparison operation")
			}
		},
	}
}

func builtinStringCompare() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt, ast.TypeString, ast.TypeString},
		Variadic:   false,
		ReturnType: ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			lhs := args[1].(string)
			rhs := args[2].(string)

			switch op {
			case ast.ArithmeticOpEqual:
				return lhs == rhs, nil
			case ast.ArithmeticOpNotEqual:
				return lhs != rhs, nil
			default:
				return nil, errors.New("invalid comparison operation")
			}
		},
	}
}

func builtinLogical() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeInt},
		Variadic:     true,
		VariadicType: ast.TypeBool,
		ReturnType:   ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			op := args[0].(ast.ArithmeticOp)
			result := args[1].(bool)
			for _, raw := range args[2:] {
				arg := raw.(bool)
				switch op {
				case ast.ArithmeticOpLogicalOr:
					result = result || arg
				case ast.ArithmeticOpLogicalAnd:
					result = result && arg
				default:
					return nil, errors.New("invalid logical operator")
				}
			}

			return result, nil
		},
	}
}

func builtinFloatToInt() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat},
		ReturnType: ast.TypeInt,
		Callback: func(args []interface{}) (interface{}, error) {
			return int(args[0].(float64)), nil
		},
	}
}

func builtinFloatToString() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return strconv.FormatFloat(
				args[0].(float64), 'g', -1, 64), nil
		},
	}
}

func builtinIntToFloat() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt},
		ReturnType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			return float64(args[0].(int)), nil
		},
	}
}

func builtinIntToString() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return strconv.FormatInt(int64(args[0].(int)), 10), nil
		},
	}
}

func builtinStringToInt() ast.Function {
	return ast.Function{
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

func builtinStringToFloat() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			v, err := strconv.ParseFloat(args[0].(string), 64)
			if err != nil {
				return nil, err
			}

			return v, nil
		},
	}
}

func builtinBoolToString() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeBool},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return strconv.FormatBool(args[0].(bool)), nil
		},
	}
}

func builtinStringToBool() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			v, err := strconv.ParseBool(args[0].(string))
			if err != nil {
				return nil, err
			}

			return v, nil
		},
	}
}
