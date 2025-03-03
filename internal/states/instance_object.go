// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/marks"
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

	// Private is an opaque value set by the provider when this object was
	// last created or updated. Terraform Core does not use this value in
	// any way and it is not exposed anywhere in the user interface, so
	// a provider can use it for retaining any necessary private state.
	Private []byte

	// Status represents the "readiness" of the object as of the last time
	// it was updated.
	Status ObjectStatus

	// Dependencies is a set of absolute address to other resources this
	// instance dependeded on when it was applied. This is used to construct
	// the dependency relationships for an object whose configuration is no
	// longer available, such as if it has been removed from configuration
	// altogether, or is now deposed.
	Dependencies []addrs.ConfigResource

	// CreateBeforeDestroy reflects the status of the lifecycle
	// create_before_destroy option when this instance was last updated.
	// Because create_before_destroy also effects the overall ordering of the
	// destroy operations, we need to record the status to ensure a resource
	// removed from the config will still be destroyed in the same manner.
	CreateBeforeDestroy bool
}

// ObjectStatus represents the status of a RemoteObject.
type ObjectStatus rune

//go:generate go tool golang.org/x/tools/cmd/stringer -type ObjectStatus

const (
	// ObjectReady is an object status for an object that is ready to use.
	ObjectReady ObjectStatus = 'R'

	// ObjectTainted is an object status representing an object that is in
	// an unrecoverable bad state due to a partial failure during a create,
	// update, or delete operation. Since it cannot be moved into the
	// ObjectRead state, a tainted object must be replaced.
	ObjectTainted ObjectStatus = 'T'

	// ObjectPlanned is a special object status used only for the transient
	// placeholder objects we place into state during the refresh and plan
	// walks to stand in for objects that will be created during apply.
	//
	// Any object of this status must have a corresponding change recorded
	// in the current plan, whose value must then be used in preference to
	// the value stored in state when evaluating expressions. A planned
	// object stored in state will be incomplete if any of its attributes are
	// not yet known, and the plan must be consulted in order to "see" those
	// unknown values, because the state is not able to represent them.
	ObjectPlanned ObjectStatus = 'P'
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
func (o *ResourceInstanceObject) Encode(ty cty.Type, schemaVersion uint64) (*ResourceInstanceObjectSrc, error) {
	// If it contains marks, remove these marks before traversing the
	// structure with UnknownAsNull, and save the PathValueMarks
	// so we can save them in state.
	val, sensitivePaths, err := unmarkValueForStorage(o.Value)
	if err != nil {
		return nil, err
	}

	// Our state serialization can't represent unknown values, so we convert
	// them to nulls here. This is lossy, but nobody should be writing unknown
	// values here and expecting to get them out again later.
	//
	// We get unknown values here while we're building out a "planned state"
	// during the plan phase, but the value stored in the plan takes precedence
	// for expression evaluation. The apply step should never produce unknown
	// values, but if it does it's the responsibility of the caller to detect
	// and raise an error about that.
	val = cty.UnknownAsNull(val)

	src, err := ctyjson.Marshal(val, ty)
	if err != nil {
		return nil, err
	}

	// Dependencies are collected and merged in an unordered format (using map
	// keys as a set), then later changed to a slice (in random ordering) to be
	// stored in state as an array. To avoid pointless thrashing of state in
	// refresh-only runs, we can either override comparison of dependency lists
	// (more desirable, but tricky for Reasons) or just sort when encoding.
	// Encoding of instances can happen concurrently, so we must copy the
	// dependencies to avoid mutating what may be a shared array of values.
	dependencies := make([]addrs.ConfigResource, len(o.Dependencies))
	copy(dependencies, o.Dependencies)

	sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].String() < dependencies[j].String() })

	return &ResourceInstanceObjectSrc{
		SchemaVersion:       schemaVersion,
		AttrsJSON:           src,
		AttrSensitivePaths:  sensitivePaths,
		Private:             o.Private,
		Status:              o.Status,
		Dependencies:        dependencies,
		CreateBeforeDestroy: o.CreateBeforeDestroy,
		// The cached value must have all its marks since it bypasses decoding.
		decodeValueCache: o.Value,
	}, nil
}

// AsTainted returns a deep copy of the receiver with the status updated to
// ObjectTainted.
func (o *ResourceInstanceObject) AsTainted() *ResourceInstanceObject {
	if o == nil {
		// A nil object can't be tainted, but we'll allow this anyway to
		// avoid a crash, since we presumably intend to eventually record
		// the object has having been deleted anyway.
		return nil
	}
	ret := o.DeepCopy()
	ret.Status = ObjectTainted
	return ret
}

// unmarkValueForStorage takes a value that possibly contains marked values
// and returns an equal value without markings along with the separated mark
// metadata that should be stored alongside the value in another field.
//
// This function only accepts the marks that are valid to store, and so will
// return an error if other marks are present. Marks that this package doesn't
// know how to store must be dealt with somehow by a caller -- presumably by
// replacing each marked value with some sort of storage placeholder -- before
// writing a value into the state.
func unmarkValueForStorage(v cty.Value) (unmarkedV cty.Value, sensitivePaths []cty.Path, err error) {
	val, pvms := v.UnmarkDeepWithPaths()
	sensitivePaths, withOtherMarks := marks.PathsWithMark(pvms, marks.Sensitive)
	if len(withOtherMarks) != 0 {
		return cty.NilVal, nil, fmt.Errorf(
			"%s: cannot serialize value marked as %#v for inclusion in a state snapshot (this is a bug in Terraform)",
			format.CtyPath(withOtherMarks[0].Path), withOtherMarks[0].Marks,
		)
	}

	// sort the sensitive paths for consistency in comparison and serialization
	sort.Slice(sensitivePaths, func(i, j int) bool {
		// use our human-readable format of paths for comparison
		return format.CtyPath(sensitivePaths[i]) < format.CtyPath(sensitivePaths[j])
	})

	return val, sensitivePaths, nil
}
