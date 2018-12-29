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

	// If all of the given types are of the same structural kind, we may be
	// able to construct a new type that they can all be unified to, even if
	// that is not one of the given types. We must try this before the general
	// behavior below because in unsafe mode we can convert an object type to
	// a subset of that type, which would be a much less useful conversion for
	// unification purposes.
	{
		objectCt := 0
		tupleCt := 0
		dynamicCt := 0
		for _, ty := range types {
			switch {
			case ty.IsObjectType():
				objectCt++
			case ty.IsTupleType():
				tupleCt++
			case ty == cty.DynamicPseudoType:
				dynamicCt++
			default:
				break
			}
		}
		switch {
		case objectCt > 0 && (objectCt+dynamicCt) == len(types):
			return unifyObjectTypes(types, unsafe, dynamicCt > 0)
		case tupleCt > 0 && (tupleCt+dynamicCt) == len(types):
			return unifyTupleTypes(types, unsafe, dynamicCt > 0)
		case objectCt > 0 && tupleCt > 0:
			// Can never unify object and tuple types since they have incompatible kinds
			return cty.NilType, nil
		}
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

	// If we fall out here, no unification is possible
	return cty.NilType, nil
}

func unifyObjectTypes(types []cty.Type, unsafe bool, hasDynamic bool) (cty.Type, []Conversion) {
	// If we had any dynamic types in the input here then we can't predict
	// what path we'll take through here once these become known types, so
	// we'll conservatively produce DynamicVal for these.
	if hasDynamic {
		return unifyAllAsDynamic(types)
	}

	// There are two different ways we can succeed here:
	// - If all of the given object types have the same set of attribute names
	//   and the corresponding types are all unifyable, then we construct that
	//   type.
	// - If the given object types have different attribute names or their
	//   corresponding types are not unifyable, we'll instead try to unify
	//   all of the attribute types together to produce a map type.
	//
	// Our unification behavior is intentionally stricter than our conversion
	// behavior for subset object types because user intent is different with
	// unification use-cases: it makes sense to allow {"foo":true} to convert
	// to emptyobjectval, but unifying an object with an attribute with the
	// empty object type should be an error because unifying to the empty
	// object type would be suprising and useless.

	firstAttrs := types[0].AttributeTypes()
	for _, ty := range types[1:] {
		thisAttrs := ty.AttributeTypes()
		if len(thisAttrs) != len(firstAttrs) {
			// If number of attributes is different then there can be no
			// object type in common.
			return unifyObjectTypesToMap(types, unsafe)
		}
		for name := range thisAttrs {
			if _, ok := firstAttrs[name]; !ok {
				// If attribute names don't exactly match then there can be
				// no object type in common.
				return unifyObjectTypesToMap(types, unsafe)
			}
		}
	}

	// If we get here then we've proven that all of the given object types
	// have exactly the same set of attribute names, though the types may
	// differ.
	retAtys := make(map[string]cty.Type)
	atysAcross := make([]cty.Type, len(types))
	for name := range firstAttrs {
		for i, ty := range types {
			atysAcross[i] = ty.AttributeType(name)
		}
		retAtys[name], _ = unify(atysAcross, unsafe)
		if retAtys[name] == cty.NilType {
			// Cannot unify this attribute alone, which means that unification
			// of everything down to a map type can't be possible either.
			return cty.NilType, nil
		}
	}
	retTy := cty.Object(retAtys)

	conversions := make([]Conversion, len(types))
	for i, ty := range types {
		if ty.Equals(retTy) {
			continue
		}
		if unsafe {
			conversions[i] = GetConversionUnsafe(ty, retTy)
		} else {
			conversions[i] = GetConversion(ty, retTy)
		}
		if conversions[i] == nil {
			// Shouldn't be reachable, since we were able to unify
			return unifyObjectTypesToMap(types, unsafe)
		}
	}

	return retTy, conversions
}

func unifyObjectTypesToMap(types []cty.Type, unsafe bool) (cty.Type, []Conversion) {
	// This is our fallback case for unifyObjectTypes, where we see if we can
	// construct a map type that can accept all of the attribute types.

	var atys []cty.Type
	for _, ty := range types {
		for _, aty := range ty.AttributeTypes() {
			atys = append(atys, aty)
		}
	}

	ety, _ := unify(atys, unsafe)
	if ety == cty.NilType {
		return cty.NilType, nil
	}

	retTy := cty.Map(ety)
	conversions := make([]Conversion, len(types))
	for i, ty := range types {
		if ty.Equals(retTy) {
			continue
		}
		if unsafe {
			conversions[i] = GetConversionUnsafe(ty, retTy)
		} else {
			conversions[i] = GetConversion(ty, retTy)
		}
		if conversions[i] == nil {
			// Shouldn't be reachable, since we were able to unify
			return unifyObjectTypesToMap(types, unsafe)
		}
	}
	return retTy, conversions
}

func unifyTupleTypes(types []cty.Type, unsafe bool, hasDynamic bool) (cty.Type, []Conversion) {
	// If we had any dynamic types in the input here then we can't predict
	// what path we'll take through here once these become known types, so
	// we'll conservatively produce DynamicVal for these.
	if hasDynamic {
		return unifyAllAsDynamic(types)
	}

	// There are two different ways we can succeed here:
	// - If all of the given tuple types have the same sequence of element types
	//   and the corresponding types are all unifyable, then we construct that
	//   type.
	// - If the given tuple types have different element types or their
	//   corresponding types are not unifyable, we'll instead try to unify
	//   all of the elements types together to produce a list type.

	firstEtys := types[0].TupleElementTypes()
	for _, ty := range types[1:] {
		thisEtys := ty.TupleElementTypes()
		if len(thisEtys) != len(firstEtys) {
			// If number of elements is different then there can be no
			// tuple type in common.
			return unifyTupleTypesToList(types, unsafe)
		}
	}

	// If we get here then we've proven that all of the given tuple types
	// have the same number of elements, though the types may differ.
	retEtys := make([]cty.Type, len(firstEtys))
	atysAcross := make([]cty.Type, len(types))
	for idx := range firstEtys {
		for tyI, ty := range types {
			atysAcross[tyI] = ty.TupleElementTypes()[idx]
		}
		retEtys[idx], _ = unify(atysAcross, unsafe)
		if retEtys[idx] == cty.NilType {
			// Cannot unify this element alone, which means that unification
			// of everything down to a map type can't be possible either.
			return cty.NilType, nil
		}
	}
	retTy := cty.Tuple(retEtys)

	conversions := make([]Conversion, len(types))
	for i, ty := range types {
		if ty.Equals(retTy) {
			continue
		}
		if unsafe {
			conversions[i] = GetConversionUnsafe(ty, retTy)
		} else {
			conversions[i] = GetConversion(ty, retTy)
		}
		if conversions[i] == nil {
			// Shouldn't be reachable, since we were able to unify
			return unifyTupleTypesToList(types, unsafe)
		}
	}

	return retTy, conversions
}

func unifyTupleTypesToList(types []cty.Type, unsafe bool) (cty.Type, []Conversion) {
	// This is our fallback case for unifyTupleTypes, where we see if we can
	// construct a list type that can accept all of the element types.

	var etys []cty.Type
	for _, ty := range types {
		for _, ety := range ty.TupleElementTypes() {
			etys = append(etys, ety)
		}
	}

	ety, _ := unify(etys, unsafe)
	if ety == cty.NilType {
		return cty.NilType, nil
	}

	retTy := cty.List(ety)
	conversions := make([]Conversion, len(types))
	for i, ty := range types {
		if ty.Equals(retTy) {
			continue
		}
		if unsafe {
			conversions[i] = GetConversionUnsafe(ty, retTy)
		} else {
			conversions[i] = GetConversion(ty, retTy)
		}
		if conversions[i] == nil {
			// Shouldn't be reachable, since we were able to unify
			return unifyObjectTypesToMap(types, unsafe)
		}
	}
	return retTy, conversions
}

func unifyAllAsDynamic(types []cty.Type) (cty.Type, []Conversion) {
	conversions := make([]Conversion, len(types))
	for i := range conversions {
		conversions[i] = func(cty.Value) (cty.Value, error) {
			return cty.DynamicVal, nil
		}
	}
	return cty.DynamicPseudoType, conversions
}
