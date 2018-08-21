package objchange

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
)

// ProposedNewObject constructs a proposed new object value by combining the
// computed attribute values from "prior" with the configured attribute values
// from "config".
//
// Both value must conform to the given schema's implied type, or this function
// will panic.
//
// The prior value must be wholly known, but the config value may be unknown
// or have nested unknown values.
//
// The merging of the two objects includes the attributes of any nested blocks,
// which will be correlated in a manner appropriate for their nesting mode.
// Note in particular that the correlation for blocks backed by sets is a
// heuristic based on matching non-computed attribute values and so it may
// produce strange results with more "extreme" cases, such as a nested set
// block where _all_ attributes are computed.
func ProposedNewObject(schema *configschema.Block, prior, config cty.Value) cty.Value {
	if prior.IsNull() {
		// This is the easy case... no prior value to merge, so we can just
		// return the config as-is.
		return config
	}
	if config.IsNull() || !config.IsKnown() {
		// This is a weird situation, but we'll allow it anyway to free
		// callers from needing to specifically check for these cases.
		return prior
	}
	if (!prior.Type().IsObjectType()) || (!config.Type().IsObjectType()) {
		panic("ProposedNewObject only supports object-typed values")
	}

	// From this point onwards, we can assume that both values are non-null
	// object types, and that the config value itself is known (though it
	// may contain nested values that are unknown.)

	newAttrs := map[string]cty.Value{}
	for name, attr := range schema.Attributes {
		priorV := prior.GetAttr(name)
		configV := config.GetAttr(name)
		var newV cty.Value
		switch {
		case attr.Computed && attr.Optional:
			// This is the trickiest scenario: we want to keep the prior value
			// if the config isn't overriding it. Note that due to some
			// ambiguity here, setting an optional+computed attribute from
			// config and then later switching the config to null in a
			// subsequent change causes the initial config value to be "sticky"
			// unless the provider specifically overrides it during its own
			// plan customization step.
			if configV.IsNull() {
				newV = priorV
			} else {
				newV = configV
			}
		case attr.Computed:
			// configV will always be null in this case, by definition.
			// priorV may also be null, but that's okay.
			newV = priorV
		default:
			// For non-computed attributes, we always take the config value,
			// even if it is null. If it's _required_ then null values
			// should've been caught during an earlier validation step, and
			// so we don't really care about that here.
			newV = configV
		}
		newAttrs[name] = newV
	}

	// Merging nested blocks is a little more complex, since we need to
	// correlate blocks between both objects and then recursively propose
	// a new object for each. The correlation logic depends on the nesting
	// mode for each block type.
	for name, blockType := range schema.BlockTypes {
		priorV := prior.GetAttr(name)
		configV := config.GetAttr(name)
		var newV cty.Value
		switch blockType.Nesting {

		case configschema.NestingSingle:
			newV = ProposedNewObject(&blockType.Block, priorV, configV)

		case configschema.NestingList:
			if !configV.Type().IsTupleType() {
				// Despite the name, we expect NestingList to produce a tuple
				// type so that different elements may have dynamically-typed
				// attributes that have a different actual type.
				panic("configschema.NestingList value is not a tuple as expected")
			}

			// Nested blocks are correlated by index.
			if l := configV.LengthInt(); l > 0 {
				newVals := make([]cty.Value, 0, l)
				for it := configV.ElementIterator(); it.Next(); {
					idx, configEV := it.Element()
					if !priorV.HasIndex(idx).True() {
						// If there is no corresponding prior element then
						// we just take the config value as-is.
						newVals = append(newVals, configEV)
						continue
					}
					priorEV := priorV.Index(idx)

					newEV := ProposedNewObject(&blockType.Block, priorEV, configEV)
					newVals = append(newVals, newEV)
				}
				// Although we call the nesting mode "list", we actually use
				// tuple values so that elements might have different types
				// in case of dynamically-typed attributes.
				newV = cty.TupleVal(newVals)
			} else {
				newV = cty.EmptyTupleVal
			}

		case configschema.NestingMap:
			if !configV.Type().IsObjectType() {
				// Despite the name, we expect NestingMap to produce an object
				// type so that different elements may have dynamically-typed
				// attributes that have a different actual type.
				panic("configschema.NestingMap value is not an object as expected")
			}

			// Nested blocks are correlated by key.
			if l := configV.LengthInt(); l > 0 {
				newVals := make(map[string]cty.Value, l)
				atys := configV.Type().AttributeTypes()
				for name := range atys {
					configEV := configV.GetAttr(name)
					if !priorV.Type().HasAttribute(name) {
						// If there is no corresponding prior element then
						// we just take the config value as-is.
						newVals[name] = configEV
						continue
					}
					priorEV := priorV.GetAttr(name)

					newEV := ProposedNewObject(&blockType.Block, priorEV, configEV)
					newVals[name] = newEV
				}
				// Although we call the nesting mode "map", we actually use
				// object values so that elements might have different types
				// in case of dynamically-typed attributes.
				newV = cty.ObjectVal(newVals)
			} else {
				newV = cty.EmptyObjectVal
			}

		case configschema.NestingSet:
			if !configV.Type().IsSetType() {
				panic("configschema.NestingSet value is not a set as expected")
			}

			// Nested blocks are correlated by comparing the element values
			// after eliminating all of the computed attributes. In practice,
			// this means that any config change produces an entirely new
			// nested object, and we only propagate prior computed values
			// if the non-computed attribute values are identical.
			cmpVals := setElementCompareValues(&blockType.Block, priorV, false)
			if l := configV.LengthInt(); l > 0 {
				used := make([]bool, len(cmpVals)) // track used elements in case multiple have the same compare value
				newVals := make([]cty.Value, 0, l)
				for it := configV.ElementIterator(); it.Next(); {
					_, configEV := it.Element()
					var priorEV cty.Value
					for i, cmp := range cmpVals {
						if used[i] {
							continue
						}
						if cmp[1].RawEquals(configEV) {
							priorEV = cmp[0]
							used[i] = true // we can't use this value on a future iteration
							break
						}
					}
					if priorEV == cty.NilVal {
						priorEV = cty.NullVal(blockType.ImpliedType())
					}

					newEV := ProposedNewObject(&blockType.Block, priorEV, configEV)
					newVals = append(newVals, newEV)
				}
				newV = cty.SetVal(newVals)
			} else {
				newV = cty.SetValEmpty(blockType.Block.ImpliedType())
			}

		default:
			// Should never happen, since the above cases are comprehensive.
			panic(fmt.Sprintf("unsupported block nesting mode %s", blockType.Nesting))
		}

		newAttrs[name] = newV
	}

	return cty.ObjectVal(newAttrs)
}

// setElementCompareValues takes a known, non-null value of a cty.Set type and
// returns a table -- constructed of two-element arrays -- that maps original
// set element values to corresponding values that have all of the computed
// values removed, making them suitable for comparison with values obtained
// from configuration. The element type of the set must conform to the implied
// type of the given schema, or this function will panic.
//
// In the resulting slice, the zeroth element of each array is the original
// value and the one-indexed element is the corresponding "compare value".
//
// This is intended to help correlate prior elements with configured elements
// in ProposedNewObject. The result is a heuristic rather than an exact science,
// since e.g. two separate elements may reduce to the same value through this
// process. The caller must therefore be ready to deal with duplicates.
func setElementCompareValues(schema *configschema.Block, set cty.Value, isConfig bool) [][2]cty.Value {
	ret := make([][2]cty.Value, 0, set.LengthInt())
	for it := set.ElementIterator(); it.Next(); {
		_, ev := it.Element()
		ret = append(ret, [2]cty.Value{ev, setElementCompareValue(schema, ev, isConfig)})
	}
	return ret
}

// setElementCompareValue creates a new value that has all of the same
// non-computed attribute values as the one given but has all computed
// attribute values forced to null.
//
// If isConfig is true then non-null Optional+Computed attribute values will
// be preserved. Otherwise, they will also be set to null.
//
// The input value must conform to the schema's implied type, and the return
// value is guaranteed to conform to it.
func setElementCompareValue(schema *configschema.Block, v cty.Value, isConfig bool) cty.Value {
	if v.IsNull() || !v.IsKnown() {
		return v
	}

	attrs := map[string]cty.Value{}
	for name, attr := range schema.Attributes {
		switch {
		case attr.Computed && attr.Optional:
			if isConfig {
				attrs[name] = v.GetAttr(name)
			} else {
				attrs[name] = cty.NullVal(attr.Type)
			}
		case attr.Computed:
			attrs[name] = cty.NullVal(attr.Type)
		default:
			attrs[name] = v.GetAttr(name)
		}
	}

	for name, blockType := range schema.BlockTypes {
		switch blockType.Nesting {

		case configschema.NestingSingle:
			attrs[name] = setElementCompareValue(&blockType.Block, v.GetAttr(name), isConfig)

		case configschema.NestingList, configschema.NestingSet:
			cv := v.GetAttr(name)
			if cv.IsNull() || !cv.IsKnown() {
				attrs[name] = cv
				continue
			}
			if l := cv.LengthInt(); l > 0 {
				elems := make([]cty.Value, 0, l)
				for it := cv.ElementIterator(); it.Next(); {
					_, ev := it.Element()
					elems = append(elems, setElementCompareValue(&blockType.Block, ev, isConfig))
				}
				if blockType.Nesting == configschema.NestingSet {
					// SetValEmpty would panic if given elements that are not
					// all of the same type, but that's guaranteed not to
					// happen here because our input value was _already_ a
					// set and we've not changed the types of any elements here.
					attrs[name] = cty.SetVal(elems)
				} else {
					attrs[name] = cty.TupleVal(elems)
				}
			} else {
				if blockType.Nesting == configschema.NestingSet {
					attrs[name] = cty.SetValEmpty(blockType.Block.ImpliedType())
				} else {
					attrs[name] = cty.EmptyTupleVal
				}
			}

		case configschema.NestingMap:
			cv := v.GetAttr(name)
			if cv.IsNull() || !cv.IsKnown() {
				attrs[name] = cv
				continue
			}
			elems := make(map[string]cty.Value)
			for it := cv.ElementIterator(); it.Next(); {
				kv, ev := it.Element()
				elems[kv.AsString()] = setElementCompareValue(&blockType.Block, ev, isConfig)
			}
			attrs[name] = cty.ObjectVal(elems)

		default:
			// Should never happen, since the above cases are comprehensive.
			panic(fmt.Sprintf("unsupported block nesting mode %s", blockType.Nesting))
		}
	}

	return cty.ObjectVal(attrs)
}
