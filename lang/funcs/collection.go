package funcs

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

var ElementFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.DynamicPseudoType,
		},
		{
			Name: "index",
			Type: cty.Number,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		list := args[0]
		listTy := list.Type()
		switch {
		case listTy.IsListType():
			return listTy.ElementType(), nil
		case listTy.IsTupleType():
			etys := listTy.TupleElementTypes()
			var index int
			err := gocty.FromCtyValue(args[1], &index)
			if err != nil {
				// e.g. fractional number where whole number is required
				return cty.DynamicPseudoType, fmt.Errorf("invalid index: %s", err)
			}
			if len(etys) == 0 {
				return cty.DynamicPseudoType, fmt.Errorf("cannot use element function with an empty list")
			}
			index = index % len(etys)
			return etys[index], nil
		default:
			return cty.DynamicPseudoType, fmt.Errorf("cannot read elements from %s", listTy.FriendlyName())
		}
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		var index int
		err := gocty.FromCtyValue(args[1], &index)
		if err != nil {
			// can't happen because we checked this in the Type function above
			return cty.DynamicVal, fmt.Errorf("invalid index: %s", err)
		}
		l := args[0].LengthInt()
		if l == 0 {
			return cty.DynamicVal, fmt.Errorf("cannot use element function with an empty list")
		}
		index = index % l

		// We did all the necessary type checks in the type function above,
		// so this is guaranteed not to fail.
		return args[0].Index(cty.NumberIntVal(int64(index))), nil
	},
})

var LengthFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
			AllowUnknown:     true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		collTy := args[0].Type()
		switch {
		case collTy == cty.String || collTy.IsTupleType() || collTy.IsListType() || collTy.IsMapType() || collTy.IsSetType() || collTy == cty.DynamicPseudoType:
			return cty.Number, nil
		default:
			return cty.Number, fmt.Errorf("argument must be a string, a collection type, or a structural type")
		}
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		coll := args[0]
		collTy := args[0].Type()
		switch {
		case collTy == cty.DynamicPseudoType:
			return cty.UnknownVal(cty.Number), nil
		case collTy.IsTupleType():
			l := len(collTy.TupleElementTypes())
			return cty.NumberIntVal(int64(l)), nil
		case collTy.IsObjectType():
			l := len(collTy.AttributeTypes())
			return cty.NumberIntVal(int64(l)), nil
		case collTy == cty.String:
			// We'll delegate to the cty stdlib strlen function here, because
			// it deals with all of the complexities of tokenizing unicode
			// grapheme clusters.
			return stdlib.Strlen(coll)
		case collTy.IsListType() || collTy.IsSetType() || collTy.IsMapType():
			return coll.Length(), nil
		default:
			// Should never happen, because of the checks in our Type func above
			return cty.UnknownVal(cty.Number), fmt.Errorf("impossible value type for length(...)")
		}
	},
})

// CoalesceListFunc contructs a function that takes any number of list arguments
// and returns the first one that isn't empty.
var CoalesceListFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name:             "vals",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		if len(args) == 0 {
			return cty.NilType, fmt.Errorf("at least one argument is required")
		}

		argTypes := make([]cty.Type, len(args))

		for i, arg := range args {
			argTypes[i] = arg.Type()
		}

		retType, _ := convert.UnifyUnsafe(argTypes)
		if retType == cty.NilType {
			return cty.NilType, fmt.Errorf("all arguments must have the same type")
		}

		return retType, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {

		vals := make([]cty.Value, 0, len(args))
		for _, arg := range args {

			// We already know this will succeed because of the checks in our Type func above
			arg, _ = convert.Convert(arg, retType)

			it := arg.ElementIterator()
			for it.Next() {
				_, v := it.Element()
				vals = append(vals, v)
			}

			if len(vals) > 0 {
				return cty.ListVal(vals), nil
			}
		}

		return cty.NilVal, fmt.Errorf("no non-null arguments")
	},
})

// CompactFunc contructs a function that takes a list of strings and returns a new list
// with any empty string elements removed.
var CompactFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.String),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {

		var outputList []cty.Value

		it := args[0].ElementIterator()
		for it.Next() {
			_, v := it.Element()
			if v.AsString() == "" {
				continue
			}
			outputList = append(outputList, v)
		}
		return cty.ListVal(outputList), nil
	},
})

// Element returns a single element from a given list at the given index. If
// index is greater than the length of the list then it is wrapped modulo
// the list length.
func Element(list, index cty.Value) (cty.Value, error) {
	return ElementFunc.Call([]cty.Value{list, index})
}

// Length returns the number of elements in the given collection or number of
// Unicode characters in the given string.
func Length(collection cty.Value) (cty.Value, error) {
	return LengthFunc.Call([]cty.Value{collection})
}

// CoalesceList takes any number of list arguments and returns the first one that isn't empty.
func CoalesceList(args ...cty.Value) (cty.Value, error) {
	return CoalesceListFunc.Call(args)
}

// Compact takes a list of strings and returns a new list
// with any empty string elements removed.
func Compact(list cty.Value) (cty.Value, error) {
	return CompactFunc.Call([]cty.Value{list})
}
