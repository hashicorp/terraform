// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plans

import (
	"github.com/zclconf/go-cty/cty"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
)

// DynamicValue is the representation in the plan of a value whose type cannot
// be determined at compile time, such as because it comes from a schema
// defined in a plugin.
//
// This type is used as an indirection so that the overall plan structure can
// be decoded without schema available, and then the dynamic values accessed
// at a later time once the appropriate schema has been determined.
//
// Internally, DynamicValue is a serialized version of a cty.Value created
// against a particular type constraint. Callers should not access directly
// the serialized form, whose format may change in future. Values of this
// type must always be created by calling NewDynamicValue.
//
// The zero value of DynamicValue is nil, and represents the absense of a
// value within the Go type system. This is distinct from a cty.NullVal
// result, which represents the absense of a value within the cty type system.
type DynamicValue []byte

// NewDynamicValue creates a DynamicValue by serializing the given value
// against the given type constraint. The value must conform to the type
// constraint, or the result is undefined.
//
// If the value to be encoded has no predefined schema (for example, for
// module output values and input variables), set the type constraint to
// cty.DynamicPseudoType in order to save type information as part of the
// value, and then also pass cty.DynamicPseudoType to method Decode to recover
// the original value.
//
// cty.NilVal can be used to represent the absense of a value, but callers
// must be careful to distinguish values that are absent at the Go layer
// (cty.NilVal) vs. values that are absent at the cty layer (cty.NullVal
// results).
func NewDynamicValue(val cty.Value, ty cty.Type) (DynamicValue, error) {
	// If we're given cty.NilVal (the zero value of cty.Value, which is
	// distinct from a typed null value created by cty.NullVal) then we'll
	// assume the caller is trying to represent the _absense_ of a value,
	// and so we'll return a nil DynamicValue.
	if val == cty.NilVal {
		return DynamicValue(nil), nil
	}

	// Currently our internal encoding is msgpack, via ctymsgpack.
	buf, err := ctymsgpack.Marshal(val, ty)
	if err != nil {
		return nil, err
	}

	return DynamicValue(buf), nil
}

// Decode retrieves the effective value from the receiever by interpreting the
// serialized form against the given type constraint. For correct results,
// the type constraint must match (or be consistent with) the one that was
// used to create the receiver.
//
// A nil DynamicValue decodes to cty.NilVal, which is not a valid value and
// instead represents the absense of a value.
func (v DynamicValue) Decode(ty cty.Type) (cty.Value, error) {
	if v == nil {
		return cty.NilVal, nil
	}

	return ctymsgpack.Unmarshal([]byte(v), ty)
}

// ImpliedType returns the type implied by the serialized structure of the
// receiving value.
//
// This will not necessarily be exactly the type that was given when the
// value was encoded, and in particular must not be used for values that
// were encoded with their static type given as cty.DynamicPseudoType.
// It is however safe to use this method for values that were encoded using
// their runtime type as the conforming type, with the result being
// semantically equivalent but with all lists and sets represented as tuples,
// and maps as objects, due to ambiguities of the serialization.
func (v DynamicValue) ImpliedType() (cty.Type, error) {
	return ctymsgpack.ImpliedType([]byte(v))
}

// Copy produces a copy of the receiver with a distinct backing array.
func (v DynamicValue) Copy() DynamicValue {
	if v == nil {
		return nil
	}

	ret := make(DynamicValue, len(v))
	copy(ret, v)
	return ret
}
