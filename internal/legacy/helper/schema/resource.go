// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package schema

import (
	"github.com/hashicorp/terraform/internal/legacy/terraform"
	"github.com/zclconf/go-cty/cty"
)

var ReservedDataSourceFields = []string{
	"connection",
	"count",
	"depends_on",
	"lifecycle",
	"provider",
	"provisioner",
}

var ReservedResourceFields = []string{
	"connection",
	"count",
	"depends_on",
	"id",
	"lifecycle",
	"provider",
	"provisioner",
}

// Resource represents a thing in Terraform that has a set of configurable
// attributes and a lifecycle (create, read, update, delete).
//
// The Resource schema is an abstraction that allows provider writers to
// worry only about CRUD operations while off-loading validation, diff
// generation, etc. to this higher level library.
//
// In spite of the name, this struct is not used only for terraform resources,
// but also for data sources. In the case of data sources, the Create,
// Update and Delete functions must not be provided.
type Resource struct {
	// Schema is the schema for the configuration of this resource.
	//
	// The keys of this map are the configuration keys, and the values
	// describe the schema of the configuration value.
	//
	// The schema is used to represent both configurable data as well
	// as data that might be computed in the process of creating this
	// resource.
	Schema map[string]*Schema

	// SchemaVersion is the version number for this resource's Schema
	// definition. The current SchemaVersion stored in the state for each
	// resource. Provider authors can increment this version number
	// when Schema semantics change. If the State's SchemaVersion is less than
	// the current SchemaVersion, the InstanceState is yielded to the
	// MigrateState callback, where the provider can make whatever changes it
	// needs to update the state to be compatible to the latest version of the
	// Schema.
	//
	// When unset, SchemaVersion defaults to 0, so provider authors can start
	// their Versioning at any integer >= 1
	SchemaVersion int

	// MigrateState is deprecated and any new changes to a resource's schema
	// should be handled by StateUpgraders. Existing MigrateState implementations
	// should remain for compatibility with existing state. MigrateState will
	// still be called if the stored SchemaVersion is less than the
	// first version of the StateUpgraders.
	//
	// MigrateState is responsible for updating an InstanceState with an old
	// version to the format expected by the current version of the Schema.
	//
	// It is called during Refresh if the State's stored SchemaVersion is less
	// than the current SchemaVersion of the Resource.
	//
	// The function is yielded the state's stored SchemaVersion and a pointer to
	// the InstanceState that needs updating, as well as the configured
	// provider's configured meta interface{}, in case the migration process
	// needs to make any remote API calls.
	MigrateState StateMigrateFunc

	// StateUpgraders contains the functions responsible for upgrading an
	// existing state with an old schema version to a newer schema. It is
	// called specifically by Terraform when the stored schema version is less
	// than the current SchemaVersion of the Resource.
	//
	// StateUpgraders map specific schema versions to a StateUpgrader
	// function. The registered versions are expected to be ordered,
	// consecutive values. The initial value may be greater than 0 to account
	// for legacy schemas that weren't recorded and can be handled by
	// MigrateState.
	StateUpgraders []StateUpgrader

	// The functions below are the CRUD operations for this resource.
	//
	// The only optional operation is Update. If Update is not implemented,
	// then updates will not be supported for this resource.
	//
	// The ResourceData parameter in the functions below are used to
	// query configuration and changes for the resource as well as to set
	// the ID, computed data, etc.
	//
	// The interface{} parameter is the result of the ConfigureFunc in
	// the provider for this resource. If the provider does not define
	// a ConfigureFunc, this will be nil. This parameter should be used
	// to store API clients, configuration structures, etc.
	//
	// If any errors occur during each of the operation, an error should be
	// returned. If a resource was partially updated, be careful to enable
	// partial state mode for ResourceData and use it accordingly.
	//
	// Exists is a function that is called to check if a resource still
	// exists. If this returns false, then this will affect the diff
	// accordingly. If this function isn't set, it will not be called. You
	// can also signal existence in the Read method by calling d.SetId("")
	// if the Resource is no longer present and should be removed from state.
	// The *ResourceData passed to Exists should _not_ be modified.
	Create CreateFunc
	Read   ReadFunc
	Update UpdateFunc
	Delete DeleteFunc
	Exists ExistsFunc

	// CustomizeDiff is a custom function for working with the diff that
	// Terraform has created for this resource - it can be used to customize the
	// diff that has been created, diff values not controlled by configuration,
	// or even veto the diff altogether and abort the plan. It is passed a
	// *ResourceDiff, a structure similar to ResourceData but lacking most write
	// functions like Set, while introducing new functions that work with the
	// diff such as SetNew, SetNewComputed, and ForceNew.
	//
	// The phases Terraform runs this in, and the state available via functions
	// like Get and GetChange, are as follows:
	//
	//  * New resource: One run with no state
	//  * Existing resource: One run with state
	//   * Existing resource, forced new: One run with state (before ForceNew),
	//     then one run without state (as if new resource)
	//  * Tainted resource: No runs (custom diff logic is skipped)
	//  * Destroy: No runs (standard diff logic is skipped on destroy diffs)
	//
	// This function needs to be resilient to support all scenarios.
	//
	// If this function needs to access external API resources, remember to flag
	// the RequiresRefresh attribute mentioned below to ensure that
	// -refresh=false is blocked when running plan or apply, as this means that
	// this resource requires refresh-like behaviour to work effectively.
	//
	// For the most part, only computed fields can be customized by this
	// function.
	//
	// This function is only allowed on regular resources (not data sources).
	CustomizeDiff CustomizeDiffFunc

	// Importer is the ResourceImporter implementation for this resource.
	// If this is nil, then this resource does not support importing. If
	// this is non-nil, then it supports importing and ResourceImporter
	// must be validated. The validity of ResourceImporter is verified
	// by InternalValidate on Resource.
	Importer *ResourceImporter

	// If non-empty, this string is emitted as a warning during Validate.
	DeprecationMessage string

	// Timeouts allow users to specify specific time durations in which an
	// operation should time out, to allow them to extend an action to suit their
	// usage. For example, a user may specify a large Creation timeout for their
	// AWS RDS Instance due to it's size, or restoring from a snapshot.
	// Resource implementors must enable Timeout support by adding the allowed
	// actions (Create, Read, Update, Delete, Default) to the Resource struct, and
	// accessing them in the matching methods.
	Timeouts *ResourceTimeout
}

// See Resource documentation.
type CreateFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type ReadFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type UpdateFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type DeleteFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type ExistsFunc func(*ResourceData, interface{}) (bool, error)

// See Resource documentation.
type StateMigrateFunc func(
	int, *terraform.InstanceState, interface{}) (*terraform.InstanceState, error)

type StateUpgrader struct {
	// Version is the version schema that this Upgrader will handle, converting
	// it to Version+1.
	Version int

	// Type describes the schema that this function can upgrade. Type is
	// required to decode the schema if the state was stored in a legacy
	// flatmap format.
	Type cty.Type

	// Upgrade takes the JSON encoded state and the provider meta value, and
	// upgrades the state one single schema version. The provided state is
	// deocded into the default json types using a map[string]interface{}. It
	// is up to the StateUpgradeFunc to ensure that the returned value can be
	// encoded using the new schema.
	Upgrade StateUpgradeFunc
}

// See StateUpgrader
type StateUpgradeFunc func(rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error)

// See Resource documentation.
type CustomizeDiffFunc func(*ResourceDiff, interface{}) error
