package diffs

import (
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/zclconf/go-cty/cty"
)

// PreserveComputedAttrs takes an old and a new object value, the latter of
// which may contain unknown values, and produces a new value where any unknown
// values in new are replaced with corresponding values from old.
//
// Both given values must have types that conform to the implied type of the
// given schema, or else this function may panic or produce a nonsensical
// result. The old value must never be unknown or contain any unknown values,
// which will also cause this function to panic.
//
// This is primarily useful when preparing an Update change for an existing
// resource, where concrete values from its state (passed as "old") should
// be used in place of unknown values in its config (passed as "new") under
// the assumption that they were decided by the provider during a previous
// apply and so should be retained for future updates unless overridden.
//
// The preservation applies only to direct values of attributes that are
// marked as Computed in the given schema. Unknown values nested within
// collections are not subject to any merging, and non-computed attributes
// are left untouched.
//
// When the schema contains nested blocks backed by collections (NestingList,
// NestingSet or NestingMap) the blocks are correlated using their keys for
// the sake of preserving values: lists are correlated by index, maps are
// correlated by key, and sets are correlated by a heuristic that considers
// two elements as equivalent if their non-computed attributes have equal
// values. This may produce unexpected results in the face of drastic changes
// to configuration, such as reordering of elements in a list. It is best to
// minimize the use of computed attributes in such structures to avoid user
// confusion in such situations.
func PreserveComputedAttrs(old, new cty.Value, schema *configschema.Block) cty.Value {
	if old.IsNull() || new.IsNull() {
		return new
	}
	if !new.IsKnown() {
		// Should never happen in any reasonable case, since we never produce
		// a wholly-unknown resource, but we'll allow it anyway since there's
		// an easy, obvious result for this situation.
		return old
	}

	retVals := make(map[string]cty.Value)

	for name, attrS := range schema.Attributes {
		oldVal := old.GetAttr(name)
		newVal := new.GetAttr(name)

		switch {
		case !attrS.Computed:
			// Non-computed attributes always use their new value, which
			// may be unknown if assigned a value from a computed attribute
			// on another resource.
			retVals[name] = newVal
		case !newVal.IsKnown():
			// If a computed attribute has a new value of unknown _and_ if
			// the old value is non-null then we'll "preserve" that non-null
			// value in our result.
			retVals[name] = oldVal
		default:
			// In all other cases, the new value just passes through.
			retVals[name] = newVal
		}
	}

	// Now we need to recursively do the same work for all of our nested blocks
	for name, blockS := range schema.BlockTypes {
		switch blockS.Nesting {
		case configschema.NestingSingle:
			oldVal := old.GetAttr(name)
			newVal := new.GetAttr(name)
			retVals[name] = PreserveComputedAttrs(oldVal, newVal, &blockS.Block)
		case configschema.NestingList:
			oldList := old.GetAttr(name)
			newList := new.GetAttr(name)

			if oldList.IsNull() || newList.IsNull() || !newList.IsKnown() {
				retVals[name] = newList
				continue
			}

			length := newList.LengthInt()
			if length == 0 {
				retVals[name] = newList
				continue
			}

			retElems := make([]cty.Value, 0, length)
			for it := newList.ElementIterator(); it.Next(); {
				idx, newElem := it.Element()
				if oldList.HasIndex(idx).True() {
					oldElem := oldList.Index(idx)
					retElems = append(retElems, PreserveComputedAttrs(oldElem, newElem, &blockS.Block))
				} else {
					retElems = append(retElems, newElem)
				}
			}
			retVals[name] = cty.ListVal(retElems)
		case configschema.NestingMap:
			oldMap := old.GetAttr(name)
			newMap := new.GetAttr(name)

			if oldMap.IsNull() || newMap.IsNull() || !newMap.IsKnown() {
				retVals[name] = newMap
				continue
			}
			if newMap.LengthInt() == 0 {
				retVals[name] = newMap
				continue
			}

			retElems := make(map[string]cty.Value)
			for it := newMap.ElementIterator(); it.Next(); {
				key, newElem := it.Element()
				if oldMap.HasIndex(key).True() {
					oldElem := oldMap.Index(key)
					retElems[key.AsString()] = PreserveComputedAttrs(oldElem, newElem, &blockS.Block)
				} else {
					retElems[key.AsString()] = newElem
				}
			}
			retVals[name] = cty.MapVal(retElems)
		case configschema.NestingSet:
			oldSet := old.GetAttr(name)
			newSet := new.GetAttr(name)

			if oldSet.IsNull() || newSet.IsNull() || !newSet.IsKnown() {
				retVals[name] = newSet
				continue
			}
			if newSet.LengthInt() == 0 {
				retVals[name] = newSet
				continue
			}

			// Correlating set elements is tricky because their value is also
			// their key, and so there is no precise way to correlate a
			// new object that has unknown attributes with an existing value
			// that has those attributes populated.
			//
			// As an approximation, the technique here is to null out all of
			// the computed attribute values in both old and new where new
			// has an unknown value and then look for matching pairs that
			// produce the same result, which effectively then uses the
			// Non-Computed attributes (as well as any explicitly-set
			// Optional+Computed attributes in new) as the "key". We must
			// do this normalization recursively because our block may contain
			// nested blocks of its own that _also_ have computed attributes.
			//
			// This will be successful as long as the attributes we use for
			// matching form a unique key once the computed attributes are
			// taken out of consideration. If not, we will arbitrarily select
			// one of the two-or-more corresponding elements to propagate
			// the computed values into, and leave the others untouched
			// with their unknown values exactly as given in "new".
			//
			// This correlation work ends up being ~O(oldLen * newLen) because
			// Optional+Computed attributes require us to compare to old
			// separately for each new element. This is generally fine because
			// oldLen and newLen are small in all reasonable configurations.
			// We may start to see some disagreeable performance on structures
			// where set blocks are nested deeply, however, since the descendent
			// will be visited many times as we traverse each nesting level.

			oldVals := make([]cty.Value, 0, oldSet.LengthInt())
			for it := oldSet.ElementIterator(); it.Next(); {
				_, oldVal := it.Element()
				oldVals = append(oldVals, oldVal)
			}
			oldValsUsed := make([]bool, len(oldVals))

			retElems := make([]cty.Value, 0, newSet.LengthInt())
			for it := newSet.ElementIterator(); it.Next(); {
				_, newVal := it.Element()
				var oldVal cty.Value
				for i, candidate := range oldVals {
					if oldValsUsed[i] {
						// Can't propagate attribute values from an object
						// we already propagated.
						continue
					}

					if blockElementsCorrelated(candidate, newVal, &blockS.Block) {
						oldVal = candidate
						oldValsUsed[i] = true
						break
					}
				}

				if oldVal != cty.NilVal {
					retElems = append(retElems, PreserveComputedAttrs(oldVal, newVal, &blockS.Block))
				} else {
					retElems = append(retElems, newVal)
				}
			}
			retVals[name] = cty.SetVal(retElems)

		default:
			// Should never happen since the above is exhaustive, but we'll
			// preserve the new value if not just to ensure that we produce
			// something that conforms to the schema.
			retVals[name] = new.GetAttr(name)
		}
	}

	return cty.ObjectVal(retVals)
}

// blockElementsCorrelated determines whether two values (which must both
// conform to the given schema) are correlatable based on non-computed attributes.
//
// For any attribute that is non-Computed, the old and new values must be
// equal. Attributes that are Computed are considered only if they have known
// values in "new", which suggests an Optional+Computed attribute that is
// being explicitly set in config, overriding any computed value.
//
// If there are any nested blocks in the given schema then these are
// recursively tested.
func blockElementsCorrelated(old, new cty.Value, schema *configschema.Block) bool {
	for name, attrS := range schema.Attributes {
		oldVal := old.GetAttr(name)
		newVal := new.GetAttr(name)

		switch {
		case !attrS.Computed:
			if !newVal.IsKnown() {
				// Should not be possible per the schema, but we'll silently
				// accept it and disallow correlation, rather than crashing
				// below.
				return false
			}
			if oldVal.Equals(newVal).False() {
				return false
			}
		case newVal.IsKnown():
			if oldVal.Equals(newVal).False() {
				return false
			}
		}
	}

	// Now we need to recursively do the same work for all of our nested blocks
	for name, blockS := range schema.BlockTypes {

		switch blockS.Nesting {
		case configschema.NestingSingle:
			oldVal := old.GetAttr(name)
			newVal := new.GetAttr(name)
			if !blockElementsCorrelated(oldVal, newVal, &blockS.Block) {
				return false
			}
		case configschema.NestingSet:
			// Nested sets are, as usual, quite tricky: we need to recursively
			// try to correlate all of the members of the old and new sets
			// to make sure that there is a bijection for the two sets.
			oldColl := old.GetAttr(name)
			newColl := new.GetAttr(name)

			if newColl.IsNull() || newColl.IsNull() {
				if oldColl.IsNull() != newColl.IsNull() {
					return false
				}
				continue
			}

			if newColl.Length().Equals(oldColl.Length()).False() {
				// Two sets of different lengths can't possibly have a bijection
				return false
			}

			oldVals := make([]cty.Value, 0, oldColl.LengthInt())
			for it := oldColl.ElementIterator(); it.Next(); {
				_, oldVal := it.Element()
				oldVals = append(oldVals, oldVal)
			}
			oldValsUsed := make([]bool, len(oldVals))
			for it := newColl.ElementIterator(); it.Next(); {
				_, newVal := it.Element()
				var oldVal cty.Value
				for i, candidate := range oldVals {
					if oldValsUsed[i] {
						// Can't re-use a collection element, so we must keep
						// searching to see if a later one correlates.
						continue
					}

					if blockElementsCorrelated(candidate, newVal, &blockS.Block) {
						oldVal = candidate
						oldValsUsed[i] = true
						break
					}
				}

				if oldVal == cty.NilVal {
					return false
				}
			}

			return false
		default:
			// Assume everything else is an indexable collection and just do a
			// per-element recursive check. (This covers NestingList and NestingMap)
			oldColl := old.GetAttr(name)
			newColl := new.GetAttr(name)

			if newColl.IsNull() || newColl.IsNull() {
				if oldColl.IsNull() != newColl.IsNull() {
					return false
				}
				continue
			}

			for it := newColl.ElementIterator(); it.Next(); {
				key, newVal := it.Element()

				// If our collection keys don't match exactly then we don't
				// correlate.
				if oldColl.HasIndex(key).False() {
					return false
				}

				oldVal := oldColl.Index(key)
				if !blockElementsCorrelated(oldVal, newVal, &blockS.Block) {
					return false
				}
			}
		}
	}

	return true
}
