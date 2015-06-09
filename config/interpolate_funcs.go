package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config/lang/ast"
	"github.com/mitchellh/go-homedir"
)

// Funcs is the mapping of built-in functions for configuration.
var Funcs map[string]ast.Function

func init() {
	Funcs = map[string]ast.Function{
		"file":       interpolationFuncFile(),
		"format":     interpolationFuncFormat(),
		"formatlist": interpolationFuncFormatList(),
		"join":       interpolationFuncJoin(),
		"element":    interpolationFuncElement(),
		"replace":    interpolationFuncReplace(),
		"split":      interpolationFuncSplit(),
		"length":     interpolationFuncLength(),

		// Concat is a little useless now since we supported embeddded
		// interpolations but we keep it around for backwards compat reasons.
		"concat": interpolationFuncConcat(),
	}
}

// interpolationFuncConcat implements the "concat" function that
// concatenates multiple strings. This isn't actually necessary anymore
// since our language supports string concat natively, but for backwards
// compat we do this.
func interpolationFuncConcat() ast.Function {
	return ast.Function{
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
func interpolationFuncFile() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			path, err := homedir.Expand(args[0].(string))
			if err != nil {
				return "", err
			}
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return "", err
			}

			return string(data), nil
		},
	}
}

// interpolationFuncFormat implements the "format" function that does
// string formatting.
func interpolationFuncFormat() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeString},
		Variadic:     true,
		VariadicType: ast.TypeAny,
		ReturnType:   ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			format := args[0].(string)
			return fmt.Sprintf(format, args[1:]...), nil
		},
	}
}

// interpolationFuncFormatList implements the "formatlist" function that does
// string formatting on lists.
func interpolationFuncFormatList() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeString},
		Variadic:     true,
		VariadicType: ast.TypeAny,
		ReturnType:   ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			// Make a copy of the variadic part of args
			// to avoid modifying the original.
			varargs := make([]interface{}, len(args)-1)
			copy(varargs, args[1:])

			// Convert arguments that are lists into slices.
			// Confirm along the way that all lists have the same length (n).
			var n int
			for i := 1; i < len(args); i++ {
				s, ok := args[i].(string)
				if !ok {
					continue
				}
				parts := strings.Split(s, InterpSplitDelim)
				if len(parts) == 1 {
					continue
				}
				varargs[i-1] = parts
				if n == 0 {
					// first list we've seen
					n = len(parts)
					continue
				}
				if n != len(parts) {
					return nil, fmt.Errorf("format: mismatched list lengths: %d != %d", n, len(parts))
				}
			}

			if n == 0 {
				return nil, errors.New("no lists in arguments to formatlist")
			}

			// Do the formatting.
			format := args[0].(string)

			// Generate a list of formatted strings.
			list := make([]string, n)
			fmtargs := make([]interface{}, len(varargs))
			for i := 0; i < n; i++ {
				for j, arg := range varargs {
					switch arg := arg.(type) {
					default:
						fmtargs[j] = arg
					case []string:
						fmtargs[j] = arg[i]
					}
				}
				list[i] = fmt.Sprintf(format, fmtargs...)
			}
			return strings.Join(list, InterpSplitDelim), nil
		},
	}
}

// interpolationFuncJoin implements the "join" function that allows
// multi-variable values to be joined by some character.
func interpolationFuncJoin() ast.Function {
	return ast.Function{
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

// interpolationFuncReplace implements the "replace" function that does
// string replacement.
func interpolationFuncReplace() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			search := args[1].(string)
			replace := args[2].(string)

			// We search/replace using a regexp if the string is surrounded
			// in forward slashes.
			if len(search) > 1 && search[0] == '/' && search[len(search)-1] == '/' {
				re, err := regexp.Compile(search[1 : len(search)-1])
				if err != nil {
					return nil, err
				}

				return re.ReplaceAllString(s, replace), nil
			}

			return strings.Replace(s, search, replace, -1), nil
		},
	}
}

func interpolationFuncLength() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeInt,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			if !strings.Contains(args[0].(string), InterpSplitDelim) {
				return len(args[0].(string)), nil
			}

			var list []string
			for _, arg := range args {
				parts := strings.Split(arg.(string), InterpSplitDelim)
				for _, part := range parts {
					list = append(list, part)
				}
			}
			return len(list), nil
		},
	}
}

// interpolationFuncSplit implements the "split" function that allows
// strings to split into multi-variable values
func interpolationFuncSplit() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return strings.Replace(args[1].(string), args[0].(string), InterpSplitDelim, -1), nil
		},
	}
}

// interpolationFuncLookup implements the "lookup" function that allows
// dynamic lookups of map types within a Terraform configuration.
func interpolationFuncLookup(vs map[string]ast.Variable) ast.Function {
	return ast.Function{
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
			if v.Type != ast.TypeString {
				return "", fmt.Errorf(
					"lookup in '%s' for '%s' has bad type %s",
					args[0].(string), args[1].(string), v.Type)
			}

			return v.Value.(string), nil
		},
	}
}

// interpolationFuncElement implements the "element" function that allows
// a specific index to be looked up in a multi-variable value. Note that this will
// wrap if the index is larger than the number of elements in the multi-variable value.
func interpolationFuncElement() ast.Function {
	return ast.Function{
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

// interpolationFuncKeys implements the "keys" function that yields a list of
// keys of map types within a Terraform configuration.
func interpolationFuncKeys(vs map[string]ast.Variable) ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			// Prefix must include ending dot to be a map
			prefix := fmt.Sprintf("var.%s.", args[0].(string))
			keys := make([]string, 0, len(vs))
			for k, _ := range vs {
				if !strings.HasPrefix(k, prefix) {
					continue
				}
				keys = append(keys, k[len(prefix):])
			}

			if len(keys) <= 0 {
				return "", fmt.Errorf(
					"failed to find map '%s'",
					args[0].(string))
			}

			sort.Strings(keys)

			return strings.Join(keys, InterpSplitDelim), nil
		},
	}
}

// interpolationFuncValues implements the "values" function that yields a list of
// keys of map types within a Terraform configuration.
func interpolationFuncValues(vs map[string]ast.Variable) ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			// Prefix must include ending dot to be a map
			prefix := fmt.Sprintf("var.%s.", args[0].(string))
			keys := make([]string, 0, len(vs))
			for k, _ := range vs {
				if !strings.HasPrefix(k, prefix) {
					continue
				}
				keys = append(keys, k)
			}

			if len(keys) <= 0 {
				return "", fmt.Errorf(
					"failed to find map '%s'",
					args[0].(string))
			}

			sort.Strings(keys)

			vals := make([]string, 0, len(keys))

			for _, k := range keys {
				v := vs[k]
				if v.Type != ast.TypeString {
					return "", fmt.Errorf("values(): %q has bad type %s", k, v.Type)
				}
				vals = append(vals, vs[k].Value.(string))
			}

			return strings.Join(vals, InterpSplitDelim), nil
		},
	}
}
