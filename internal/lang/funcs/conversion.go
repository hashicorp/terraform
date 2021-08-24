package funcs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/internal/lang/marks"
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
				Type:        cty.DynamicPseudoType,
				AllowNull:   true,
				AllowMarked: true,
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
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(TypeString(args[0].Type())).Mark(marks.Raw), nil
	},
})

// Modified copy of TypeString from go-cty:
// https://github.com/zclconf/go-cty-debug/blob/master/ctydebug/type_string.go
//
// TypeString returns a string representation of a given type that is
// reminiscent of Go syntax calling into the cty package but is mainly
// intended for easy human inspection of values in tests, debug output, etc.
//
// The resulting string will include newlines and indentation in order to
// increase the readability of complex structures. It always ends with a
// newline, so you can print this result directly to your output.
func TypeString(ty cty.Type) string {
	var b strings.Builder
	writeType(ty, &b, 0)
	return b.String()
}

func writeType(ty cty.Type, b *strings.Builder, indent int) {
	switch {
	case ty == cty.NilType:
		b.WriteString("nil")
		return
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		if len(atys) == 0 {
			b.WriteString("object({})")
			return
		}
		attrNames := make([]string, 0, len(atys))
		for name := range atys {
			attrNames = append(attrNames, name)
		}
		sort.Strings(attrNames)
		b.WriteString("object({\n")
		indent++
		for _, name := range attrNames {
			aty := atys[name]
			b.WriteString(indentSpaces(indent))
			fmt.Fprintf(b, "%s: ", name)
			writeType(aty, b, indent)
			b.WriteString(",\n")
		}
		indent--
		b.WriteString(indentSpaces(indent))
		b.WriteString("})")
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		if len(etys) == 0 {
			b.WriteString("tuple([])")
			return
		}
		b.WriteString("tuple([\n")
		indent++
		for _, ety := range etys {
			b.WriteString(indentSpaces(indent))
			writeType(ety, b, indent)
			b.WriteString(",\n")
		}
		indent--
		b.WriteString(indentSpaces(indent))
		b.WriteString("])")
	case ty.IsCollectionType():
		ety := ty.ElementType()
		switch {
		case ty.IsListType():
			b.WriteString("list(")
		case ty.IsMapType():
			b.WriteString("map(")
		case ty.IsSetType():
			b.WriteString("set(")
		default:
			// At the time of writing there are no other collection types,
			// but we'll be robust here and just pass through the GoString
			// of anything we don't recognize.
			b.WriteString(ty.FriendlyName())
			return
		}
		// Because object and tuple types render split over multiple
		// lines, a collection type container around them can end up
		// being hard to see when scanning, so we'll generate some extra
		// indentation to make a collection of structural type more visually
		// distinct from the structural type alone.
		complexElem := ety.IsObjectType() || ety.IsTupleType()
		if complexElem {
			indent++
			b.WriteString("\n")
			b.WriteString(indentSpaces(indent))
		}
		writeType(ty.ElementType(), b, indent)
		if complexElem {
			indent--
			b.WriteString(",\n")
			b.WriteString(indentSpaces(indent))
		}
		b.WriteString(")")
	default:
		// For any other type we'll just use its GoString and assume it'll
		// follow the usual GoString conventions.
		b.WriteString(ty.FriendlyName())
	}
}

func indentSpaces(level int) string {
	return strings.Repeat("    ", level)
}

func Type(input []cty.Value) (cty.Value, error) {
	return TypeFunc.Call(input)
}
