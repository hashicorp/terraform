package funcs

import (
	"regexp"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// ReplaceFunc constructs a function that searches a given string for another
// given substring, and replaces each occurence with a given replacement string.
var ReplaceFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
		{
			Name: "substr",
			Type: cty.String,
		},
		{
			Name: "replace",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		str := args[0].AsString()
		substr := args[1].AsString()
		replace := args[2].AsString()

		// We search/replace using a regexp if the string is surrounded
		// in forward slashes.
		if len(substr) > 1 && substr[0] == '/' && substr[len(substr)-1] == '/' {
			re, err := regexp.Compile(substr[1 : len(substr)-1])
			if err != nil {
				return cty.UnknownVal(cty.String), err
			}

			return cty.StringVal(re.ReplaceAllString(str, replace)), nil
		}

		return cty.StringVal(strings.Replace(str, substr, replace, -1)), nil
	},
})

var SnakeCaseFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		// Remove non alpha-numerics
		out := regexp.MustCompile(`(?m)[^a-zA-Z0-9]`).ReplaceAllString(in, "_")
		// Split on uppercase characters followed by lower case (e.g. camel case)
		out = regexp.MustCompile(`(?m)[A-Z][a-z]`).ReplaceAllString(out, "_$0")
		// Remove any consecutive underscores
		out = regexp.MustCompile(`(?m)_+`).ReplaceAllString(out, "_")
		// Remove leading/trailing underscore
		out = regexp.MustCompile(`^_|_$`).ReplaceAllString(out, "")
		return cty.StringVal(strings.ToLower(out)), nil
	},
})

var KebabCaseFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0]
		out, _ := SnakeCase(in)
		return cty.StringVal(strings.ReplaceAll(out.AsString(), "_", "-")), nil
	},
})

var CamelCaseFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0]
		snake, _ := SnakeCase(in)
		words := strings.ReplaceAll(snake.AsString(), "_", " ")
		pascal := strings.ReplaceAll(strings.Title(words), " ", "")

		if pascal != "" {
			camel := string(strings.ToLower(pascal)[0]) + pascal[1:]
			return cty.StringVal(camel), nil
		}
		return cty.StringVal(""), nil
	},
})

// Replace searches a given string for another given substring,
// and replaces all occurences with a given replacement string.
func Replace(str, substr, replace cty.Value) (cty.Value, error) {
	return ReplaceFunc.Call([]cty.Value{str, substr, replace})
}

// SnakeCase is a Function that converts a given string to snake_case.
func SnakeCase(str cty.Value) (cty.Value, error) {
	return SnakeCaseFunc.Call([]cty.Value{str})
}

// KebabCase is a Function that converts a given string to kebab-case.
func KebabCase(str cty.Value) (cty.Value, error) {
	return KebabCaseFunc.Call([]cty.Value{str})
}

// KebabCase is a Function that converts a given string to kebab-case.
func CamelCase(str cty.Value) (cty.Value, error) {
	return CamelCaseFunc.Call([]cty.Value{str})
}
