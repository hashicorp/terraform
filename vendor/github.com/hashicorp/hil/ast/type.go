package ast

import (
	"fmt"
)

// Type represents a type of value within the interpolation language.
//
// In general types are opaque and should just be compared using the standard
// equality and inequality operators. However, types that have elements
// (TypeList and TypeMap instances) can be identified using a type assertion
// or type switch to implement generic behavior that applies to lists or maps
// regardless of their element types.
type Type interface {
	Printable() string
}

// primitiveType is the internal type used to represent primitives, but
// callers will just compare them to the various singleton variables
// TypeString, TypeInt, etc, rather than interacting with this type directly.
type primitiveType string

func (t primitiveType) Printable() string {
	return string(t)
}

func (t primitiveType) String() string {
	return fmt.Sprintf("type %s", t.Printable())
}

var TypeInvalid Type = primitiveType("invalid type")
var TypeAny Type = primitiveType("any type")
var TypeBool Type = primitiveType("bool")
var TypeString Type = primitiveType("string")
var TypeInt Type = primitiveType("int")
var TypeFloat Type = primitiveType("float")
var TypeUnknown Type = primitiveType("unknown")

func (t primitiveType) GoString() string {
	// types we know get prettier representations
	switch t {
	case TypeInvalid:
		return "ast.TypeInvalid"
	case TypeAny:
		return "ast.TypeAny"
	case TypeBool:
		return "ast.TypeBool"
	case TypeString:
		return "ast.TypeString"
	case TypeInt:
		return "ast.TypeInt"
	case TypeFloat:
		return "ast.TypeFloat"
	case TypeUnknown:
		return "ast.TypeUnknown"
	default:
		return fmt.Sprintf("%T{%#v}", t, string(t))
	}
}

// TypeList is a Type implementation that represents lists of values that
// themselves have a type.
//
// It is acceptable and expected to use a type assertion to determine if
// a given Type value is a TypeList and gain access to its ElementType.
type TypeList struct {
	ElementType Type
}

func (t TypeList) Printable() string {
	return fmt.Sprintf("list of %s", t.ElementType.Printable())
}

func (t TypeList) String() string {
	return fmt.Sprintf("type %s", t.Printable())
}

func (t TypeList) GoString() string {
	return fmt.Sprintf("%T{%#v}", t, t.ElementType)
}

// TypeMap is a Type implementation that represents maps from strings to
// values that themselves have a type.
//
// It is acceptable and expected to use a type assertion to determine if
// a given Type value is a TypeMap and gain access to its ElementType.
type TypeMap struct {
	ElementType Type
}

func (t TypeMap) Printable() string {
	return fmt.Sprintf("map of %s", t.ElementType.Printable())
}

func (t TypeMap) String() string {
	return fmt.Sprintf("type %s", t.Printable())
}

func (t TypeMap) GoString() string {
	return fmt.Sprintf("%T{%#v}", t, t.ElementType)
}

// TypeIsList returns true if the given Type is a TypeList. This is equivalent
// to a type assertion for TypeList but more convenient to use in expressions
// where a single boolean value is needed.
func TypeIsList(t Type) bool {
	_, ok := t.(TypeList)
	return ok
}

// TypeIsMap returns true if the given Type is a TypeMap. This is equivalent
// to a type assertion for TypeMap but more convenient to use in expressions
// where a single boolean value is needed.
func TypeIsMap(t Type) bool {
	_, ok := t.(TypeMap)
	return ok
}
