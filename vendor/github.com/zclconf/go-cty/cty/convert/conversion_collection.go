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

// conversionCollectionToSet returns a conversion that will apply the given
// conversion to all of the elements of a collection (something that supports
// ForEachElement and LengthInt) and then returns the result as a set.
//
// "conv" can be nil if the elements are expected to already be of the
// correct type and just need to be re-wrapped into a set. (For example,
// if we're converting from a list into a set of the same element type.)
func conversionCollectionToSet(ety cty.Type, conv conversion) conversion {
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
			return cty.SetValEmpty(ety), nil
		}

		return cty.SetVal(elems), nil
	}
}

// conversionCollectionToMap returns a conversion that will apply the given
// conversion to all of the elements of a collection (something that supports
// ForEachElement and LengthInt) and then returns the result as a map.
//
// "conv" can be nil if the elements are expected to already be of the
// correct type and just need to be re-wrapped into a map.
func conversionCollectionToMap(ety cty.Type, conv conversion) conversion {
	return func(val cty.Value, path cty.Path) (cty.Value, error) {
		elems := make(map[string]cty.Value, 0)
		path = append(path, nil)
		it := val.ElementIterator()
		for it.Next() {
			key, val := it.Element()
			var err error

			path[len(path)-1] = cty.IndexStep{
				Key: key,
			}

			keyStr, err := Convert(key, cty.String)
			if err != nil {
				// Should never happen, because keys can only be numbers or
				// strings and both can convert to string.
				return cty.DynamicVal, path.NewErrorf("cannot convert key type %s to string for map", key.Type().FriendlyName())
			}

			if conv != nil {
				val, err = conv(val, path)
				if err != nil {
					return cty.NilVal, err
				}
			}

			elems[keyStr.AsString()] = val
		}

		if len(elems) == 0 {
			return cty.MapValEmpty(ety), nil
		}

		return cty.MapVal(elems), nil
	}
}

// conversionTupleToSet returns a conversion that will take a value of the
// given tuple type and return a set of the given element type.
//
// Will panic if the given tupleType isn't actually a tuple type.
func conversionTupleToSet(tupleType cty.Type, listEty cty.Type, unsafe bool) conversion {
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

		return cty.SetVal(elems), nil
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

// conversionObjectToMap returns a conversion that will take a value of the
// given object type and return a map of the given element type.
//
// Will panic if the given objectType isn't actually an object type.
func conversionObjectToMap(objectType cty.Type, mapEty cty.Type, unsafe bool) conversion {
	objectAtys := objectType.AttributeTypes()

	if len(objectAtys) == 0 {
		// Empty object short-circuit
		return func(val cty.Value, path cty.Path) (cty.Value, error) {
			return cty.MapValEmpty(mapEty), nil
		}
	}

	if mapEty == cty.DynamicPseudoType {
		// This is a special case where the caller wants us to find
		// a suitable single type that all elements can convert to, if
		// possible.
		objectAtysList := make([]cty.Type, 0, len(objectAtys))
		for _, aty := range objectAtys {
			objectAtysList = append(objectAtysList, aty)
		}
		mapEty, _ = unify(objectAtysList, unsafe)
		if mapEty == cty.NilType {
			return nil
		}
	}

	elemConvs := make(map[string]conversion, len(objectAtys))
	for name, objectAty := range objectAtys {
		if objectAty.Equals(mapEty) {
			// no conversion required
			continue
		}

		elemConvs[name] = getConversion(objectAty, mapEty, unsafe)
		if elemConvs[name] == nil {
			// If any of our element conversions are impossible, then the our
			// whole conversion is impossible.
			return nil
		}
	}

	// If we fall out here then a conversion is possible, using the
	// element conversions in elemConvs
	return func(val cty.Value, path cty.Path) (cty.Value, error) {
		elems := make(map[string]cty.Value, len(elemConvs))
		path = append(path, nil)
		it := val.ElementIterator()
		for it.Next() {
			name, val := it.Element()
			var err error

			path[len(path)-1] = cty.IndexStep{
				Key: name,
			}

			conv := elemConvs[name.AsString()]
			if conv != nil {
				val, err = conv(val, path)
				if err != nil {
					return cty.NilVal, err
				}
			}
			elems[name.AsString()] = val
		}

		return cty.MapVal(elems), nil
	}
}
