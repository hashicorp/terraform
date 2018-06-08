package states

import (
	"github.com/zclconf/go-cty/cty"

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
	// SchemaVersion identifies which version of the resource type schema the
	// Attrs or AttrsFlat value conforms to. If this is less than the schema
	// version number given by the current provider version then the value
	// must be upgraded to the latest version before use. If it is greater
	// than the current version number then the provider must be upgraded
	// before any operations can be performed.
	SchemaVersion uint64

	// AttrsJSON is a JSON-encoded representation of the object attributes,
	// encoding the value (of the object type implied by the associated resource
	// type schema) that represents this remote object in Terraform Language
	// expressions, and is compared with configuration when producing a diff.
	//
	// This is retained in JSON format here because it may require preprocessing
	// before decoding if, for example, the stored attributes are for an older
	// schema version which the provider must upgrade before use. If the
	// version is current, it is valid to simply decode this using the
	// type implied by the current schema, without the need for the provider
	// to perform an upgrade first.
	//
	// When writing a ResourceInstanceObject into the state, AttrsJSON should
	// always be conformant to the current schema version and the current
	// schema version should be recorded in the SchemaVersion field.
	AttrsJSON []byte

	// AttrsFlat is a legacy form of attributes used in older state file
	// formats, and in the new state format for objects that haven't yet been
	// upgraded. This attribute is mutually exclusive with Attrs: for any
	// ResourceInstanceObject, only one of these attributes may be populated
	// and the other must be nil.
	//
	// An instance object with this field populated should be upgraded to use
	// Attrs at the earliest opportunity, since this legacy flatmap-based
	// format will be phased out over time. AttrsFlat should not be used when
	// writing new or updated objects to state; instead, callers must follow
	// the recommendations in the AttrsJSON documentation above.
	AttrsFlat map[string]string

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
