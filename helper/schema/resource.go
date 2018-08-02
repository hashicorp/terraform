package schema

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

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
	// accordingly. If this function isn't set, it will not be called. It
	// is highly recommended to set it. The *ResourceData passed to Exists
	// should _not_ be modified.
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

// Apply creates, updates, and/or deletes a resource.
func (r *Resource) Apply(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	data, err := schemaMap(r.Schema).Data(s, d)
	if err != nil {
		return s, err
	}

	// Instance Diff shoould have the timeout info, need to copy it over to the
	// ResourceData meta
	rt := ResourceTimeout{}
	if _, ok := d.Meta[TimeoutKey]; ok {
		if err := rt.DiffDecode(d); err != nil {
			log.Printf("[ERR] Error decoding ResourceTimeout: %s", err)
		}
	} else if s != nil {
		if _, ok := s.Meta[TimeoutKey]; ok {
			if err := rt.StateDecode(s); err != nil {
				log.Printf("[ERR] Error decoding ResourceTimeout: %s", err)
			}
		}
	} else {
		log.Printf("[DEBUG] No meta timeoutkey found in Apply()")
	}
	data.timeouts = &rt

	if s == nil {
		// The Terraform API dictates that this should never happen, but
		// it doesn't hurt to be safe in this case.
		s = new(terraform.InstanceState)
	}

	if d.Destroy || d.RequiresNew() {
		if s.ID != "" {
			// Destroy the resource since it is created
			if err := r.Delete(data, meta); err != nil {
				return r.recordCurrentSchemaVersion(data.State()), err
			}

			// Make sure the ID is gone.
			data.SetId("")
		}

		// If we're only destroying, and not creating, then return
		// now since we're done!
		if !d.RequiresNew() {
			return nil, nil
		}

		// Reset the data to be stateless since we just destroyed
		data, err = schemaMap(r.Schema).Data(nil, d)
		// data was reset, need to re-apply the parsed timeouts
		data.timeouts = &rt
		if err != nil {
			return nil, err
		}
	}

	err = nil
	if data.Id() == "" {
		// We're creating, it is a new resource.
		data.MarkNewResource()
		err = r.Create(data, meta)
	} else {
		if r.Update == nil {
			return s, fmt.Errorf("doesn't support update")
		}

		err = r.Update(data, meta)
	}

	return r.recordCurrentSchemaVersion(data.State()), err
}

// Diff returns a diff of this resource.
func (r *Resource) Diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {

	t := &ResourceTimeout{}
	err := t.ConfigDecode(r, c)

	if err != nil {
		return nil, fmt.Errorf("[ERR] Error decoding timeout: %s", err)
	}

	instanceDiff, err := schemaMap(r.Schema).Diff(s, c, r.CustomizeDiff, meta)
	if err != nil {
		return instanceDiff, err
	}

	if instanceDiff != nil {
		if err := t.DiffEncode(instanceDiff); err != nil {
			log.Printf("[ERR] Error encoding timeout to instance diff: %s", err)
		}
	} else {
		log.Printf("[DEBUG] Instance Diff is nil in Diff()")
	}

	return instanceDiff, err
}

// Validate validates the resource configuration against the schema.
func (r *Resource) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	warns, errs := schemaMap(r.Schema).Validate(c)

	if r.DeprecationMessage != "" {
		warns = append(warns, r.DeprecationMessage)
	}

	return warns, errs
}

// ReadDataApply loads the data for a data source, given a diff that
// describes the configuration arguments and desired computed attributes.
func (r *Resource) ReadDataApply(
	d *terraform.InstanceDiff,
	meta interface{},
) (*terraform.InstanceState, error) {
	// Data sources are always built completely from scratch
	// on each read, so the source state is always nil.
	data, err := schemaMap(r.Schema).Data(nil, d)
	if err != nil {
		return nil, err
	}

	err = r.Read(data, meta)
	state := data.State()
	if state != nil && state.ID == "" {
		// Data sources can set an ID if they want, but they aren't
		// required to; we'll provide a placeholder if they don't,
		// to preserve the invariant that all resources have non-empty
		// ids.
		state.ID = "-"
	}

	return r.recordCurrentSchemaVersion(state), err
}

// Refresh refreshes the state of the resource.
func (r *Resource) Refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	// If the ID is already somehow blank, it doesn't exist
	if s.ID == "" {
		return nil, nil
	}

	rt := ResourceTimeout{}
	if _, ok := s.Meta[TimeoutKey]; ok {
		if err := rt.StateDecode(s); err != nil {
			log.Printf("[ERR] Error decoding ResourceTimeout: %s", err)
		}
	}

	if r.Exists != nil {
		// Make a copy of data so that if it is modified it doesn't
		// affect our Read later.
		data, err := schemaMap(r.Schema).Data(s, nil)
		data.timeouts = &rt

		if err != nil {
			return s, err
		}

		exists, err := r.Exists(data, meta)
		if err != nil {
			return s, err
		}
		if !exists {
			return nil, nil
		}
	}

	needsMigration, stateSchemaVersion := r.checkSchemaVersion(s)
	if needsMigration && r.MigrateState != nil {
		s, err := r.MigrateState(stateSchemaVersion, s, meta)
		if err != nil {
			return s, err
		}
	}

	data, err := schemaMap(r.Schema).Data(s, nil)
	data.timeouts = &rt
	if err != nil {
		return s, err
	}

	err = r.Read(data, meta)
	state := data.State()
	if state != nil && state.ID == "" {
		state = nil
	}

	return r.recordCurrentSchemaVersion(state), err
}

// InternalValidate should be called to validate the structure
// of the resource.
//
// This should be called in a unit test for any resource to verify
// before release that a resource is properly configured for use with
// this library.
//
// Provider.InternalValidate() will automatically call this for all of
// the resources it manages, so you don't need to call this manually if it
// is part of a Provider.
func (r *Resource) InternalValidate(topSchemaMap schemaMap, writable bool) error {
	if r == nil {
		return errors.New("resource is nil")
	}

	if !writable {
		if r.Create != nil || r.Update != nil || r.Delete != nil {
			return fmt.Errorf("must not implement Create, Update or Delete")
		}

		// CustomizeDiff cannot be defined for read-only resources
		if r.CustomizeDiff != nil {
			return fmt.Errorf("cannot implement CustomizeDiff")
		}
	}

	tsm := topSchemaMap

	if r.isTopLevel() && writable {
		// All non-Computed attributes must be ForceNew if Update is not defined
		if r.Update == nil {
			nonForceNewAttrs := make([]string, 0)
			for k, v := range r.Schema {
				if !v.ForceNew && !v.Computed {
					nonForceNewAttrs = append(nonForceNewAttrs, k)
				}
			}
			if len(nonForceNewAttrs) > 0 {
				return fmt.Errorf(
					"No Update defined, must set ForceNew on: %#v", nonForceNewAttrs)
			}
		} else {
			nonUpdateableAttrs := make([]string, 0)
			for k, v := range r.Schema {
				if v.ForceNew || v.Computed && !v.Optional {
					nonUpdateableAttrs = append(nonUpdateableAttrs, k)
				}
			}
			updateableAttrs := len(r.Schema) - len(nonUpdateableAttrs)
			if updateableAttrs == 0 {
				return fmt.Errorf(
					"All fields are ForceNew or Computed w/out Optional, Update is superfluous")
			}
		}

		tsm = schemaMap(r.Schema)

		// Destroy, and Read are required
		if r.Read == nil {
			return fmt.Errorf("Read must be implemented")
		}
		if r.Delete == nil {
			return fmt.Errorf("Delete must be implemented")
		}

		// If we have an importer, we need to verify the importer.
		if r.Importer != nil {
			if err := r.Importer.InternalValidate(); err != nil {
				return err
			}
		}

		for k, f := range tsm {
			if isReservedResourceFieldName(k, f) {
				return fmt.Errorf("%s is a reserved field name", k)
			}
		}
	}

	lastVersion := -1
	for _, u := range r.StateUpgraders {
		if lastVersion >= 0 && u.Version-lastVersion > 1 {
			return fmt.Errorf("missing schema version between %d and %d", lastVersion, u.Version)
		}

		if u.Version >= r.SchemaVersion {
			return fmt.Errorf("StateUpgrader version %d is >= current version %d", u.Version, r.SchemaVersion)
		}

		if !u.Type.IsObjectType() {
			return fmt.Errorf("StateUpgrader %d type is not cty.Object", u.Version)
		}

		if u.Upgrade == nil {
			return fmt.Errorf("StateUpgrader %d missing StateUpgradeFunc", u.Version)
		}

		lastVersion = u.Version
	}

	if lastVersion >= 0 && lastVersion != r.SchemaVersion-1 {
		return fmt.Errorf("missing StateUpgrader between %d and %d", lastVersion, r.SchemaVersion)
	}

	// Data source
	if r.isTopLevel() && !writable {
		tsm = schemaMap(r.Schema)
		for k, _ := range tsm {
			if isReservedDataSourceFieldName(k) {
				return fmt.Errorf("%s is a reserved field name", k)
			}
		}
	}

	return schemaMap(r.Schema).InternalValidate(tsm)
}

func isReservedDataSourceFieldName(name string) bool {
	for _, reservedName := range config.ReservedDataSourceFields {
		if name == reservedName {
			return true
		}
	}
	return false
}

func isReservedResourceFieldName(name string, s *Schema) bool {
	// Allow phasing out "id"
	// See https://github.com/terraform-providers/terraform-provider-aws/pull/1626#issuecomment-328881415
	if name == "id" && (s.Deprecated != "" || s.Removed != "") {
		return false
	}

	for _, reservedName := range config.ReservedResourceFields {
		if name == reservedName {
			return true
		}
	}
	return false
}

// Data returns a ResourceData struct for this Resource. Each return value
// is a separate copy and can be safely modified differently.
//
// The data returned from this function has no actual affect on the Resource
// itself (including the state given to this function).
//
// This function is useful for unit tests and ResourceImporter functions.
func (r *Resource) Data(s *terraform.InstanceState) *ResourceData {
	result, err := schemaMap(r.Schema).Data(s, nil)
	if err != nil {
		// At the time of writing, this isn't possible (Data never returns
		// non-nil errors). We panic to find this in the future if we have to.
		// I don't see a reason for Data to ever return an error.
		panic(err)
	}

	// load the Resource timeouts
	result.timeouts = r.Timeouts
	if result.timeouts == nil {
		result.timeouts = &ResourceTimeout{}
	}

	// Set the schema version to latest by default
	result.meta = map[string]interface{}{
		"schema_version": strconv.Itoa(r.SchemaVersion),
	}

	return result
}

// TestResourceData Yields a ResourceData filled with this resource's schema for use in unit testing
//
// TODO: May be able to be removed with the above ResourceData function.
func (r *Resource) TestResourceData() *ResourceData {
	return &ResourceData{
		schema: r.Schema,
	}
}

// Returns true if the resource is "top level" i.e. not a sub-resource.
func (r *Resource) isTopLevel() bool {
	// TODO: This is a heuristic; replace with a definitive attribute?
	return (r.Create != nil || r.Read != nil)
}

// Determines if a given InstanceState needs to be migrated by checking the
// stored version number with the current SchemaVersion
func (r *Resource) checkSchemaVersion(is *terraform.InstanceState) (bool, int) {
	// Get the raw interface{} value for the schema version. If it doesn't
	// exist or is nil then set it to zero.
	raw := is.Meta["schema_version"]
	if raw == nil {
		raw = "0"
	}

	// Try to convert it to a string. If it isn't a string then we pretend
	// that it isn't set at all. It should never not be a string unless it
	// was manually tampered with.
	rawString, ok := raw.(string)
	if !ok {
		rawString = "0"
	}

	stateSchemaVersion, _ := strconv.Atoi(rawString)
	return stateSchemaVersion < r.SchemaVersion, stateSchemaVersion
}

func (r *Resource) recordCurrentSchemaVersion(
	state *terraform.InstanceState) *terraform.InstanceState {
	if state != nil && r.SchemaVersion > 0 {
		if state.Meta == nil {
			state.Meta = make(map[string]interface{})
		}
		state.Meta["schema_version"] = strconv.Itoa(r.SchemaVersion)
	}
	return state
}

// Noop is a convenience implementation of resource function which takes
// no action and returns no error.
func Noop(*ResourceData, interface{}) error {
	return nil
}

// RemoveFromState is a convenience implementation of a resource function
// which sets the resource ID to empty string (to remove it from state)
// and returns no error.
func RemoveFromState(d *ResourceData, _ interface{}) error {
	d.SetId("")
	return nil
}
