package states

import (
	"github.com/zclconf/go-cty/cty"
)

// ImportedObject represents an object being imported into Terraform with the
// help of a provider. An ImportedObject is a RemoteObject that has been read
// by the provider's import handler but hasn't yet been committed to state.
type ImportedObject struct {
	ResourceType string

	// Value is the value (of the object type implied by the associated resource
	// type schema) that represents this remote object in Terraform Language
	// expressions and is compared with configuration when producing a diff.
	Value cty.Value

	// Private corresponds to the field of the same name on
	// ResourceInstanceObject, where the provider can record private data that
	// will be available for future operations.
	Private cty.Value
}

// AsInstanceObject converts the receiving ImportedObject into a
// ResourceInstanceObject that has status ObjectReady.
//
// The returned object does not know its own resource type, so the caller must
// retain the ResourceType value from the source object if this information is
// needed.
//
// The returned object also has no dependency addresses, but the caller may
// freely modify the direct fields of the returned object without affecting
// the receiver.
func (io *ImportedObject) AsInstanceObject() *ResourceInstanceObject {
	return &ResourceInstanceObject{
		Status:  ObjectReady,
		Value:   io.Value,
		Private: io.Private,
	}
}
