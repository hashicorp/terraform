package schema

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceDiff is used to query and make custom changes to an in-flight diff.
// It can be used to veto particular changes in the diff, customize the diff
// that has been created, or diff values not controlled by config.
//
// The object functions similar to ResourceData, however most notably lacks
// Set, SetPartial, and Partial, as it should only be used to change diff
// values only. Most other frist-class ResourceData functions exist, namely
// Get, GetOk, HasChange, and GetChange exist.
//
// All functions in ResourceDiff, save for ForceNew, can only be used on
// computed fields.
type ResourceDiff struct {
	// The underlying ResourceData object used as a refrence and diff storage.
	data *ResourceData

	// A source "copy" of the ResourceData object, designed to preserve the
	// original diff, schema, and state to allow for rollbacks.
	originalData *ResourceData

	// A writer that holds overridden old fields.
	oldWriter *MapFieldWriter

	// A reader that is tied to oldWriter's map.
	oldReader *MapFieldReader

	// A writer that holds overridden new fields.
	newWriter *MapFieldWriter

	// A reader that is tied to newWriter's map.
	newReader *MapFieldReader

	// A map of keys that will be force-flagged as computed.
	computedKeys map[string]bool

	// A catalog of top-level keys that are safely diffable. diffChange will
	// panic if the key is not found here to guard against bugs and edge cases
	// when processing diffs.
	catalog map[string]bool
}

// newResourceDiff creates a new ResourceDiff instance.
func newResourceDiff(diff *terraform.InstanceDiff, data *ResourceData) *ResourceDiff {
	d := new(ResourceDiff)
	d.originalData = &ResourceData{
		schema: data.schema,
		state:  data.state,
		config: data.config,
		diff:   diff,
	}
	d.data = new(ResourceData)
	d.data.config = data.config
	d.data.state = d.originalData.state.DeepCopy()
	d.data.diff = d.originalData.diff.DeepCopy()
	for k, v := range d.originalData.schema {
		newSchema := *v
		d.data.schema[k] = &newSchema
	}

	d.data.once.Do(d.data.init)

	d.oldWriter = &MapFieldWriter{Schema: data.schema}
	d.oldReader = &MapFieldReader{
		Schema: data.schema,
		Map:    BasicMapReader(d.oldWriter.Map()),
	}

	d.newWriter = &MapFieldWriter{Schema: data.schema}
	d.newReader = &MapFieldReader{
		Schema: data.schema,
		Map:    BasicMapReader(d.newWriter.Map()),
	}

	return d
}

// ClearAll re-creates the ResourceDiff instance and drops the old one on the
// floor. The new instance starts off without a diff.
func (d *ResourceDiff) ClearAll() {
	nd := newResourceDiff(d.originalData.diff, d.originalData)
	nd.data.diff = new(terraform.InstanceDiff)
	d = nd
}

// Reset re-creates the ResourceDiff instance, similar to ClearAll, but with
// the original diff preserved.
func (d *ResourceDiff) Reset() {
	nd := newResourceDiff(d.originalData.diff, d.originalData)
	d = nd
}

// getDiff returns the current diff as it is in the underlying ResourceData
// object.
func (d *ResourceDiff) getDiff() *terraform.InstanceDiff {
	return d.data.diff
}

// diffChange helps to implement resourceDiffer and derives its change values
// from ResourceDiff's own change data.
//
// Note that it's a currently unsupported operation to diff a field (and hence
// use this function) on a field that has not been explicitly operated on with
// SetNew, SetNewComputed, or SetDiff. The function will panic if you do.
func (d *ResourceDiff) diffChange(key string) (interface{}, interface{}, bool, bool) {
	// Panic if key was never set by any of our diff functions. It's not a legit
	// use case of this function to be used outside of very specific functions in
	// ResourceDiff.
	if _, ok := d.catalog[key]; !ok {
		panic(fmt.Errorf("ResourceDiff.diffChange: %s was not found as a valid set key", key))
	}

	old, err := d.oldReader.ReadField(strings.Split(key, "."))
	if err != nil {
		panic(fmt.Errorf("ResourceDiff.diffChange: Reading old value for %s failed: %s", key, err))
	}
	new, err := d.newReader.ReadField(strings.Split(key, "."))
	if err != nil {
		panic(fmt.Errorf("ResourceDiff.diffChange: Reading new value for %s failed: %s", key, err))
	}

	if !old.Exists {
		old.Value = nil
	}
	if !new.Exists {
		new.Value = nil
	}

	var computed bool
	if v, ok := d.computedKeys[strings.Split(key, ".")[0]]; ok {
		computed = v
	}

	return old.Value, new.Value, !reflect.DeepEqual(old.Value, new.Value), computed
}

// SetNew is used to set a new diff value for the mentioned key. The value must
// be correct for the attribute's schema (mostly relevant for maps, lists, and
// sets). The original value from the state is used as the old value.
//
// This function is only allowed on computed attributes.
//
// It is an unsupported operation to set invalid values with this function -
// doing so will taint any existing diff for this key and will remove it from
// the catalog.
func (d *ResourceDiff) SetNew(key string, value interface{}) error {
	return d.SetDiff(key, d.Get(key), value, false)
}

// SetNewComputed functions like SetNew, except that it sets the new value to
// the zero value and flags the attribute's diff as computed.
//
// This function is only allowed on computed keys.
func (d *ResourceDiff) SetNewComputed(key string) error {
	return d.SetDiff(key, d.Get(key), d.data.schema[key].ZeroValue(), true)
}

// SetDiff allows the setting of both old and new values for the diff
// referenced by a given key. This can be used to completely override
// Terraform's own diff behaviour, and can be used in conjunction with Clear or
// ClearAll to construct a compleletely new diff based off of provider logic
// alone.
//
// This function is only allowed on computed keys.
func (d *ResourceDiff) SetDiff(key string, old, new interface{}, computed bool) error {
	if !d.data.schema[key].Computed {
		return fmt.Errorf("SetNew, SetNewComputed, and SetDiff are allowed on computed attributes only - %s is not one", key)
	}

	if err := d.oldWriter.WriteField(strings.Split(key, "."), old); err != nil {
		delete(d.catalog, key)
		return fmt.Errorf("Cannot set old diff value for key %s: %s", key, err)
	}

	if err := d.newWriter.WriteField(strings.Split(key, "."), new); err != nil {
		delete(d.catalog, key)
		return fmt.Errorf("Cannot set new diff value for key %s: %s", key, err)
	}

	d.computedKeys[key] = computed
	d.catalog[key] = true

	return schemaMap(d.data.schema).diff(key, d.data.schema[key], d.data.diff, d, false)
}

// ForceNew force-flags ForceNew in the schema for a specific key, and
// re-calculates its diff. This function is a no-op/error if there is no diff.
//
// Note that the change to schema is permanent for the lifecycle of this
// specific ResourceDiff instance, until ClearAll or Reset is called to start
// anew.
func (d *ResourceDiff) ForceNew(key string) error {
	if !d.HasChange(key) {
		return fmt.Errorf("ResourceDiff.ForceNew: No changes for %s", key)
	}

	old, new := d.GetChange(key)
	d.data.schema[key].ForceNew = true
	return d.SetDiff(key, old, new, false)
}

// Get hands off to ResourceData.Get.
func (d *ResourceDiff) Get(key string) interface{} {
	return d.data.Get(key)
}

// GetChange hands off to ResourceData.GetChange.
func (d *ResourceDiff) GetChange(key string) (interface{}, interface{}) {
	return d.data.getChange(key, getSourceState, getSourceDiff)
}

// GetOk hands off to ResourceData.GetOk.
func (d *ResourceDiff) GetOk(key string) (interface{}, bool) {
	return d.data.GetOk(key)
}

// HasChange hands off to ResourceData.HasChange.
func (d *ResourceDiff) HasChange(key string) bool {
	return d.data.HasChange(key)
}

// Id hands off to ResourceData.Id.
func (d *ResourceDiff) Id() string {
	return d.data.Id()
}
