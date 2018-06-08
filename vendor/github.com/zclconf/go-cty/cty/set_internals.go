package cty

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"math/big"
	"sort"
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

func (r setRules) Hash(v interface{}) int {
	hashBytes := makeSetHashBytes(Value{
		ty: r.Type,
		v:  v,
	})
	return int(crc32.ChecksumIEEE(hashBytes))
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
