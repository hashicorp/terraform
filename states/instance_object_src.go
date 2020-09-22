package states

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/hcl2shim"
)

// ResourceInstanceObjectSrc is a not-fully-decoded version of
// ResourceInstanceObject. Decoding of it can be completed by first handling
// any schema migration steps to get to the latest schema version and then
// calling method Decode with the implied type of the latest schema.
type ResourceInstanceObjectSrc struct {
	// SchemaVersion is the resource-type-specific schema version number that
	// was current when either AttrsJSON or AttrsFlat was encoded. Migration
	// steps are required if this is less than the current version number
	// reported by the corresponding provider.
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

	// These fields all correspond to the fields of the same name on
	// ResourceInstanceObject.
	Private             []byte
	Status              ObjectStatus
	Dependencies        []addrs.ConfigResource
	CreateBeforeDestroy bool
}

// Decode unmarshals the raw representation of the object attributes. Pass the
// implied type of the corresponding resource type schema for correct operation.
//
// Before calling Decode, the caller must check that the SchemaVersion field
// exactly equals the version number of the schema whose implied type is being
// passed, or else the result is undefined.
//
// The returned object may share internal references with the receiver and
// so the caller must not mutate the receiver any further once once this
// method is called.
func (os *ResourceInstanceObjectSrc) Decode(ty cty.Type) (*ResourceInstanceObject, error) {
	var val cty.Value
	var err error
	if os.AttrsFlat != nil {
		// Legacy mode. We'll do our best to unpick this from the flatmap.
		val, err = hcl2shim.HCL2ValueFromFlatmap(os.AttrsFlat, ty)
		if err != nil {
			return nil, err
		}
	} else {
		val, err = ctyjson.Unmarshal(os.AttrsJSON, ty)
		if err != nil {
			return nil, err
		}
	}

	return &ResourceInstanceObject{
		Value:               val,
		Status:              os.Status,
		Dependencies:        os.Dependencies,
		Private:             os.Private,
		CreateBeforeDestroy: os.CreateBeforeDestroy,
	}, nil
}

// CompleteUpgrade creates a new ResourceInstanceObjectSrc by copying the
// metadata from the receiver and writing in the given new schema version
// and attribute value that are presumed to have resulted from upgrading
// from an older schema version.
func (os *ResourceInstanceObjectSrc) CompleteUpgrade(newAttrs cty.Value, newType cty.Type, newSchemaVersion uint64) (*ResourceInstanceObjectSrc, error) {
	new := os.DeepCopy()
	new.AttrsFlat = nil // We always use JSON after an upgrade, even if the source used flatmap

	// This is the same principle as ResourceInstanceObject.Encode, but
	// avoiding a decode/re-encode cycle because we don't have type info
	// available for the "old" attributes.
	newAttrs = cty.UnknownAsNull(newAttrs)
	src, err := ctyjson.Marshal(newAttrs, newType)
	if err != nil {
		return nil, err
	}

	new.AttrsJSON = src
	new.SchemaVersion = newSchemaVersion
	return new, nil
}
