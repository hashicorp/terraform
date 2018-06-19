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
