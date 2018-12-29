package cty

import (
	"fmt"
)

type typeObject struct {
	typeImplSigil
	AttrTypes map[string]Type
}

// Object creates an object type with the given attribute types.
//
// After a map is passed to this function the caller must no longer access it,
// since ownership is transferred to this library.
func Object(attrTypes map[string]Type) Type {
	attrTypesNorm := make(map[string]Type, len(attrTypes))
	for k, v := range attrTypes {
		attrTypesNorm[NormalizeString(k)] = v
	}

	return Type{
		typeObject{
			AttrTypes: attrTypesNorm,
		},
	}
}

func (t typeObject) Equals(other Type) bool {
	if ot, ok := other.typeImpl.(typeObject); ok {
		if len(t.AttrTypes) != len(ot.AttrTypes) {
			// Fast path: if we don't have the same number of attributes
			// then we can't possibly be equal. This also avoids the need
			// to test attributes in both directions below, since we know
			// there can't be extras in "other".
			return false
		}

		for attr, ty := range t.AttrTypes {
			oty, ok := ot.AttrTypes[attr]
			if !ok {
				return false
			}
			if !oty.Equals(ty) {
				return false
			}
		}

		return true
	}
	return false
}

func (t typeObject) FriendlyName(mode friendlyTypeNameMode) string {
	// There isn't really a friendly way to write an object type due to its
	// complexity, so we'll just do something English-ish. Callers will
	// probably want to make some extra effort to avoid ever printing out
	// an object type FriendlyName in its entirety. For example, could
	// produce an error message by diffing two object types and saying
	// something like "Expected attribute foo to be string, but got number".
	// TODO: Finish this
	return "object"
}

func (t typeObject) GoString() string {
	if len(t.AttrTypes) == 0 {
		return "cty.EmptyObject"
	}
	return fmt.Sprintf("cty.Object(%#v)", t.AttrTypes)
}

// EmptyObject is a shorthand for Object(map[string]Type{}), to more
// easily talk about the empty object type.
var EmptyObject Type

// EmptyObjectVal is the only possible non-null, non-unknown value of type
// EmptyObject.
var EmptyObjectVal Value

func init() {
	EmptyObject = Object(map[string]Type{})
	EmptyObjectVal = Value{
		ty: EmptyObject,
		v:  map[string]interface{}{},
	}
}

// IsObjectType returns true if the given type is an object type, regardless
// of its element type.
func (t Type) IsObjectType() bool {
	_, ok := t.typeImpl.(typeObject)
	return ok
}

// HasAttribute returns true if the receiver has an attribute with the given
// name, regardless of its type. Will panic if the reciever isn't an object
// type; use IsObjectType to determine whether this operation will succeed.
func (t Type) HasAttribute(name string) bool {
	name = NormalizeString(name)
	if ot, ok := t.typeImpl.(typeObject); ok {
		_, hasAttr := ot.AttrTypes[name]
		return hasAttr
	}
	panic("HasAttribute on non-object Type")
}

// AttributeType returns the type of the attribute with the given name. Will
// panic if the receiver is not an object type (use IsObjectType to confirm)
// or if the object type has no such attribute (use HasAttribute to confirm).
func (t Type) AttributeType(name string) Type {
	name = NormalizeString(name)
	if ot, ok := t.typeImpl.(typeObject); ok {
		aty, hasAttr := ot.AttrTypes[name]
		if !hasAttr {
			panic("no such attribute")
		}
		return aty
	}
	panic("AttributeType on non-object Type")
}

// AttributeTypes returns a map from attribute names to their associated
// types. Will panic if the receiver is not an object type (use IsObjectType
// to confirm).
//
// The returned map is part of the internal state of the type, and is provided
// for read access only. It is forbidden for any caller to modify the returned
// map. For many purposes the attribute-related methods of Value are more
// appropriate and more convenient to use.
func (t Type) AttributeTypes() map[string]Type {
	if ot, ok := t.typeImpl.(typeObject); ok {
		return ot.AttrTypes
	}
	panic("AttributeTypes on non-object Type")
}
