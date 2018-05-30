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
		Type:             cty.List(cty.DynamicPseudoType),
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

		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			if v.AsString() == "" {
				continue
			}
			outputList = append(outputList, v)
		}

		if len(outputList) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}

		return cty.ListVal(outputList), nil
	},
})

// ContainsFunc contructs a function that determines whether a given list contains
// a given single value as one of its elements.
var ContainsFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "value",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {

		_, err = Index(args[0], args[1])
		if err != nil {
			return cty.False, nil
		}

		return cty.True, nil
	},
})

// IndexFunc contructs a function that finds the element index for a given value in a list.
var IndexFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.DynamicPseudoType,
		},
		{
			Name: "value",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		if !(args[0].Type().IsListType() || args[0].Type().IsTupleType()) {
			return cty.NilVal, fmt.Errorf("argument must be a list or tuple")
		}

		if args[0].LengthInt() == 0 { // Easy path
			return cty.NilVal, fmt.Errorf("cannot search an empty list")
		}

		for it := args[0].ElementIterator(); it.Next(); {
			i, v := it.Element()
			eq, err := stdlib.Equal(v, args[1])
			if err != nil {
				return cty.NilVal, err
			}
			if !eq.IsKnown() {
				return cty.UnknownVal(cty.Number), nil
			}
			if eq.True() {
				return i, nil
			}
		}
		return cty.NilVal, fmt.Errorf("item not found")

	},
})

// DistinctFunc contructs a function that takes a list and returns a new list
// with any duplicate elements removed.
var DistinctFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.DynamicPseudoType),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.DynamicPseudoType)),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var list []cty.Value

		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			list, err = appendIfMissing(list, v)
			if err != nil {
				return cty.NilVal, err
			}
		}

		return cty.ListVal(list), nil
	},
})

// ChunklistFunc contructs a function that splits a single list into fixed-size chunks,
// returning a list of lists.
var ChunklistFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "size",
			Type: cty.Number,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		return cty.List(args[0].Type()), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var size int
		err = gocty.FromCtyValue(args[1], &size)
		if err != nil {
			return cty.NilVal, fmt.Errorf("invalid index: %s", err)
		}

		if size < 0 {
			return cty.NilVal, fmt.Errorf("the size argument must be positive")
		}

		output := make([]cty.Value, 0)

		// if size is 0, returns a list made of the initial list
		if size == 0 {
			output = append(output, args[0])
			return cty.ListVal(output), nil
		}

		chunk := make([]cty.Value, 0)

		l := args[0].LengthInt()
		i := 0

		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			chunk = append(chunk, v)

			// Chunk when index isn't 0, or when reaching the values's length
			if (i+1)%size == 0 || (i+1) == l {
				output = append(output, cty.ListVal(chunk))
				chunk = make([]cty.Value, 0)
			}
			i++
		}

		return cty.ListVal(output), nil
	},
})

// FlattenFunc contructs a function that takes a list and replaces any elements
// that are lists with a flattened sequence of the list contents.
var FlattenFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.DynamicPseudoType),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.DynamicPseudoType)),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		inputList := args[0]

		if inputList.LengthInt() == 0 {
			return cty.ListValEmpty(cty.DynamicPseudoType), nil
		}
		outputList := make([]cty.Value, 0)

		return cty.ListVal(flattener(outputList, inputList)), nil
	},
})

// Flatten until it's not a cty.List
func flattener(finalList []cty.Value, flattenList cty.Value) []cty.Value {

	for it := flattenList.ElementIterator(); it.Next(); {
		_, val := it.Element()

		if val.Type().IsListType() {
			finalList = flattener(finalList, val)
		} else {
			finalList = append(finalList, val)
		}
	}
	return finalList
}

// KeysFunc contructs a function that takes a map and returns a sorted list of the map keys.
var KeysFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "inputMap",
			Type: cty.Map(cty.DynamicPseudoType),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var keys []cty.Value

		for it := args[0].ElementIterator(); it.Next(); {
			k, _ := it.Element()
			fmt.Printf("appending %#v to %#v\n", k, keys)
			keys = append(keys, k)
			if err != nil {
				return cty.ListValEmpty(cty.String), err
			}
		}
		return cty.ListVal(keys), nil
	},
})

// ListFunc contructs a function that takes an arbitrary number of arguments
// and returns a list containing those values in the same order.
//
// This function is deprecated in Terraform v0.12
var ListFunc = function.New(&function.Spec{
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

		return cty.List(retType), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		newList := make([]cty.Value, 0, len(args))

		for _, arg := range args {
			// We already know this will succeed because of the checks in our Type func above
			arg, _ = convert.Convert(arg, retType.ElementType())
			newList = append(newList, arg)
		}

		return cty.ListVal(newList), nil
	},
})

// MapFunc contructs a function that takes an even number of arguments and
// returns a map whose elements are constructed from consecutive pairs of arguments.
//
// This function is deprecated in Terraform v0.12
var MapFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name:             "vals",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		if len(args) < 2 || len(args)%2 != 0 {
			return cty.NilType, fmt.Errorf("map requires an even number of two or more arguments, got %d", len(args))
		}

		argTypes := make([]cty.Type, len(args)/2)
		index := 0

		for i := 0; i < len(args); i += 2 {
			argTypes[index] = args[i+1].Type()
			index++
		}

		valType, _ := convert.UnifyUnsafe(argTypes)
		if valType == cty.NilType {
			return cty.NilType, fmt.Errorf("all arguments must have the same type")
		}

		return cty.Map(valType), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		outputMap := make(map[string]cty.Value)

		for i := 0; i < len(args); i += 2 {

			key := args[i].AsString()

			err := gocty.FromCtyValue(args[i], &key)
			if err != nil {
				return cty.NilVal, err
			}

			val := args[i+1]

			var variable cty.Value
			err = gocty.FromCtyValue(val, &variable)
			if err != nil {
				return cty.NilVal, err
			}

			// We already know this will succeed because of the checks in our Type func above
			variable, _ = convert.Convert(variable, retType.ElementType())

			// Check for duplicate keys
			if _, ok := outputMap[key]; ok {
				return cty.NilVal, fmt.Errorf("argument %d is a duplicate key: %q", i+1, key)
			}
			outputMap[key] = variable
		}

		return cty.MapVal(outputMap), nil
	},
})

// MatchkeysFunc contructs a function that constructs a new list by taking a
// subset of elements from one list whose indexes match the corresponding
// indexes of values in another list.
var MatchkeysFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "values",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "keys",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "searchset",
			Type: cty.List(cty.DynamicPseudoType),
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if !args[1].Type().Equals(args[2].Type()) {
			return cty.NilType, fmt.Errorf("lists must be of the same type")
		}

		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {

		if args[0].LengthInt() != args[1].LengthInt() {
			return cty.ListValEmpty(retType.ElementType()), fmt.Errorf("length of keys and values should be equal")
		}

		output := make([]cty.Value, 0)

		values := args[0]
		keys := args[1]
		searchset := args[2]

		// if searchset is empty, return an empty list.
		if searchset.LengthInt() == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}

		i := 0
		for it := keys.ElementIterator(); it.Next(); {
			_, key := it.Element()
			for iter := searchset.ElementIterator(); iter.Next(); {
				_, search := iter.Element()
				eq, err := stdlib.Equal(key, search)
				if err != nil {
					return cty.NilVal, err
				}
				if !eq.IsKnown() {
					return cty.ListValEmpty(retType.ElementType()), nil
				}
				if eq.True() {
					v := values.Index(cty.NumberIntVal(int64(i)))
					output = append(output, v)
					break
				}
			}
			i++
		}

		// if we haven't matched any key, then output is an empty list.
		if len(output) == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}
		return cty.ListVal(output), nil
	},
})

// helper function to add an element to a list, if it does not already exist
func appendIfMissing(slice []cty.Value, element cty.Value) ([]cty.Value, error) {
	for _, ele := range slice {
		eq, err := stdlib.Equal(ele, element)
		if err != nil {
			return slice, err
		}
		if eq.True() {
			return slice, nil
		}
	}
	return append(slice, element), nil
}

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

// Contains determines whether a given list contains a given single value
// as one of its elements.
func Contains(list, value cty.Value) (cty.Value, error) {
	return ContainsFunc.Call([]cty.Value{list, value})
}

// Index finds the element index for a given value in a list.
func Index(list, value cty.Value) (cty.Value, error) {
	return IndexFunc.Call([]cty.Value{list, value})
}

// Distinct takes a list and returns a new list with any duplicate elements removed.
func Distinct(list cty.Value) (cty.Value, error) {
	return DistinctFunc.Call([]cty.Value{list})
}

// Chunklist splits a single list into fixed-size chunks, returning a list of lists.
func Chunklist(list, size cty.Value) (cty.Value, error) {
	return ChunklistFunc.Call([]cty.Value{list, size})
}

// Flatten takes a list and replaces any elements that are lists with a flattened
// sequence of the list contents.
func Flatten(list cty.Value) (cty.Value, error) {
	return FlattenFunc.Call([]cty.Value{list})
}

// Keys takes a map and returns a sorted list of the map keys.
func Keys(inputMap cty.Value) (cty.Value, error) {
	return KeysFunc.Call([]cty.Value{inputMap})
}

// List takes any number of list arguments and returns a list containing those
//  values in the same order.
func List(args ...cty.Value) (cty.Value, error) {
	return ListFunc.Call(args)
}

// Map takes an even number of arguments and returns a map whose elements are constructed
// from consecutive pairs of arguments.
func Map(args ...cty.Value) (cty.Value, error) {
	return MapFunc.Call(args)
}

// Matchkeys constructs a new list by taking a subset of elements from one list
// whose indexes match the corresponding indexes of values in another list.
func Matchkeys(values, keys, searchset cty.Value) (cty.Value, error) {
	return MatchkeysFunc.Call([]cty.Value{values, keys, searchset})
}
