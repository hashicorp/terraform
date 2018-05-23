package funcs

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
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
			_, v := it.Element()
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
