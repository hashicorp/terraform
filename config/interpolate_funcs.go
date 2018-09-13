package config

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/bcrypt"
)

// stringSliceToVariableValue converts a string slice into the value
// required to be returned from interpolation functions which return
// TypeList.
func stringSliceToVariableValue(values []string) []ast.Variable {
	output := make([]ast.Variable, len(values))
	for index, value := range values {
		output[index] = ast.Variable{
			Type:  ast.TypeString,
			Value: value,
		}
	}
	return output
}

// listVariableSliceToVariableValue converts a list of lists into the value
// required to be returned from interpolation functions which return TypeList.
func listVariableSliceToVariableValue(values [][]ast.Variable) []ast.Variable {
	output := make([]ast.Variable, len(values))

	for index, value := range values {
		output[index] = ast.Variable{
			Type:  ast.TypeList,
			Value: value,
		}
	}
	return output
}

func listVariableValueToStringSlice(values []ast.Variable) ([]string, error) {
	output := make([]string, len(values))
	for index, value := range values {
		if value.Type != ast.TypeString {
			return []string{}, fmt.Errorf("list has non-string element (%T)", value.Type.String())
		}
		output[index] = value.Value.(string)
	}
	return output, nil
}

// Funcs is the mapping of built-in functions for configuration.
func Funcs() map[string]ast.Function {
	return map[string]ast.Function{
		"abs":          interpolationFuncAbs(),
		"basename":     interpolationFuncBasename(),
		"base64decode": interpolationFuncBase64Decode(),
		"base64encode": interpolationFuncBase64Encode(),
		"base64gzip":   interpolationFuncBase64Gzip(),
		"base64sha256": interpolationFuncBase64Sha256(),
		"base64sha512": interpolationFuncBase64Sha512(),
		"bcrypt":       interpolationFuncBcrypt(),
		"ceil":         interpolationFuncCeil(),
		"chomp":        interpolationFuncChomp(),
		"cidrhost":     interpolationFuncCidrHost(),
		"cidrnetmask":  interpolationFuncCidrNetmask(),
		"cidrsubnet":   interpolationFuncCidrSubnet(),
		"coalesce":     interpolationFuncCoalesce(),
		"coalescelist": interpolationFuncCoalesceList(),
		"compact":      interpolationFuncCompact(),
		"concat":       interpolationFuncConcat(),
		"contains":     interpolationFuncContains(),
		"dirname":      interpolationFuncDirname(),
		"distinct":     interpolationFuncDistinct(),
		"element":      interpolationFuncElement(),
		"chunklist":    interpolationFuncChunklist(),
		"file":         interpolationFuncFile(),
		"matchkeys":    interpolationFuncMatchKeys(),
		"flatten":      interpolationFuncFlatten(),
		"floor":        interpolationFuncFloor(),
		"format":       interpolationFuncFormat(),
		"formatlist":   interpolationFuncFormatList(),
		"indent":       interpolationFuncIndent(),
		"index":        interpolationFuncIndex(),
		"join":         interpolationFuncJoin(),
		"jsonencode":   interpolationFuncJSONEncode(),
		"length":       interpolationFuncLength(),
		"list":         interpolationFuncList(),
		"log":          interpolationFuncLog(),
		"lower":        interpolationFuncLower(),
		"map":          interpolationFuncMap(),
		"max":          interpolationFuncMax(),
		"md5":          interpolationFuncMd5(),
		"merge":        interpolationFuncMerge(),
		"min":          interpolationFuncMin(),
		"pathexpand":   interpolationFuncPathExpand(),
		"pow":          interpolationFuncPow(),
		"product":      interpolationFuncProduct(),
		"uuid":         interpolationFuncUUID(),
		"replace":      interpolationFuncReplace(),
		"rsadecrypt":   interpolationFuncRsaDecrypt(),
		"sha1":         interpolationFuncSha1(),
		"sha256":       interpolationFuncSha256(),
		"sha512":       interpolationFuncSha512(),
		"signum":       interpolationFuncSignum(),
		"slice":        interpolationFuncSlice(),
		"sort":         interpolationFuncSort(),
		"split":        interpolationFuncSplit(),
		"substr":       interpolationFuncSubstr(),
		"timestamp":    interpolationFuncTimestamp(),
		"timeadd":      interpolationFuncTimeAdd(),
		"title":        interpolationFuncTitle(),
		"transpose":    interpolationFuncTranspose(),
		"trimspace":    interpolationFuncTrimSpace(),
		"upper":        interpolationFuncUpper(),
		"urlencode":    interpolationFuncURLEncode(),
		"zipmap":       interpolationFuncZipMap(),
	}
}

// interpolationFuncList creates a list from the parameters passed
// to it.
func interpolationFuncList() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{},
		ReturnType:   ast.TypeList,
		Variadic:     true,
		VariadicType: ast.TypeAny,
		Callback: func(args []interface{}) (interface{}, error) {
			var outputList []ast.Variable

			for i, val := range args {
				switch v := val.(type) {
				case string:
					outputList = append(outputList, ast.Variable{Type: ast.TypeString, Value: v})
				case []ast.Variable:
					outputList = append(outputList, ast.Variable{Type: ast.TypeList, Value: v})
				case map[string]ast.Variable:
					outputList = append(outputList, ast.Variable{Type: ast.TypeMap, Value: v})
				default:
					return nil, fmt.Errorf("unexpected type %T for argument %d in list", v, i)
				}
			}

			// we don't support heterogeneous types, so make sure all types match the first
			if len(outputList) > 0 {
				firstType := outputList[0].Type
				for i, v := range outputList[1:] {
					if v.Type != firstType {
						return nil, fmt.Errorf("unexpected type %s for argument %d in list", v.Type, i+1)
					}
				}
			}

			return outputList, nil
		},
	}
}

// interpolationFuncMap creates a map from the parameters passed
// to it.
func interpolationFuncMap() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{},
		ReturnType:   ast.TypeMap,
		Variadic:     true,
		VariadicType: ast.TypeAny,
		Callback: func(args []interface{}) (interface{}, error) {
			outputMap := make(map[string]ast.Variable)

			if len(args)%2 != 0 {
				return nil, fmt.Errorf("requires an even number of arguments, got %d", len(args))
			}

			var firstType *ast.Type
			for i := 0; i < len(args); i += 2 {
				key, ok := args[i].(string)
				if !ok {
					return nil, fmt.Errorf("argument %d represents a key, so it must be a string", i+1)
				}
				val := args[i+1]
				variable, err := hil.InterfaceToVariable(val)
				if err != nil {
					return nil, err
				}
				// Enforce map type homogeneity
				if firstType == nil {
					firstType = &variable.Type
				} else if variable.Type != *firstType {
					return nil, fmt.Errorf("all map values must have the same type, got %s then %s", firstType.Printable(), variable.Type.Printable())
				}
				// Check for duplicate keys
				if _, ok := outputMap[key]; ok {
					return nil, fmt.Errorf("argument %d is a duplicate key: %q", i+1, key)
				}
				outputMap[key] = variable
			}

			return outputMap, nil
		},
	}
}

// interpolationFuncCompact strips a list of multi-variable values
// (e.g. as returned by "split") of any empty strings.
func interpolationFuncCompact() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeList},
		ReturnType: ast.TypeList,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			inputList := args[0].([]ast.Variable)

			var outputList []string
			for _, val := range inputList {
				strVal, ok := val.Value.(string)
				if !ok {
					return nil, fmt.Errorf(
						"compact() may only be used with flat lists, this list contains elements of %s",
						val.Type.Printable())
				}
				if strVal == "" {
					continue
				}

				outputList = append(outputList, strVal)
			}
			return stringSliceToVariableValue(outputList), nil
		},
	}
}

// interpolationFuncCidrHost implements the "cidrhost" function that
// fills in the host part of a CIDR range address to create a single
// host address
func interpolationFuncCidrHost() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeString, // starting CIDR mask
			ast.TypeInt,    // host number to insert
		},
		ReturnType: ast.TypeString,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			hostNum := args[1].(int)
			_, network, err := net.ParseCIDR(args[0].(string))
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR expression: %s", err)
			}

			ip, err := cidr.Host(network, hostNum)
			if err != nil {
				return nil, err
			}

			return ip.String(), nil
		},
	}
}

// interpolationFuncCidrNetmask implements the "cidrnetmask" function
// that returns the subnet mask in IP address notation.
func interpolationFuncCidrNetmask() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeString, // CIDR mask
		},
		ReturnType: ast.TypeString,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			_, network, err := net.ParseCIDR(args[0].(string))
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR expression: %s", err)
			}

			return net.IP(network.Mask).String(), nil
		},
	}
}

// interpolationFuncCidrSubnet implements the "cidrsubnet" function that
// adds an additional subnet of the given length onto an existing
// IP block expressed in CIDR notation.
func interpolationFuncCidrSubnet() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeString, // starting CIDR mask
			ast.TypeInt,    // number of bits to extend the prefix
			ast.TypeInt,    // network number to append to the prefix
		},
		ReturnType: ast.TypeString,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			extraBits := args[1].(int)
			subnetNum := args[2].(int)
			_, network, err := net.ParseCIDR(args[0].(string))
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR expression: %s", err)
			}

			// For portability with 32-bit systems where the subnet number
			// will be a 32-bit int, we only allow extension of 32 bits in
			// one call even if we're running on a 64-bit machine.
			// (Of course, this is significant only for IPv6.)
			if extraBits > 32 {
				return nil, fmt.Errorf("may not extend prefix by more than 32 bits")
			}

			newNetwork, err := cidr.Subnet(network, extraBits, subnetNum)
			if err != nil {
				return nil, err
			}

			return newNetwork.String(), nil
		},
	}
}

// interpolationFuncCoalesce implements the "coalesce" function that
// returns the first non null / empty string from the provided input
func interpolationFuncCoalesce() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeString},
		ReturnType:   ast.TypeString,
		Variadic:     true,
		VariadicType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("must provide at least two arguments")
			}
			for _, arg := range args {
				argument := arg.(string)

				if argument != "" {
					return argument, nil
				}
			}
			return "", nil
		},
	}
}

// interpolationFuncCoalesceList implements the "coalescelist" function that
// returns the first non empty list from the provided input
func interpolationFuncCoalesceList() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeList},
		ReturnType:   ast.TypeList,
		Variadic:     true,
		VariadicType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("must provide at least two arguments")
			}
			for _, arg := range args {
				argument := arg.([]ast.Variable)

				if len(argument) > 0 {
					return argument, nil
				}
			}
			return make([]ast.Variable, 0), nil
		},
	}
}

// interpolationFuncContains returns true if an element is in the list
// and return false otherwise
func interpolationFuncContains() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeList, ast.TypeString},
		ReturnType: ast.TypeBool,
		Callback: func(args []interface{}) (interface{}, error) {
			_, err := interpolationFuncIndex().Callback(args)
			if err != nil {
				return false, nil
			}
			return true, nil
		},
	}
}

// interpolationFuncConcat implements the "concat" function that concatenates
// multiple lists.
func interpolationFuncConcat() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeList},
		ReturnType:   ast.TypeList,
		Variadic:     true,
		VariadicType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			var outputList []ast.Variable

			for _, arg := range args {
				for _, v := range arg.([]ast.Variable) {
					switch v.Type {
					case ast.TypeString:
						outputList = append(outputList, v)
					case ast.TypeList:
						outputList = append(outputList, v)
					case ast.TypeMap:
						outputList = append(outputList, v)
					default:
						return nil, fmt.Errorf("concat() does not support lists of %s", v.Type.Printable())
					}
				}
			}

			// we don't support heterogeneous types, so make sure all types match the first
			if len(outputList) > 0 {
				firstType := outputList[0].Type
				for _, v := range outputList[1:] {
					if v.Type != firstType {
						return nil, fmt.Errorf("unexpected %s in list of %s", v.Type.Printable(), firstType.Printable())
					}
				}
			}

			return outputList, nil
		},
	}
}

// interpolationFuncPow returns base x exponential of y.
func interpolationFuncPow() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat, ast.TypeFloat},
		ReturnType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			return math.Pow(args[0].(float64), args[1].(float64)), nil
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

// interpolationFuncMax returns the maximum of the numeric arguments
func interpolationFuncMax() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeFloat},
		ReturnType:   ast.TypeFloat,
		Variadic:     true,
		VariadicType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			max := args[0].(float64)

			for i := 1; i < len(args); i++ {
				max = math.Max(max, args[i].(float64))
			}

			return max, nil
		},
	}
}

// interpolationFuncMin returns the minimum of the numeric arguments
func interpolationFuncMin() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeFloat},
		ReturnType:   ast.TypeFloat,
		Variadic:     true,
		VariadicType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			min := args[0].(float64)

			for i := 1; i < len(args); i++ {
				min = math.Min(min, args[i].(float64))
			}

			return min, nil
		},
	}
}

// interpolationFuncPathExpand will expand any `~`'s found with the full file path
func interpolationFuncPathExpand() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return homedir.Expand(args[0].(string))
		},
	}
}

// interpolationFuncCeil returns the the least integer value greater than or equal to the argument
func interpolationFuncCeil() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat},
		ReturnType: ast.TypeInt,
		Callback: func(args []interface{}) (interface{}, error) {
			return int(math.Ceil(args[0].(float64))), nil
		},
	}
}

// interpolationFuncLog returns the logarithnm.
func interpolationFuncLog() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat, ast.TypeFloat},
		ReturnType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			return math.Log(args[0].(float64)) / math.Log(args[1].(float64)), nil
		},
	}
}

// interpolationFuncChomp removes trailing newlines from the given string
func interpolationFuncChomp() ast.Function {
	newlines := regexp.MustCompile(`(?:\r\n?|\n)*\z`)
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return newlines.ReplaceAllString(args[0].(string), ""), nil
		},
	}
}

// interpolationFuncFloorreturns returns the greatest integer value less than or equal to the argument
func interpolationFuncFloor() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat},
		ReturnType: ast.TypeInt,
		Callback: func(args []interface{}) (interface{}, error) {
			return int(math.Floor(args[0].(float64))), nil
		},
	}
}

func interpolationFuncZipMap() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeList, // Keys
			ast.TypeList, // Values
		},
		ReturnType: ast.TypeMap,
		Callback: func(args []interface{}) (interface{}, error) {
			keys := args[0].([]ast.Variable)
			values := args[1].([]ast.Variable)

			if len(keys) != len(values) {
				return nil, fmt.Errorf("count of keys (%d) does not match count of values (%d)",
					len(keys), len(values))
			}

			for i, val := range keys {
				if val.Type != ast.TypeString {
					return nil, fmt.Errorf("keys must be strings. value at position %d is %s",
						i, val.Type.Printable())
				}
			}

			result := map[string]ast.Variable{}
			for i := 0; i < len(keys); i++ {
				result[keys[i].Value.(string)] = values[i]
			}

			return result, nil
		},
	}
}

// interpolationFuncFormatList implements the "formatlist" function that does
// string formatting on lists.
func interpolationFuncFormatList() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeAny},
		Variadic:     true,
		VariadicType: ast.TypeAny,
		ReturnType:   ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			// Make a copy of the variadic part of args
			// to avoid modifying the original.
			varargs := make([]interface{}, len(args)-1)
			copy(varargs, args[1:])

			// Verify we have some arguments
			if len(varargs) == 0 {
				return nil, fmt.Errorf("no arguments to formatlist")
			}

			// Convert arguments that are lists into slices.
			// Confirm along the way that all lists have the same length (n).
			var n int
			listSeen := false
			for i := 1; i < len(args); i++ {
				s, ok := args[i].([]ast.Variable)
				if !ok {
					continue
				}

				// Mark that we've seen at least one list
				listSeen = true

				// Convert the ast.Variable to a slice of strings
				parts, err := listVariableValueToStringSlice(s)
				if err != nil {
					return nil, err
				}

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

			// If we didn't see a list this is an error because we
			// can't determine the return value length.
			if !listSeen {
				return nil, fmt.Errorf(
					"formatlist requires at least one list argument")
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
			return stringSliceToVariableValue(list), nil
		},
	}
}

// interpolationFuncIndent indents a multi-line string with the
// specified number of spaces
func interpolationFuncIndent() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			spaces := args[0].(int)
			data := args[1].(string)
			pad := strings.Repeat(" ", spaces)
			return strings.Replace(data, "\n", "\n"+pad, -1), nil
		},
	}
}

// interpolationFuncIndex implements the "index" function that allows one to
// find the index of a specific element in a list
func interpolationFuncIndex() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeList, ast.TypeString},
		ReturnType: ast.TypeInt,
		Callback: func(args []interface{}) (interface{}, error) {
			haystack := args[0].([]ast.Variable)
			needle := args[1].(string)
			for index, element := range haystack {
				if needle == element.Value {
					return index, nil
				}
			}
			return nil, fmt.Errorf("Could not find '%s' in '%s'", needle, haystack)
		},
	}
}

// interpolationFuncBasename implements the "dirname" function.
func interpolationFuncDirname() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return filepath.Dir(args[0].(string)), nil
		},
	}
}

// interpolationFuncDistinct implements the "distinct" function that
// removes duplicate elements from a list.
func interpolationFuncDistinct() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeList},
		ReturnType:   ast.TypeList,
		Variadic:     true,
		VariadicType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			var list []string

			if len(args) != 1 {
				return nil, fmt.Errorf("accepts only one argument.")
			}

			if argument, ok := args[0].([]ast.Variable); ok {
				for _, element := range argument {
					if element.Type != ast.TypeString {
						return nil, fmt.Errorf(
							"only works for flat lists, this list contains elements of %s",
							element.Type.Printable())
					}
					list = appendIfMissing(list, element.Value.(string))
				}
			}

			return stringSliceToVariableValue(list), nil
		},
	}
}

// helper function to add an element to a list, if it does not already exsit
func appendIfMissing(slice []string, element string) []string {
	for _, ele := range slice {
		if ele == element {
			return slice
		}
	}
	return append(slice, element)
}

// for two lists `keys` and `values` of equal length, returns all elements
// from `values` where the corresponding element from `keys` is in `searchset`.
func interpolationFuncMatchKeys() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeList, ast.TypeList, ast.TypeList},
		ReturnType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			output := make([]ast.Variable, 0)

			values, _ := args[0].([]ast.Variable)
			keys, _ := args[1].([]ast.Variable)
			searchset, _ := args[2].([]ast.Variable)

			if len(keys) != len(values) {
				return nil, fmt.Errorf("length of keys and values should be equal")
			}

			for i, key := range keys {
				for _, search := range searchset {
					if res, err := compareSimpleVariables(key, search); err != nil {
						return nil, err
					} else if res == true {
						output = append(output, values[i])
						break
					}
				}
			}
			// if searchset is empty, then output is an empty list as well.
			// if we haven't matched any key, then output is an empty list.
			return output, nil
		},
	}
}

// compare two variables of the same type, i.e. non complex one, such as TypeList or TypeMap
func compareSimpleVariables(a, b ast.Variable) (bool, error) {
	if a.Type != b.Type {
		return false, fmt.Errorf(
			"won't compare items of different types %s and %s",
			a.Type.Printable(), b.Type.Printable())
	}
	switch a.Type {
	case ast.TypeString:
		return a.Value.(string) == b.Value.(string), nil
	default:
		return false, fmt.Errorf(
			"can't compare items of type %s",
			a.Type.Printable())
	}
}

// interpolationFuncJoin implements the "join" function that allows
// multi-variable values to be joined by some character.
func interpolationFuncJoin() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeString},
		Variadic:     true,
		VariadicType: ast.TypeList,
		ReturnType:   ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			var list []string

			if len(args) < 2 {
				return nil, fmt.Errorf("not enough arguments to join()")
			}

			for _, arg := range args[1:] {
				for _, part := range arg.([]ast.Variable) {
					if part.Type != ast.TypeString {
						return nil, fmt.Errorf(
							"only works on flat lists, this list contains elements of %s",
							part.Type.Printable())
					}
					list = append(list, part.Value.(string))
				}
			}

			return strings.Join(list, args[0].(string)), nil
		},
	}
}

// interpolationFuncJSONEncode implements the "jsonencode" function that encodes
// a string, list, or map as its JSON representation.
func interpolationFuncJSONEncode() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeAny},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			var toEncode interface{}

			switch typedArg := args[0].(type) {
			case string:
				toEncode = typedArg

			case []ast.Variable:
				strings := make([]string, len(typedArg))

				for i, v := range typedArg {
					if v.Type != ast.TypeString {
						variable, _ := hil.InterfaceToVariable(typedArg)
						toEncode, _ = hil.VariableToInterface(variable)

						jEnc, err := json.Marshal(toEncode)
						if err != nil {
							return "", fmt.Errorf("failed to encode JSON data '%s'", toEncode)
						}
						return string(jEnc), nil

					}
					strings[i] = v.Value.(string)
				}
				toEncode = strings

			case map[string]ast.Variable:
				stringMap := make(map[string]string)
				for k, v := range typedArg {
					if v.Type != ast.TypeString {
						variable, _ := hil.InterfaceToVariable(typedArg)
						toEncode, _ = hil.VariableToInterface(variable)

						jEnc, err := json.Marshal(toEncode)
						if err != nil {
							return "", fmt.Errorf("failed to encode JSON data '%s'", toEncode)
						}
						return string(jEnc), nil
					}
					stringMap[k] = v.Value.(string)
				}
				toEncode = stringMap

			default:
				return "", fmt.Errorf("unknown type for JSON encoding: %T", args[0])
			}

			jEnc, err := json.Marshal(toEncode)
			if err != nil {
				return "", fmt.Errorf("failed to encode JSON data '%s'", toEncode)
			}
			return string(jEnc), nil
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
		ArgTypes:   []ast.Type{ast.TypeAny},
		ReturnType: ast.TypeInt,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			subject := args[0]

			switch typedSubject := subject.(type) {
			case string:
				return len(typedSubject), nil
			case []ast.Variable:
				return len(typedSubject), nil
			case map[string]ast.Variable:
				return len(typedSubject), nil
			}

			return 0, fmt.Errorf("arguments to length() must be a string, list, or map")
		},
	}
}

func interpolationFuncSignum() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeInt},
		ReturnType: ast.TypeInt,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			num := args[0].(int)
			switch {
			case num < 0:
				return -1, nil
			case num > 0:
				return +1, nil
			default:
				return 0, nil
			}
		},
	}
}

// interpolationFuncSlice returns a portion of the input list between from, inclusive and to, exclusive.
func interpolationFuncSlice() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeList, // inputList
			ast.TypeInt,  // from
			ast.TypeInt,  // to
		},
		ReturnType: ast.TypeList,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			inputList := args[0].([]ast.Variable)
			from := args[1].(int)
			to := args[2].(int)

			if from < 0 {
				return nil, fmt.Errorf("from index must be >= 0")
			}
			if to > len(inputList) {
				return nil, fmt.Errorf("to index must be <= length of the input list")
			}
			if from > to {
				return nil, fmt.Errorf("from index must be <= to index")
			}

			var outputList []ast.Variable
			for i, val := range inputList {
				if i >= from && i < to {
					outputList = append(outputList, val)
				}
			}
			return outputList, nil
		},
	}
}

// interpolationFuncSort sorts a list of a strings lexographically
func interpolationFuncSort() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeList},
		ReturnType: ast.TypeList,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			inputList := args[0].([]ast.Variable)

			// Ensure that all the list members are strings and
			// create a string slice from them
			members := make([]string, len(inputList))
			for i, val := range inputList {
				if val.Type != ast.TypeString {
					return nil, fmt.Errorf(
						"sort() may only be used with lists of strings - %s at index %d",
						val.Type.String(), i)
				}

				members[i] = val.Value.(string)
			}

			sort.Strings(members)
			return stringSliceToVariableValue(members), nil
		},
	}
}

// interpolationFuncSplit implements the "split" function that allows
// strings to split into multi-variable values
func interpolationFuncSplit() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			sep := args[0].(string)
			s := args[1].(string)
			elements := strings.Split(s, sep)
			return stringSliceToVariableValue(elements), nil
		},
	}
}

// interpolationFuncLookup implements the "lookup" function that allows
// dynamic lookups of map types within a Terraform configuration.
func interpolationFuncLookup(vs map[string]ast.Variable) ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeMap, ast.TypeString},
		ReturnType:   ast.TypeString,
		Variadic:     true,
		VariadicType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			defaultValue := ""
			defaultValueSet := false
			if len(args) > 2 {
				defaultValue = args[2].(string)
				defaultValueSet = true
			}
			if len(args) > 3 {
				return "", fmt.Errorf("lookup() takes no more than three arguments")
			}
			index := args[1].(string)
			mapVar := args[0].(map[string]ast.Variable)

			v, ok := mapVar[index]
			if !ok {
				if defaultValueSet {
					return defaultValue, nil
				} else {
					return "", fmt.Errorf(
						"lookup failed to find '%s'",
						args[1].(string))
				}
			}
			if v.Type != ast.TypeString {
				return nil, fmt.Errorf(
					"lookup() may only be used with flat maps, this map contains elements of %s",
					v.Type.Printable())
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
		ArgTypes:   []ast.Type{ast.TypeList, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			list := args[0].([]ast.Variable)
			if len(list) == 0 {
				return nil, fmt.Errorf("element() may not be used with an empty list")
			}

			index, err := strconv.Atoi(args[1].(string))
			if err != nil || index < 0 {
				return "", fmt.Errorf(
					"invalid number for index, got %s", args[1])
			}

			resolvedIndex := index % len(list)

			v := list[resolvedIndex]
			if v.Type != ast.TypeString {
				return nil, fmt.Errorf(
					"element() may only be used with flat lists, this list contains elements of %s",
					v.Type.Printable())
			}
			return v.Value, nil
		},
	}
}

// returns the `list` items chunked by `size`.
func interpolationFuncChunklist() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeList, // inputList
			ast.TypeInt,  // size
		},
		ReturnType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			output := make([]ast.Variable, 0)

			values, _ := args[0].([]ast.Variable)
			size, _ := args[1].(int)

			// errors if size is negative
			if size < 0 {
				return nil, fmt.Errorf("The size argument must be positive")
			}

			// if size is 0, returns a list made of the initial list
			if size == 0 {
				output = append(output, ast.Variable{
					Type:  ast.TypeList,
					Value: values,
				})
				return output, nil
			}

			variables := make([]ast.Variable, 0)
			chunk := ast.Variable{
				Type:  ast.TypeList,
				Value: variables,
			}
			l := len(values)
			for i, v := range values {
				variables = append(variables, v)

				// Chunk when index isn't 0, or when reaching the values's length
				if (i+1)%size == 0 || (i+1) == l {
					chunk.Value = variables
					output = append(output, chunk)
					variables = make([]ast.Variable, 0)
				}
			}

			return output, nil
		},
	}
}

// interpolationFuncKeys implements the "keys" function that yields a list of
// keys of map types within a Terraform configuration.
func interpolationFuncKeys(vs map[string]ast.Variable) ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeMap},
		ReturnType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			mapVar := args[0].(map[string]ast.Variable)
			keys := make([]string, 0)

			for k, _ := range mapVar {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			// Keys are guaranteed to be strings
			return stringSliceToVariableValue(keys), nil
		},
	}
}

// interpolationFuncValues implements the "values" function that yields a list of
// keys of map types within a Terraform configuration.
func interpolationFuncValues(vs map[string]ast.Variable) ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeMap},
		ReturnType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			mapVar := args[0].(map[string]ast.Variable)
			keys := make([]string, 0)

			for k, _ := range mapVar {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			values := make([]string, len(keys))
			for index, key := range keys {
				if value, ok := mapVar[key].Value.(string); ok {
					values[index] = value
				} else {
					return "", fmt.Errorf("values(): %q has element with bad type %s",
						key, mapVar[key].Type)
				}
			}

			variable, err := hil.InterfaceToVariable(values)
			if err != nil {
				return nil, err
			}

			return variable.Value, nil
		},
	}
}

// interpolationFuncBasename implements the "basename" function.
func interpolationFuncBasename() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return filepath.Base(args[0].(string)), nil
		},
	}
}

// interpolationFuncBase64Encode implements the "base64encode" function that
// allows Base64 encoding.
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

// interpolationFuncBase64Decode implements the "base64decode" function that
// allows Base64 decoding.
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

// interpolationFuncBase64Gzip implements the "gzip" function that allows gzip
// compression encoding the result using base64
func interpolationFuncBase64Gzip() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)

			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			if _, err := gz.Write([]byte(s)); err != nil {
				return "", fmt.Errorf("failed to write gzip raw data: '%s'", s)
			}
			if err := gz.Flush(); err != nil {
				return "", fmt.Errorf("failed to flush gzip writer: '%s'", s)
			}
			if err := gz.Close(); err != nil {
				return "", fmt.Errorf("failed to close gzip writer: '%s'", s)
			}

			return base64.StdEncoding.EncodeToString(b.Bytes()), nil
		},
	}
}

// interpolationFuncLower implements the "lower" function that does
// string lower casing.
func interpolationFuncLower() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			toLower := args[0].(string)
			return strings.ToLower(toLower), nil
		},
	}
}

func interpolationFuncMd5() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			h := md5.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			return hash, nil
		},
	}
}

func interpolationFuncMerge() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeMap},
		ReturnType:   ast.TypeMap,
		Variadic:     true,
		VariadicType: ast.TypeMap,
		Callback: func(args []interface{}) (interface{}, error) {
			outputMap := make(map[string]ast.Variable)

			for _, arg := range args {
				for k, v := range arg.(map[string]ast.Variable) {
					outputMap[k] = v
				}
			}

			return outputMap, nil
		},
	}
}

// interpolationFuncUpper implements the "upper" function that does
// string upper casing.
func interpolationFuncUpper() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			toUpper := args[0].(string)
			return strings.ToUpper(toUpper), nil
		},
	}
}

func interpolationFuncSha1() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			return hash, nil
		},
	}
}

// hexadecimal representation of sha256 sum
func interpolationFuncSha256() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			h := sha256.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			return hash, nil
		},
	}
}

func interpolationFuncSha512() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			h := sha512.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			return hash, nil
		},
	}
}

func interpolationFuncTrimSpace() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			trimSpace := args[0].(string)
			return strings.TrimSpace(trimSpace), nil
		},
	}
}

func interpolationFuncBase64Sha256() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			h := sha256.New()
			h.Write([]byte(s))
			shaSum := h.Sum(nil)
			encoded := base64.StdEncoding.EncodeToString(shaSum[:])
			return encoded, nil
		},
	}
}

func interpolationFuncBase64Sha512() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			h := sha512.New()
			h.Write([]byte(s))
			shaSum := h.Sum(nil)
			encoded := base64.StdEncoding.EncodeToString(shaSum[:])
			return encoded, nil
		},
	}
}

func interpolationFuncBcrypt() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeString},
		Variadic:     true,
		VariadicType: ast.TypeString,
		ReturnType:   ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			defaultCost := 10

			if len(args) > 1 {
				costStr := args[1].(string)
				cost, err := strconv.Atoi(costStr)
				if err != nil {
					return "", err
				}

				defaultCost = cost
			}

			if len(args) > 2 {
				return "", fmt.Errorf("bcrypt() takes no more than two arguments")
			}

			input := args[0].(string)
			out, err := bcrypt.GenerateFromPassword([]byte(input), defaultCost)
			if err != nil {
				return "", fmt.Errorf("error occured generating password %s", err.Error())
			}

			return string(out), nil
		},
	}
}

func interpolationFuncUUID() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return uuid.GenerateUUID()
		},
	}
}

// interpolationFuncTimestamp
func interpolationFuncTimestamp() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return time.Now().UTC().Format(time.RFC3339), nil
		},
	}
}

func interpolationFuncTimeAdd() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeString, // input timestamp string in RFC3339 format
			ast.TypeString, // duration to add to input timestamp that should be parsable by time.ParseDuration
		},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {

			ts, err := time.Parse(time.RFC3339, args[0].(string))
			if err != nil {
				return nil, err
			}
			duration, err := time.ParseDuration(args[1].(string))
			if err != nil {
				return nil, err
			}

			return ts.Add(duration).Format(time.RFC3339), nil
		},
	}
}

// interpolationFuncTitle implements the "title" function that returns a copy of the
// string in which first characters of all the words are capitalized.
func interpolationFuncTitle() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			toTitle := args[0].(string)
			return strings.Title(toTitle), nil
		},
	}
}

// interpolationFuncSubstr implements the "substr" function that allows strings
// to be truncated.
func interpolationFuncSubstr() ast.Function {
	return ast.Function{
		ArgTypes: []ast.Type{
			ast.TypeString, // input string
			ast.TypeInt,    // offset
			ast.TypeInt,    // length
		},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			str := args[0].(string)
			offset := args[1].(int)
			length := args[2].(int)

			// Interpret a negative offset as being equivalent to a positive
			// offset taken from the end of the string.
			if offset < 0 {
				offset += len(str)
			}

			// Interpret a length of `-1` as indicating that the substring
			// should start at `offset` and continue until the end of the
			// string. Any other negative length (other than `-1`) is invalid.
			if length == -1 {
				length = len(str)
			} else if length >= 0 {
				length += offset
			} else {
				return nil, fmt.Errorf("length should be a non-negative integer")
			}

			if offset > len(str) || offset < 0 {
				return nil, fmt.Errorf("offset cannot be larger than the length of the string")
			}

			if length > len(str) {
				return nil, fmt.Errorf("'offset + length' cannot be larger than the length of the string")
			}

			return str[offset:length], nil
		},
	}
}

// Flatten until it's not ast.TypeList
func flattener(finalList []ast.Variable, flattenList []ast.Variable) []ast.Variable {
	for _, val := range flattenList {
		if val.Type == ast.TypeList {
			finalList = flattener(finalList, val.Value.([]ast.Variable))
		} else {
			finalList = append(finalList, val)
		}
	}
	return finalList
}

// Flatten to single list
func interpolationFuncFlatten() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeList},
		ReturnType: ast.TypeList,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			inputList := args[0].([]ast.Variable)

			var outputList []ast.Variable
			return flattener(outputList, inputList), nil
		},
	}
}

func interpolationFuncURLEncode() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			return url.QueryEscape(s), nil
		},
	}
}

// interpolationFuncTranspose implements the "transpose" function
// that converts a map (string,list) to a map (string,list) where
// the unique values of the original lists become the keys of the
// new map and the keys of the original map become values for the
// corresponding new keys.
func interpolationFuncTranspose() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeMap},
		ReturnType: ast.TypeMap,
		Callback: func(args []interface{}) (interface{}, error) {

			inputMap := args[0].(map[string]ast.Variable)
			outputMap := make(map[string]ast.Variable)
			tmpMap := make(map[string][]string)

			for inKey, inVal := range inputMap {
				if inVal.Type != ast.TypeList {
					return nil, fmt.Errorf("transpose requires a map of lists of strings")
				}
				values := inVal.Value.([]ast.Variable)
				for _, listVal := range values {
					if listVal.Type != ast.TypeString {
						return nil, fmt.Errorf("transpose requires the given map values to be lists of strings")
					}
					outKey := listVal.Value.(string)
					if _, ok := tmpMap[outKey]; !ok {
						tmpMap[outKey] = make([]string, 0)
					}
					outVal := tmpMap[outKey]
					outVal = append(outVal, inKey)
					sort.Strings(outVal)
					tmpMap[outKey] = outVal
				}
			}

			for outKey, outVal := range tmpMap {
				values := make([]ast.Variable, 0)
				for _, v := range outVal {
					values = append(values, ast.Variable{Type: ast.TypeString, Value: v})
				}
				outputMap[outKey] = ast.Variable{Type: ast.TypeList, Value: values}
			}
			return outputMap, nil
		},
	}
}

// interpolationFuncAbs returns the absolute value of a given float.
func interpolationFuncAbs() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeFloat},
		ReturnType: ast.TypeFloat,
		Callback: func(args []interface{}) (interface{}, error) {
			return math.Abs(args[0].(float64)), nil
		},
	}
}

// interpolationFuncRsaDecrypt implements the "rsadecrypt" function that does
// RSA decryption.
func interpolationFuncRsaDecrypt() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString, ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			key := args[1].(string)

			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", fmt.Errorf("Failed to decode input %q: cipher text must be base64-encoded", s)
			}

			block, _ := pem.Decode([]byte(key))
			if block == nil {
				return "", fmt.Errorf("Failed to read key %q: no key found", key)
			}
			if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
				return "", fmt.Errorf(
					"Failed to read key %q: password protected keys are\n"+
						"not supported. Please decrypt the key prior to use.", key)
			}

			x509Key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return "", err
			}

			out, err := rsa.DecryptPKCS1v15(nil, x509Key, b)
			if err != nil {
				return "", err
			}

			return string(out), nil
		},
	}
}

// interpolationFuncProduct implements the "product" function
// that returns the cartesian product of two or more lists
func interpolationFuncProduct() ast.Function {
	return ast.Function{
		ArgTypes:     []ast.Type{ast.TypeList},
		ReturnType:   ast.TypeList,
		Variadic:     true,
		VariadicType: ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("must provide at least two arguments")
			}

			total := 1
			for _, arg := range args {
				total *= len(arg.([]ast.Variable))
			}

			if total == 0 {
				return nil, fmt.Errorf("empty list provided")
			}

			product := make([][]ast.Variable, total)

			b := make([]ast.Variable, total*len(args))
			n := make([]int, len(args))
			s := 0

			for i := range product {
				e := s + len(args)
				pi := b[s:e]
				product[i] = pi
				s = e

				for j, n := range n {
					pi[j] = args[j].([]ast.Variable)[n]
				}

				for j := len(n) - 1; j >= 0; j-- {
					n[j]++
					if n[j] < len(args[j].([]ast.Variable)) {
						break
					}
					n[j] = 0
				}
			}

			return listVariableSliceToVariableValue(product), nil
		},
	}
}
