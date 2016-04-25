package config

import (
	"bytes"
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
	"github.com/hashicorp/hil/ast"
	"github.com/mitchellh/go-homedir"
)

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
		"element":      interpolationFuncElement(),
		"file":         interpolationFuncFile(),
		"format":       interpolationFuncFormat(),
		"formatlist":   interpolationFuncFormatList(),
		"index":        interpolationFuncIndex(),
		"join":         interpolationFuncJoin(),
		"jsonencode":   interpolationFuncJSONEncode(),
		"length":       interpolationFuncLength(),
		"lower":        interpolationFuncLower(),
		"md5":          interpolationFuncMd5(),
		"uuid":         interpolationFuncUUID(),
		"replace":      interpolationFuncReplace(),
		"sha1":         interpolationFuncSha1(),
		"sha256":       interpolationFuncSha256(),
		"signum":       interpolationFuncSignum(),
		"split":        interpolationFuncSplit(),
		"trimspace":    interpolationFuncTrimSpace(),
		"upper":        interpolationFuncUpper(),
	}
}

// interpolationFuncCompact strips a list of multi-variable values
// (e.g. as returned by "split") of any empty strings.
func interpolationFuncCompact() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Variadic:   false,
		Callback: func(args []interface{}) (interface{}, error) {
			if !IsStringList(args[0].(string)) {
				return args[0].(string), nil
			}
			return StringList(args[0].(string)).Compact().String(), nil
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

// interpolationFuncJSONEncode implements the "jsonencode" function that encodes
// a string as its JSON representation.
func interpolationFuncJSONEncode() ast.Function {
	return ast.Function{
		ArgTypes:   []ast.Type{ast.TypeString},
		ReturnType: ast.TypeString,
		Callback: func(args []interface{}) (interface{}, error) {
			s := args[0].(string)
			jEnc, err := json.Marshal(s)
			if err != nil {
				return "", fmt.Errorf("failed to encode JSON data '%s'", s)
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
			if err != nil || index < 0 {
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
