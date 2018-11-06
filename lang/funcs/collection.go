package funcs

import (
	"fmt"
	"sort"

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
			if !args[1].IsKnown() {
				// If the index isn't known yet then we can't predict the
				// result type since each tuple element can have its own type.
				return cty.DynamicPseudoType, nil
			}

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

		if !args[0].IsKnown() {
			return cty.UnknownVal(retType), nil
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
		case collTy == cty.String || collTy.IsTupleType() || collTy.IsObjectType() || collTy.IsListType() || collTy.IsMapType() || collTy.IsSetType() || collTy == cty.DynamicPseudoType:
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
			if !arg.IsKnown() {
				// If we run into an unknown list at some point, we can't
				// predict the final result yet. (If there's a known, non-empty
				// arg before this then we won't get here.)
				return cty.UnknownVal(retType), nil
			}

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
		listVal := args[0]
		if !listVal.IsWhollyKnown() {
			// If some of the element values aren't known yet then we
			// can't yet return a compacted list
			return cty.UnknownVal(retType), nil
		}

		var outputList []cty.Value

		for it := listVal.ElementIterator(); it.Next(); {
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

		if !args[0].IsKnown() {
			return cty.UnknownVal(cty.Number), nil
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
	Type: func(args []cty.Value) (cty.Type, error) {
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		listVal := args[0]

		if !listVal.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}
		var list []cty.Value

		for it := listVal.ElementIterator(); it.Next(); {
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
		listVal := args[0]
		if !listVal.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

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
			output = append(output, listVal)
			return cty.ListVal(output), nil
		}

		chunk := make([]cty.Value, 0)

		l := args[0].LengthInt()
		i := 0

		for it := listVal.ElementIterator(); it.Next(); {
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
		if !inputList.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

		if inputList.LengthInt() == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
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
			Type: cty.DynamicPseudoType,
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var keys []cty.Value
		ty := args[0].Type()

		if !ty.IsObjectType() && !ty.IsMapType() {
			return cty.NilVal, fmt.Errorf("keys() requires a map")
		}

		for it := args[0].ElementIterator(); it.Next(); {
			k, _ := it.Element()
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

// LookupFunc contructs a function that performs dynamic lookups of map types.
var LookupFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "inputMap",
			Type: cty.DynamicPseudoType,
		},
		{
			Name: "key",
			Type: cty.String,
		},
	},
	VarParam: &function.Parameter{
		Name:             "default",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		if len(args) < 1 || len(args) > 3 {
			return cty.NilType, fmt.Errorf("lookup() takes two or three arguments, got %d", len(args))
		}

		ty := args[0].Type()

		switch {
		case ty.IsObjectType():
			if !args[1].IsKnown() {
				return cty.DynamicPseudoType, nil
			}

			key := args[1].AsString()
			if ty.HasAttribute(key) {
				return args[0].GetAttr(key).Type(), nil
			} else if len(args) == 3 {
				// if the key isn't found but a default is provided,
				// return the default type
				return args[2].Type(), nil
			}
			return cty.DynamicPseudoType, function.NewArgErrorf(0, "the given object has no attribute %q", key)
		case ty.IsMapType():
			return ty.ElementType(), nil
		default:
			return cty.NilType, function.NewArgErrorf(0, "lookup() requires a map as the first argument")
		}
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var defaultVal cty.Value
		defaultValueSet := false

		if len(args) == 3 {
			defaultVal = args[2]
			defaultValueSet = true
		}

		mapVar := args[0]
		lookupKey := args[1].AsString()

		if !mapVar.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

		if mapVar.Type().IsObjectType() {
			if mapVar.Type().HasAttribute(lookupKey) {
				return mapVar.GetAttr(lookupKey), nil
			}
		} else if mapVar.HasIndex(cty.StringVal(lookupKey)) == cty.True {
			v := mapVar.Index(cty.StringVal(lookupKey))
			if ty := v.Type(); !ty.Equals(cty.NilType) {
				switch {
				case ty.Equals(cty.String):
					return cty.StringVal(v.AsString()), nil
				case ty.Equals(cty.Number):
					return cty.NumberVal(v.AsBigFloat()), nil
				default:
					return cty.NilVal, fmt.Errorf("lookup() can only be used with flat lists")
				}
			}
		}

		if defaultValueSet {
			defaultVal, err = convert.Convert(defaultVal, retType)
			if err != nil {
				return cty.NilVal, err
			}
			return defaultVal, nil
		}

		return cty.UnknownVal(cty.DynamicPseudoType), fmt.Errorf(
			"lookup failed to find '%s'", lookupKey)
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
		for _, arg := range args {
			if !arg.IsWhollyKnown() {
				return cty.UnknownVal(retType), nil
			}
		}

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
		if !args[0].IsKnown() {
			return cty.UnknownVal(cty.List(retType.ElementType())), nil
		}

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

		if !values.IsWhollyKnown() || !keys.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
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

// MergeFunc contructs a function that takes an arbitrary number of maps and
// returns a single map that contains a merged set of elements from all of the maps.
//
// If more than one given map defines the same key then the one that is later in
// the argument sequence takes precedence.
var MergeFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name:             "maps",
		Type:             cty.DynamicPseudoType,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: function.StaticReturnType(cty.DynamicPseudoType),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		outputMap := make(map[string]cty.Value)

		for _, arg := range args {
			if !arg.IsWhollyKnown() {
				return cty.UnknownVal(retType), nil
			}
			if !arg.Type().IsObjectType() && !arg.Type().IsMapType() {
				return cty.NilVal, fmt.Errorf("arguments must be maps or objects, got %#v", arg.Type().FriendlyName())
			}
			for it := arg.ElementIterator(); it.Next(); {
				k, v := it.Element()
				outputMap[k.AsString()] = v
			}
		}
		return cty.ObjectVal(outputMap), nil
	},
})

// SliceFunc contructs a function that extracts some consecutive elements
// from within a list.
var SliceFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "startIndex",
			Type: cty.Number,
		},
		{
			Name: "endIndex",
			Type: cty.Number,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		inputList := args[0]
		if !inputList.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}
		var startIndex, endIndex int

		if err = gocty.FromCtyValue(args[1], &startIndex); err != nil {
			return cty.NilVal, fmt.Errorf("invalid start index: %s", err)
		}
		if err = gocty.FromCtyValue(args[2], &endIndex); err != nil {
			return cty.NilVal, fmt.Errorf("invalid start index: %s", err)
		}

		if startIndex < 0 {
			return cty.NilVal, fmt.Errorf("from index must be >= 0")
		}
		if endIndex > inputList.LengthInt() {
			return cty.NilVal, fmt.Errorf("to index must be <= length of the input list")
		}
		if startIndex > endIndex {
			return cty.NilVal, fmt.Errorf("from index must be <= to index")
		}

		var outputList []cty.Value

		i := 0
		for it := inputList.ElementIterator(); it.Next(); {
			_, v := it.Element()
			if i >= startIndex && i < endIndex {
				outputList = append(outputList, v)
			}
			i++
		}

		if len(outputList) == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}
		return cty.ListVal(outputList), nil
	},
})

// TransposeFunc contructs a function that takes a map of lists of strings and
// swaps the keys and values to produce a new map of lists of strings.
var TransposeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "values",
			Type: cty.Map(cty.List(cty.String)),
		},
	},
	Type: function.StaticReturnType(cty.Map(cty.List(cty.String))),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		inputMap := args[0]
		if !inputMap.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

		outputMap := make(map[string]cty.Value)
		tmpMap := make(map[string][]string)

		for it := inputMap.ElementIterator(); it.Next(); {
			inKey, inVal := it.Element()
			for iter := inVal.ElementIterator(); iter.Next(); {
				_, val := iter.Element()
				if !val.Type().Equals(cty.String) {
					return cty.MapValEmpty(cty.List(cty.String)), fmt.Errorf("input must be a map of lists of strings")
				}

				outKey := val.AsString()
				if _, ok := tmpMap[outKey]; !ok {
					tmpMap[outKey] = make([]string, 0)
				}
				outVal := tmpMap[outKey]
				outVal = append(outVal, inKey.AsString())
				sort.Strings(outVal)
				tmpMap[outKey] = outVal
			}
		}

		for outKey, outVal := range tmpMap {
			values := make([]cty.Value, 0)
			for _, v := range outVal {
				values = append(values, cty.StringVal(v))
			}
			outputMap[outKey] = cty.ListVal(values)
		}

		return cty.MapVal(outputMap), nil
	},
})

// ValuesFunc contructs a function that returns a list of the map values,
// in the order of the sorted keys.
var ValuesFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "values",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		ty := args[0].Type()
		if ty.IsMapType() {
			return cty.List(ty.ElementType()), nil
		} else if ty.IsObjectType() {
			var tys []cty.Type
			for _, v := range ty.AttributeTypes() {
				tys = append(tys, v)
			}
			return cty.Tuple(tys), nil
		}
		return cty.NilType, fmt.Errorf("values() requires a map as the first argument")
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		mapVar := args[0]

		if !mapVar.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

		if mapVar.LengthInt() == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}

		keys, err := Keys(mapVar)
		if err != nil {
			return cty.NilVal, err
		}

		var values []cty.Value

		for it := keys.ElementIterator(); it.Next(); {
			_, key := it.Element()
			k := key.AsString()
			if mapVar.Type().IsObjectType() {
				if mapVar.Type().HasAttribute(k) {
					value := mapVar.GetAttr(k)
					values = append(values, value)
				}
			} else {
				value := mapVar.Index(cty.StringVal(k))
				values = append(values, value)
			}
		}

		if retType.IsTupleType() {
			return cty.TupleVal(values), nil
		}
		if len(values) == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}
		return cty.ListVal(values), nil
	},
})

// ZipmapFunc contructs a function that constructs a map from a list of keys
// and a corresponding list of values.
var ZipmapFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "keys",
			Type: cty.List(cty.String),
		},
		{
			Name: "values",
			Type: cty.List(cty.DynamicPseudoType),
		},
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		keys := args[0]
		values := args[1]

		if !keys.IsKnown() || !values.IsKnown() || keys.LengthInt() == 0 {
			return cty.Map(cty.DynamicPseudoType), nil
		}

		if keys.LengthInt() != values.LengthInt() {
			return cty.NilType, fmt.Errorf("count of keys (%d) does not match count of values (%d)",
				keys.LengthInt(), values.LengthInt())
		}
		return cty.Map(values.Type().ElementType()), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		keys := args[0]
		values := args[1]

		if !keys.IsKnown() || !values.IsKnown() || keys.LengthInt() == 0 {
			return cty.MapValEmpty(cty.DynamicPseudoType), nil
		}

		output := make(map[string]cty.Value)

		i := 0
		for it := keys.ElementIterator(); it.Next(); {
			_, v := it.Element()
			val := values.Index(cty.NumberIntVal(int64(i)))
			output[v.AsString()] = val
			i++
		}

		return cty.MapVal(output), nil
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

// Lookup performs a dynamic lookup into a map.
// There are two required arguments, map and key, plus an optional default,
// which is a value to return if no key is found in map.
func Lookup(args ...cty.Value) (cty.Value, error) {
	return LookupFunc.Call(args)
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

// Merge takes an arbitrary number of maps and returns a single map that contains
// a merged set of elements from all of the maps.
//
// If more than one given map defines the same key then the one that is later in
// the argument sequence takes precedence.
func Merge(maps ...cty.Value) (cty.Value, error) {
	return MergeFunc.Call(maps)
}

// Slice extracts some consecutive elements from within a list.
func Slice(list, start, end cty.Value) (cty.Value, error) {
	return SliceFunc.Call([]cty.Value{list, start, end})
}

// Transpose takes a map of lists of strings and swaps the keys and values to
// produce a new map of lists of strings.
func Transpose(values cty.Value) (cty.Value, error) {
	return TransposeFunc.Call([]cty.Value{values})
}

// Values returns a list of the map values, in the order of the sorted keys.
// This function only works on flat maps.
func Values(values cty.Value) (cty.Value, error) {
	return ValuesFunc.Call([]cty.Value{values})
}

// Zipmap constructs a map from a list of keys and a corresponding list of values.
func Zipmap(keys, values cty.Value) (cty.Value, error) {
	return ZipmapFunc.Call([]cty.Value{keys, values})
}
