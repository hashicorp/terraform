// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package schema

import (
	"reflect"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/internal/legacy/terraform"
)

// newValueWriter is a minor re-implementation of MapFieldWriter to include
// keys that should be marked as computed, to represent the new part of a
// pseudo-diff.
type newValueWriter struct {
	*MapFieldWriter

	// A list of keys that should be marked as computed.
	computedKeys map[string]bool

	// A lock to prevent races on writes. The underlying writer will have one as
	// well - this is for computed keys.
	lock sync.Mutex

	// To be used with init.
	once sync.Once
}

// init performs any initialization tasks for the newValueWriter.
func (w *newValueWriter) init() {
	if w.computedKeys == nil {
		w.computedKeys = make(map[string]bool)
	}
}

// ComputedKeysMap returns the underlying computed keys map.
func (w *newValueWriter) ComputedKeysMap() map[string]bool {
	w.once.Do(w.init)
	return w.computedKeys
}

// newValueReader is a minor re-implementation of MapFieldReader and is the
// read counterpart to MapValueWriter, allowing the read of keys flagged as
// computed to accommodate the diff override logic in ResourceDiff.
type newValueReader struct {
	*MapFieldReader

	// The list of computed keys from a newValueWriter.
	computedKeys map[string]bool
}

// ReadField reads the values from the underlying writer, returning the
// computed value if it is found as well.
func (r *newValueReader) ReadField(address []string) (FieldReadResult, error) {
	addrKey := strings.Join(address, ".")
	v, err := r.MapFieldReader.ReadField(address)
	if err != nil {
		return FieldReadResult{}, err
	}
	for computedKey := range r.computedKeys {
		if childAddrOf(addrKey, computedKey) {
			if strings.HasSuffix(addrKey, ".#") {
				// This is a count value for a list or set that has been marked as
				// computed, or a sub-list/sub-set of a complex resource that has
				// been marked as computed.  We need to pass through to other readers
				// so that an accurate previous count can be fetched for the diff.
				v.Exists = false
			}
			v.Computed = true
		}
	}

	return v, nil
}

// ResourceDiff is used to query and make custom changes to an in-flight diff.
// It can be used to veto particular changes in the diff, customize the diff
// that has been created, or diff values not controlled by config.
//
// The object functions similar to ResourceData, however most notably lacks
// Set, SetPartial, and Partial, as it should be used to change diff values
// only.  Most other first-class ResourceData functions exist, namely Get,
// GetOk, HasChange, and GetChange exist.
//
// All functions in ResourceDiff, save for ForceNew, can only be used on
// computed fields.
type ResourceDiff struct {
	// The schema for the resource being worked on.
	schema map[string]*Schema

	// The current config for this resource.
	config *terraform.ResourceConfig

	// The state for this resource as it exists post-refresh, after the initial
	// diff.
	state *terraform.InstanceState

	// The diff created by Terraform. This diff is used, along with state,
	// config, and custom-set diff data, to provide a multi-level reader
	// experience similar to ResourceData.
	diff *terraform.InstanceDiff

	// The internal reader structure that contains the state, config, the default
	// diff, and the new diff.
	multiReader *MultiLevelFieldReader

	// A writer that writes overridden new fields.
	newWriter *newValueWriter

	// Tracks which keys have been updated by ResourceDiff to ensure that the
	// diff does not get re-run on keys that were not touched, or diffs that were
	// just removed (re-running on the latter would just roll back the removal).
	updatedKeys map[string]bool

	// Tracks which keys were flagged as forceNew. These keys are not saved in
	// newWriter, but we need to track them so that they can be re-diffed later.
	forcedNewKeys map[string]bool
}

// newResourceDiff creates a new ResourceDiff instance.
func newResourceDiff(schema map[string]*Schema, config *terraform.ResourceConfig, state *terraform.InstanceState, diff *terraform.InstanceDiff) *ResourceDiff {
	d := &ResourceDiff{
		config: config,
		state:  state,
		diff:   diff,
		schema: schema,
	}

	d.newWriter = &newValueWriter{
		MapFieldWriter: &MapFieldWriter{Schema: d.schema},
	}
	readers := make(map[string]FieldReader)
	var stateAttributes map[string]string
	if d.state != nil {
		stateAttributes = d.state.Attributes
		readers["state"] = &MapFieldReader{
			Schema: d.schema,
			Map:    BasicMapReader(stateAttributes),
		}
	}
	if d.config != nil {
		readers["config"] = &ConfigFieldReader{
			Schema: d.schema,
			Config: d.config,
		}
	}
	if d.diff != nil {
		readers["diff"] = &DiffFieldReader{
			Schema: d.schema,
			Diff:   d.diff,
			Source: &MultiLevelFieldReader{
				Levels:  []string{"state", "config"},
				Readers: readers,
			},
		}
	}
	readers["newDiff"] = &newValueReader{
		MapFieldReader: &MapFieldReader{
			Schema: d.schema,
			Map:    BasicMapReader(d.newWriter.Map()),
		},
		computedKeys: d.newWriter.ComputedKeysMap(),
	}
	d.multiReader = &MultiLevelFieldReader{
		Levels: []string{
			"state",
			"config",
			"diff",
			"newDiff",
		},

		Readers: readers,
	}

	d.updatedKeys = make(map[string]bool)
	d.forcedNewKeys = make(map[string]bool)

	return d
}

// UpdatedKeys returns the keys that were updated by this ResourceDiff run.
// These are the only keys that a diff should be re-calculated for.
//
// This is the combined result of both keys for which diff values were updated
// for or cleared, and also keys that were flagged to be re-diffed as a result
// of ForceNew.
func (d *ResourceDiff) UpdatedKeys() []string {
	var s []string
	for k := range d.updatedKeys {
		s = append(s, k)
	}
	for k := range d.forcedNewKeys {
		for _, l := range s {
			if k == l {
				break
			}
		}
		s = append(s, k)
	}
	return s
}

// diffChange helps to implement resourceDiffer and derives its change values
// from ResourceDiff's own change data, in addition to existing diff, config, and state.
func (d *ResourceDiff) diffChange(key string) (interface{}, interface{}, bool, bool, bool) {
	old, new, customized := d.getChange(key)

	if !old.Exists {
		old.Value = nil
	}
	if !new.Exists || d.removed(key) {
		new.Value = nil
	}

	return old.Value, new.Value, !reflect.DeepEqual(old.Value, new.Value), new.Computed, customized
}

// GetOk functions the same way as ResourceData.GetOk, but it also checks the
// new diff levels to provide data consistent with the current state of the
// customized diff.
func (d *ResourceDiff) GetOk(key string) (interface{}, bool) {
	r := d.get(strings.Split(key, "."), "newDiff")
	exists := r.Exists && !r.Computed
	if exists {
		// If it exists, we also want to verify it is not the zero-value.
		value := r.Value
		zero := r.Schema.Type.Zero()

		if eq, ok := value.(Equal); ok {
			exists = !eq.Equal(zero)
		} else {
			exists = !reflect.DeepEqual(value, zero)
		}
	}

	return r.Value, exists
}

// Id returns the ID of this resource.
//
// Note that technically, ID does not change during diffs (it either has
// already changed in the refresh, or will change on update), hence we do not
// support updating the ID or fetching it from anything else other than state.
func (d *ResourceDiff) Id() string {
	var result string

	if d.state != nil {
		result = d.state.ID
	}
	return result
}

// getChange gets values from two different levels, designed for use in
// diffChange, HasChange, and GetChange.
//
// This implementation differs from ResourceData's in the way that we first get
// results from the exact levels for the new diff, then from state and diff as
// per normal.
func (d *ResourceDiff) getChange(key string) (getResult, getResult, bool) {
	old := d.get(strings.Split(key, "."), "state")
	var new getResult
	for p := range d.updatedKeys {
		if childAddrOf(key, p) {
			new = d.getExact(strings.Split(key, "."), "newDiff")
			return old, new, true
		}
	}
	new = d.get(strings.Split(key, "."), "newDiff")
	return old, new, false
}

// removed checks to see if the key is present in the existing, pre-customized
// diff and if it was marked as NewRemoved.
func (d *ResourceDiff) removed(k string) bool {
	diff, ok := d.diff.Attributes[k]
	if !ok {
		return false
	}
	return diff.NewRemoved
}

// get performs the appropriate multi-level reader logic for ResourceDiff,
// starting at source. Refer to newResourceDiff for the level order.
func (d *ResourceDiff) get(addr []string, source string) getResult {
	result, err := d.multiReader.ReadFieldMerge(addr, source)
	if err != nil {
		panic(err)
	}

	return d.finalizeResult(addr, result)
}

// getExact gets an attribute from the exact level referenced by source.
func (d *ResourceDiff) getExact(addr []string, source string) getResult {
	result, err := d.multiReader.ReadFieldExact(addr, source)
	if err != nil {
		panic(err)
	}

	return d.finalizeResult(addr, result)
}

// finalizeResult does some post-processing of the result produced by get and getExact.
func (d *ResourceDiff) finalizeResult(addr []string, result FieldReadResult) getResult {
	// If the result doesn't exist, then we set the value to the zero value
	var schema *Schema
	if schemaL := addrToSchema(addr, d.schema); len(schemaL) > 0 {
		schema = schemaL[len(schemaL)-1]
	}

	if result.Value == nil && schema != nil {
		result.Value = result.ValueOrZero(schema)
	}

	// Transform the FieldReadResult into a getResult. It might be worth
	// merging these two structures one day.
	return getResult{
		Value:          result.Value,
		ValueProcessed: result.ValueProcessed,
		Computed:       result.Computed,
		Exists:         result.Exists,
		Schema:         schema,
	}
}

// childAddrOf does a comparison of two addresses to see if one is the child of
// the other.
func childAddrOf(child, parent string) bool {
	cs := strings.Split(child, ".")
	ps := strings.Split(parent, ".")
	if len(ps) > len(cs) {
		return false
	}
	return reflect.DeepEqual(ps, cs[:len(ps)])
}
