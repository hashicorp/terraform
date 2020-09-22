package typeexpr

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// TypeConstraintType is a cty capsule type that allows cty type constraints to
// be used as values.
//
// If TypeConstraintType is used in a context supporting the
// customdecode.CustomExpressionDecoder extension then it will implement
// expression decoding using the TypeConstraint function, thus allowing
// type expressions to be used in contexts where value expressions might
// normally be expected, such as in arguments to function calls.
var TypeConstraintType cty.Type

// TypeConstraintVal constructs a cty.Value whose type is
// TypeConstraintType.
func TypeConstraintVal(ty cty.Type) cty.Value {
	return cty.CapsuleVal(TypeConstraintType, &ty)
}

// TypeConstraintFromVal extracts the type from a cty.Value of
// TypeConstraintType that was previously constructed using TypeConstraintVal.
//
// If the given value isn't a known, non-null value of TypeConstraintType
// then this function will panic.
func TypeConstraintFromVal(v cty.Value) cty.Type {
	if !v.Type().Equals(TypeConstraintType) {
		panic("value is not of TypeConstraintType")
	}
	ptr := v.EncapsulatedValue().(*cty.Type)
	return *ptr
}

// ConvertFunc is a cty function that implements type conversions.
//
// Its signature is as follows:
//     convert(value, type_constraint)
//
// ...where type_constraint is a type constraint expression as defined by
// typeexpr.TypeConstraint.
//
// It relies on HCL's customdecode extension and so it's not suitable for use
// in non-HCL contexts or if you are using a HCL syntax implementation that
// does not support customdecode for function arguments. However, it _is_
// supported for function calls in the HCL native expression syntax.
var ConvertFunc function.Function

func init() {
	TypeConstraintType = cty.CapsuleWithOps("type constraint", reflect.TypeOf(cty.Type{}), &cty.CapsuleOps{
		ExtensionData: func(key interface{}) interface{} {
			switch key {
			case customdecode.CustomExpressionDecoder:
				return customdecode.CustomExpressionDecoderFunc(
					func(expr hcl.Expression, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
						ty, diags := TypeConstraint(expr)
						if diags.HasErrors() {
							return cty.NilVal, diags
						}
						return TypeConstraintVal(ty), nil
					},
				)
			default:
				return nil
			}
		},
		TypeGoString: func(_ reflect.Type) string {
			return "typeexpr.TypeConstraintType"
		},
		GoString: func(raw interface{}) string {
			tyPtr := raw.(*cty.Type)
			return fmt.Sprintf("typeexpr.TypeConstraintVal(%#v)", *tyPtr)
		},
		RawEquals: func(a, b interface{}) bool {
			aPtr := a.(*cty.Type)
			bPtr := b.(*cty.Type)
			return (*aPtr).Equals(*bPtr)
		},
	})

	ConvertFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "value",
				Type:             cty.DynamicPseudoType,
				AllowNull:        true,
				AllowDynamicType: true,
			},
			{
				Name: "type",
				Type: TypeConstraintType,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			wantTypePtr := args[1].EncapsulatedValue().(*cty.Type)
			got, err := convert.Convert(args[0], *wantTypePtr)
			if err != nil {
				return cty.NilType, function.NewArgError(0, err)
			}
			return got.Type(), nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			v, err := convert.Convert(args[0], retType)
			if err != nil {
				return cty.NilVal, function.NewArgError(0, err)
			}
			return v, nil
		},
	})
}
