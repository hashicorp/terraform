package cty

import (
	"fmt"
	"math/big"

	"reflect"

	"github.com/zclconf/go-cty/cty/set"
)

func (val Value) GoString() string {
	if val == NilVal {
		return "cty.NilVal"
	}

	if val.IsNull() {
		return fmt.Sprintf("cty.NullVal(%#v)", val.ty)
	}
	if val == DynamicVal { // is unknown, so must be before the IsKnown check below
		return "cty.DynamicVal"
	}
	if !val.IsKnown() {
		return fmt.Sprintf("cty.UnknownVal(%#v)", val.ty)
	}

	// By the time we reach here we've dealt with all of the exceptions around
	// unknowns and nulls, so we're guaranteed that the values are the
	// canonical internal representation of the given type.

	switch val.ty {
	case Bool:
		if val.v.(bool) {
			return "cty.True"
		} else {
			return "cty.False"
		}
	case Number:
		fv := val.v.(*big.Float)
		// We'll try to use NumberIntVal or NumberFloatVal if we can, since
		// the fully-general initializer call is pretty ugly-looking.
		if fv.IsInt() {
			return fmt.Sprintf("cty.NumberIntVal(%#v)", fv)
		}
		if rfv, accuracy := fv.Float64(); accuracy == big.Exact {
			return fmt.Sprintf("cty.NumberFloatVal(%#v)", rfv)
		}
		return fmt.Sprintf("cty.NumberVal(new(big.Float).Parse(\"%#v\", 10))", fv)
	case String:
		return fmt.Sprintf("cty.StringVal(%#v)", val.v)
	}

	switch {
	case val.ty.IsSetType():
		vals := val.v.(set.Set).Values()
		if vals == nil || len(vals) == 0 {
			return fmt.Sprintf("cty.SetValEmpty()")
		} else {
			return fmt.Sprintf("cty.SetVal(%#v)", vals)
		}
	case val.ty.IsCapsuleType():
		return fmt.Sprintf("cty.CapsuleVal(%#v, %#v)", val.ty, val.v)
	}

	// Default exposes implementation details, so should actually cover
	// all of the cases above for good caller UX.
	return fmt.Sprintf("cty.Value{ty: %#v, v: %#v}", val.ty, val.v)
}

// Equals returns True if the receiver and the given other value have the
// same type and are exactly equal in value.
//
// The usual short-circuit rules apply, so the result can be unknown or typed
// as dynamic if either of the given values are. Use RawEquals to compare
// if two values are equal *ignoring* the short-circuit rules.
func (val Value) Equals(other Value) Value {
	if val.ty.HasDynamicTypes() || other.ty.HasDynamicTypes() {
		return UnknownVal(Bool)
	}

	if !val.ty.Equals(other.ty) {
		return BoolVal(false)
	}

	if !(val.IsKnown() && other.IsKnown()) {
		return UnknownVal(Bool)
	}

	if val.IsNull() || other.IsNull() {
		if val.IsNull() && other.IsNull() {
			return BoolVal(true)
		}
		return BoolVal(false)
	}

	ty := val.ty
	result := false

	switch {
	case ty == Number:
		result = val.v.(*big.Float).Cmp(other.v.(*big.Float)) == 0
	case ty == Bool:
		result = val.v.(bool) == other.v.(bool)
	case ty == String:
		// Simple equality is safe because we NFC-normalize strings as they
		// enter our world from StringVal, and so we can assume strings are
		// always in normal form.
		result = val.v.(string) == other.v.(string)
	case ty.IsObjectType():
		oty := ty.typeImpl.(typeObject)
		result = true
		for attr, aty := range oty.AttrTypes {
			lhs := Value{
				ty: aty,
				v:  val.v.(map[string]interface{})[attr],
			}
			rhs := Value{
				ty: aty,
				v:  other.v.(map[string]interface{})[attr],
			}
			eq := lhs.Equals(rhs)
			if !eq.IsKnown() {
				return UnknownVal(Bool)
			}
			if eq.False() {
				result = false
				break
			}
		}
	case ty.IsTupleType():
		tty := ty.typeImpl.(typeTuple)
		result = true
		for i, ety := range tty.ElemTypes {
			lhs := Value{
				ty: ety,
				v:  val.v.([]interface{})[i],
			}
			rhs := Value{
				ty: ety,
				v:  other.v.([]interface{})[i],
			}
			eq := lhs.Equals(rhs)
			if !eq.IsKnown() {
				return UnknownVal(Bool)
			}
			if eq.False() {
				result = false
				break
			}
		}
	case ty.IsListType():
		ety := ty.typeImpl.(typeList).ElementTypeT
		if len(val.v.([]interface{})) == len(other.v.([]interface{})) {
			result = true
			for i := range val.v.([]interface{}) {
				lhs := Value{
					ty: ety,
					v:  val.v.([]interface{})[i],
				}
				rhs := Value{
					ty: ety,
					v:  other.v.([]interface{})[i],
				}
				eq := lhs.Equals(rhs)
				if !eq.IsKnown() {
					return UnknownVal(Bool)
				}
				if eq.False() {
					result = false
					break
				}
			}
		}
	case ty.IsSetType():
		s1 := val.v.(set.Set)
		s2 := other.v.(set.Set)
		equal := true

		// Note that by our definition of sets it's never possible for two
		// sets that contain unknown values (directly or indicrectly) to
		// ever be equal, even if they are otherwise identical.

		// FIXME: iterating both lists and checking each item is not the
		// ideal implementation here, but it works with the primitives we
		// have in the set implementation. Perhaps the set implementation
		// can provide its own equality test later.
		s1.EachValue(func(v interface{}) {
			if !s2.Has(v) {
				equal = false
			}
		})
		s2.EachValue(func(v interface{}) {
			if !s1.Has(v) {
				equal = false
			}
		})

		result = equal
	case ty.IsMapType():
		ety := ty.typeImpl.(typeMap).ElementTypeT
		if len(val.v.(map[string]interface{})) == len(other.v.(map[string]interface{})) {
			result = true
			for k := range val.v.(map[string]interface{}) {
				if _, ok := other.v.(map[string]interface{})[k]; !ok {
					result = false
					break
				}
				lhs := Value{
					ty: ety,
					v:  val.v.(map[string]interface{})[k],
				}
				rhs := Value{
					ty: ety,
					v:  other.v.(map[string]interface{})[k],
				}
				eq := lhs.Equals(rhs)
				if !eq.IsKnown() {
					return UnknownVal(Bool)
				}
				if eq.False() {
					result = false
					break
				}
			}
		}
	case ty.IsCapsuleType():
		// A capsule type's encapsulated value is a pointer to a value of its
		// native type, so we can just compare these to get the identity test
		// we need.
		return BoolVal(val.v == other.v)

	default:
		// should never happen
		panic(fmt.Errorf("unsupported value type %#v in Equals", ty))
	}

	return BoolVal(result)
}

// NotEqual is a shorthand for Equals followed by Not.
func (val Value) NotEqual(other Value) Value {
	return val.Equals(other).Not()
}

// True returns true if the receiver is True, false if False, and panics if
// the receiver is not of type Bool.
//
// This is a helper function to help write application logic that works with
// values, rather than a first-class operation. It does not work with unknown
// or null values. For more robust handling with unknown value
// short-circuiting, use val.Equals(cty.True).
func (val Value) True() bool {
	if val.ty != Bool {
		panic("not bool")
	}
	return val.Equals(True).v.(bool)
}

// False is the opposite of True.
func (val Value) False() bool {
	return !val.True()
}

// RawEquals returns true if and only if the two given values have the same
// type and equal value, ignoring the usual short-circuit rules about
// unknowns and dynamic types.
//
// This method is more appropriate for testing than for real use, since it
// skips over usual semantics around unknowns but as a consequence allows
// testing the result of another operation that is expected to return unknown.
// It returns a primitive Go bool rather than a Value to remind us that it
// is not a first-class value operation.
func (val Value) RawEquals(other Value) bool {
	if !val.ty.Equals(other.ty) {
		return false
	}
	if (!val.IsKnown()) && (!other.IsKnown()) {
		return true
	}
	if (val.IsKnown() && !other.IsKnown()) || (other.IsKnown() && !val.IsKnown()) {
		return false
	}
	if val.IsNull() && other.IsNull() {
		return true
	}
	if (val.IsNull() && !other.IsNull()) || (other.IsNull() && !val.IsNull()) {
		return false
	}
	if val.ty == DynamicPseudoType && other.ty == DynamicPseudoType {
		return true
	}

	ty := val.ty
	switch {
	case ty == Number || ty == Bool || ty == String || ty == DynamicPseudoType:
		return val.Equals(other).True()
	case ty.IsObjectType():
		oty := ty.typeImpl.(typeObject)
		for attr, aty := range oty.AttrTypes {
			lhs := Value{
				ty: aty,
				v:  val.v.(map[string]interface{})[attr],
			}
			rhs := Value{
				ty: aty,
				v:  other.v.(map[string]interface{})[attr],
			}
			eq := lhs.RawEquals(rhs)
			if !eq {
				return false
			}
		}
		return true
	case ty.IsTupleType():
		tty := ty.typeImpl.(typeTuple)
		for i, ety := range tty.ElemTypes {
			lhs := Value{
				ty: ety,
				v:  val.v.([]interface{})[i],
			}
			rhs := Value{
				ty: ety,
				v:  other.v.([]interface{})[i],
			}
			eq := lhs.RawEquals(rhs)
			if !eq {
				return false
			}
		}
		return true
	case ty.IsListType():
		ety := ty.typeImpl.(typeList).ElementTypeT
		if len(val.v.([]interface{})) == len(other.v.([]interface{})) {
			for i := range val.v.([]interface{}) {
				lhs := Value{
					ty: ety,
					v:  val.v.([]interface{})[i],
				}
				rhs := Value{
					ty: ety,
					v:  other.v.([]interface{})[i],
				}
				eq := lhs.RawEquals(rhs)
				if !eq {
					return false
				}
			}
			return true
		}
		return false
	case ty.IsSetType():
		s1 := val.v.(set.Set)
		s2 := other.v.(set.Set)

		// Since we're intentionally ignoring our rule that two unknowns
		// are never equal, we can cheat here.
		// (This isn't 100% right since e.g. it will fail if the set contains
		// numbers that are infinite, which DeepEqual can't compare properly.
		// We're accepting that limitation for simplicity here, since this
		// function is here primarily for testing.)
		return reflect.DeepEqual(s1, s2)

	case ty.IsMapType():
		ety := ty.typeImpl.(typeMap).ElementTypeT
		if len(val.v.(map[string]interface{})) == len(other.v.(map[string]interface{})) {
			for k := range val.v.(map[string]interface{}) {
				if _, ok := other.v.(map[string]interface{})[k]; !ok {
					return false
				}
				lhs := Value{
					ty: ety,
					v:  val.v.(map[string]interface{})[k],
				}
				rhs := Value{
					ty: ety,
					v:  other.v.(map[string]interface{})[k],
				}
				eq := lhs.RawEquals(rhs)
				if !eq {
					return false
				}
			}
			return true
		}
		return false
	case ty.IsCapsuleType():
		// A capsule type's encapsulated value is a pointer to a value of its
		// native type, so we can just compare these to get the identity test
		// we need.
		return val.v == other.v

	default:
		// should never happen
		panic(fmt.Errorf("unsupported value type %#v in RawEquals", ty))
	}
}

// Add returns the sum of the receiver and the given other value. Both values
// must be numbers; this method will panic if not.
func (val Value) Add(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Number, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	ret := new(big.Float)
	ret.Add(val.v.(*big.Float), other.v.(*big.Float))
	return NumberVal(ret)
}

// Subtract returns receiver minus the given other value. Both values must be
// numbers; this method will panic if not.
func (val Value) Subtract(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Number, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	return val.Add(other.Negate())
}

// Negate returns the numeric negative of the receiver, which must be a number.
// This method will panic when given a value of any other type.
func (val Value) Negate() Value {
	if shortCircuit := mustTypeCheck(Number, Number, val); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	ret := new(big.Float).Neg(val.v.(*big.Float))
	return NumberVal(ret)
}

// Multiply returns the product of the receiver and the given other value.
// Both values must be numbers; this method will panic if not.
func (val Value) Multiply(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Number, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	ret := new(big.Float)
	ret.Mul(val.v.(*big.Float), other.v.(*big.Float))
	return NumberVal(ret)
}

// Divide returns the quotient of the receiver and the given other value.
// Both values must be numbers; this method will panic if not.
//
// If the "other" value is exactly zero, this operation will return either
// PositiveInfinity or NegativeInfinity, depending on the sign of the
// receiver value. For some use-cases the presence of infinities may be
// undesirable, in which case the caller should check whether the
// other value equals zero before calling and raise an error instead.
//
// If both values are zero or infinity, this function will panic with
// an instance of big.ErrNaN.
func (val Value) Divide(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Number, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	ret := new(big.Float)
	ret.Quo(val.v.(*big.Float), other.v.(*big.Float))
	return NumberVal(ret)
}

// Modulo returns the remainder of an integer division of the receiver and
// the given other value. Both values must be numbers; this method will panic
// if not.
//
// If the "other" value is exactly zero, this operation will return either
// PositiveInfinity or NegativeInfinity, depending on the sign of the
// receiver value. For some use-cases the presence of infinities may be
// undesirable, in which case the caller should check whether the
// other value equals zero before calling and raise an error instead.
//
// This operation is primarily here for use with nonzero natural numbers.
// Modulo with "other" as a non-natural number gets somewhat philosophical,
// and this function takes a position on what that should mean, but callers
// may wish to disallow such things outright or implement their own modulo
// if they disagree with the interpretation used here.
func (val Value) Modulo(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Number, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	// We cheat a bit here with infinities, just abusing the Multiply operation
	// to get an infinite result of the correct sign.
	if val == PositiveInfinity || val == NegativeInfinity || other == PositiveInfinity || other == NegativeInfinity {
		return val.Multiply(other)
	}

	if other.RawEquals(Zero) {
		return val
	}

	// FIXME: This is a bit clumsy. Should come back later and see if there's a
	// more straightforward way to do this.
	rat := val.Divide(other)
	ratFloorInt := &big.Int{}
	rat.v.(*big.Float).Int(ratFloorInt)
	work := (&big.Float{}).SetInt(ratFloorInt)
	work.Mul(other.v.(*big.Float), work)
	work.Sub(val.v.(*big.Float), work)

	return NumberVal(work)
}

// Absolute returns the absolute (signless) value of the receiver, which must
// be a number or this method will panic.
func (val Value) Absolute() Value {
	if shortCircuit := mustTypeCheck(Number, Number, val); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Number)
		return *shortCircuit
	}

	ret := (&big.Float{}).Abs(val.v.(*big.Float))
	return NumberVal(ret)
}

// GetAttr returns the value of the given attribute of the receiver, which
// must be of an object type that has an attribute of the given name.
// This method will panic if the receiver type is not compatible.
//
// The method will also panic if the given attribute name is not defined
// for the value's type. Use the attribute-related methods on Type to
// check for the validity of an attribute before trying to use it.
//
// This method may be called on a value whose type is DynamicPseudoType,
// in which case the result will also be DynamicVal.
func (val Value) GetAttr(name string) Value {
	if val.ty == DynamicPseudoType {
		return DynamicVal
	}

	if !val.ty.IsObjectType() {
		panic("value is not an object")
	}

	name = NormalizeString(name)
	if !val.ty.HasAttribute(name) {
		panic("value has no attribute of that name")
	}

	attrType := val.ty.AttributeType(name)

	if !val.IsKnown() {
		return UnknownVal(attrType)
	}

	return Value{
		ty: attrType,
		v:  val.v.(map[string]interface{})[name],
	}
}

// Index returns the value of an element of the receiver, which must have
// either a list, map or tuple type. This method will panic if the receiver
// type is not compatible.
//
// The key value must be the correct type for the receving collection: a
// number if the collection is a list or tuple, or a string if it is a map.
// In the case of a list or tuple, the given number must be convertable to int
// or this method will panic. The key may alternatively be of
// DynamicPseudoType, in which case the result itself is an unknown of the
// collection's element type.
//
// The result is of the receiver collection's element type, or in the case
// of a tuple the type of the specific element index requested.
//
// This method may be called on a value whose type is DynamicPseudoType,
// in which case the result will also be the DynamicValue.
func (val Value) Index(key Value) Value {
	if val.ty == DynamicPseudoType {
		return DynamicVal
	}

	switch {
	case val.Type().IsListType():
		elty := val.Type().ElementType()
		if key.Type() == DynamicPseudoType {
			return UnknownVal(elty)
		}

		if key.Type() != Number {
			panic("element key for list must be number")
		}
		if !key.IsKnown() {
			return UnknownVal(elty)
		}

		if !val.IsKnown() {
			return UnknownVal(elty)
		}

		index, accuracy := key.v.(*big.Float).Int64()
		if accuracy != big.Exact || index < 0 {
			panic("element key for list must be non-negative integer")
		}

		return Value{
			ty: elty,
			v:  val.v.([]interface{})[index],
		}
	case val.Type().IsMapType():
		elty := val.Type().ElementType()
		if key.Type() == DynamicPseudoType {
			return UnknownVal(elty)
		}

		if key.Type() != String {
			panic("element key for map must be string")
		}
		if !key.IsKnown() {
			return UnknownVal(elty)
		}

		if !val.IsKnown() {
			return UnknownVal(elty)
		}

		keyStr := key.v.(string)

		return Value{
			ty: elty,
			v:  val.v.(map[string]interface{})[keyStr],
		}
	case val.Type().IsTupleType():
		if key.Type() == DynamicPseudoType {
			return DynamicVal
		}

		if key.Type() != Number {
			panic("element key for tuple must be number")
		}
		if !key.IsKnown() {
			return DynamicVal
		}

		index, accuracy := key.v.(*big.Float).Int64()
		if accuracy != big.Exact || index < 0 {
			panic("element key for list must be non-negative integer")
		}

		eltys := val.Type().TupleElementTypes()

		if !val.IsKnown() {
			return UnknownVal(eltys[index])
		}

		return Value{
			ty: eltys[index],
			v:  val.v.([]interface{})[index],
		}
	default:
		panic("not a list, map, or tuple type")
	}
}

// HasIndex returns True if the receiver (which must be supported for Index)
// has an element with the given index key, or False if it does not.
//
// The result will be UnknownVal(Bool) if either the collection or the
// key value are unknown.
//
// This method will panic if the receiver is not indexable, but does not
// impose any panic-causing type constraints on the key.
func (val Value) HasIndex(key Value) Value {
	if val.ty == DynamicPseudoType {
		return UnknownVal(Bool)
	}

	switch {
	case val.Type().IsListType():
		if key.Type() == DynamicPseudoType {
			return UnknownVal(Bool)
		}

		if key.Type() != Number {
			return False
		}
		if !key.IsKnown() {
			return UnknownVal(Bool)
		}
		if !val.IsKnown() {
			return UnknownVal(Bool)
		}

		index, accuracy := key.v.(*big.Float).Int64()
		if accuracy != big.Exact || index < 0 {
			return False
		}

		return BoolVal(int(index) < len(val.v.([]interface{})) && index >= 0)
	case val.Type().IsMapType():
		if key.Type() == DynamicPseudoType {
			return UnknownVal(Bool)
		}

		if key.Type() != String {
			return False
		}
		if !key.IsKnown() {
			return UnknownVal(Bool)
		}
		if !val.IsKnown() {
			return UnknownVal(Bool)
		}

		keyStr := key.v.(string)
		_, exists := val.v.(map[string]interface{})[keyStr]

		return BoolVal(exists)
	case val.Type().IsTupleType():
		if key.Type() == DynamicPseudoType {
			return UnknownVal(Bool)
		}

		if key.Type() != Number {
			return False
		}
		if !key.IsKnown() {
			return UnknownVal(Bool)
		}

		index, accuracy := key.v.(*big.Float).Int64()
		if accuracy != big.Exact || index < 0 {
			return False
		}

		length := val.Type().Length()
		return BoolVal(int(index) < length && index >= 0)
	default:
		panic("not a list, map, or tuple type")
	}
}

// HasElement returns True if the receiver (which must be of a set type)
// has the given value as an element, or False if it does not.
//
// The result will be UnknownVal(Bool) if either the set or the
// given value are unknown.
//
// This method will panic if the receiver is not a set, or if it is a null set.
func (val Value) HasElement(elem Value) Value {
	ty := val.Type()

	if !ty.IsSetType() {
		panic("not a set type")
	}
	if !val.IsKnown() || !elem.IsKnown() {
		return UnknownVal(Bool)
	}
	if val.IsNull() {
		panic("can't call HasElement on a nil value")
	}
	if !ty.ElementType().Equals(elem.Type()) {
		return False
	}

	s := val.v.(set.Set)
	return BoolVal(s.Has(elem.v))
}

// Length returns the length of the receiver, which must be a collection type
// or tuple type, as a number value. If the receiver is not a compatible type
// then this method will panic.
//
// If the receiver is unknown then the result is also unknown.
//
// If the receiver is null then this function will panic.
//
// Note that Length is not supported for strings. To determine the length
// of a string, call AsString and take the length of the native Go string
// that is returned.
func (val Value) Length() Value {
	if val.Type().IsTupleType() {
		// For tuples, we can return the length even if the value is not known.
		return NumberIntVal(int64(val.Type().Length()))
	}

	if !val.IsKnown() {
		return UnknownVal(Number)
	}

	return NumberIntVal(int64(val.LengthInt()))
}

// LengthInt is like Length except it returns an int. It has the same behavior
// as Length except that it will panic if the receiver is unknown.
//
// This is an integration method provided for the convenience of code bridging
// into Go's type system.
func (val Value) LengthInt() int {
	if val.Type().IsTupleType() {
		// For tuples, we can return the length even if the value is not known.
		return val.Type().Length()
	}
	if val.Type().IsObjectType() {
		// For objects, the length is the number of attributes associated with the type.
		return len(val.Type().AttributeTypes())
	}
	if !val.IsKnown() {
		panic("value is not known")
	}
	if val.IsNull() {
		panic("value is null")
	}

	switch {

	case val.ty.IsListType():
		return len(val.v.([]interface{}))

	case val.ty.IsSetType():
		return val.v.(set.Set).Length()

	case val.ty.IsMapType():
		return len(val.v.(map[string]interface{}))

	default:
		panic("value is not a collection")
	}
}

// ElementIterator returns an ElementIterator for iterating the elements
// of the receiver, which must be a collection type, a tuple type, or an object
// type. If called on a method of any other type, this method will panic.
//
// The value must be Known and non-Null, or this method will panic.
//
// If the receiver is of a list type, the returned keys will be of type Number
// and the values will be of the list's element type.
//
// If the receiver is of a map type, the returned keys will be of type String
// and the value will be of the map's element type. Elements are passed in
// ascending lexicographical order by key.
//
// If the receiver is of a set type, each element is returned as both the
// key and the value, since set members are their own identity.
//
// If the receiver is of a tuple type, the returned keys will be of type Number
// and the value will be of the corresponding element's type.
//
// If the receiver is of an object type, the returned keys will be of type
// String and the value will be of the corresponding attributes's type.
//
// ElementIterator is an integration method, so it cannot handle Unknown
// values. This method will panic if the receiver is Unknown.
func (val Value) ElementIterator() ElementIterator {
	if !val.IsKnown() {
		panic("can't use ElementIterator on unknown value")
	}
	if val.IsNull() {
		panic("can't use ElementIterator on null value")
	}
	return elementIterator(val)
}

// CanIterateElements returns true if the receiver can support the
// ElementIterator method (and by extension, ForEachElement) without panic.
func (val Value) CanIterateElements() bool {
	return canElementIterator(val)
}

// ForEachElement executes a given callback function for each element of
// the receiver, which must be a collection type or tuple type, or this method
// will panic.
//
// ForEachElement uses ElementIterator internally, and so the values passed
// to the callback are as described for ElementIterator.
//
// Returns true if the iteration exited early due to the callback function
// returning true, or false if the loop ran to completion.
//
// ForEachElement is an integration method, so it cannot handle Unknown
// values. This method will panic if the receiver is Unknown.
func (val Value) ForEachElement(cb ElementCallback) bool {
	it := val.ElementIterator()
	for it.Next() {
		key, val := it.Element()
		stop := cb(key, val)
		if stop {
			return true
		}
	}
	return false
}

// Not returns the logical inverse of the receiver, which must be of type
// Bool or this method will panic.
func (val Value) Not() Value {
	if shortCircuit := mustTypeCheck(Bool, Bool, val); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Bool)
		return *shortCircuit
	}

	return BoolVal(!val.v.(bool))
}

// And returns the result of logical AND with the receiver and the other given
// value, which must both be of type Bool or this method will panic.
func (val Value) And(other Value) Value {
	if shortCircuit := mustTypeCheck(Bool, Bool, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Bool)
		return *shortCircuit
	}

	return BoolVal(val.v.(bool) && other.v.(bool))
}

// Or returns the result of logical OR with the receiver and the other given
// value, which must both be of type Bool or this method will panic.
func (val Value) Or(other Value) Value {
	if shortCircuit := mustTypeCheck(Bool, Bool, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Bool)
		return *shortCircuit
	}

	return BoolVal(val.v.(bool) || other.v.(bool))
}

// LessThan returns True if the receiver is less than the other given value,
// which must both be numbers or this method will panic.
func (val Value) LessThan(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Bool, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Bool)
		return *shortCircuit
	}

	return BoolVal(val.v.(*big.Float).Cmp(other.v.(*big.Float)) < 0)
}

// GreaterThan returns True if the receiver is greater than the other given
// value, which must both be numbers or this method will panic.
func (val Value) GreaterThan(other Value) Value {
	if shortCircuit := mustTypeCheck(Number, Bool, val, other); shortCircuit != nil {
		shortCircuit = forceShortCircuitType(shortCircuit, Bool)
		return *shortCircuit
	}

	return BoolVal(val.v.(*big.Float).Cmp(other.v.(*big.Float)) > 0)
}

// LessThanOrEqualTo is equivalent to LessThan and Equal combined with Or.
func (val Value) LessThanOrEqualTo(other Value) Value {
	return val.LessThan(other).Or(val.Equals(other))
}

// GreaterThanOrEqualTo is equivalent to GreaterThan and Equal combined with Or.
func (val Value) GreaterThanOrEqualTo(other Value) Value {
	return val.GreaterThan(other).Or(val.Equals(other))
}

// AsString returns the native string from a non-null, non-unknown cty.String
// value, or panics if called on any other value.
func (val Value) AsString() string {
	if val.ty != String {
		panic("not a string")
	}
	if val.IsNull() {
		panic("value is null")
	}
	if !val.IsKnown() {
		panic("value is unknown")
	}

	return val.v.(string)
}

// AsBigFloat returns a big.Float representation of a non-null, non-unknown
// cty.Number value, or panics if called on any other value.
//
// For more convenient conversions to other native numeric types, use the
// "gocty" package.
func (val Value) AsBigFloat() *big.Float {
	if val.ty != Number {
		panic("not a number")
	}
	if val.IsNull() {
		panic("value is null")
	}
	if !val.IsKnown() {
		panic("value is unknown")
	}

	// Copy the float so that callers can't mutate our internal state
	ret := *(val.v.(*big.Float))

	return &ret
}

// AsValueSlice returns a []cty.Value representation of a non-null, non-unknown
// value of any type that CanIterateElements, or panics if called on
// any other value.
//
// For more convenient conversions to slices of more specific types, use
// the "gocty" package.
func (val Value) AsValueSlice() []Value {
	l := val.LengthInt()
	if l == 0 {
		return nil
	}

	ret := make([]Value, 0, l)
	for it := val.ElementIterator(); it.Next(); {
		_, v := it.Element()
		ret = append(ret, v)
	}
	return ret
}

// AsValueMap returns a map[string]cty.Value representation of a non-null,
// non-unknown value of any type that CanIterateElements, or panics if called
// on any other value.
//
// For more convenient conversions to maps of more specific types, use
// the "gocty" package.
func (val Value) AsValueMap() map[string]Value {
	l := val.LengthInt()
	if l == 0 {
		return nil
	}

	ret := make(map[string]Value, l)
	for it := val.ElementIterator(); it.Next(); {
		k, v := it.Element()
		ret[k.AsString()] = v
	}
	return ret
}

// AsValueSet returns a ValueSet representation of a non-null,
// non-unknown value of any collection type, or panics if called
// on any other value.
//
// Unlike AsValueSlice and AsValueMap, this method requires specifically a
// collection type (list, set or map) and does not allow structural types
// (tuple or object), because the ValueSet type requires homogenous
// element types.
//
// The returned ValueSet can store only values of the receiver's element type.
func (val Value) AsValueSet() ValueSet {
	if !val.Type().IsCollectionType() {
		panic("not a collection type")
	}

	// We don't give the caller our own set.Set (assuming we're a cty.Set value)
	// because then the caller could mutate our internals, which is forbidden.
	// Instead, we will construct a new set and append our elements into it.
	ret := NewValueSet(val.Type().ElementType())
	for it := val.ElementIterator(); it.Next(); {
		_, v := it.Element()
		ret.Add(v)
	}
	return ret
}

// EncapsulatedValue returns the native value encapsulated in a non-null,
// non-unknown capsule-typed value, or panics if called on any other value.
//
// The result is the same pointer that was passed to CapsuleVal to create
// the value. Since cty considers values to be immutable, it is strongly
// recommended to treat the encapsulated value itself as immutable too.
func (val Value) EncapsulatedValue() interface{} {
	if !val.Type().IsCapsuleType() {
		panic("not a capsule-typed value")
	}

	return val.v
}
