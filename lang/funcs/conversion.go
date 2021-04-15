package funcs

import (
	"strconv"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// MakeToFunc constructs a "to..." function, like "tostring", which converts
// its argument to a specific type or type kind.
//
// The given type wantTy can be any type constraint that cty's "convert" package
// would accept. In particular, this means that you can pass
// cty.List(cty.DynamicPseudoType) to mean "list of any single type", which
// will then cause cty to attempt to unify all of the element types when given
// a tuple.
func MakeToFunc(wantTy cty.Type) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "v",
				// We use DynamicPseudoType rather than wantTy here so that
				// all values will pass through the function API verbatim and
				// we can handle the conversion logic within the Type and
				// Impl functions. This allows us to customize the error
				// messages to be more appropriate for an explicit type
				// conversion, whereas the cty function system produces
				// messages aimed at _implicit_ type conversions.
				Type:      cty.DynamicPseudoType,
				AllowNull: true,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			gotTy := args[0].Type()
			if gotTy.Equals(wantTy) {
				return wantTy, nil
			}
			conv := convert.GetConversionUnsafe(args[0].Type(), wantTy)
			if conv == nil {
				// We'll use some specialized errors for some trickier cases,
				// but most we can handle in a simple way.
				switch {
				case gotTy.IsTupleType() && wantTy.IsTupleType():
					return cty.NilType, function.NewArgErrorf(0, "incompatible tuple type for conversion: %s", convert.MismatchMessage(gotTy, wantTy))
				case gotTy.IsObjectType() && wantTy.IsObjectType():
					return cty.NilType, function.NewArgErrorf(0, "incompatible object type for conversion: %s", convert.MismatchMessage(gotTy, wantTy))
				default:
					return cty.NilType, function.NewArgErrorf(0, "cannot convert %s to %s", gotTy.FriendlyName(), wantTy.FriendlyNameForConstraint())
				}
			}
			// If a conversion is available then everything is fine.
			return wantTy, nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// We didn't set "AllowUnknown" on our argument, so it is guaranteed
			// to be known here but may still be null.
			ret, err := convert.Convert(args[0], retType)
			if err != nil {
				// Because we used GetConversionUnsafe above, conversion can
				// still potentially fail in here. For example, if the user
				// asks to convert the string "a" to bool then we'll
				// optimistically permit it during type checking but fail here
				// once we note that the value isn't either "true" or "false".
				gotTy := args[0].Type()
				switch {
				case gotTy == cty.String && wantTy == cty.Bool:
					what := "string"
					if !args[0].IsNull() {
						what = strconv.Quote(args[0].AsString())
					}
					return cty.NilVal, function.NewArgErrorf(0, `cannot convert %s to bool; only the strings "true" or "false" are allowed`, what)
				case gotTy == cty.String && wantTy == cty.Number:
					what := "string"
					if !args[0].IsNull() {
						what = strconv.Quote(args[0].AsString())
					}
					return cty.NilVal, function.NewArgErrorf(0, `cannot convert %s to number; given string must be a decimal representation of a number`, what)
				default:
					return cty.NilVal, function.NewArgErrorf(0, "cannot convert %s to %s", gotTy.FriendlyName(), wantTy.FriendlyNameForConstraint())
				}
			}
			return ret, nil
		},
	})
}

// ExpandNullFunc is a specialized function that recursively walks a given
// data structure looking for null values of structural types or collection
// types.
//
// It will then replace each structural type with a non-null instance of that
// type whose attributes are all either null or recursively expanded, and
// each collection type with an empty collection of that type.
//
// This is intended as a helper function for modules that accept complex
// data structures containing optional attributes where they'd rather push
// all of the "null-ness" to the primitive-typed leaves of the data structure
// in order to traverse it more easily elsewhere in the module.
var ExpandNullFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:      "v",
			Type:      cty.DynamicPseudoType,
			AllowNull: true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		// This function only replaces null values with non-null values of
		// the same type, so the return type is always the same.
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.Transform(args[0], func(path cty.Path, v cty.Value) (cty.Value, error) {
			if !v.IsKnown() || !v.IsNull() {
				// We're only interested in known, null values here
				return v, nil
			}
			return expandedNull(v.Type()), nil
		})
	},
})

func expandedNull(ty cty.Type) cty.Value {
	switch {
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		attrs := make(map[string]cty.Value, len(atys))
		for name, aty := range atys {
			attrs[name] = expandedNull(aty)
		}
		return cty.ObjectVal(attrs)
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		elems := make([]cty.Value, len(etys))
		for i, ety := range etys {
			elems[i] = expandedNull(ety)
		}
		return cty.TupleVal(elems)
	case ty.IsListType():
		return cty.ListValEmpty(ty.ElementType())
	case ty.IsMapType():
		return cty.MapValEmpty(ty.ElementType())
	case ty.IsSetType():
		return cty.SetValEmpty(ty.ElementType())
	default:
		// Otherwise we just return a direct null of the requested type
		return cty.NullVal(ty)
	}
}
