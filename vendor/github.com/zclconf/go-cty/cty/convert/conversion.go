package convert

import (
	"github.com/zclconf/go-cty/cty"
)

// conversion is an internal variant of Conversion that carries around
// a cty.Path to be used in error responses.
type conversion func(cty.Value, cty.Path) (cty.Value, error)

func getConversion(in cty.Type, out cty.Type, unsafe bool) conversion {
	conv := getConversionKnown(in, out, unsafe)
	if conv == nil {
		return nil
	}

	// Wrap the conversion in some standard checks that we don't want to
	// have to repeat in every conversion function.
	return func(in cty.Value, path cty.Path) (cty.Value, error) {
		if !in.IsKnown() {
			return cty.UnknownVal(out), nil
		}
		if in.IsNull() {
			// We'll pass through nulls, albeit type converted, and let
			// the caller deal with whatever handling they want to do in
			// case null values are considered valid in some applications.
			return cty.NullVal(out), nil
		}

		return conv(in, path)
	}
}

func getConversionKnown(in cty.Type, out cty.Type, unsafe bool) conversion {
	switch {

	case out == cty.DynamicPseudoType:
		// Conversion *to* DynamicPseudoType means that the caller wishes
		// to allow any type in this position, so we'll produce a do-nothing
		// conversion that just passes through the value as-is.
		return dynamicPassthrough

	case unsafe && in == cty.DynamicPseudoType:
		// Conversion *from* DynamicPseudoType means that we have a value
		// whose type isn't yet known during type checking. For these we will
		// assume that conversion will succeed and deal with any errors that
		// result (which is why we can only do this when "unsafe" is set).
		return dynamicFixup(out)

	case in.IsPrimitiveType() && out.IsPrimitiveType():
		conv := primitiveConversionsSafe[in][out]
		if conv != nil {
			return conv
		}
		if unsafe {
			return primitiveConversionsUnsafe[in][out]
		}
		return nil

	case out.IsListType() && (in.IsListType() || in.IsSetType()):
		inEty := in.ElementType()
		outEty := out.ElementType()
		if inEty.Equals(outEty) {
			// This indicates that we're converting from list to set with
			// the same element type, so we don't need an element converter.
			return conversionCollectionToList(outEty, nil)
		}

		convEty := getConversion(inEty, outEty, unsafe)
		if convEty == nil {
			return nil
		}
		return conversionCollectionToList(outEty, convEty)

	case out.IsSetType() && (in.IsListType() || in.IsSetType()):
		if in.IsListType() && !unsafe {
			// Conversion from list to map is unsafe because it will lose
			// information: the ordering will not be preserved, and any
			// duplicate elements will be conflated.
			return nil
		}
		inEty := in.ElementType()
		outEty := out.ElementType()
		convEty := getConversion(inEty, outEty, unsafe)
		if inEty.Equals(outEty) {
			// This indicates that we're converting from set to list with
			// the same element type, so we don't need an element converter.
			return conversionCollectionToSet(outEty, nil)
		}

		if convEty == nil {
			return nil
		}
		return conversionCollectionToSet(outEty, convEty)

	case out.IsMapType() && in.IsMapType():
		inEty := in.ElementType()
		outEty := out.ElementType()
		convEty := getConversion(inEty, outEty, unsafe)
		if convEty == nil {
			return nil
		}
		return conversionCollectionToMap(outEty, convEty)

	case out.IsListType() && in.IsTupleType():
		outEty := out.ElementType()
		return conversionTupleToList(in, outEty, unsafe)

	case out.IsMapType() && in.IsObjectType():
		outEty := out.ElementType()
		return conversionObjectToMap(in, outEty, unsafe)

	default:
		return nil

	}
}

// retConversion wraps a conversion (internal type) so it can be returned
// as a Conversion (public type).
func retConversion(conv conversion) Conversion {
	if conv == nil {
		return nil
	}

	return func(in cty.Value) (cty.Value, error) {
		return conv(in, cty.Path(nil))
	}
}
