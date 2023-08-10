// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

var LengthFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
			AllowUnknown:     true,
			AllowMarked:      true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		collTy := args[0].Type()
		switch {
		case collTy == cty.String || collTy.IsTupleType() || collTy.IsObjectType() || collTy.IsListType() || collTy.IsMapType() || collTy.IsSetType() || collTy == cty.DynamicPseudoType:
			return cty.Number, nil
		default:
			return cty.Number, errors.New("argument must be a string, a collection type, or a structural type")
		}
	},
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		coll := args[0]
		collTy := args[0].Type()
		marks := coll.Marks()
		switch {
		case collTy == cty.DynamicPseudoType:
			return cty.UnknownVal(cty.Number).WithMarks(marks), nil
		case collTy.IsTupleType():
			l := len(collTy.TupleElementTypes())
			return cty.NumberIntVal(int64(l)).WithMarks(marks), nil
		case collTy.IsObjectType():
			l := len(collTy.AttributeTypes())
			return cty.NumberIntVal(int64(l)).WithMarks(marks), nil
		case collTy == cty.String:
			// We'll delegate to the cty stdlib strlen function here, because
			// it deals with all of the complexities of tokenizing unicode
			// grapheme clusters.
			return stdlib.Strlen(coll)
		case collTy.IsListType() || collTy.IsSetType() || collTy.IsMapType():
			return coll.Length(), nil
		default:
			// Should never happen, because of the checks in our Type func above
			return cty.UnknownVal(cty.Number), errors.New("impossible value type for length(...)")
		}
	},
})

// AllTrueFunc constructs a function that returns true if all elements of the
// list are true. If the list is empty, return true.
var AllTrueFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.Bool),
		},
	},
	Type:         function.StaticReturnType(cty.Bool),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		result := cty.True
		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			if !v.IsKnown() {
				return cty.UnknownVal(cty.Bool), nil
			}
			if v.IsNull() {
				return cty.False, nil
			}
			result = result.And(v)
			if result.False() {
				return cty.False, nil
			}
		}
		return result, nil
	},
})

// AnyTrueFunc constructs a function that returns true if any element of the
// list is true. If the list is empty, return false.
var AnyTrueFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.Bool),
		},
	},
	Type:         function.StaticReturnType(cty.Bool),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		result := cty.False
		var hasUnknown bool
		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			if !v.IsKnown() {
				hasUnknown = true
				continue
			}
			if v.IsNull() {
				continue
			}
			result = result.Or(v)
			if result.True() {
				return cty.True, nil
			}
		}
		if hasUnknown {
			return cty.UnknownVal(cty.Bool), nil
		}
		return result, nil
	},
})

// CoalesceFunc constructs a function that takes any number of arguments and
// returns the first one that isn't empty. This function was copied from go-cty
// stdlib and modified so that it returns the first *non-empty* non-null element
// from a sequence, instead of merely the first non-null.
var CoalesceFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name:             "vals",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		argTypes := make([]cty.Type, len(args))
		for i, val := range args {
			argTypes[i] = val.Type()
		}
		retType, _ := convert.UnifyUnsafe(argTypes)
		if retType == cty.NilType {
			return cty.NilType, errors.New("all arguments must have the same type")
		}
		return retType, nil
	},
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		for _, argVal := range args {
			// We already know this will succeed because of the checks in our Type func above
			argVal, _ = convert.Convert(argVal, retType)
			if !argVal.IsKnown() {
				return cty.UnknownVal(retType), nil
			}
			if argVal.IsNull() {
				continue
			}
			if retType == cty.String && argVal.RawEquals(cty.StringVal("")) {
				continue
			}

			return argVal, nil
		}
		return cty.NilVal, errors.New("no non-null, non-empty-string arguments")
	},
})

// IndexFunc constructs a function that finds the element index for a given value in a list.
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
	Type:         function.StaticReturnType(cty.Number),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		if !(args[0].Type().IsListType() || args[0].Type().IsTupleType()) {
			return cty.NilVal, errors.New("argument must be a list or tuple")
		}

		if !args[0].IsKnown() {
			return cty.UnknownVal(cty.Number), nil
		}

		if args[0].LengthInt() == 0 { // Easy path
			return cty.NilVal, errors.New("cannot search an empty list")
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
		return cty.NilVal, errors.New("item not found")

	},
})

// LookupFunc constructs a function that performs dynamic lookups of map types.
var LookupFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:        "inputMap",
			Type:        cty.DynamicPseudoType,
			AllowMarked: true,
		},
		{
			Name:        "key",
			Type:        cty.String,
			AllowMarked: true,
		},
	},
	VarParam: &function.Parameter{
		Name:             "default",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
		AllowMarked:      true,
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

			keyVal, _ := args[1].Unmark()
			key := keyVal.AsString()
			if ty.HasAttribute(key) {
				return args[0].GetAttr(key).Type(), nil
			} else if len(args) == 3 {
				// if the key isn't found but a default is provided,
				// return the default type
				return args[2].Type(), nil
			}
			return cty.DynamicPseudoType, function.NewArgErrorf(0, "the given object has no attribute %q", key)
		case ty.IsMapType():
			if len(args) == 3 {
				_, err = convert.Convert(args[2], ty.ElementType())
				if err != nil {
					return cty.NilType, function.NewArgErrorf(2, "the default value must have the same type as the map elements")
				}
			}
			return ty.ElementType(), nil
		default:
			return cty.NilType, function.NewArgErrorf(0, "lookup() requires a map as the first argument")
		}
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var defaultVal cty.Value
		defaultValueSet := false

		if len(args) == 3 {
			// intentionally leave default value marked
			defaultVal = args[2]
			defaultValueSet = true
		}

		// keep track of marks from the collection and key
		var markses []cty.ValueMarks

		// unmark collection, retain marks to reapply later
		mapVar, mapMarks := args[0].Unmark()
		markses = append(markses, mapMarks)

		// include marks on the key in the result
		keyVal, keyMarks := args[1].Unmark()
		if len(keyMarks) > 0 {
			markses = append(markses, keyMarks)
		}
		lookupKey := keyVal.AsString()

		if !mapVar.IsKnown() {
			return cty.UnknownVal(retType).WithMarks(markses...), nil
		}

		if mapVar.Type().IsObjectType() {
			if mapVar.Type().HasAttribute(lookupKey) {
				return mapVar.GetAttr(lookupKey).WithMarks(markses...), nil
			}
		} else if mapVar.HasIndex(cty.StringVal(lookupKey)) == cty.True {
			return mapVar.Index(cty.StringVal(lookupKey)).WithMarks(markses...), nil
		}

		if defaultValueSet {
			defaultVal, err = convert.Convert(defaultVal, retType)
			if err != nil {
				return cty.NilVal, err
			}
			return defaultVal.WithMarks(markses...), nil
		}

		return cty.UnknownVal(cty.DynamicPseudoType), fmt.Errorf(
			"lookup failed to find key %s", redactIfSensitive(lookupKey, keyMarks))
	},
})

// MatchkeysFunc constructs a function that constructs a new list by taking a
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
		ty, _ := convert.UnifyUnsafe([]cty.Type{args[1].Type(), args[2].Type()})
		if ty == cty.NilType {
			return cty.NilType, errors.New("keys and searchset must be of the same type")
		}

		// the return type is based on args[0] (values)
		return args[0].Type(), nil
	},
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		if !args[0].IsKnown() {
			return cty.UnknownVal(cty.List(retType.ElementType())), nil
		}

		if args[0].LengthInt() != args[1].LengthInt() {
			return cty.ListValEmpty(retType.ElementType()), errors.New("length of keys and values should be equal")
		}

		output := make([]cty.Value, 0)
		values := args[0]

		// Keys and searchset must be the same type.
		// We can skip error checking here because we've already verified that
		// they can be unified in the Type function
		ty, _ := convert.UnifyUnsafe([]cty.Type{args[1].Type(), args[2].Type()})
		keys, _ := convert.Convert(args[1], ty)
		searchset, _ := convert.Convert(args[2], ty)

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

// OneFunc returns either the first element of a one-element list, or null
// if given a zero-element list.
var OneFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		ty := args[0].Type()
		switch {
		case ty.IsListType() || ty.IsSetType():
			return ty.ElementType(), nil
		case ty.IsTupleType():
			etys := ty.TupleElementTypes()
			switch len(etys) {
			case 0:
				// No specific type information, so we'll ultimately return
				// a null value of unknown type.
				return cty.DynamicPseudoType, nil
			case 1:
				return etys[0], nil
			}
		}
		return cty.NilType, function.NewArgErrorf(0, "must be a list, set, or tuple value with either zero or one elements")
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		val := args[0]
		ty := val.Type()

		// Our parameter spec above doesn't set AllowUnknown or AllowNull,
		// so we can assume our top-level collection is both known and non-null
		// in here.

		switch {
		case ty.IsListType() || ty.IsSetType():
			lenVal := val.Length()
			if !lenVal.IsKnown() {
				return cty.UnknownVal(retType), nil
			}
			var l int
			err := gocty.FromCtyValue(lenVal, &l)
			if err != nil {
				// It would be very strange to get here, because that would
				// suggest that the length is either not a number or isn't
				// an integer, which would suggest a bug in cty.
				return cty.NilVal, fmt.Errorf("invalid collection length: %s", err)
			}
			switch l {
			case 0:
				return cty.NullVal(retType), nil
			case 1:
				var ret cty.Value
				// We'll use an iterator here because that works for both lists
				// and sets, whereas indexing directly would only work for lists.
				// Since we've just checked the length, we should only actually
				// run this loop body once.
				for it := val.ElementIterator(); it.Next(); {
					_, ret = it.Element()
				}
				return ret, nil
			}
		case ty.IsTupleType():
			etys := ty.TupleElementTypes()
			switch len(etys) {
			case 0:
				return cty.NullVal(retType), nil
			case 1:
				ret := val.Index(cty.NumberIntVal(0))
				return ret, nil
			}
		}
		return cty.NilVal, function.NewArgErrorf(0, "must be a list, set, or tuple value with either zero or one elements")
	},
})

// SumFunc constructs a function that returns the sum of all
// numbers provided in a list
var SumFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.DynamicPseudoType,
		},
	},
	Type:         function.StaticReturnType(cty.Number),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {

		if !args[0].CanIterateElements() {
			return cty.NilVal, function.NewArgErrorf(0, "cannot sum noniterable")
		}

		if args[0].LengthInt() == 0 { // Easy path
			return cty.NilVal, function.NewArgErrorf(0, "cannot sum an empty list")
		}

		arg := args[0].AsValueSlice()
		ty := args[0].Type()

		if !ty.IsListType() && !ty.IsSetType() && !ty.IsTupleType() {
			return cty.NilVal, function.NewArgErrorf(0, fmt.Sprintf("argument must be list, set, or tuple. Received %s", ty.FriendlyName()))
		}

		if !args[0].IsWhollyKnown() {
			return cty.UnknownVal(cty.Number), nil
		}

		// big.Float.Add can panic if the input values are opposing infinities,
		// so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = fmt.Errorf("can't compute sum of opposing infinities")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()

		s := arg[0]
		if s.IsNull() {
			return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
		}
		s, err = convert.Convert(s, cty.Number)
		if err != nil {
			return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
		}
		for _, v := range arg[1:] {
			if v.IsNull() {
				return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
			}
			v, err = convert.Convert(v, cty.Number)
			if err != nil {
				return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
			}
			s = s.Add(v)
		}

		return s, nil
	},
})

// TransposeFunc constructs a function that takes a map of lists of strings and
// swaps the keys and values to produce a new map of lists of strings.
var TransposeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "values",
			Type: cty.Map(cty.List(cty.String)),
		},
	},
	Type:         function.StaticReturnType(cty.Map(cty.List(cty.String))),
	RefineResult: refineNotNull,
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
					return cty.MapValEmpty(cty.List(cty.String)), errors.New("input must be a map of lists of strings")
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

		if len(outputMap) == 0 {
			return cty.MapValEmpty(cty.List(cty.String)), nil
		}

		return cty.MapVal(outputMap), nil
	},
})

// ListFunc constructs a function that takes an arbitrary number of arguments
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
		return cty.DynamicPseudoType, fmt.Errorf("the \"list\" function was deprecated in Terraform v0.12 and is no longer available; use tolist([ ... ]) syntax to write a literal list")
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return cty.DynamicVal, fmt.Errorf("the \"list\" function was deprecated in Terraform v0.12 and is no longer available; use tolist([ ... ]) syntax to write a literal list")
	},
})

// MapFunc constructs a function that takes an even number of arguments and
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
		return cty.DynamicPseudoType, fmt.Errorf("the \"map\" function was deprecated in Terraform v0.12 and is no longer available; use tomap({ ... }) syntax to write a literal map")
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return cty.DynamicVal, fmt.Errorf("the \"map\" function was deprecated in Terraform v0.12 and is no longer available; use tomap({ ... }) syntax to write a literal map")
	},
})

// Length returns the number of elements in the given collection or number of
// Unicode characters in the given string.
func Length(collection cty.Value) (cty.Value, error) {
	return LengthFunc.Call([]cty.Value{collection})
}

// AllTrue returns true if all elements of the list are true. If the list is empty,
// return true.
func AllTrue(collection cty.Value) (cty.Value, error) {
	return AllTrueFunc.Call([]cty.Value{collection})
}

// AnyTrue returns true if any element of the list is true. If the list is empty,
// return false.
func AnyTrue(collection cty.Value) (cty.Value, error) {
	return AnyTrueFunc.Call([]cty.Value{collection})
}

// Coalesce takes any number of arguments and returns the first one that isn't empty.
func Coalesce(args ...cty.Value) (cty.Value, error) {
	return CoalesceFunc.Call(args)
}

// Index finds the element index for a given value in a list.
func Index(list, value cty.Value) (cty.Value, error) {
	return IndexFunc.Call([]cty.Value{list, value})
}

// List takes any number of arguments of types that can unify into a single
// type and returns a list containing those values in the same order, or
// returns an error if there is no single element type that all values can
// convert to.
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

// One returns either the first element of a one-element list, or null
// if given a zero-element list..
func One(list cty.Value) (cty.Value, error) {
	return OneFunc.Call([]cty.Value{list})
}

// Sum adds numbers in a list, set, or tuple
func Sum(list cty.Value) (cty.Value, error) {
	return SumFunc.Call([]cty.Value{list})
}

// Transpose takes a map of lists of strings and swaps the keys and values to
// produce a new map of lists of strings.
func Transpose(values cty.Value) (cty.Value, error) {
	return TransposeFunc.Call([]cty.Value{values})
}
