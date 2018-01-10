package diffs

import (
	"github.com/zclconf/go-cty/cty"
)

// UnknownAsNull takes a value that may contain unknown values and returns
// a new value where those unknown values are replaced with null values of
// the same type.
//
// This is useful to help complete the construction of a resource whose
// provider may not have necessarily populated all computed attributes,
// to default any straggling attributes to null before storing them in state.
//
// However, it can also potentially mask bugs where values are not properly
// resolved during the apply phase, so care should be taken by callers.
//
// When this function is used with a set, it is possible that several set
// elements will collide to become a single element after conversion, since
// unknown values never compare equal but null values can. This may cause
// the resulting set to have fewer elements than the input set. In all other
// cases, input-to-output correspondence is preserved in nested structures.
func UnknownAsNull(val cty.Value) cty.Value {
	ty := val.Type()

	if !val.IsKnown() {
		return cty.NullVal(ty)
	}

	// The above is the main functionality here. The rest of this is just
	// to deal with recursively digging into data structures.

	if val.IsNull() {
		// Can't recurse into a null, so we're done
		return val
	}

	switch {
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		newVals := make(map[string]cty.Value, len(atys))
		for name := range atys {
			newVals[name] = UnknownAsNull(val.GetAttr(name))
		}
		return cty.ObjectVal(newVals)
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		newVals := make([]cty.Value, len(etys))
		for i := range etys {
			newVals[i] = UnknownAsNull(val.Index(cty.NumberIntVal(int64(i))))
		}
		return cty.TupleVal(newVals)
	case ty.IsListType() || ty.IsSetType():
		length := val.LengthInt()
		if length == 0 {
			return val
		}
		newVals := make([]cty.Value, 0, length)
		for it := val.ElementIterator(); it.Next(); {
			_, eVal := it.Element()
			newVals = append(newVals, UnknownAsNull(eVal))
		}
		if ty.IsSetType() {
			return cty.SetVal(newVals)
		}
		return cty.ListVal(newVals)
	case ty.IsMapType():
		length := val.LengthInt()
		if length == 0 {
			return val
		}
		newVals := make(map[string]cty.Value)
		for it := val.ElementIterator(); it.Next(); {
			eKey, eVal := it.Element()
			newVals[eKey.AsString()] = UnknownAsNull(eVal)
		}
		return cty.MapVal(newVals)
	default:
		return val
	}

}
