package cty

// Walk visits all of the values in a possibly-complex structure, calling
// a given function for each value.
//
// For example, given a list of strings the callback would first be called
// with the whole list and then called once for each element of the list.
//
// The callback function may prevent recursive visits to child values by
// returning false. The callback function my halt the walk altogether by
// returning a non-nil error. If the returned error is about the element
// currently being visited, it is recommended to use the provided path
// value to produce a PathError describing that context.
//
// The path passed to the given function may not be used after that function
// returns, since its backing array is re-used for other calls.
func Walk(val Value, cb func(Path, Value) (bool, error)) error {
	var path Path
	return walk(path, val, cb)
}

func walk(path Path, val Value, cb func(Path, Value) (bool, error)) error {
	deeper, err := cb(path, val)
	if err != nil {
		return err
	}
	if !deeper {
		return nil
	}

	if val.IsNull() || !val.IsKnown() {
		// Can't recurse into null or unknown values, regardless of type
		return nil
	}

	ty := val.Type()
	switch {
	case ty.IsObjectType():
		for it := val.ElementIterator(); it.Next(); {
			nameVal, av := it.Element()
			path := append(path, GetAttrStep{
				Name: nameVal.AsString(),
			})
			err := walk(path, av, cb)
			if err != nil {
				return err
			}
		}
	case val.CanIterateElements():
		for it := val.ElementIterator(); it.Next(); {
			kv, ev := it.Element()
			path := append(path, IndexStep{
				Key: kv,
			})
			err := walk(path, ev, cb)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Transform visits all of the values in a possibly-complex structure,
// calling a given function for each value which has an opportunity to
// replace that value.
//
// Unlike Walk, Transform visits child nodes first, so for a list of strings
// it would first visit the strings and then the _new_ list constructed
// from the transformed values of the list items.
//
// This is useful for creating the effect of being able to make deep mutations
// to a value even though values are immutable. However, it's the responsibility
// of the given function to preserve expected invariants, such as homogenity of
// element types in collections; this function can panic if such invariants
// are violated, just as if new values were constructed directly using the
// value constructor functions. An easy way to preserve invariants is to
// ensure that the transform function never changes the value type.
//
// The callback function my halt the walk altogether by
// returning a non-nil error. If the returned error is about the element
// currently being visited, it is recommended to use the provided path
// value to produce a PathError describing that context.
//
// The path passed to the given function may not be used after that function
// returns, since its backing array is re-used for other calls.
func Transform(val Value, cb func(Path, Value) (Value, error)) (Value, error) {
	var path Path
	return transform(path, val, cb)
}

func transform(path Path, val Value, cb func(Path, Value) (Value, error)) (Value, error) {
	ty := val.Type()
	var newVal Value

	switch {

	case val.IsNull() || !val.IsKnown():
		// Can't recurse into null or unknown values, regardless of type
		newVal = val

	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		l := val.LengthInt()
		switch l {
		case 0:
			// No deep transform for an empty sequence
			newVal = val
		default:
			elems := make([]Value, 0, l)
			for it := val.ElementIterator(); it.Next(); {
				kv, ev := it.Element()
				path := append(path, IndexStep{
					Key: kv,
				})
				newEv, err := transform(path, ev, cb)
				if err != nil {
					return DynamicVal, err
				}
				elems = append(elems, newEv)
			}
			switch {
			case ty.IsListType():
				newVal = ListVal(elems)
			case ty.IsSetType():
				newVal = SetVal(elems)
			case ty.IsTupleType():
				newVal = TupleVal(elems)
			default:
				panic("unknown sequence type") // should never happen because of the case we are in
			}
		}

	case ty.IsMapType():
		l := val.LengthInt()
		switch l {
		case 0:
			// No deep transform for an empty map
			newVal = val
		default:
			elems := make(map[string]Value)
			for it := val.ElementIterator(); it.Next(); {
				kv, ev := it.Element()
				path := append(path, IndexStep{
					Key: kv,
				})
				newEv, err := transform(path, ev, cb)
				if err != nil {
					return DynamicVal, err
				}
				elems[kv.AsString()] = newEv
			}
			newVal = MapVal(elems)
		}

	case ty.IsObjectType():
		switch {
		case ty.Equals(EmptyObject):
			// No deep transform for an empty object
			newVal = val
		default:
			atys := ty.AttributeTypes()
			newAVs := make(map[string]Value)
			for name := range atys {
				av := val.GetAttr(name)
				path := append(path, GetAttrStep{
					Name: name,
				})
				newAV, err := transform(path, av, cb)
				if err != nil {
					return DynamicVal, err
				}
				newAVs[name] = newAV
			}
			newVal = ObjectVal(newAVs)
		}

	default:
		newVal = val
	}

	return cb(path, newVal)
}
