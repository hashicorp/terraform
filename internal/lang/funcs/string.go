package funcs

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
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

// SortSemVerFunc constructs a function that takes a version constraint string
// and a list of semantic version strings and returns the versions matching that
// constraint in precedence order.
var SortSemVerFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "constraint",
			Type: cty.String,
		},
		{
			Name: "list",
			Type: cty.List(cty.String),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		constStr := args[0].AsString()
		listVal := args[1]

		// Create the constraints to check against.
		constraints := version.Constraints{}
		if strings.TrimSpace(constStr) != "" {
			var err error
			constraints, err = version.NewConstraint(constStr)
			if err != nil {
				return cty.UnknownVal(retType), err
			}
		}

		if !listVal.IsWhollyKnown() {
			// If some of the element values aren't known yet then we
			// can't yet preduct the order of the result.
			return cty.UnknownVal(retType), nil
		}
		if listVal.LengthInt() == 0 { // Easy path
			return listVal, nil
		}

		list := make([]*version.Version, 0, listVal.LengthInt())
		for it := listVal.ElementIterator(); it.Next(); {
			iv, v := it.Element()
			version, err := version.NewSemver(v.AsString())
			if err != nil {
				return cty.UnknownVal(retType), fmt.Errorf("given list element %s is not parseable as a semantic version", iv.AsBigFloat().String())
			}
			if constraints.Check(version) {
				list = append(list, version)
			}
		}

		sort.Stable(version.Collection(list))
		retVals := make([]cty.Value, len(list))
		for i, s := range list {
			retVals[i] = cty.StringVal(s.String())
		}
		return cty.ListVal(retVals), nil
	},
})

// Replace searches a given string for another given substring,
// and replaces all occurences with a given replacement string.
func Replace(str, substr, replace cty.Value) (cty.Value, error) {
	return ReplaceFunc.Call([]cty.Value{str, substr, replace})
}

// SortSemVer re-orders the elements of a given list of strings so that the
// elements matching a given version constraint are returned in precedence
// order.
func SortSemVer(constraint cty.Value, list cty.Value) (cty.Value, error) {
	return SortSemVerFunc.Call([]cty.Value{constraint, list})
}
