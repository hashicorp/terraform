package addrs

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// InstanceKey represents the key of an instance within an object that
// contains multiple instances due to using "count" or "for_each" arguments
// in configuration.
//
// IntKey and StringKey are the two implementations of this type. No other
// implementations are allowed. The single instance of an object that _isn't_
// using "count" or "for_each" is represented by NoKey, which is a nil
// InstanceKey.
type InstanceKey interface {
	instanceKeySigil()
	String() string

	// Value returns the cty.Value of the appropriate type for the InstanceKey
	// value.
	Value() cty.Value
}

// ParseInstanceKey returns the instance key corresponding to the given value,
// which must be known and non-null.
//
// If an unknown or null value is provided then this function will panic. This
// function is intended to deal with the values that would naturally be found
// in a hcl.TraverseIndex, which (when parsed from source, at least) can never
// contain unknown or null values.
func ParseInstanceKey(key cty.Value) (InstanceKey, error) {
	switch key.Type() {
	case cty.String:
		return StringKey(key.AsString()), nil
	case cty.Number:
		var idx int
		err := gocty.FromCtyValue(key, &idx)
		return IntKey(idx), err
	default:
		return NoKey, fmt.Errorf("either a string or an integer is required")
	}
}

// NoKey represents the absense of an InstanceKey, for the single instance
// of a configuration object that does not use "count" or "for_each" at all.
var NoKey InstanceKey

// IntKey is the InstanceKey representation representing integer indices, as
// used when the "count" argument is specified or if for_each is used with
// a sequence type.
type IntKey int

func (k IntKey) instanceKeySigil() {
}

func (k IntKey) String() string {
	return fmt.Sprintf("[%d]", int(k))
}

func (k IntKey) Value() cty.Value {
	return cty.NumberIntVal(int64(k))
}

// StringKey is the InstanceKey representation representing string indices, as
// used when the "for_each" argument is specified with a map or object type.
type StringKey string

func (k StringKey) instanceKeySigil() {
}

func (k StringKey) String() string {
	// FIXME: This isn't _quite_ right because Go's quoted string syntax is
	// slightly different than HCL's, but we'll accept it for now.
	return fmt.Sprintf("[%q]", string(k))
}

func (k StringKey) Value() cty.Value {
	return cty.StringVal(string(k))
}

// InstanceKeyLess returns true if the first given instance key i should sort
// before the second key j, and false otherwise.
func InstanceKeyLess(i, j InstanceKey) bool {
	iTy := instanceKeyType(i)
	jTy := instanceKeyType(j)

	switch {
	case i == j:
		return false
	case i == NoKey:
		return true
	case j == NoKey:
		return false
	case iTy != jTy:
		// The ordering here is arbitrary except that we want NoKeyType
		// to sort before the others, so we'll just use the enum values
		// of InstanceKeyType here (where NoKey is zero, sorting before
		// any other).
		return uint32(iTy) < uint32(jTy)
	case iTy == IntKeyType:
		return int(i.(IntKey)) < int(j.(IntKey))
	case iTy == StringKeyType:
		return string(i.(StringKey)) < string(j.(StringKey))
	default:
		// Shouldn't be possible to get down here in practice, since the
		// above is exhaustive.
		return false
	}
}

func instanceKeyType(k InstanceKey) InstanceKeyType {
	if _, ok := k.(StringKey); ok {
		return StringKeyType
	}
	if _, ok := k.(IntKey); ok {
		return IntKeyType
	}
	return NoKeyType
}

// InstanceKeyType represents the different types of instance key that are
// supported. Usually it is sufficient to simply type-assert an InstanceKey
// value to either IntKey or StringKey, but this type and its values can be
// used to represent the types themselves, rather than specific values
// of those types.
type InstanceKeyType rune

const (
	NoKeyType     InstanceKeyType = 0
	IntKeyType    InstanceKeyType = 'I'
	StringKeyType InstanceKeyType = 'S'
)
