package convert

import (
	"github.com/zclconf/go-cty/cty"
)

// conversionCollectionToList returns a conversion that will apply the given
// conversion to all of the elements of a collection (something that supports
// ForEachElement and LengthInt) and then returns the result as a list.
//
// "conv" can be nil if the elements are expected to already be of the
// correct type and just need to be re-wrapped into a list. (For example,
// if we're converting from a set into a list of the same element type.)
func conversionCollectionToList(ety cty.Type, conv conversion) conversion {
	return func(val cty.Value, path cty.Path) (cty.Value, error) {
		elems := make([]cty.Value, 0, val.LengthInt())
		i := int64(0)
		path = append(path, nil)
		it := val.ElementIterator()
		for it.Next() {
			_, val := it.Element()
			var err error

			path[len(path)-1] = cty.IndexStep{
				Key: cty.NumberIntVal(i),
			}

			if conv != nil {
				val, err = conv(val, path)
				if err != nil {
					return cty.NilVal, err
				}
			}
			elems = append(elems, val)

			i++
		}

		if len(elems) == 0 {
			return cty.ListValEmpty(ety), nil
		}

		return cty.ListVal(elems), nil
	}
}

// conversionTupleToList returns a conversion that will take a value of the
// given tuple type and return a list of the given element type.
//
// Will panic if the given tupleType isn't actually a tuple type.
func conversionTupleToList(tupleType cty.Type, listEty cty.Type, unsafe bool) conversion {
	tupleEtys := tupleType.TupleElementTypes()

	if len(tupleEtys) == 0 {
		// Empty tuple short-circuit
		return func(val cty.Value, path cty.Path) (cty.Value, error) {
			return cty.ListValEmpty(listEty), nil
		}
	}

	if listEty == cty.DynamicPseudoType {
		// This is a special case where the caller wants us to find
		// a suitable single type that all elements can convert to, if
		// possible.
		listEty, _ = unify(tupleEtys, unsafe)
		if listEty == cty.NilType {
			return nil
		}
	}

	elemConvs := make([]conversion, len(tupleEtys))
	for i, tupleEty := range tupleEtys {
		if tupleEty.Equals(listEty) {
			// no conversion required
			continue
		}

		elemConvs[i] = getConversion(tupleEty, listEty, unsafe)
		if elemConvs[i] == nil {
			// If any of our element conversions are impossible, then the our
			// whole conversion is impossible.
			return nil
		}
	}

	// If we fall out here then a conversion is possible, using the
	// element conversions in elemConvs
	return func(val cty.Value, path cty.Path) (cty.Value, error) {
		elems := make([]cty.Value, 0, len(elemConvs))
		path = append(path, nil)
		i := int64(0)
		it := val.ElementIterator()
		for it.Next() {
			_, val := it.Element()
			var err error

			path[len(path)-1] = cty.IndexStep{
				Key: cty.NumberIntVal(i),
			}

			conv := elemConvs[i]
			if conv != nil {
				val, err = conv(val, path)
				if err != nil {
					return cty.NilVal, err
				}
			}
			elems = append(elems, val)

			i++
		}

		return cty.ListVal(elems), nil
	}
}
