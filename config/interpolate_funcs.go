package config

import (
	"bytes"
	"encoding/base64"
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
		"concat":     interpolationFuncConcat(),
		"element":    interpolationFuncElement(),
		"file":       interpolationFuncFile(),
		"format":     interpolationFuncFormat(),
		"formatlist": interpolationFuncFormatList(),
		"index":      interpolationFuncIndex(),
		"join":       interpolationFuncJoin(),
		"length":     interpolationFuncLength(),
		"replace":    interpolationFuncReplace(),
		"split":      interpolationFuncSplit(),
		"base64enc":  interpolationFuncBase64Encode(),
		"base64dec":  interpolationFuncBase64Decode(),
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
			var finalList []string

			var isDeprecated = true

			for _, arg := range args {
				argument := arg.(string)

				if len(argument) == 0 {
					continue
				}

				if IsStringList(argument) {
					isDeprecated = false
					finalList = append(finalList, StringList(argument).Slice()...)
				} else {
					finalList = append(finalList, argument)
				}

				// Deprecated concat behaviour
				b.WriteString(argument)
			}

			if isDeprecated {
				return b.String(), nil
			}

			return NewStringList(finalList).String(), nil
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
				if !IsStringList(s) {
					continue
				}

				parts := StringList(s).Slice()

				// otherwise the list is sent down to be indexed
				varargs[i-1] = parts

				// Check length
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
			return NewStringList(list).String(), nil
		},
	}
}

// interpolationFuncIndex implements the "index" function that allows one to
// find the index of a specific element in a list
func interpolationFuncIndex() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeInt,
		Callback: func(args []interface{}) (interface{}, error) {
			haystack := StringList(args[0].(string)).Slice()
			needle := args[1].(string)
			for index, element := range haystack {
				if needle == element {
					return index, nil
				}
			}
			return nil, fmt.Errorf("Could not find '%s' in '%s'", needle, haystack)
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
				parts := StringList(arg.(string)).Slice()
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
			if !IsStringList(args[0].(string)) {
				return len(args[0].(string)), nil
			}

			length := 0
			for _, arg := range args {
				length += StringList(arg.(string)).Length()
			}
			return length, nil
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
			sep := args[0].(string)
			s := args[1].(string)
			return NewStringList(strings.Split(s, sep)).String(), nil
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
			list := StringList(args[0].(string))

			index, err := strconv.Atoi(args[1].(string))
			if err != nil {
				return "", fmt.Errorf(
					"invalid number for index, got %s", args[1])
			}

			v := list.Element(index)
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

			return NewStringList(keys).String(), nil
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

			return NewStringList(vals).String(), nil
		},
	}
}

// interpolationFuncBase64Encode implements the "base64enc" function that allows
// Base64 encoding.
func interpolationFuncBase64Encode() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			return base64.StdEncoding.EncodeToString([]byte(s)), nil
		},
	}
}

// interpolationFuncBase64Decode implements the "base64dec" function that allows
// Base64 decoding.
func interpolationFuncBase64Decode() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			sDec, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", fmt.Errorf("failed to decode base64 data '%s'", s)
			}
			return string(sDec), nil
		},
	}
}
