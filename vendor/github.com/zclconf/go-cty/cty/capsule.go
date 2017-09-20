package cty

import (
	"fmt"
	"reflect"
)

type capsuleType struct {
	typeImplSigil
	Name   string
	GoType reflect.Type
}

func (t *capsuleType) Equals(other Type) bool {
	if otherP, ok := other.typeImpl.(*capsuleType); ok {
		// capsule types compare by pointer identity
		return otherP == t
	}
	return false
}

func (t *capsuleType) FriendlyName() string {
	return t.Name
}

func (t *capsuleType) GoString() string {
	// To get a useful representation of our native type requires some
	// shenanigans.
	victimVal := reflect.Zero(t.GoType)
	return fmt.Sprintf("cty.Capsule(%q, reflect.TypeOf(%#v))", t.Name, victimVal.Interface())
}

// Capsule creates a new Capsule type.
//
// A Capsule type is a special type that can be used to transport arbitrary
// Go native values of a given type through the cty type system. A language
// that uses cty as its type system might, for example, provide functions
// that return capsule-typed values and then other functions that operate
// on those values.
//
// From cty's perspective, Capsule types have a few interesting characteristics,
// described in the following paragraphs.
//
// Each capsule type has an associated Go native type that it is able to
// transport. Capsule types compare by identity, so each call to the
// Capsule function creates an entirely-distinct cty Type, even if two calls
// use the same native type.
//
// Each capsule-typed value contains a pointer to a value of the given native
// type. A capsule-typed value supports no operations except equality, and
// equality is implemented by pointer identity of the encapsulated pointer.
//
// The given name is used as the new type's "friendly name". This can be any
// string in principle, but will usually be a short, all-lowercase name aimed
// at users of the embedding language (i.e. not mention Go-specific details)
// and will ideally not create ambiguity with any predefined cty type.
//
// Capsule types are never introduced by any standard cty operation, so a
// calling application opts in to including them within its own type system
// by creating them and introducing them via its own functions. At that point,
// the application is responsible for dealing with any capsule-typed values
// that might be returned.
func Capsule(name string, nativeType reflect.Type) Type {
	return Type{
		&capsuleType{
			Name:   name,
			GoType: nativeType,
		},
	}
}

// IsCapsuleType returns true if this type is a capsule type, as created
// by cty.Capsule .
func (t Type) IsCapsuleType() bool {
	_, ok := t.typeImpl.(*capsuleType)
	return ok
}

// EncapsulatedType returns the encapsulated native type of a capsule type,
// or panics if the receiver is not a Capsule type.
//
// Is IsCapsuleType to determine if this method is safe to call.
func (t Type) EncapsulatedType() reflect.Type {
	impl, ok := t.typeImpl.(*capsuleType)
	if !ok {
		panic("not a capsule type")
	}
	return impl.GoType
}
