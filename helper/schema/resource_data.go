package schema

import (
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceData is used to query and set the attributes of a resource.
//
// ResourceData is the primary argument received for CRUD operations on
// a resource as well as configuration of a provider. It is a powerful
// structure that can be used to not only query data, but check for changes,
// define partial state updates, etc.
//
// The most relevant methods to take a look at are Get, Set, and Partial.
type ResourceData struct {
	// Settable (internally)
	schema   map[string]*Schema
	config   *terraform.ResourceConfig
	state    *terraform.InstanceState
	diff     *terraform.InstanceDiff
	meta     map[string]interface{}
	timeouts *ResourceTimeout

	// Don't set
	multiReader *MultiLevelFieldReader
	setWriter   *MapFieldWriter
	newState    *terraform.InstanceState
	partial     bool
	partialMap  map[string]struct{}
	once        sync.Once
	isNew       bool

	panicOnError bool
}

// getResult is the internal structure that is generated when a Get
// is called that contains some extra data that might be used.
type getResult struct {
	Value          interface{}
	ValueProcessed interface{}
	Computed       bool
	Exists         bool
	Schema         *Schema
}

// UnsafeSetFieldRaw allows setting arbitrary values in state to arbitrary
// values, bypassing schema. This MUST NOT be used in normal circumstances -
// it exists only to support the remote_state data source.
func (d *ResourceData) UnsafeSetFieldRaw(key string, value string) {
	d.once.Do(d.init)

	d.setWriter.unsafeWriteField(key, value)
}

// Get returns the data for the given key, or nil if the key doesn't exist
// in the schema.
//
// If the key does exist in the schema but doesn't exist in the configuration,
// then the default value for that type will be returned. For strings, this is
// "", for numbers it is 0, etc.
//
// If you want to test if something is set at all in the configuration,
// use GetOk.
func (d *ResourceData) Get(key string) interface{} {
	v, _ := d.GetOk(key)
	return v
}

// GetChange returns the old and new value for a given key.
//
// HasChange should be used to check if a change exists. It is possible
// that both the old and new value are the same if the old value was not
// set and the new value is. This is common, for example, for boolean
// fields which have a zero value of false.
func (d *ResourceData) GetChange(key string) (interface{}, interface{}) {
	o, n := d.getChange(key, getSourceState, getSourceDiff)
	return o.Value, n.Value
}

// GetOk returns the data for the given key and whether or not the key
// has been set to a non-zero value at some point.
//
// The first result will not necessarilly be nil if the value doesn't exist.
// The second result should be checked to determine this information.
func (d *ResourceData) GetOk(key string) (interface{}, bool) {
	r := d.getRaw(key, getSourceSet)
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

// GetOkExists returns the data for a given key and whether or not the key
// has been set to a non-zero value. This is only useful for determining
// if boolean attributes have been set, if they are Optional but do not
// have a Default value.
//
// This is nearly the same function as GetOk, yet it does not check
// for the zero value of the attribute's type. This allows for attributes
// without a default, to fully check for a literal assignment, regardless
// of the zero-value for that type.
// This should only be used if absolutely required/needed.
func (d *ResourceData) GetOkExists(key string) (interface{}, bool) {
	r := d.getRaw(key, getSourceSet)
	exists := r.Exists && !r.Computed
	return r.Value, exists
}

func (d *ResourceData) getRaw(key string, level getSource) getResult {
	var parts []string
	if key != "" {
		parts = strings.Split(key, ".")
	}

	return d.get(parts, level)
}

// HasChange returns whether or not the given key has been changed.
func (d *ResourceData) HasChange(key string) bool {
	o, n := d.GetChange(key)

	// If the type implements the Equal interface, then call that
	// instead of just doing a reflect.DeepEqual. An example where this is
	// needed is *Set
	if eq, ok := o.(Equal); ok {
		return !eq.Equal(n)
	}

	return !reflect.DeepEqual(o, n)
}

// Partial turns partial state mode on/off.
//
// When partial state mode is enabled, then only key prefixes specified
// by SetPartial will be in the final state. This allows providers to return
// partial states for partially applied resources (when errors occur).
func (d *ResourceData) Partial(on bool) {
	d.partial = on
	if on {
		if d.partialMap == nil {
			d.partialMap = make(map[string]struct{})
		}
	} else {
		d.partialMap = nil
	}
}

// Set sets the value for the given key.
//
// If the key is invalid or the value is not a correct type, an error
// will be returned.
func (d *ResourceData) Set(key string, value interface{}) error {
	d.once.Do(d.init)

	// If the value is a pointer to a non-struct, get its value and
	// use that. This allows Set to take a pointer to primitives to
	// simplify the interface.
	reflectVal := reflect.ValueOf(value)
	if reflectVal.Kind() == reflect.Ptr {
		if reflectVal.IsNil() {
			// If the pointer is nil, then the value is just nil
			value = nil
		} else {
			// Otherwise, we dereference the pointer as long as its not
			// a pointer to a struct, since struct pointers are allowed.
			reflectVal = reflect.Indirect(reflectVal)
			if reflectVal.Kind() != reflect.Struct {
				value = reflectVal.Interface()
			}
		}
	}

	err := d.setWriter.WriteField(strings.Split(key, "."), value)
	if err != nil && d.panicOnError {
		panic(err)
	}
	return err
}

// SetPartial adds the key to the final state output while
// in partial state mode. The key must be a root key in the schema (i.e.
// it cannot be "list.0").
//
// If partial state mode is disabled, then this has no effect. Additionally,
// whenever partial state mode is toggled, the partial data is cleared.
func (d *ResourceData) SetPartial(k string) {
	if d.partial {
		d.partialMap[k] = struct{}{}
	}
}

func (d *ResourceData) MarkNewResource() {
	d.isNew = true
}

func (d *ResourceData) IsNewResource() bool {
	return d.isNew
}

// Id returns the ID of the resource.
func (d *ResourceData) Id() string {
	var result string

	if d.state != nil {
		result = d.state.ID
		if result == "" {
			result = d.state.Attributes["id"]
		}
	}

	if d.newState != nil {
		result = d.newState.ID
		if result == "" {
			result = d.newState.Attributes["id"]
		}
	}

	return result
}

// ConnInfo returns the connection info for this resource.
func (d *ResourceData) ConnInfo() map[string]string {
	if d.newState != nil {
		return d.newState.Ephemeral.ConnInfo
	}

	if d.state != nil {
		return d.state.Ephemeral.ConnInfo
	}

	return nil
}

// SetId sets the ID of the resource. If the value is blank, then the
// resource is destroyed.
func (d *ResourceData) SetId(v string) {
	d.once.Do(d.init)
	d.newState.ID = v

	// once we transition away from the legacy state types, "id" will no longer
	// be a special field, and will become a normal attribute.
	// set the attribute normally
	d.setWriter.unsafeWriteField("id", v)

	// Make sure the newState is also set, otherwise the old value
	// may get precedence.
	if d.newState.Attributes == nil {
		d.newState.Attributes = map[string]string{}
	}
	d.newState.Attributes["id"] = v
}

// SetConnInfo sets the connection info for a resource.
func (d *ResourceData) SetConnInfo(v map[string]string) {
	d.once.Do(d.init)
	d.newState.Ephemeral.ConnInfo = v
}

// SetType sets the ephemeral type for the data. This is only required
// for importing.
func (d *ResourceData) SetType(t string) {
	d.once.Do(d.init)
	d.newState.Ephemeral.Type = t
}

// State returns the new InstanceState after the diff and any Set
// calls.
func (d *ResourceData) State() *terraform.InstanceState {
	var result terraform.InstanceState
	result.ID = d.Id()
	result.Meta = d.meta

	// If we have no ID, then this resource doesn't exist and we just
	// return nil.
	if result.ID == "" {
		return nil
	}

	if d.timeouts != nil {
		if err := d.timeouts.StateEncode(&result); err != nil {
			log.Printf("[ERR] Error encoding Timeout meta to Instance State: %s", err)
		}
	}

	// Look for a magic key in the schema that determines we skip the
	// integrity check of fields existing in the schema, allowing dynamic
	// keys to be created.
	hasDynamicAttributes := false
	for k, _ := range d.schema {
		if k == "__has_dynamic_attributes" {
			hasDynamicAttributes = true
			log.Printf("[INFO] Resource %s has dynamic attributes", result.ID)
		}
	}

	// In order to build the final state attributes, we read the full
	// attribute set as a map[string]interface{}, write it to a MapFieldWriter,
	// and then use that map.
	rawMap := make(map[string]interface{})
	for k := range d.schema {
		source := getSourceSet
		if d.partial {
			source = getSourceState
			if _, ok := d.partialMap[k]; ok {
				source = getSourceSet
			}
		}

		raw := d.get([]string{k}, source)
		if raw.Exists && !raw.Computed {
			rawMap[k] = raw.Value
			if raw.ValueProcessed != nil {
				rawMap[k] = raw.ValueProcessed
			}
		}
	}

	mapW := &MapFieldWriter{Schema: d.schema}
	if err := mapW.WriteField(nil, rawMap); err != nil {
		log.Printf("[ERR] Error writing fields: %s", err)
		return nil
	}

	result.Attributes = mapW.Map()

	if hasDynamicAttributes {
		// If we have dynamic attributes, just copy the attributes map
		// one for one into the result attributes.
		for k, v := range d.setWriter.Map() {
			// Don't clobber schema values. This limits usage of dynamic
			// attributes to names which _do not_ conflict with schema
			// keys!
			if _, ok := result.Attributes[k]; !ok {
				result.Attributes[k] = v
			}
		}
	}

	if d.newState != nil {
		result.Ephemeral = d.newState.Ephemeral
	}

	// TODO: This is hacky and we can remove this when we have a proper
	// state writer. We should instead have a proper StateFieldWriter
	// and use that.
	for k, schema := range d.schema {
		if schema.Type != TypeMap {
			continue
		}

		if result.Attributes[k] == "" {
			delete(result.Attributes, k)
		}
	}

	if v := d.Id(); v != "" {
		result.Attributes["id"] = d.Id()
	}

	if d.state != nil {
		result.Tainted = d.state.Tainted
	}

	return &result
}

// Timeout returns the data for the given timeout key
// Returns a duration of 20 minutes for any key not found, or not found and no default.
func (d *ResourceData) Timeout(key string) time.Duration {
	key = strings.ToLower(key)

	// System default of 20 minutes
	defaultTimeout := 20 * time.Minute

	if d.timeouts == nil {
		return defaultTimeout
	}

	var timeout *time.Duration
	switch key {
	case TimeoutCreate:
		timeout = d.timeouts.Create
	case TimeoutRead:
		timeout = d.timeouts.Read
	case TimeoutUpdate:
		timeout = d.timeouts.Update
	case TimeoutDelete:
		timeout = d.timeouts.Delete
	}

	if timeout != nil {
		return *timeout
	}

	if d.timeouts.Default != nil {
		return *d.timeouts.Default
	}

	return defaultTimeout
}

func (d *ResourceData) init() {
	// Initialize the field that will store our new state
	var copyState terraform.InstanceState
	if d.state != nil {
		copyState = *d.state.DeepCopy()
	}
	d.newState = &copyState

	// Initialize the map for storing set data
	d.setWriter = &MapFieldWriter{Schema: d.schema}

	// Initialize the reader for getting data from the
	// underlying sources (config, diff, etc.)
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
	readers["set"] = &MapFieldReader{
		Schema: d.schema,
		Map:    BasicMapReader(d.setWriter.Map()),
	}
	d.multiReader = &MultiLevelFieldReader{
		Levels: []string{
			"state",
			"config",
			"diff",
			"set",
		},

		Readers: readers,
	}
}

func (d *ResourceData) diffChange(
	k string) (interface{}, interface{}, bool, bool, bool) {
	// Get the change between the state and the config.
	o, n := d.getChange(k, getSourceState, getSourceConfig|getSourceExact)
	if !o.Exists {
		o.Value = nil
	}
	if !n.Exists {
		n.Value = nil
	}

	// Return the old, new, and whether there is a change
	return o.Value, n.Value, !reflect.DeepEqual(o.Value, n.Value), n.Computed, false
}

func (d *ResourceData) getChange(
	k string,
	oldLevel getSource,
	newLevel getSource) (getResult, getResult) {
	var parts, parts2 []string
	if k != "" {
		parts = strings.Split(k, ".")
		parts2 = strings.Split(k, ".")
	}

	o := d.get(parts, oldLevel)
	n := d.get(parts2, newLevel)
	return o, n
}

func (d *ResourceData) get(addr []string, source getSource) getResult {
	d.once.Do(d.init)

	level := "set"
	flags := source & ^getSourceLevelMask
	exact := flags&getSourceExact != 0
	source = source & getSourceLevelMask
	if source >= getSourceSet {
		level = "set"
	} else if source >= getSourceDiff {
		level = "diff"
	} else if source >= getSourceConfig {
		level = "config"
	} else {
		level = "state"
	}

	var result FieldReadResult
	var err error
	if exact {
		result, err = d.multiReader.ReadFieldExact(addr, level)
	} else {
		result, err = d.multiReader.ReadFieldMerge(addr, level)
	}
	if err != nil {
		panic(err)
	}

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
