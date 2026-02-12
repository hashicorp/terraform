// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/lang/types"
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
				Type:             cty.DynamicPseudoType,
				AllowNull:        true,
				AllowMarked:      true,
				AllowDynamicType: true,
				AllowUnknown:     true,
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
			if !args[0].IsKnown() {
				return cty.UnknownVal(retType).WithSameMarks(args[0]), nil
			}

			ret, err := convert.Convert(args[0], retType)
			if err != nil {
				val, _ := args[0].UnmarkDeep()
				// Because we used GetConversionUnsafe above, conversion can
				// still potentially fail in here. For example, if the user
				// asks to convert the string "a" to bool then we'll
				// optimistically permit it during type checking but fail here
				// once we note that the value isn't either "true" or "false".
				gotTy := val.Type()
				switch {
				case marks.Contains(args[0], marks.Sensitive):
					// Generic message so we won't inadvertently disclose
					// information about sensitive values.
					return cty.NilVal, function.NewArgErrorf(0, "cannot convert this sensitive %s to %s", gotTy.FriendlyName(), wantTy.FriendlyNameForConstraint())

				case gotTy == cty.String && wantTy == cty.Bool:
					what := "string"
					if !val.IsNull() {
						what = strconv.Quote(val.AsString())
					}
					return cty.NilVal, function.NewArgErrorf(0, `cannot convert %s to bool; only the strings "true" or "false" are allowed`, what)
				case gotTy == cty.String && wantTy == cty.Number:
					what := "string"
					if !val.IsNull() {
						what = strconv.Quote(val.AsString())
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

// EphemeralAsNullFunc is a cty function that takes a value of any type and
// returns a similar value with any ephemeral-marked values anywhere in the
// structure replaced with a null value of the same type that is not marked
// as ephemeral.
//
// This is intended as a convenience for returning the non-ephemeral parts of
// a partially-ephemeral data structure through an output value that isn't
// ephemeral itself.
var EphemeralAsNullFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
			AllowUnknown:     true,
			AllowNull:        true,
			AllowMarked:      true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		// This function always preserves the type of the given argument.
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return ephemeral.RemoveEphemeralValues(args[0]), nil
	},
})

func EphemeralAsNull(input cty.Value) (cty.Value, error) {
	return EphemeralAsNullFunc.Call([]cty.Value{input})
}

// TypeFunc returns an encapsulated value containing its argument's type. This
// value is marked to allow us to limit the use of this function at the moment
// to only a few supported use cases.
var TypeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
			AllowUnknown:     true,
			AllowNull:        true,
		},
	},
	Type: function.StaticReturnType(types.TypeType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		givenType := args[0].Type()
		return cty.CapsuleVal(types.TypeType, &givenType).Mark(marks.TypeType), nil
	},
})

func Type(input []cty.Value) (cty.Value, error) {
	return TypeFunc.Call(input)
}

// ConvertFunc is a cty function which takes any value as the first argument,
// and returns the result of converting the first argument to the type
// constraint literal given as the second argument. We allow type constraint
// literals by injecting a custom decoder into HCL using a cty capsule type.
var ConvertFunc = makeConvertFunc()

// makeConvertFunc is a constructor function because of the unusual method we
// have for passing a custom decoder into HCL. We need to be able to declare a
// recursive closure that can return the same value that it's assigned to, hence
// there needs some procedural code to construct it.
func makeConvertFunc() function.Function {
	// We want to be able to use optional and default values in our type
	// constrains, so we need to be able to track both the type and the default
	// values.
	type typeConstraintArg struct {
		Type     cty.Type
		Defaults *typeexpr.Defaults
	}

	var typeConstraintType cty.Type
	typeConstraintType = cty.CapsuleWithOps("type_constraint", reflect.TypeFor[typeConstraintArg](), &cty.CapsuleOps{
		ExtensionData: func(key any) any {
			switch key {
			// HCL will look for a capsule with CustomExpressionDecoder when
			// decoding function arguments, and then insert this decoder
			// allowing us to use our standard type expression syntax.
			case customdecode.CustomExpressionDecoder:
				return customdecode.CustomExpressionDecoderFunc(
					func(expr hcl.Expression, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
						ty, defs, diags := typeexpr.TypeConstraintWithDefaults(expr)
						if diags.HasErrors() {
							return cty.NilVal, diags
						}
						return cty.CapsuleVal(typeConstraintType, &typeConstraintArg{Type: ty, Defaults: defs}), nil
					},
				)
			default:
				return nil
			}
		},
		TypeGoString: func(_ reflect.Type) string {
			return "typeConstraint"
		},
		GoString: func(raw any) string {
			tyPtr := raw.(*typeConstraintArg)
			// The GoString value from our constraint will suffice here.
			return fmt.Sprintf("typeConstraint(%#v)", tyPtr.Type)
		},
	})

	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "value",
				Type:             cty.DynamicPseudoType,
				AllowNull:        true,
				AllowDynamicType: true,
			},
			{
				Name: "type",
				Type: typeConstraintType,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			constraint := args[1].EncapsulatedValue().(*typeConstraintArg)
			// optional attributes are only used during the conversion process,
			// the final type must be fully defined.
			return constraint.Type.WithoutOptionalAttributesDeep(), nil
		},
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			// the retType parameter tells us the final type, but it does not
			// contain optional attributes or defaults, so we need to extract
			// our typeConstraintArg from the arguments again.
			constraint := args[1].EncapsulatedValue().(*typeConstraintArg)
			v, err := convert.Convert(args[0], constraint.Type)
			if err != nil {
				return cty.NilVal, function.NewArgError(0, err)
			}
			if constraint.Defaults != nil {
				v = constraint.Defaults.Apply(v)
			}

			return v, nil
		},
	})
}
