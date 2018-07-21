package states

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
)

// ResourceInstanceObject is the local representation of a specific remote
// object associated with a resource instance. In practice not all remote
// objects are actually remote in the sense of being accessed over the network,
// but this is the most common case.
//
// It is not valid to mutate a ResourceInstanceObject once it has been created.
// Instead, create a new object and replace the existing one.
type ResourceInstanceObject struct {
	// Value is the object-typed value representing the remote object within
	// Terraform.
	Value cty.Value

	// Internal is an opaque value set by the provider when this object was
	// last created or updated. Terraform Core does not use this value in
	// any way and it is not exposed anywhere in the user interface, so
	// a provider can use it for retaining any necessary private state.
	Private cty.Value

	// Status represents the "readiness" of the object as of the last time
	// it was updated.
	Status ObjectStatus

	// Dependencies is a set of other addresses in the same module which
	// this instance depended on when the given attributes were evaluated.
	// This is used to construct the dependency relationships for an object
	// whose configuration is no longer available, such as if it has been
	// removed from configuration altogether, or is now deposed.
	Dependencies []addrs.Referenceable
}

// ObjectStatus represents the status of a RemoteObject.
type ObjectStatus rune

//go:generate stringer -type ObjectStatus

const (
	// ObjectReady is an object status for an object that is ready to use.
	ObjectReady ObjectStatus = 'R'

	// ObjectTainted is an object status representing an object that is in
	// an unrecoverable bad state due to a partial failure during a create,
	// update, or delete operation. Since it cannot be moved into the
	// ObjectRead state, a tainted object must be replaced.
	ObjectTainted ObjectStatus = 'T'
)

// Encode marshals the value within the receiver to produce a
// ResourceInstanceObjectSrc ready to be written to a state file.
//
// The given type must be the implied type of the resource type schema, and
// the given value must conform to it. It is important to pass the schema
// type and not the object's own type so that dynamically-typed attributes
// will be stored correctly. The caller must also provide the version number
// of the schema that the given type was derived from, which will be recorded
// in the source object so it can be used to detect when schema migration is
// required on read.
//
// The returned object may share internal references with the receiver and
// so the caller must not mutate the receiver any further once once this
// method is called.
func (o *ResourceInstanceObject) Encode(val cty.Value, ty cty.Type, schemaVersion uint64) (*ResourceInstanceObjectSrc, error) {
	src, err := ctyjson.Marshal(val, ty)
	if err != nil {
		return nil, err
	}

	return &ResourceInstanceObjectSrc{
		SchemaVersion: schemaVersion,
		AttrsJSON:     src,
		Private:       o.Private,
		Status:        o.Status,
		Dependencies:  o.Dependencies,
	}, nil
}
