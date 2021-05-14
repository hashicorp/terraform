package templatevals

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// Type constructs a cty type representing a lazily-evaluated template.
//
// Template types are parameterized by their set of required argument names and
// associated type constraints.
func Type(atys map[string]cty.Type) cty.Type {
	var ret cty.Type
	ops := &cty.CapsuleOps{
		TypeGoString: func(goTy reflect.Type) string {
			return fmt.Sprintf("templatevals.Type(%#v)", atys)
		},
		GoString: func(rv interface{}) string {
			tv := rv.(*templateVal)
			return fmt.Sprintf("templatevals.Val(%#v, %#v, %#v)", atys, tv.expr, tv.ctx)
		},
		ConversionFrom: func(dest cty.Type) func(interface{}, cty.Path) (cty.Value, error) {
			if !IsTemplateType(dest) {
				// Can only convert between template types
				return nil
			}

			// There are some other constraints on successful conversion but
			// we'll wait until inside the conversion function to deal with
			// those, so we can return specialized errors.
			dstAtys := TypeArgs(dest)
			return func(rv interface{}, path cty.Path) (cty.Value, error) {
				// Conversion is allowed only if the destination arguments are
				// all assignable to the source arguments, thus making the
				// result potentially _more_ constrained in what arguments
				// the template can accept. To check that we make some
				// temporary object types to borrow the object type conversion
				// behavior.
				srcObjTy := cty.Object(atys)
				dstObjTy := cty.Object(dstAtys)
				if srcObjTy.Equals(dstObjTy) {
					// Easy case then: the two types are equivalent.
				} else if conv := convert.GetConversionUnsafe(dstObjTy, srcObjTy); conv != nil {
					// Also valid, from an interface-conformance perspective
					// (note that dst and src are intentionally inverted above
					// because we're effectively testing if an object of the
					// destination type (the final template arguments) would be
					// assignable to the source type (the arguments that the
					// template actually expects.)
				} else {
					// TODO: A better error message, saying something about
					// what is wrong.
					return cty.NilVal, path.NewErrorf("incompatible template arguments")
				}

				// Even if the static type information suggests compatibility,
				// our template argument constraints start off very broad
				// at the point of definition (everything is "any") and
				// constraining further requires that the template can pass
				// type checking when given arguments of the destination
				// types.
				tv := rv.(*templateVal)
				expr := tv.expr
				parentCtx := tv.ctx
				ctx := parentCtx.NewChild()
				ctx.Variables = map[string]cty.Value{
					"template": cty.UnknownVal(dstObjTy),
				}
				_, diags := expr.Value(ctx)
				if diags.HasErrors() {
					// TODO: Again, a better error message. Doing better here
					// probably in practice means trying more surgical type
					// checks with only one argument at a time set to a
					// specific type constraint, to see which ones fail and
					// which ones succeed. Although even that wouldn't be
					// 100% because it might be the combination of two
					// arguments that makes it invalid!
					return cty.NilVal, path.NewErrorf("incompatible usage of template arguments")
				}

				// If we get down here without returning an error then this
				// conversion seems acceptable from a type-checking standpoint,
				// and so we can wrap our same templateVal value up in the
				// destination type.
				// Note that the resulting template might still fail for
				// dynamic reasons, e.g. if it's expecting valid JSON but
				// doesn't _get_ valid JSON, but we'll catch that sort of
				// problem at evaluation time.
				return cty.CapsuleVal(dest, tv), nil
			}
		},
		ExtensionData: func(key interface{}) interface{} {
			switch key {
			case templateTypeAtys:
				return atys
			default:
				return nil
			}
		},
	}
	ret = cty.CapsuleWithOps("template", templateValReflect, ops)
	return ret
}

// Val constructs a new value of a template type with the given expression and
// evaluation context.
//
// The given type must be a template type, or this function will panic.
//
// This function does no validation of whether the given expression and context
// are compatible with one another or whether the the expression can support
// the given argument types. The caller must guarantee such compatibility.
func Val(ty cty.Type, expr hcl.Expression, ctx *hcl.EvalContext) cty.Value {
	if !IsTemplateType(ty) {
		panic(fmt.Sprintf("can't construct template value of non-template type %#v", ty))
	}
	rv := &templateVal{
		expr: expr,
		ctx:  ctx,
	}
	return cty.CapsuleVal(ty, rv)
}

func IsTemplateType(ty cty.Type) bool {
	if !ty.IsCapsuleType() {
		return false
	}
	return ty.EncapsulatedType() == templateValReflect
}

// TypeArgs returns the arguments and their associated types for the given
// type, which must be a template type or this function will panic.
//
// Do not modify the returned array. It is part of the internal state of
// the template type.
func TypeArgs(ty cty.Type) map[string]cty.Type {
	if !IsTemplateType(ty) {
		panic("templatevals.TypeArgs on non-template type")
	}
	return ty.CapsuleExtensionData(templateTypeAtys).(map[string]cty.Type)
}

func IsTemplateVal(v cty.Value) bool {
	return IsTemplateType(v.Type())
}

type templateVal struct {
	expr hcl.Expression
	ctx  *hcl.EvalContext
}

var templateValReflect = reflect.TypeOf(templateVal{})

type templateTypeAtysKey int

var templateTypeAtys templateTypeAtysKey = 0
