package cty

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"math/big"
	"sort"

	"github.com/zclconf/go-cty/cty/set"
)

// setRules provides a Rules implementation for the ./set package that
// respects the equality rules for cty values of the given type.
//
// This implementation expects that values added to the set will be
// valid internal values for the given Type, which is to say that wrapping
// the given value in a Value struct along with the ruleset's type should
// produce a valid, working Value.
type setRules struct {
	Type Type
}

var _ set.OrderedRules = setRules{}

// Hash returns a hash value for the receiver that can be used for equality
// checks where some inaccuracy is tolerable.
//
// The hash function is value-type-specific, so it is not meaningful to compare
// hash results for values of different types.
//
// This function is not safe to use for security-related applications, since
// the hash used is not strong enough.
func (val Value) Hash() int {
	hashBytes := makeSetHashBytes(val)
	return int(crc32.ChecksumIEEE(hashBytes))
}

func (r setRules) Hash(v interface{}) int {
	return Value{
		ty: r.Type,
		v:  v,
	}.Hash()
}

func (r setRules) Equivalent(v1 interface{}, v2 interface{}) bool {
	v1v := Value{
		ty: r.Type,
		v:  v1,
	}
	v2v := Value{
		ty: r.Type,
		v:  v2,
	}

	eqv := v1v.Equals(v2v)

	// By comparing the result to true we ensure that an Unknown result,
	// which will result if either value is unknown, will be considered
	// as non-equivalent. Two unknown values are not equivalent for the
	// sake of set membership.
	return eqv.v == true
}

// Less is an implementation of set.OrderedRules so that we can iterate over
// set elements in a consistent order, where such an order is possible.
func (r setRules) Less(v1, v2 interface{}) bool {
	v1v := Value{
		ty: r.Type,
		v:  v1,
	}
	v2v := Value{
		ty: r.Type,
		v:  v2,
	}

	if v1v.RawEquals(v2v) { // Easy case: if they are equal then v1 can't be less
		return false
	}

	// Null values always sort after non-null values
	if v2v.IsNull() && !v1v.IsNull() {
		return true
	} else if v1v.IsNull() {
		return false
	}
	// Unknown values always sort after known values
	if v1v.IsKnown() && !v2v.IsKnown() {
		return true
	} else if !v1v.IsKnown() {
		return false
	}

	switch r.Type {
	case String:
		// String values sort lexicographically
		return v1v.AsString() < v2v.AsString()
	case Bool:
		// Weird to have a set of bools, but if we do then false sorts before true.
		if v2v.True() || !v1v.True() {
			return true
		}
		return false
	case Number:
		v1f := v1v.AsBigFloat()
		v2f := v2v.AsBigFloat()
		return v1f.Cmp(v2f) < 0
	default:
		// No other types have a well-defined ordering, so we just produce a
		// default consistent-but-undefined ordering then. This situation is
		// not considered a compatibility constraint; callers should rely only
		// on the ordering rules for primitive values.
		v1h := makeSetHashBytes(v1v)
		v2h := makeSetHashBytes(v2v)
		return bytes.Compare(v1h, v2h) < 0
	}
}

func makeSetHashBytes(val Value) []byte {
	var buf bytes.Buffer
	appendSetHashBytes(val, &buf)
	return buf.Bytes()
}

func appendSetHashBytes(val Value, buf *bytes.Buffer) {
	// Exactly what bytes we generate here don't matter as long as the following
	// constraints hold:
	// - Unknown and null values all generate distinct strings from
	//   each other and from any normal value of the given type.
	// - The delimiter used to separate items in a compound structure can
	//   never appear literally in any of its elements.
	// Since we don't support hetrogenous lists we don't need to worry about
	// collisions between values of different types, apart from
	// PseudoTypeDynamic.
	// If in practice we *do* get a collision then it's not a big deal because
	// the Equivalent function will still distinguish values, but set
	// performance will be best if we are able to produce a distinct string
	// for each distinct value, unknown values notwithstanding.
	if !val.IsKnown() {
		buf.WriteRune('?')
		return
	}
	if val.IsNull() {
		buf.WriteRune('~')
		return
	}

	switch val.ty {
	case Number:
		buf.WriteString(val.v.(*big.Float).String())
		return
	case Bool:
		if val.v.(bool) {
			buf.WriteRune('T')
		} else {
			buf.WriteRune('F')
		}
		return
	case String:
		buf.WriteString(fmt.Sprintf("%q", val.v.(string)))
		return
	}

	if val.ty.IsMapType() {
		buf.WriteRune('{')
		val.ForEachElement(func(keyVal, elementVal Value) bool {
			appendSetHashBytes(keyVal, buf)
			buf.WriteRune(':')
			appendSetHashBytes(elementVal, buf)
			buf.WriteRune(';')
			return false
		})
		buf.WriteRune('}')
		return
	}

	if val.ty.IsListType() || val.ty.IsSetType() {
		buf.WriteRune('[')
		val.ForEachElement(func(keyVal, elementVal Value) bool {
			appendSetHashBytes(elementVal, buf)
			buf.WriteRune(';')
			return false
		})
		buf.WriteRune(']')
		return
	}

	if val.ty.IsObjectType() {
		buf.WriteRune('<')
		attrNames := make([]string, 0, len(val.ty.AttributeTypes()))
		for attrName := range val.ty.AttributeTypes() {
			attrNames = append(attrNames, attrName)
		}
		sort.Strings(attrNames)
		for _, attrName := range attrNames {
			appendSetHashBytes(val.GetAttr(attrName), buf)
			buf.WriteRune(';')
		}
		buf.WriteRune('>')
		return
	}

	if val.ty.IsTupleType() {
		buf.WriteRune('<')
		val.ForEachElement(func(keyVal, elementVal Value) bool {
			appendSetHashBytes(elementVal, buf)
			buf.WriteRune(';')
			return false
		})
		buf.WriteRune('>')
		return
	}

	// should never get down here
	panic("unsupported type in set hash")
}
