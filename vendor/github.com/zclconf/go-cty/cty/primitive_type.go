package cty

import "math/big"

// primitiveType is the hidden implementation of the various primitive types
// that are exposed as variables in this package.
type primitiveType struct {
	typeImplSigil
	Kind primitiveTypeKind
}

type primitiveTypeKind byte

const (
	primitiveTypeBool   primitiveTypeKind = 'B'
	primitiveTypeNumber primitiveTypeKind = 'N'
	primitiveTypeString primitiveTypeKind = 'S'
)

func (t primitiveType) Equals(other Type) bool {
	if otherP, ok := other.typeImpl.(primitiveType); ok {
		return otherP.Kind == t.Kind
	}
	return false
}

func (t primitiveType) FriendlyName() string {
	switch t.Kind {
	case primitiveTypeBool:
		return "bool"
	case primitiveTypeNumber:
		return "number"
	case primitiveTypeString:
		return "string"
	default:
		// should never happen
		panic("invalid primitive type")
	}
}

func (t primitiveType) GoString() string {
	switch t.Kind {
	case primitiveTypeBool:
		return "cty.Bool"
	case primitiveTypeNumber:
		return "cty.Number"
	case primitiveTypeString:
		return "cty.String"
	default:
		// should never happen
		panic("invalid primitive type")
	}
}

// Number is the numeric type. Number values are arbitrary-precision
// decimal numbers, which can then be converted into Go's various numeric
// types only if they are in the appropriate range.
var Number Type

// String is the string type. String values are sequences of unicode codepoints
// encoded internally as UTF-8.
var String Type

// Bool is the boolean type. The two values of this type are True and False.
var Bool Type

// True is the truthy value of type Bool
var True Value

// False is the falsey value of type Bool
var False Value

// Zero is a number value representing exactly zero.
var Zero Value

// PositiveInfinity is a Number value representing positive infinity
var PositiveInfinity Value

// NegativeInfinity is a Number value representing negative infinity
var NegativeInfinity Value

func init() {
	Number = Type{
		primitiveType{Kind: primitiveTypeNumber},
	}
	String = Type{
		primitiveType{Kind: primitiveTypeString},
	}
	Bool = Type{
		primitiveType{Kind: primitiveTypeBool},
	}
	True = Value{
		ty: Bool,
		v:  true,
	}
	False = Value{
		ty: Bool,
		v:  false,
	}
	Zero = Value{
		ty: Number,
		v:  big.NewFloat(0),
	}
	PositiveInfinity = Value{
		ty: Number,
		v:  (&big.Float{}).SetInf(false),
	}
	NegativeInfinity = Value{
		ty: Number,
		v:  (&big.Float{}).SetInf(true),
	}
}

// IsPrimitiveType returns true if and only if the reciever is a primitive
// type, which means it's either number, string, or bool. Any two primitive
// types can be safely compared for equality using the standard == operator
// without panic, which is not a guarantee that holds for all types. Primitive
// types can therefore also be used in switch statements.
func (t Type) IsPrimitiveType() bool {
	_, ok := t.typeImpl.(primitiveType)
	return ok
}
