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

	for name := range schema.Attributes {
		plannedV := planned.GetAttr(name)
		actualV := actual.GetAttr(name)

		path := append(path, cty.GetAttrStep{Name: name})
		moreErrs := assertValueCompatible(plannedV, actualV, path)
		errs = append(errs, moreErrs...)
	}
	for name, blockS := range schema.BlockTypes {
		plannedV := planned.GetAttr(name)
		actualV := actual.GetAttr(name)

		// As a special case, we permit a "planned" block with exactly one
		// element where all of the "leaf" values are unknown, since that's
		// what HCL's dynamic block extension generates if the for_each
		// expression is itself unknown and thus it cannot predict how many
		// child blocks will get created.
		switch blockS.Nesting {
		case configschema.NestingSingle:
			if allLeafValuesUnknown(plannedV) && !plannedV.IsNull() {
				return errs
			}
		case configschema.NestingList, configschema.NestingMap, configschema.NestingSet:
			if plannedV.IsKnown() && !plannedV.IsNull() && plannedV.LengthInt() == 1 {
				elemVs := plannedV.AsValueSlice()
				if allLeafValuesUnknown(elemVs[0]) {
					return errs
				}
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", blockS.Nesting))
		}

		path := append(path, cty.GetAttrStep{Name: name})
		switch blockS.Nesting {
		case configschema.NestingSingle:
			moreErrs := assertObjectCompatible(&blockS.Block, plannedV, actualV, path)
			errs = append(errs, moreErrs...)
		case configschema.NestingList:
			// A NestingList might either be a list or a tuple, depending on
			// whether there are dynamically-typed attributes inside. However,
			// both support a similar-enough API that we can treat them the
			// same for our purposes here.
			if !plannedV.IsKnown() || plannedV.IsNull() || actualV.IsNull() {
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
				for k := range actualAtys {
					if _, ok := plannedAtys[k]; !ok {
						errs = append(errs, path.NewErrorf("new block key %q has appeared", k))
						continue
					}
				}
			} else {
				if !plannedV.IsKnown() || plannedV.IsNull() || actualV.IsNull() {
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
			}
		case configschema.NestingSet:
			// We can't do any reasonable matching of set elements since their
			// content is also their key, and so we have no way to correlate
			// them. Because of this, we simply verify that we still have the
			// same number of elements.
			if !plannedV.IsKnown() || plannedV.IsNull() || actualV.IsNull() {
				continue
			}
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
	}
	if planned.IsNull() {
		errs = append(errs, path.NewErrorf("was null, but now %#v", actual))
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

func allLeafValuesUnknown(v cty.Value) bool {
	seenKnownValue := false
	cty.Walk(v, func(path cty.Path, cv cty.Value) (bool, error) {
		if cv.IsNull() {
			seenKnownValue = true
		}
		if cv.Type().IsPrimitiveType() && cv.IsKnown() {
			seenKnownValue = true
		}
		return true, nil
	})
	return !seenKnownValue
}
