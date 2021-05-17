package funcs

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// DefaultsFunc is a helper function for substituting default values in
// place of null values in a given data structure.
//
// See the documentation for function Defaults for more information.
var DefaultsFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:        "input",
			Type:        cty.DynamicPseudoType,
			AllowNull:   true,
			AllowMarked: true,
		},
		{
			Name:        "defaults",
			Type:        cty.DynamicPseudoType,
			AllowMarked: true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		// The result type is guaranteed to be the same as the input type,
		// since all we're doing is replacing null values with non-null
		// values of the same type.
		retType := args[0].Type()
		defaultsType := args[1].Type()

		// This function is aimed at filling in object types or collections
		// of object types where some of the attributes might be null, so
		// it doesn't make sense to use a primitive type directly with it.
		// (The "coalesce" function may be appropriate for such cases.)
		if retType.IsPrimitiveType() {
			// This error message is a bit of a fib because we can actually
			// apply defaults to tuples too, but we expect that to be so
			// unusual as to not be worth mentioning here, because mentioning
			// it would require using some less-well-known Terraform language
			// terminology in the message (tuple types, structural types).
			return cty.DynamicPseudoType, function.NewArgErrorf(1, "only object types and collections of object types can have defaults applied")
		}

		defaultsPath := make(cty.Path, 0, 4) // some capacity so that most structures won't reallocate
		if err := defaultsAssertSuitableFallback(retType, defaultsType, defaultsPath); err != nil {
			errMsg := tfdiags.FormatError(err) // add attribute path prefix
			return cty.DynamicPseudoType, function.NewArgErrorf(1, "%s", errMsg)
		}

		return retType, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if args[0].Type().HasDynamicTypes() {
			// If the types our input object aren't known yet for some reason
			// then we'll defer all of our work here, because our
			// interpretation of the defaults depends on the types in
			// the input.
			return cty.UnknownVal(retType), nil
		}

		v := defaultsApply(args[0], args[1])
		return v, nil
	},
})

func defaultsApply(input, fallback cty.Value) cty.Value {
	wantTy := input.Type()

	umInput, inputMarks := input.Unmark()
	umFb, fallbackMarks := fallback.Unmark()

	// If neither are known, we very conservatively return an unknown value
	// with the union of marks on both input and default.
	if !(umInput.IsKnown() && umFb.IsKnown()) {
		return cty.UnknownVal(wantTy).WithMarks(inputMarks).WithMarks(fallbackMarks)
	}

	// For the rest of this function we're assuming that the given defaults
	// will always be valid, because we expect to have caught any problems
	// during the type checking phase. Any inconsistencies that reach here are
	// therefore considered to be implementation bugs, and so will panic.

	// Our strategy depends on the kind of type we're working with.
	switch {
	case wantTy.IsPrimitiveType():
		// For leaf primitive values the rule is relatively simple: use the
		// input if it's non-null, or fallback if input is null.
		if !umInput.IsNull() {
			return input
		}
		v, err := convert.Convert(umFb, wantTy)
		if err != nil {
			// Should not happen because we checked in defaultsAssertSuitableFallback
			panic(err.Error())
		}
		return v.WithMarks(fallbackMarks)

	case wantTy.IsObjectType():
		// For structural types, a null input value must be passed through. We
		// do not apply default values for missing optional structural values,
		// only their contents.
		//
		// We also pass through the input if the fallback value is null. This
		// can happen if the given defaults do not include a value for this
		// attribute.
		if umInput.IsNull() || umFb.IsNull() {
			return input
		}
		atys := wantTy.AttributeTypes()
		ret := map[string]cty.Value{}
		for attr, aty := range atys {
			inputSub := umInput.GetAttr(attr)
			fallbackSub := cty.NullVal(aty)
			if umFb.Type().HasAttribute(attr) {
				fallbackSub = umFb.GetAttr(attr)
			}
			ret[attr] = defaultsApply(inputSub.WithMarks(inputMarks), fallbackSub.WithMarks(fallbackMarks))
		}
		return cty.ObjectVal(ret)

	case wantTy.IsTupleType():
		// For structural types, a null input value must be passed through. We
		// do not apply default values for missing optional structural values,
		// only their contents.
		//
		// We also pass through the input if the fallback value is null. This
		// can happen if the given defaults do not include a value for this
		// attribute.
		if umInput.IsNull() || umFb.IsNull() {
			return input
		}

		l := wantTy.Length()
		ret := make([]cty.Value, l)
		for i := 0; i < l; i++ {
			inputSub := umInput.Index(cty.NumberIntVal(int64(i)))
			fallbackSub := umFb.Index(cty.NumberIntVal(int64(i)))
			ret[i] = defaultsApply(inputSub.WithMarks(inputMarks), fallbackSub.WithMarks(fallbackMarks))
		}
		return cty.TupleVal(ret)

	case wantTy.IsCollectionType():
		// For collection types we apply a single fallback value to each
		// element of the input collection, because in the situations this
		// function is intended for we assume that the number of elements
		// is the caller's decision, and so we'll just apply the same defaults
		// to all of the elements.
		ety := wantTy.ElementType()
		switch {
		case wantTy.IsMapType():
			newVals := map[string]cty.Value{}

			if !umInput.IsNull() {
				for it := umInput.ElementIterator(); it.Next(); {
					k, v := it.Element()
					newVals[k.AsString()] = defaultsApply(v.WithMarks(inputMarks), fallback.WithMarks(fallbackMarks))
				}
			}

			if len(newVals) == 0 {
				return cty.MapValEmpty(ety)
			}
			return cty.MapVal(newVals)
		case wantTy.IsListType(), wantTy.IsSetType():
			var newVals []cty.Value

			if !umInput.IsNull() {
				for it := umInput.ElementIterator(); it.Next(); {
					_, v := it.Element()
					newV := defaultsApply(v.WithMarks(inputMarks), fallback.WithMarks(fallbackMarks))
					newVals = append(newVals, newV)
				}
			}

			if len(newVals) == 0 {
				if wantTy.IsSetType() {
					return cty.SetValEmpty(ety)
				}
				return cty.ListValEmpty(ety)
			}
			if wantTy.IsSetType() {
				return cty.SetVal(newVals)
			}
			return cty.ListVal(newVals)
		default:
			// There are no other collection types, so this should not happen
			panic(fmt.Sprintf("invalid collection type %#v", wantTy))
		}
	default:
		// We should've caught anything else in defaultsAssertSuitableFallback,
		// so this should not happen.
		panic(fmt.Sprintf("invalid target type %#v", wantTy))
	}
}

func defaultsAssertSuitableFallback(wantTy, fallbackTy cty.Type, fallbackPath cty.Path) error {
	// If the type we want is a collection type then we need to keep peeling
	// away collection type wrappers until we find the non-collection-type
	// that's underneath, which is what the fallback will actually be applied
	// to.
	inCollection := false
	for wantTy.IsCollectionType() {
		wantTy = wantTy.ElementType()
		inCollection = true
	}

	switch {
	case wantTy.IsPrimitiveType():
		// The fallback is valid if it's equal to or convertible to what we want.
		if fallbackTy.Equals(wantTy) {
			return nil
		}
		conversion := convert.GetConversion(fallbackTy, wantTy)
		if conversion == nil {
			msg := convert.MismatchMessage(fallbackTy, wantTy)
			return fallbackPath.NewErrorf("invalid default value for %s: %s", wantTy.FriendlyName(), msg)
		}
		return nil
	case wantTy.IsObjectType():
		if !fallbackTy.IsObjectType() {
			if inCollection {
				return fallbackPath.NewErrorf("the default value for a collection of an object type must itself be an object type, not %s", fallbackTy.FriendlyName())
			}
			return fallbackPath.NewErrorf("the default value for an object type must itself be an object type, not %s", fallbackTy.FriendlyName())
		}
		for attr, wantAty := range wantTy.AttributeTypes() {
			if !fallbackTy.HasAttribute(attr) {
				continue // it's always okay to not have a default value
			}
			fallbackSubpath := fallbackPath.GetAttr(attr)
			fallbackSubTy := fallbackTy.AttributeType(attr)
			err := defaultsAssertSuitableFallback(wantAty, fallbackSubTy, fallbackSubpath)
			if err != nil {
				return err
			}
		}
		for attr := range fallbackTy.AttributeTypes() {
			if !wantTy.HasAttribute(attr) {
				fallbackSubpath := fallbackPath.GetAttr(attr)
				return fallbackSubpath.NewErrorf("target type does not expect an attribute named %q", attr)
			}
		}
		return nil
	case wantTy.IsTupleType():
		if !fallbackTy.IsTupleType() {
			if inCollection {
				return fallbackPath.NewErrorf("the default value for a collection of a tuple type must itself be a tuple type, not %s", fallbackTy.FriendlyName())
			}
			return fallbackPath.NewErrorf("the default value for a tuple type must itself be a tuple type, not %s", fallbackTy.FriendlyName())
		}
		wantEtys := wantTy.TupleElementTypes()
		fallbackEtys := fallbackTy.TupleElementTypes()
		if got, want := len(wantEtys), len(fallbackEtys); got != want {
			return fallbackPath.NewErrorf("the default value for a tuple type of length %d must also have length %d, not %d", want, want, got)
		}
		for i := 0; i < len(wantEtys); i++ {
			fallbackSubpath := fallbackPath.IndexInt(i)
			wantSubTy := wantEtys[i]
			fallbackSubTy := fallbackEtys[i]
			err := defaultsAssertSuitableFallback(wantSubTy, fallbackSubTy, fallbackSubpath)
			if err != nil {
				return err
			}
		}
		return nil
	default:
		// No other types are supported right now.
		return fallbackPath.NewErrorf("cannot apply defaults to %s", wantTy.FriendlyName())
	}
}

// Defaults is a helper function for substituting default values in
// place of null values in a given data structure.
//
// This is primarily intended for use with a module input variable that
// has an object type constraint (or a collection thereof) that has optional
// attributes, so that the receiver of a value that omits those attributes
// can insert non-null default values in place of the null values caused by
// omitting the attributes.
func Defaults(input, defaults cty.Value) (cty.Value, error) {
	return DefaultsFunc.Call([]cty.Value{input, defaults})
}
