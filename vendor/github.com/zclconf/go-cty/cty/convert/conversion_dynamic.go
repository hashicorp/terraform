package convert

import (
	"github.com/zclconf/go-cty/cty"
)

// dynamicFixup deals with just-in-time conversions of values that were
// input-typed as cty.DynamicPseudoType during analysis, ensuring that
// we end up with the desired output type once the value is known, or
// failing with an error if that is not possible.
//
// This is in the spirit of the cty philosophy of optimistically assuming that
// DynamicPseudoType values will become the intended value eventually, and
// dealing with any inconsistencies during final evaluation.
func dynamicFixup(wantType cty.Type) conversion {
	return func(in cty.Value, path cty.Path) (cty.Value, error) {
		ret, err := Convert(in, wantType)
		if err != nil {
			// Re-wrap this error so that the returned path is relative
			// to the caller's original value, rather than relative to our
			// conversion value here.
			return cty.NilVal, path.NewError(err)
		}
		return ret, nil
	}
}

// dynamicPassthrough is an identity conversion that is used when the
// target type is DynamicPseudoType, indicating that the caller doesn't care
// which type is returned.
func dynamicPassthrough(in cty.Value, path cty.Path) (cty.Value, error) {
	return in, nil
}
