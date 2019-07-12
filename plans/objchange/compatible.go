package objchange

import (
	"fmt"
	"strconv"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/configs/configschema"
)

// AssertObjectCompatible checks whether the given "actual" value is a valid
// completion of the possibly-partially-unknown "planned" value.
//
// This means that any known leaf value in "planned" must be equal to the
// corresponding value in "actual", and various other similar constraints.
//
// Any inconsistencies are reported by returning a non-zero number of errors.
// These errors are usually (but not necessarily) cty.PathError values
// referring to a particular nested value within the "actual" value.
//
// The two values must have types that conform to the given schema's implied
// type, or this function will panic.
func AssertObjectCompatible(schema *configschema.Block, planned, actual cty.Value) []error {
	return assertObjectCompatible(schema, planned, actual, nil)
}

func assertObjectCompatible(schema *configschema.Block, planned, actual cty.Value, path cty.Path) []error {
	var errs []error
	if planned.IsNull() && !actual.IsNull() {
		errs = append(errs, path.NewErrorf("was absent, but now present"))
		return errs
	}
	if actual.IsNull() && !planned.IsNull() {
		errs = append(errs, path.NewErrorf("was present, but now absent"))
		return errs
	}
	if planned.IsNull() {
		// No further checks possible if both values are null
		return errs
	}

	for name, attrS := range schema.Attributes {
		plannedV := planned.GetAttr(name)
		actualV := actual.GetAttr(name)

		path := append(path, cty.GetAttrStep{Name: name})
		moreErrs := assertValueCompatible(plannedV, actualV, path)
		if attrS.Sensitive {
			if len(moreErrs) > 0 {
				// Use a vague placeholder message instead, to avoid disclosing
				// sensitive information.
				errs = append(errs, path.NewErrorf("inconsistent values for sensitive attribute"))
			}
		} else {
			errs = append(errs, moreErrs...)
		}
	}
	for name, blockS := range schema.BlockTypes {
		plannedV := planned.GetAttr(name)
		actualV := actual.GetAttr(name)

		// As a special case, if there were any blocks whose leaf attributes
		// are all unknown then we assume (possibly incorrectly) that the
		// HCL dynamic block extension is in use with an unknown for_each
		// argument, and so we will do looser validation here that allows
		// for those blocks to have expanded into a different number of blocks
		// if the for_each value is now known.
		maybeUnknownBlocks := couldHaveUnknownBlockPlaceholder(plannedV, blockS, false)

		path := append(path, cty.GetAttrStep{Name: name})
		switch blockS.Nesting {
		case configschema.NestingSingle, configschema.NestingGroup:
			// If an unknown block placeholder was present then the placeholder
			// may have expanded out into zero blocks, which is okay.
			if maybeUnknownBlocks && actualV.IsNull() {
				continue
			}
			moreErrs := assertObjectCompatible(&blockS.Block, plannedV, actualV, path)
			errs = append(errs, moreErrs...)
		case configschema.NestingList:
			// A NestingList might either be a list or a tuple, depending on
			// whether there are dynamically-typed attributes inside. However,
			// both support a similar-enough API that we can treat them the
			// same for our purposes here.
			if !plannedV.IsKnown() || !actualV.IsKnown() || plannedV.IsNull() || actualV.IsNull() {
				continue
			}

			if maybeUnknownBlocks {
				// When unknown blocks are present the final blocks may be
				// at different indices than the planned blocks, so unfortunately
				// we can't do our usual checks in this case without generating
				// false negatives.
				continue
			}

			plannedL := plannedV.LengthInt()
			actualL := actualV.LengthInt()
			if plannedL != actualL {
				errs = append(errs, path.NewErrorf("block count changed from %d to %d", plannedL, actualL))
				continue
			}
			for it := plannedV.ElementIterator(); it.Next(); {
				idx, plannedEV := it.Element()
				if !actualV.HasIndex(idx).True() {
					continue
				}
				actualEV := actualV.Index(idx)
				moreErrs := assertObjectCompatible(&blockS.Block, plannedEV, actualEV, append(path, cty.IndexStep{Key: idx}))
				errs = append(errs, moreErrs...)
			}
		case configschema.NestingMap:
			// A NestingMap might either be a map or an object, depending on
			// whether there are dynamically-typed attributes inside, but
			// that's decided statically and so both values will have the same
			// kind.
			if plannedV.Type().IsObjectType() {
				plannedAtys := plannedV.Type().AttributeTypes()
				actualAtys := actualV.Type().AttributeTypes()
				for k := range plannedAtys {
					if _, ok := actualAtys[k]; !ok {
						errs = append(errs, path.NewErrorf("block key %q has vanished", k))
						continue
					}

					plannedEV := plannedV.GetAttr(k)
					actualEV := actualV.GetAttr(k)
					moreErrs := assertObjectCompatible(&blockS.Block, plannedEV, actualEV, append(path, cty.GetAttrStep{Name: k}))
					errs = append(errs, moreErrs...)
				}
				if !maybeUnknownBlocks { // new blocks may appear if unknown blocks were present in the plan
					for k := range actualAtys {
						if _, ok := plannedAtys[k]; !ok {
							errs = append(errs, path.NewErrorf("new block key %q has appeared", k))
							continue
						}
					}
				}
			} else {
				if !plannedV.IsKnown() || plannedV.IsNull() || actualV.IsNull() {
					continue
				}
				plannedL := plannedV.LengthInt()
				actualL := actualV.LengthInt()
				if plannedL != actualL && !maybeUnknownBlocks { // new blocks may appear if unknown blocks were persent in the plan
					errs = append(errs, path.NewErrorf("block count changed from %d to %d", plannedL, actualL))
					continue
				}
				for it := plannedV.ElementIterator(); it.Next(); {
					idx, plannedEV := it.Element()
					if !actualV.HasIndex(idx).True() {
						continue
					}
					actualEV := actualV.Index(idx)
					moreErrs := assertObjectCompatible(&blockS.Block, plannedEV, actualEV, append(path, cty.IndexStep{Key: idx}))
					errs = append(errs, moreErrs...)
				}
			}
		case configschema.NestingSet:
			if !plannedV.IsKnown() || !actualV.IsKnown() || plannedV.IsNull() || actualV.IsNull() {
				continue
			}

			setErrs := assertSetValuesCompatible(plannedV, actualV, path, func(plannedEV, actualEV cty.Value) bool {
				errs := assertObjectCompatible(&blockS.Block, plannedEV, actualEV, append(path, cty.IndexStep{Key: actualEV}))
				return len(errs) == 0
			})
			errs = append(errs, setErrs...)

			if maybeUnknownBlocks {
				// When unknown blocks are present the final number of blocks
				// may be different, either because the unknown set values
				// become equal and are collapsed, or the count is unknown due
				// a dynamic block. Unfortunately this means we can't do our
				// usual checks in this case without generating false
				// negatives.
				continue
			}

			// There can be fewer elements in a set after its elements are all
			// known (values that turn out to be equal will coalesce) but the
			// number of elements must never get larger.
			plannedL := plannedV.LengthInt()
			actualL := actualV.LengthInt()
			if plannedL < actualL {
				errs = append(errs, path.NewErrorf("block set length changed from %d to %d", plannedL, actualL))
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", blockS.Nesting))
		}
	}
	return errs
}

func assertValueCompatible(planned, actual cty.Value, path cty.Path) []error {
	// NOTE: We don't normally use the GoString rendering of cty.Value in
	// user-facing error messages as a rule, but we make an exception
	// for this function because we expect the user to pass this message on
	// verbatim to the provider development team and so more detail is better.

	var errs []error
	if planned.Type() == cty.DynamicPseudoType {
		// Anything goes, then
		return errs
	}
	if problems := planned.Type().TestConformance(actual.Type()); len(problems) > 0 {
		errs = append(errs, path.NewErrorf("wrong final value type: %s", convert.MismatchMessage(actual.Type(), planned.Type())))
		// If the types don't match then we can't do any other comparisons,
		// so we bail early.
		return errs
	}

	if !planned.IsKnown() {
		// We didn't know what were going to end up with during plan, so
		// anything goes during apply.
		return errs
	}

	if actual.IsNull() {
		if planned.IsNull() {
			return nil
		}
		errs = append(errs, path.NewErrorf("was %#v, but now null", planned))
		return errs
	}
	if planned.IsNull() {
		errs = append(errs, path.NewErrorf("was null, but now %#v", actual))
		return errs
	}

	ty := planned.Type()
	switch {

	case !actual.IsKnown():
		errs = append(errs, path.NewErrorf("was known, but now unknown"))

	case ty.IsPrimitiveType():
		if !actual.Equals(planned).True() {
			errs = append(errs, path.NewErrorf("was %#v, but now %#v", planned, actual))
		}

	case ty.IsListType() || ty.IsMapType() || ty.IsTupleType():
		for it := planned.ElementIterator(); it.Next(); {
			k, plannedV := it.Element()
			if !actual.HasIndex(k).True() {
				errs = append(errs, path.NewErrorf("element %s has vanished", indexStrForErrors(k)))
				continue
			}

			actualV := actual.Index(k)
			moreErrs := assertValueCompatible(plannedV, actualV, append(path, cty.IndexStep{Key: k}))
			errs = append(errs, moreErrs...)
		}

		for it := actual.ElementIterator(); it.Next(); {
			k, _ := it.Element()
			if !planned.HasIndex(k).True() {
				errs = append(errs, path.NewErrorf("new element %s has appeared", indexStrForErrors(k)))
			}
		}

	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		for name := range atys {
			// Because we already tested that the two values have the same type,
			// we can assume that the same attributes are present in both and
			// focus just on testing their values.
			plannedV := planned.GetAttr(name)
			actualV := actual.GetAttr(name)
			moreErrs := assertValueCompatible(plannedV, actualV, append(path, cty.GetAttrStep{Name: name}))
			errs = append(errs, moreErrs...)
		}

	case ty.IsSetType():
		// We can't really do anything useful for sets here because changing
		// an unknown element to known changes the identity of the element, and
		// so we can't correlate them properly. However, we will at least check
		// to ensure that the number of elements is consistent, along with
		// the general type-match checks we ran earlier in this function.
		if planned.IsKnown() && !planned.IsNull() && !actual.IsNull() {

			setErrs := assertSetValuesCompatible(planned, actual, path, func(plannedV, actualV cty.Value) bool {
				errs := assertValueCompatible(plannedV, actualV, append(path, cty.IndexStep{Key: actualV}))
				return len(errs) == 0
			})
			errs = append(errs, setErrs...)

			// There can be fewer elements in a set after its elements are all
			// known (values that turn out to be equal will coalesce) but the
			// number of elements must never get larger.

			plannedL := planned.LengthInt()
			actualL := actual.LengthInt()
			if plannedL < actualL {
				errs = append(errs, path.NewErrorf("length changed from %d to %d", plannedL, actualL))
			}
		}
	}

	return errs
}

func indexStrForErrors(v cty.Value) string {
	switch v.Type() {
	case cty.Number:
		return v.AsBigFloat().Text('f', -1)
	case cty.String:
		return strconv.Quote(v.AsString())
	default:
		// Should be impossible, since no other index types are allowed!
		return fmt.Sprintf("%#v", v)
	}
}

// couldHaveUnknownBlockPlaceholder is a heuristic that recognizes how the
// HCL dynamic block extension behaves when it's asked to expand a block whose
// for_each argument is unknown. In such cases, it generates a single placeholder
// block with all leaf attribute values unknown, and once the for_each
// expression becomes known the placeholder may be replaced with any number
// of blocks, so object compatibility checks would need to be more liberal.
//
// Set "nested" if testing a block that is nested inside a candidate block
// placeholder; this changes the interpretation of there being no blocks of
// a type to allow for there being zero nested blocks.
func couldHaveUnknownBlockPlaceholder(v cty.Value, blockS *configschema.NestedBlock, nested bool) bool {
	switch blockS.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		if nested && v.IsNull() {
			return true // for nested blocks, a single block being unset doesn't disqualify from being an unknown block placeholder
		}
		return couldBeUnknownBlockPlaceholderElement(v, &blockS.Block)
	default:
		// These situations should be impossible for correct providers, but
		// we permit the legacy SDK to produce some incorrect outcomes
		// for compatibility with its existing logic, and so we must be
		// tolerant here.
		if !v.IsKnown() {
			return true
		}
		if v.IsNull() {
			return false // treated as if the list were empty, so we would see zero iterations below
		}

		// For all other nesting modes, our value should be something iterable.
		for it := v.ElementIterator(); it.Next(); {
			_, ev := it.Element()
			if couldBeUnknownBlockPlaceholderElement(ev, &blockS.Block) {
				return true
			}
		}

		// Our default changes depending on whether we're testing the candidate
		// block itself or something nested inside of it: zero blocks of a type
		// can never contain a dynamic block placeholder, but a dynamic block
		// placeholder might contain zero blocks of one of its own nested block
		// types, if none were set in the config at all.
		return nested
	}
}

func couldBeUnknownBlockPlaceholderElement(v cty.Value, schema *configschema.Block) bool {
	if v.IsNull() {
		return false // null value can never be a placeholder element
	}
	if !v.IsKnown() {
		return true // this should never happen for well-behaved providers, but can happen with the legacy SDK opt-outs
	}
	for name := range schema.Attributes {
		av := v.GetAttr(name)

		// Unknown block placeholders contain only unknown or null attribute
		// values, depending on whether or not a particular attribute was set
		// explicitly inside the content block. Note that this is imprecise:
		// non-placeholders can also match this, so this function can generate
		// false positives.
		if av.IsKnown() && !av.IsNull() {
			return false
		}
	}
	for name, blockS := range schema.BlockTypes {
		if !couldHaveUnknownBlockPlaceholder(v.GetAttr(name), blockS, true) {
			return false
		}
	}
	return true
}

// assertSetValuesCompatible checks that each of the elements in a can
// be correlated with at least one equivalent element in b and vice-versa,
// using the given correlation function.
//
// This allows the number of elements in the sets to change as long as all
// elements in both sets can be correlated, making this function safe to use
// with sets that may contain unknown values as long as the unknown case is
// addressed in some reasonable way in the callback function.
//
// The callback always recieves values from set a as its first argument and
// values from set b in its second argument, so it is safe to use with
// non-commutative functions.
//
// As with assertValueCompatible, we assume that the target audience of error
// messages here is a provider developer (via a bug report from a user) and so
// we intentionally violate our usual rule of keeping cty implementation
// details out of error messages.
func assertSetValuesCompatible(planned, actual cty.Value, path cty.Path, f func(aVal, bVal cty.Value) bool) []error {
	a := planned
	b := actual

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
			if f(av, bv) {
				aeqs[ai] = true
				beqs[bi] = true
			}
		}
	}

	var errs []error
	for i, eq := range aeqs {
		if !eq {
			errs = append(errs, path.NewErrorf("planned set element %#v does not correlate with any element in actual", as[i]))
		}
	}
	if len(errs) > 0 {
		// Exit early since otherwise we're likely to generate duplicate
		// error messages from the other perspective in the subsequent loop.
		return errs
	}
	for i, eq := range beqs {
		if !eq {
			errs = append(errs, path.NewErrorf("actual set element %#v does not correlate with any element in plan", bs[i]))
		}
	}
	return errs
}
