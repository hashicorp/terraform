package stdlib

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

var ConcatFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name: "seqs",
		Type: cty.DynamicPseudoType,
	},
	Type: func(args []cty.Value) (ret cty.Type, err error) {
		if len(args) == 0 {
			return cty.NilType, fmt.Errorf("at least one argument is required")
		}

		if args[0].Type().IsListType() {
			// Possibly we're going to return a list, if all of our other
			// args are also lists and we can find a common element type.
			tys := make([]cty.Type, len(args))
			for i, val := range args {
				ty := val.Type()
				if !ty.IsListType() {
					tys = nil
					break
				}
				tys[i] = ty
			}

			if tys != nil {
				commonType, _ := convert.UnifyUnsafe(tys)
				if commonType != cty.NilType {
					return commonType, nil
				}
			}
		}

		etys := make([]cty.Type, 0, len(args))
		for i, val := range args {
			ety := val.Type()
			switch {
			case ety.IsTupleType():
				etys = append(etys, ety.TupleElementTypes()...)
			case ety.IsListType():
				if !val.IsKnown() {
					// We need to know the list to count its elements to
					// build our tuple type, so any concat of an unknown
					// list can't be typed yet.
					return cty.DynamicPseudoType, nil
				}

				l := val.LengthInt()
				subEty := ety.ElementType()
				for j := 0; j < l; j++ {
					etys = append(etys, subEty)
				}
			default:
				return cty.NilType, function.NewArgErrorf(
					i, "all arguments must be lists or tuples; got %s",
					ety.FriendlyName(),
				)
			}
		}
		return cty.Tuple(etys), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		switch {
		case retType.IsListType():
			// If retType is a list type then we know that all of the
			// given values will be lists and that they will either be of
			// retType or of something we can convert to retType.
			vals := make([]cty.Value, 0, len(args))
			for i, list := range args {
				list, err = convert.Convert(list, retType)
				if err != nil {
					// Conversion might fail because we used UnifyUnsafe
					// to choose our return type.
					return cty.NilVal, function.NewArgError(i, err)
				}

				it := list.ElementIterator()
				for it.Next() {
					_, v := it.Element()
					vals = append(vals, v)
				}
			}
			if len(vals) == 0 {
				return cty.ListValEmpty(retType.ElementType()), nil
			}

			return cty.ListVal(vals), nil
		case retType.IsTupleType():
			// If retType is a tuple type then we could have a mixture of
			// lists and tuples but we know they all have known values
			// (because our params don't AllowUnknown) and we know that
			// concatenating them all together will produce a tuple of
			// retType because of the work we did in the Type function above.
			vals := make([]cty.Value, 0, len(args))

			for _, seq := range args {
				// Both lists and tuples support ElementIterator, so this is easy.
				it := seq.ElementIterator()
				for it.Next() {
					_, v := it.Element()
					vals = append(vals, v)
				}
			}

			return cty.TupleVal(vals), nil
		default:
			// should never happen if Type is working correctly above
			panic("unsupported return type")
		}
	},
})

// Concat takes one or more sequences (lists or tuples) and returns the single
// sequence that results from concatenating them together in order.
//
// If all of the given sequences are lists of the same element type then the
// result is a list of that type. Otherwise, the result is a of a tuple type
// constructed from the given sequence types.
func Concat(seqs ...cty.Value) (cty.Value, error) {
	return ConcatFunc.Call(seqs)
}
