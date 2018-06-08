package cty

// Type represents value types within the type system.
//
// This is a closed interface type, meaning that only the concrete
// implementations provided within this package are considered valid.
type Type struct {
	typeImpl
}

type typeImpl interface {
	// isTypeImpl is a do-nothing method that exists only to express
	// that a type is an implementation of typeImpl.
	isTypeImpl() typeImplSigil

	// Equals returns true if the other given Type exactly equals the
	// receiver Type.
	Equals(other Type) bool

	// FriendlyName returns a human-friendly *English* name for the given
	// type.
	FriendlyName() string

	// GoString implements the GoStringer interface from package fmt.
	GoString() string
}

// Base implementation of Type to embed into concrete implementations
// to signal that they are implementations of Type.
type typeImplSigil struct{}

func (t typeImplSigil) isTypeImpl() typeImplSigil {
	return typeImplSigil{}
}

// Equals returns true if the other given Type exactly equals the receiver
// type.
func (t Type) Equals(other Type) bool {
	return t.typeImpl.Equals(other)
}

// FriendlyName returns a human-friendly *English* name for the given type.
func (t Type) FriendlyName() string {
	return t.typeImpl.FriendlyName()
}

// GoString returns a string approximating how the receiver type would be
// expressed in Go source code.
func (t Type) GoString() string {
	if t.typeImpl == nil {
		return "cty.NilType"
	}

	return t.typeImpl.GoString()
}

// NilType is an invalid type used when a function is returning an error
// and has no useful type to return. It should not be used and any methods
// called on it will panic.
var NilType = Type{}

// HasDynamicTypes returns true either if the receiver is itself
// DynamicPseudoType or if it is a compound type whose descendent elements
// are DynamicPseudoType.
func (t Type) HasDynamicTypes() bool {
	switch {
	case t == DynamicPseudoType:
		return true
	case t.IsPrimitiveType():
		return false
	case t.IsCollectionType():
		return false
	case t.IsObjectType():
		attrTypes := t.AttributeTypes()
		for _, at := range attrTypes {
			if at.HasDynamicTypes() {
				return true
			}
		}
		return false
	case t.IsTupleType():
		elemTypes := t.TupleElementTypes()
		for _, et := range elemTypes {
			if et.HasDynamicTypes() {
				return true
			}
		}
		return false
	case t.IsCapsuleType():
		return false
	default:
		// Should never happen, since above should be exhaustive
		panic("HasDynamicTypes does not support the given type")
	}
}
