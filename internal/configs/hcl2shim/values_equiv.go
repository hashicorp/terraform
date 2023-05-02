// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcl2shim

import (
	"github.com/zclconf/go-cty/cty"
)

// ValuesSDKEquivalent returns true if both of the given values seem equivalent
// as far as the legacy SDK diffing code would be concerned.
//
// Since SDK diffing is a fuzzy, inexact operation, this function is also
// fuzzy and inexact. It will err on the side of returning false if it
// encounters an ambiguous situation. Ambiguity is most common in the presence
// of sets because in practice it is impossible to exactly correlate
// nonequal-but-equivalent set elements because they have no identity separate
// from their value.
//
// This must be used _only_ for comparing values for equivalence within the
// SDK planning code. It is only meaningful to compare the "prior state"
// provided by Terraform Core with the "planned new state" produced by the
// legacy SDK code via shims. In particular it is not valid to use this
// function with their the config value or the "proposed new state" value
// because they contain only the subset of data that Terraform Core itself is
// able to determine.
func ValuesSDKEquivalent(a, b cty.Value) bool {
	if a == cty.NilVal || b == cty.NilVal {
		// We don't generally expect nils to appear, but we'll allow them
		// for robustness since the data structures produced by legacy SDK code
		// can sometimes be non-ideal.
		return a == b // equivalent if they are _both_ nil
	}
	if a.RawEquals(b) {
		// Easy case. We use RawEquals because we want two unknowns to be
		// considered equal here, whereas "Equals" would return unknown.
		return true
	}
	if !a.IsKnown() || !b.IsKnown() {
		// Two unknown values are equivalent regardless of type. A known is
		// never equivalent to an unknown.
		return a.IsKnown() == b.IsKnown()
	}
	if aZero, bZero := valuesSDKEquivalentIsNullOrZero(a), valuesSDKEquivalentIsNullOrZero(b); aZero || bZero {
		// Two null/zero values are equivalent regardless of type. A non-zero is
		// never equivalent to a zero.
		return aZero == bZero
	}

	// If we get down here then we are guaranteed that both a and b are known,
	// non-null values.

	aTy := a.Type()
	bTy := b.Type()
	switch {
	case aTy.IsSetType() && bTy.IsSetType():
		return valuesSDKEquivalentSets(a, b)
	case aTy.IsListType() && bTy.IsListType():
		return valuesSDKEquivalentSequences(a, b)
	case aTy.IsTupleType() && bTy.IsTupleType():
		return valuesSDKEquivalentSequences(a, b)
	case aTy.IsMapType() && bTy.IsMapType():
		return valuesSDKEquivalentMappings(a, b)
	case aTy.IsObjectType() && bTy.IsObjectType():
		return valuesSDKEquivalentMappings(a, b)
	case aTy == cty.Number && bTy == cty.Number:
		return valuesSDKEquivalentNumbers(a, b)
	default:
		// We've now covered all the interesting cases, so anything that falls
		// down here cannot be equivalent.
		return false
	}
}

// valuesSDKEquivalentIsNullOrZero returns true if the given value is either
// null or is the "zero value" (in the SDK/Go sense) for its type.
func valuesSDKEquivalentIsNullOrZero(v cty.Value) bool {
	if v == cty.NilVal {
		return true
	}

	ty := v.Type()
	switch {
	case !v.IsKnown():
		return false
	case v.IsNull():
		return true

	// After this point, v is always known and non-null
	case ty.IsListType() || ty.IsSetType() || ty.IsMapType() || ty.IsObjectType() || ty.IsTupleType():
		return v.LengthInt() == 0
	case ty == cty.String:
		return v.RawEquals(cty.StringVal(""))
	case ty == cty.Number:
		return v.RawEquals(cty.Zero)
	case ty == cty.Bool:
		return v.RawEquals(cty.False)
	default:
		// The above is exhaustive, but for robustness we'll consider anything
		// else to _not_ be zero unless it is null.
		return false
	}
}

// valuesSDKEquivalentSets returns true only if each of the elements in a can
// be correlated with at least one equivalent element in b and vice-versa.
// This is a fuzzy operation that prefers to signal non-equivalence if it cannot
// be certain that all elements are accounted for.
func valuesSDKEquivalentSets(a, b cty.Value) bool {
	if aLen, bLen := a.LengthInt(), b.LengthInt(); aLen != bLen {
		return false
	}

	// Our methodology here is a little tricky, to deal with the fact that
	// it's impossible to directly correlate two non-equal set elements because
	// they don't have identities separate from their values.
	// The approach is to count the number of equivalent elements each element
	// of a has in b and vice-versa, and then return true only if each element
	// in both sets has at least one equivalent.
	as := a.AsValueSlice()
	bs := b.AsValueSlice()
	aeqs := make([]bool, len(as))
	beqs := make([]bool, len(bs))
	for ai, av := range as {
		for bi, bv := range bs {
			if ValuesSDKEquivalent(av, bv) {
				aeqs[ai] = true
				beqs[bi] = true
			}
		}
	}

	for _, eq := range aeqs {
		if !eq {
			return false
		}
	}
	for _, eq := range beqs {
		if !eq {
			return false
		}
	}
	return true
}

// valuesSDKEquivalentSequences decides equivalence for two sequence values
// (lists or tuples).
func valuesSDKEquivalentSequences(a, b cty.Value) bool {
	as := a.AsValueSlice()
	bs := b.AsValueSlice()
	if len(as) != len(bs) {
		return false
	}

	for i := range as {
		if !ValuesSDKEquivalent(as[i], bs[i]) {
			return false
		}
	}
	return true
}

// valuesSDKEquivalentMappings decides equivalence for two mapping values
// (maps or objects).
func valuesSDKEquivalentMappings(a, b cty.Value) bool {
	as := a.AsValueMap()
	bs := b.AsValueMap()
	if len(as) != len(bs) {
		return false
	}

	for k, av := range as {
		bv, ok := bs[k]
		if !ok {
			return false
		}
		if !ValuesSDKEquivalent(av, bv) {
			return false
		}
	}
	return true
}

// valuesSDKEquivalentNumbers decides equivalence for two number values based
// on the fact that the SDK uses int and float64 representations while
// cty (and thus Terraform Core) uses big.Float, and so we expect to lose
// precision in the round-trip.
//
// This does _not_ attempt to allow for an epsilon difference that may be
// caused by accumulated innacuracy in a float calculation, under the
// expectation that providers generally do not actually do compuations on
// floats and instead just pass string representations of them on verbatim
// to remote APIs. A remote API _itself_ may introduce inaccuracy, but that's
// a problem for the provider itself to deal with, based on its knowledge of
// the remote system, e.g. using DiffSuppressFunc.
func valuesSDKEquivalentNumbers(a, b cty.Value) bool {
	if a.RawEquals(b) {
		return true // easy
	}

	af := a.AsBigFloat()
	bf := b.AsBigFloat()

	if af.IsInt() != bf.IsInt() {
		return false
	}
	if af.IsInt() && bf.IsInt() {
		return false // a.RawEquals(b) test above is good enough for integers
	}

	// The SDK supports only int and float64, so if it's not an integer
	// we know that only a float64-level of precision can possibly be
	// significant.
	af64, _ := af.Float64()
	bf64, _ := bf.Float64()
	return af64 == bf64
}
