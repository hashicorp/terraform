package stdlib

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

var HasIndexFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "collection",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
		{
			Name:             "key",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		collTy := args[0].Type()
		if !(collTy.IsTupleType() || collTy.IsListType() || collTy.IsMapType() || collTy == cty.DynamicPseudoType) {
			return cty.NilType, fmt.Errorf("collection must be a list, a map or a tuple")
		}
		return cty.Bool, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].HasIndex(args[1]), nil
	},
})

var IndexFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "collection",
			Type: cty.DynamicPseudoType,
		},
		{
			Name:             "key",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		collTy := args[0].Type()
		key := args[1]
		keyTy := key.Type()
		switch {
		case collTy.IsTupleType():
			if keyTy != cty.Number && keyTy != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("key for tuple must be number")
			}
			if !key.IsKnown() {
				return cty.DynamicPseudoType, nil
			}
			var idx int
			err := gocty.FromCtyValue(key, &idx)
			if err != nil {
				return cty.NilType, fmt.Errorf("invalid key for tuple: %s", err)
			}

			etys := collTy.TupleElementTypes()

			if idx >= len(etys) || idx < 0 {
				return cty.NilType, fmt.Errorf("key must be between 0 and %d inclusive", len(etys))
			}

			return etys[idx], nil

		case collTy.IsListType():
			if keyTy != cty.Number && keyTy != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("key for list must be number")
			}

			return collTy.ElementType(), nil

		case collTy.IsMapType():
			if keyTy != cty.String && keyTy != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("key for map must be string")
			}

			return collTy.ElementType(), nil

		default:
			return cty.NilType, fmt.Errorf("collection must be a list, a map or a tuple")
		}
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		has, err := HasIndex(args[0], args[1])
		if err != nil {
			return cty.NilVal, err
		}
		if has.False() { // safe because collection and key are guaranteed known here
			return cty.NilVal, fmt.Errorf("invalid index")
		}

		return args[0].Index(args[1]), nil
	},
})

var LengthFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "collection",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		collTy := args[0].Type()
		if !(collTy.IsTupleType() || collTy.IsListType() || collTy.IsMapType() || collTy.IsSetType() || collTy == cty.DynamicPseudoType) {
			return cty.NilType, fmt.Errorf("collection must be a list, a map or a tuple")
		}
		return cty.Number, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].Length(), nil
	},
})

// HasIndex determines whether the given collection can be indexed with the
// given key.
func HasIndex(collection cty.Value, key cty.Value) (cty.Value, error) {
	return HasIndexFunc.Call([]cty.Value{collection, key})
}

// Index returns an element from the given collection using the given key,
// or returns an error if there is no element for the given key.
func Index(collection cty.Value, key cty.Value) (cty.Value, error) {
	return IndexFunc.Call([]cty.Value{collection, key})
}

// Length returns the number of elements in the given collection.
func Length(collection cty.Value) (cty.Value, error) {
	return LengthFunc.Call([]cty.Value{collection})
}
