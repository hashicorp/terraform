package cty

import (
	"fmt"
	"math/big"
	"reflect"

	"golang.org/x/text/unicode/norm"

	"github.com/zclconf/go-cty/cty/set"
)

// BoolVal returns a Value of type Number whose internal value is the given
// bool.
func BoolVal(v bool) Value {
	return Value{
		ty: Bool,
		v:  v,
	}
}

// NumberVal returns a Value of type Number whose internal value is the given
// big.Float. The returned value becomes the owner of the big.Float object,
// and so it's forbidden for the caller to mutate the object after it's
// wrapped in this way.
func NumberVal(v *big.Float) Value {
	return Value{
		ty: Number,
		v:  v,
	}
}

// NumberIntVal returns a Value of type Number whose internal value is equal
// to the given integer.
func NumberIntVal(v int64) Value {
	return NumberVal(new(big.Float).SetInt64(v))
}

// NumberUIntVal returns a Value of type Number whose internal value is equal
// to the given unsigned integer.
func NumberUIntVal(v uint64) Value {
	return NumberVal(new(big.Float).SetUint64(v))
}

// NumberFloatVal returns a Value of type Number whose internal value is
// equal to the given float.
func NumberFloatVal(v float64) Value {
	return NumberVal(new(big.Float).SetFloat64(v))
}

// StringVal returns a Value of type String whose internal value is the
// given string.
//
// Strings must be UTF-8 encoded sequences of valid unicode codepoints, and
// they are NFC-normalized on entry into the world of cty values.
//
// If the given string is not valid UTF-8 then behavior of string operations
// is undefined.
func StringVal(v string) Value {
	return Value{
		ty: String,
		v:  NormalizeString(v),
	}
}

// NormalizeString applies the same normalization that cty applies when
// constructing string values.
//
// A return value from this function can be meaningfully compared byte-for-byte
// with a Value.AsString result.
func NormalizeString(s string) string {
	return norm.NFC.String(s)
}

// ObjectVal returns a Value of an object type whose structure is defined
// by the key names and value types in the given map.
func ObjectVal(attrs map[string]Value) Value {
	attrTypes := make(map[string]Type, len(attrs))
	attrVals := make(map[string]interface{}, len(attrs))

	for attr, val := range attrs {
		attr = NormalizeString(attr)
		attrTypes[attr] = val.ty
		attrVals[attr] = val.v
	}

	return Value{
		ty: Object(attrTypes),
		v:  attrVals,
	}
}

// TupleVal returns a Value of a tuple type whose element types are
// defined by the value types in the given slice.
func TupleVal(elems []Value) Value {
	elemTypes := make([]Type, len(elems))
	elemVals := make([]interface{}, len(elems))

	for i, val := range elems {
		elemTypes[i] = val.ty
		elemVals[i] = val.v
	}

	return Value{
		ty: Tuple(elemTypes),
		v:  elemVals,
	}
}

// ListVal returns a Value of list type whose element type is defined by
// the types of the given values, which must be homogenous.
//
// If the types are not all consistent (aside from elements that are of the
// dynamic pseudo-type) then this function will panic. It will panic also
// if the given list is empty, since then the element type cannot be inferred.
// (See also ListValEmpty.)
func ListVal(vals []Value) Value {
	if len(vals) == 0 {
		panic("must not call ListVal with empty slice")
	}
	elementType := DynamicPseudoType
	rawList := make([]interface{}, len(vals))

	for i, val := range vals {
		if elementType == DynamicPseudoType {
			elementType = val.ty
		} else if val.ty != DynamicPseudoType && !elementType.Equals(val.ty) {
			panic(fmt.Errorf(
				"inconsistent list element types (%#v then %#v)",
				elementType, val.ty,
			))
		}

		rawList[i] = val.v
	}

	return Value{
		ty: List(elementType),
		v:  rawList,
	}
}

// ListValEmpty returns an empty list of the given element type.
func ListValEmpty(element Type) Value {
	return Value{
		ty: List(element),
		v:  []interface{}{},
	}
}

// MapVal returns a Value of a map type whose element type is defined by
// the types of the given values, which must be homogenous.
//
// If the types are not all consistent (aside from elements that are of the
// dynamic pseudo-type) then this function will panic. It will panic also
// if the given map is empty, since then the element type cannot be inferred.
// (See also MapValEmpty.)
func MapVal(vals map[string]Value) Value {
	if len(vals) == 0 {
		panic("must not call MapVal with empty map")
	}
	elementType := DynamicPseudoType
	rawMap := make(map[string]interface{}, len(vals))

	for key, val := range vals {
		if elementType == DynamicPseudoType {
			elementType = val.ty
		} else if val.ty != DynamicPseudoType && !elementType.Equals(val.ty) {
			panic(fmt.Errorf(
				"inconsistent map element types (%#v then %#v)",
				elementType, val.ty,
			))
		}

		rawMap[NormalizeString(key)] = val.v
	}

	return Value{
		ty: Map(elementType),
		v:  rawMap,
	}
}

// MapValEmpty returns an empty map of the given element type.
func MapValEmpty(element Type) Value {
	return Value{
		ty: Map(element),
		v:  map[string]interface{}{},
	}
}

// SetVal returns a Value of set type whose element type is defined by
// the types of the given values, which must be homogenous.
//
// If the types are not all consistent (aside from elements that are of the
// dynamic pseudo-type) then this function will panic. It will panic also
// if the given list is empty, since then the element type cannot be inferred.
// (See also SetValEmpty.)
func SetVal(vals []Value) Value {
	if len(vals) == 0 {
		panic("must not call SetVal with empty slice")
	}
	elementType := DynamicPseudoType
	rawList := make([]interface{}, len(vals))

	for i, val := range vals {
		if elementType == DynamicPseudoType {
			elementType = val.ty
		} else if val.ty != DynamicPseudoType && !elementType.Equals(val.ty) {
			panic(fmt.Errorf(
				"inconsistent set element types (%#v then %#v)",
				elementType, val.ty,
			))
		}

		rawList[i] = val.v
	}

	rawVal := set.NewSetFromSlice(setRules{elementType}, rawList)

	return Value{
		ty: Set(elementType),
		v:  rawVal,
	}
}

// SetValFromValueSet returns a Value of set type based on an already-constructed
// ValueSet.
//
// The element type of the returned value is the element type of the given
// set.
func SetValFromValueSet(s ValueSet) Value {
	ety := s.ElementType()
	rawVal := s.s.Copy() // copy so caller can't mutate what we wrap

	return Value{
		ty: Set(ety),
		v:  rawVal,
	}
}

// SetValEmpty returns an empty set of the given element type.
func SetValEmpty(element Type) Value {
	return Value{
		ty: Set(element),
		v:  set.NewSet(setRules{element}),
	}
}

// CapsuleVal creates a value of the given capsule type using the given
// wrapVal, which must be a pointer to a value of the capsule type's native
// type.
//
// This function will panic if the given type is not a capsule type, if
// the given wrapVal is not compatible with the given capsule type, or if
// wrapVal is not a pointer.
func CapsuleVal(ty Type, wrapVal interface{}) Value {
	if !ty.IsCapsuleType() {
		panic("not a capsule type")
	}

	wv := reflect.ValueOf(wrapVal)
	if wv.Kind() != reflect.Ptr {
		panic("wrapVal is not a pointer")
	}

	it := ty.typeImpl.(*capsuleType).GoType
	if !wv.Type().Elem().AssignableTo(it) {
		panic("wrapVal target is not compatible with the given capsule type")
	}

	return Value{
		ty: ty,
		v:  wrapVal,
	}
}
