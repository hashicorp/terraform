package convert

import (
	"github.com/zclconf/go-cty/cty"
)

// The current unify implementation is somewhat inefficient, but we accept this
// under the assumption that it will generally be used with small numbers of
// types and with types of reasonable complexity. However, it does have a
// "happy path" where all of the given types are equal.
//
// This function is likely to have poor performance in cases where any given
// types are very complex (lots of deeply-nested structures) or if the list
// of types itself is very large. In particular, it will walk the nested type
// structure under the given types several times, especially when given a
// list of types for which unification is not possible, since each permutation
// will be tried to determine that result.
func unify(types []cty.Type, unsafe bool) (cty.Type, []Conversion) {
	if len(types) == 0 {
		// Degenerate case
		return cty.NilType, nil
	}

	prefOrder := sortTypes(types)

	// sortTypes gives us an order where earlier items are preferable as
	// our result type. We'll now walk through these and choose the first
	// one we encounter for which conversions exist for all source types.
	conversions := make([]Conversion, len(types))
Preferences:
	for _, wantTypeIdx := range prefOrder {
		wantType := types[wantTypeIdx]
		for i, tryType := range types {
			if i == wantTypeIdx {
				// Don't need to convert our wanted type to itself
				conversions[i] = nil
				continue
			}

			if tryType.Equals(wantType) {
				conversions[i] = nil
				continue
			}

			if unsafe {
				conversions[i] = GetConversionUnsafe(tryType, wantType)
			} else {
				conversions[i] = GetConversion(tryType, wantType)
			}

			if conversions[i] == nil {
				// wantType is not a suitable unification type, so we'll
				// try the next one in our preference order.
				continue Preferences
			}
		}

		return wantType, conversions
	}

	// TODO: For structural types, try to invent a new type that they
	// can all be unified to, by unifying their respective attributes.

	// If we fall out here, no unification is possible
	return cty.NilType, nil
}
