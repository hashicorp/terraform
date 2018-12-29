package funcs

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

var JoinFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "separator",
			Type: cty.String,
		},
	},
	VarParam: &function.Parameter{
		Name: "lists",
		Type: cty.List(cty.String),
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		sep := args[0].AsString()
		listVals := args[1:]
		if len(listVals) < 1 {
			return cty.UnknownVal(cty.String), fmt.Errorf("at least one list is required")
		}

		l := 0
		for _, list := range listVals {
			if !list.IsWhollyKnown() {
				return cty.UnknownVal(cty.String), nil
			}
			l += list.LengthInt()
		}

		items := make([]string, 0, l)
		for _, list := range listVals {
			for it := list.ElementIterator(); it.Next(); {
				_, val := it.Element()
				items = append(items, val.AsString())
			}
		}

		return cty.StringVal(strings.Join(items, sep)), nil
	},
})

var SortFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.String),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		listVal := args[0]

		if !listVal.IsWhollyKnown() {
			// If some of the element values aren't known yet then we
			// can't yet preduct the order of the result.
			return cty.UnknownVal(retType), nil
		}
		if listVal.LengthInt() == 0 { // Easy path
			return listVal, nil
		}

		list := make([]string, 0, listVal.LengthInt())
		for it := listVal.ElementIterator(); it.Next(); {
			iv, v := it.Element()
			if v.IsNull() {
				return cty.UnknownVal(retType), fmt.Errorf("given list element %s is null; a null string cannot be sorted", iv.AsBigFloat().String())
			}
			list = append(list, v.AsString())
		}

		sort.Strings(list)
		retVals := make([]cty.Value, len(list))
		for i, s := range list {
			retVals[i] = cty.StringVal(s)
		}
		return cty.ListVal(retVals), nil
	},
})

var SplitFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "separator",
			Type: cty.String,
		},
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		sep := args[0].AsString()
		str := args[1].AsString()
		elems := strings.Split(str, sep)
		elemVals := make([]cty.Value, len(elems))
		for i, s := range elems {
			elemVals[i] = cty.StringVal(s)
		}
		if len(elemVals) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}
		return cty.ListVal(elemVals), nil
	},
})

// ChompFunc constructions a function that removes newline characters at the end of a string.
var ChompFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		newlines := regexp.MustCompile(`(?:\r\n?|\n)*\z`)
		return cty.StringVal(newlines.ReplaceAllString(args[0].AsString(), "")), nil
	},
})

// IndentFunc constructions a function that adds a given number of spaces to the
// beginnings of all but the first line in a given multi-line string.
var IndentFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "spaces",
			Type: cty.Number,
		},
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var spaces int
		if err := gocty.FromCtyValue(args[0], &spaces); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		data := args[1].AsString()
		pad := strings.Repeat(" ", spaces)
		return cty.StringVal(strings.Replace(data, "\n", "\n"+pad, -1)), nil
	},
})

// ReplaceFunc constructions a function that searches a given string for another
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

// TitleFunc constructions a function that converts the first letter of each word
// in the given string to uppercase.
var TitleFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return cty.StringVal(strings.Title(args[0].AsString())), nil
	},
})

// TrimSpaceFunc constructions a function that removes any space characters from
// the start and end of the given string.
var TrimSpaceFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return cty.StringVal(strings.TrimSpace(args[0].AsString())), nil
	},
})

// Join concatenates together the string elements of one or more lists with a
// given separator.
func Join(sep cty.Value, lists ...cty.Value) (cty.Value, error) {
	args := make([]cty.Value, len(lists)+1)
	args[0] = sep
	copy(args[1:], lists)
	return JoinFunc.Call(args)
}

// Sort re-orders the elements of a given list of strings so that they are
// in ascending lexicographical order.
func Sort(list cty.Value) (cty.Value, error) {
	return SortFunc.Call([]cty.Value{list})
}

// Split divides a given string by a given separator, returning a list of
// strings containing the characters between the separator sequences.
func Split(sep, str cty.Value) (cty.Value, error) {
	return SplitFunc.Call([]cty.Value{sep, str})
}

// Chomp removes newline characters at the end of a string.
func Chomp(str cty.Value) (cty.Value, error) {
	return ChompFunc.Call([]cty.Value{str})
}

// Indent adds a given number of spaces to the beginnings of all but the first
// line in a given multi-line string.
func Indent(spaces, str cty.Value) (cty.Value, error) {
	return IndentFunc.Call([]cty.Value{spaces, str})
}

// Replace searches a given string for another given substring,
// and replaces all occurences with a given replacement string.
func Replace(str, substr, replace cty.Value) (cty.Value, error) {
	return ReplaceFunc.Call([]cty.Value{str, substr, replace})
}

// Title converts the first letter of each word in the given string to uppercase.
func Title(str cty.Value) (cty.Value, error) {
	return TitleFunc.Call([]cty.Value{str})
}

// TrimSpace removes any space characters from the start and end of the given string.
func TrimSpace(str cty.Value) (cty.Value, error) {
	return TrimSpaceFunc.Call([]cty.Value{str})
}
