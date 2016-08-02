package config

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/mitchellh/go-homedir"
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
		"base64decode": interpolationFuncBase64Decode(),
		"base64encode": interpolationFuncBase64Encode(),
		"base64sha256": interpolationFuncBase64Sha256(),
		"cidrhost":     interpolationFuncCidrHost(),
		"cidrnetmask":  interpolationFuncCidrNetmask(),
		"cidrsubnet":   interpolationFuncCidrSubnet(),
		"coalesce":     interpolationFuncCoalesce(),
		"compact":      interpolationFuncCompact(),
		"concat":       interpolationFuncConcat(),
		"distinct":     interpolationFuncDistinct(),
		"element":      interpolationFuncElement(),
		"file":         interpolationFuncFile(),
		"format":       interpolationFuncFormat(),
		"formatlist":   interpolationFuncFormatList(),
		"index":        interpolationFuncIndex(),
		"join":         interpolationFuncJoin(),
		"jsonencode":   interpolationFuncJSONEncode(),
		"length":       interpolationFuncLength(),
		"list":         interpolationFuncList(),
		"lower":        interpolationFuncLower(),
		"map":          interpolationFuncMap(),
		"md5":          interpolationFuncMd5(),
		"merge":        interpolationFuncMerge(),
		"uuid":         interpolationFuncUUID(),
		"replace":      interpolationFuncReplace(),
		"sha1":         interpolationFuncSha1(),
		"sha256":       interpolationFuncSha256(),
		"signum":       interpolationFuncSignum(),
		"sort":         interpolationFuncSort(),
		"split":        interpolationFuncSplit(),
		"trimspace":    interpolationFuncTrimSpace(),
		"upper":        interpolationFuncUpper(),
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
		ArgTypes:     []ast.Type{ast.TypeAny},
		Variadic:     true,
		VariadicType: ast.TypeAny,
		ReturnType:   ast.TypeList,
		Callback: func(args []interface{}) (interface{}, error) {
			// Make a copy of the variadic part of args
			// to avoid modifying the original.
			varargs := make([]interface{}, len(args)-1)
			copy(varargs, args[1:])

			// Convert arguments that are lists into slices.
			// Confirm along the way that all lists have the same length (n).
			var n int
			for i := 1; i < len(args); i++ {
				s, ok := args[i].([]ast.Variable)
				if !ok {
					continue
				}

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
			return stringSliceToVariableValue(list), nil
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
// a string, list, or map as its JSON representation. For now, values in the
// list or map may only be strings.
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
				// We preallocate the list here. Note that it's important that in
				// the length 0 case, we have an empty list rather than nil, as
				// they encode differently.
				// XXX It would be nice to support arbitrarily nested data here. Is
				// there an inverse of hil.InterfaceToVariable?
				strings := make([]string, len(typedArg))

				for i, v := range typedArg {
					if v.Type != ast.TypeString {
						return "", fmt.Errorf("list elements must be strings")
					}
					strings[i] = v.Value.(string)
				}
				toEncode = strings

			case map[string]ast.Variable:
				// XXX It would be nice to support arbitrarily nested data here. Is
				// there an inverse of hil.InterfaceToVariable?
				stringMap := make(map[string]string)
				for k, v := range typedArg {
					if v.Type != ast.TypeString {
						return "", fmt.Errorf("map values must be strings")
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

func interpolationFuncUUID() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			return uuid.GenerateUUID()
		},
	}
}
