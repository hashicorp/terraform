package stdlib

import (
	"regexp"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// ReplaceFunc is a function that searches a given string for another given
// substring, and replaces each occurence with a given replacement string.
// The substr argument is a simple string.
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
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		substr := args[1].AsString()
		replace := args[2].AsString()

		return cty.StringVal(strings.Replace(str, substr, replace, -1)), nil
	},
})

// RegexReplaceFunc is a function that searches a given string for another
// given substring, and replaces each occurence with a given replacement
// string. The substr argument must be a valid regular expression.
var RegexReplaceFunc = function.New(&function.Spec{
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

		re, err := regexp.Compile(substr)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(re.ReplaceAllString(str, replace)), nil
	},
})

// Replace searches a given string for another given substring,
// and replaces all occurrences with a given replacement string.
func Replace(str, substr, replace cty.Value) (cty.Value, error) {
	return ReplaceFunc.Call([]cty.Value{str, substr, replace})
}

func RegexReplace(str, substr, replace cty.Value) (cty.Value, error) {
	return RegexReplaceFunc.Call([]cty.Value{str, substr, replace})
}
