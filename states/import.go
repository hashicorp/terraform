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
