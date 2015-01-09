package schema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
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
	schema  map[string]*Schema
	config  *terraform.ResourceConfig
	state   *terraform.InstanceState
	diff    *terraform.InstanceDiff
	diffing bool

	// Don't set
	multiReader *MultiLevelFieldReader
	setMap      map[string]string
	newState    *terraform.InstanceState
	partial     bool
	partialMap  map[string]struct{}
	once        sync.Once
}

// getSource represents the level we want to get for a value (internally).
// Any source less than or equal to the level will be loaded (whichever
// has a value first).
type getSource byte

const (
	getSourceState getSource = 1 << iota
	getSourceConfig
	getSourceSet
	getSourceExact               // Only get from the _exact_ level
	getSourceDiff                // Apply the diff on top our level
	getSourceLevelMask getSource = getSourceState | getSourceConfig | getSourceSet
	getSourceMax       getSource = getSourceSet
)

// getResult is the internal structure that is generated when a Get
// is called that contains some extra data that might be used.
type getResult struct {
	Value          interface{}
	ValueProcessed interface{}
	Computed       bool
	Exists         bool
	Schema         *Schema
}

var getResultEmpty getResult

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
	o, n := d.getChange(key, getSourceConfig, getSourceConfig|getSourceDiff)
	return o.Value, n.Value
}

// GetOk returns the data for the given key and whether or not the key
// existed or not in the configuration. The second boolean result will also
// be false if a key is given that isn't in the schema at all.
//
// The first result will not necessarilly be nil if the value doesn't exist.
// The second result should be checked to determine this information.
func (d *ResourceData) GetOk(key string) (interface{}, bool) {
	r := d.getRaw(key, getSourceSet|getSourceDiff)
	return r.Value, r.Exists
}

func (d *ResourceData) getRaw(key string, level getSource) getResult {
	var parts []string
	if key != "" {
		parts = strings.Split(key, ".")
	}

	return d.getObject("", parts, d.schema, level)
}

// HasChange returns whether or not the given key has been changed.
func (d *ResourceData) HasChange(key string) bool {
	o, n := d.GetChange(key)
	return !reflect.DeepEqual(o, n)
}

// hasComputedSubKeys walks through a schema and returns whether or not the
// given key contains any subkeys that are computed.
func (d *ResourceData) hasComputedSubKeys(key string, schema *Schema) bool {
	prefix := key + "."

	switch t := schema.Elem.(type) {
	case *Resource:
		for k, schema := range t.Schema {
			if d.config.IsComputed(prefix + k) {
				return true
			}
			if d.hasComputedSubKeys(prefix+k, schema) {
				return true
			}
		}
	}
	return false
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
	parts := strings.Split(key, ".")
	return d.setObject("", parts, d.schema, value)
}

// SetPartial adds the key prefix to the final state output while
// in partial state mode.
//
// If partial state mode is disabled, then this has no effect. Additionally,
// whenever partial state mode is toggled, the partial data is cleared.
func (d *ResourceData) SetPartial(k string) {
	if d.partial {
		d.partialMap[k] = struct{}{}
	}
}

// Id returns the ID of the resource.
func (d *ResourceData) Id() string {
	var result string

	if d.state != nil {
		result = d.state.ID
	}

	if d.newState != nil {
		result = d.newState.ID
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
}

// SetConnInfo sets the connection info for a resource.
func (d *ResourceData) SetConnInfo(v map[string]string) {
	d.once.Do(d.init)
	d.newState.Ephemeral.ConnInfo = v
}

// State returns the new InstanceState after the diff and any Set
// calls.
func (d *ResourceData) State() *terraform.InstanceState {
	var result terraform.InstanceState
	result.ID = d.Id()

	// If we have no ID, then this resource doesn't exist and we just
	// return nil.
	if result.ID == "" {
		return nil
	}

	result.Attributes = d.stateObject("", d.schema)
	result.Ephemeral.ConnInfo = d.ConnInfo()

	if v := d.Id(); v != "" {
		result.Attributes["id"] = d.Id()
	}

	return &result
}

func (d *ResourceData) init() {
	// Initialize the field that will store our new state
	var copyState terraform.InstanceState
	if d.state != nil {
		copyState = *d.state
	}
	d.newState = &copyState

	// Initialize the map for storing set data
	d.setMap = make(map[string]string)

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
		Map:    BasicMapReader(d.setMap),
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
	k string) (interface{}, interface{}, bool, bool) {
	// Get the change between the state and the config.
	o, n := d.getChange(k, getSourceState, getSourceConfig|getSourceExact)
	if !o.Exists {
		o.Value = nil
	}
	if !n.Exists {
		n.Value = nil
	}

	// Return the old, new, and whether there is a change
	return o.Value, n.Value, !reflect.DeepEqual(o.Value, n.Value), n.Computed
}

func (d *ResourceData) getChange(
	key string,
	oldLevel getSource,
	newLevel getSource) (getResult, getResult) {
	var parts, parts2 []string
	if key != "" {
		parts = strings.Split(key, ".")
		parts2 = strings.Split(key, ".")
	}

	o := d.getObject("", parts, d.schema, oldLevel)
	n := d.getObject("", parts2, d.schema, newLevel)
	return o, n
}

func (d *ResourceData) get(
	k string,
	parts []string,
	schema *Schema,
	source getSource) getResult {
	d.once.Do(d.init)

	level := "set"
	flags := source & ^getSourceLevelMask
	diff := flags&getSourceDiff != 0
	exact := flags&getSourceExact != 0
	source = source & getSourceLevelMask
	if source >= getSourceSet {
		level = "set"
	} else if diff {
		level = "diff"
	} else if source >= getSourceConfig {
		level = "config"
	} else {
		level = "state"
	}

	// Build the address of the key we're looking for and ask the FieldReader
	addr := append(strings.Split(k, "."), parts...)
	for i, v := range addr {
		if v[0] == '~' {
			addr[i] = v[1:]
		}
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
	if result.Value == nil {
		if schema := addrToSchema(addr, d.schema); len(schema) > 0 {
			result.Value = schema[len(schema)-1].Type.Zero()
		}
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

func (d *ResourceData) getSet(
	k string,
	parts []string,
	schema *Schema,
	source getSource) getResult {
	s := &Set{F: schema.Set}
	result := getResult{Schema: schema, Value: s}
	prefix := k + "."

	// Get the set. For sets, the entire source must be exact: the
	// entire set must come from set, diff, state, etc. So we go backwards
	// and once we get a result, we take it. Or, we never get a result.
	var indexMap map[int]int
	codes := make(map[string]int)
	sourceLevel := source & getSourceLevelMask
	sourceFlags := source & ^getSourceLevelMask
	sourceDiff := sourceFlags&getSourceDiff != 0
	for setSource := sourceLevel; setSource > 0; setSource >>= 1 {
		// If we're already asking for an exact source and it doesn't
		// match, then leave since the original source was the match.
		if sourceFlags&getSourceExact != 0 && setSource != sourceLevel {
			break
		}

		if d.config != nil && setSource == getSourceConfig {
			raw := d.getList(k, nil, schema, setSource)
			// If the entire list is computed, then the entire set is
			// necessarilly computed.
			if raw.Computed {
				result.Computed = true
				if len(parts) > 0 {
					break
				}
				return result
			}

			if raw.Exists {
				result.Exists = true

				list := raw.Value.([]interface{})
				indexMap = make(map[int]int, len(list))

				// Build the set from all the items using the given hash code
				for i, v := range list {
					code := s.add(v)

					// Check if any of the keys in this item are computed
					computed := false
					if len(d.config.ComputedKeys) > 0 {
						prefix := fmt.Sprintf("%s.%d", k, i)
						computed = d.hasComputedSubKeys(prefix, schema)
					}

					// Check if we are computed and if so negatate the hash to
					// this is a approximate hash
					if computed {
						s.m[-code] = s.m[code]
						delete(s.m, code)
						code = -code
					}
					indexMap[code] = i
				}

				break
			}
		}

		if d.state != nil && setSource == getSourceState {
			for k, _ := range d.state.Attributes {
				if !strings.HasPrefix(k, prefix) || strings.HasPrefix(k, prefix+"#") {
					continue
				}
				parts := strings.Split(k[len(prefix):], ".")
				idx := parts[0]
				if _, ok := codes[idx]; ok {
					continue
				}

				code, err := strconv.Atoi(strings.Replace(parts[0], "~", "-", -1))
				if err != nil {
					panic(fmt.Sprintf("unable to convert %s to int: %v", idx, err))
				}
				codes[idx] = code
			}
		}

		if d.setMap != nil && setSource == getSourceSet {
			for k, _ := range d.setMap {
				if !strings.HasPrefix(k, prefix) || strings.HasPrefix(k, prefix+"#") {
					continue
				}
				parts := strings.Split(k[len(prefix):], ".")
				idx := parts[0]
				if _, ok := codes[idx]; ok {
					continue
				}

				code, err := strconv.Atoi(strings.Replace(parts[0], "~", "-", -1))
				if err != nil {
					panic(fmt.Sprintf("unable to convert %s to int: %v", idx, err))
				}
				codes[idx] = code
			}
		}

		if d.diff != nil && sourceDiff {
			for k, _ := range d.diff.Attributes {
				if !strings.HasPrefix(k, prefix) || strings.HasPrefix(k, prefix+"#") {
					continue
				}
				parts := strings.Split(k[len(prefix):], ".")
				idx := parts[0]
				if _, ok := codes[idx]; ok {
					continue
				}

				code, err := strconv.Atoi(strings.Replace(parts[0], "~", "-", -1))
				if err != nil {
					panic(fmt.Sprintf("unable to convert %s to int: %v", idx, err))
				}
				codes[idx] = code
			}
		}

		if len(codes) > 0 {
			break
		}
	}

	if indexMap == nil {
		s.m = make(map[int]interface{})
		for idx, code := range codes {
			switch t := schema.Elem.(type) {
			case *Resource:
				// Get the entire object
				m := make(map[string]interface{})
				for field, _ := range t.Schema {
					m[field] = d.getObject(prefix+idx, []string{field}, t.Schema, source).Value
				}
				s.m[code] = m
				result.Exists = true
			case *Schema:
				// Get a single value
				s.m[code] = d.get(prefix+idx, nil, t, source).Value
				result.Exists = true
			}
		}
	}

	if len(parts) > 0 {
		// We still have parts left over meaning we're accessing an
		// element of this set.
		idx := parts[0]
		parts = parts[1:]

		// Special case if we're accessing the count of the set
		if idx == "#" {
			schema := &Schema{Type: TypeInt}
			return d.get(prefix+"#", parts, schema, source)
		}

		if source&getSourceLevelMask == getSourceConfig {
			i, err := strconv.Atoi(strings.Replace(idx, "~", "-", -1))
			if err != nil {
				panic(fmt.Sprintf("unable to convert %s to int: %v", idx, err))
			}
			if i, ok := indexMap[i]; ok {
				idx = strconv.Itoa(i)
			}
		}

		switch t := schema.Elem.(type) {
		case *Resource:
			return d.getObject(prefix+idx, parts, t.Schema, source)
		case *Schema:
			return d.get(prefix+idx, parts, t, source)
		}
	}

	return result
}

func (d *ResourceData) getMap(
	k string,
	parts []string,
	schema *Schema,
	source getSource) getResult {
	elemSchema := &Schema{Type: TypeString}

	result := make(map[string]interface{})
	resultSet := false
	prefix := k + "."

	flags := source & ^getSourceLevelMask
	level := source & getSourceLevelMask
	exact := flags&getSourceExact != 0
	diff := flags&getSourceDiff != 0

	if !exact || level == getSourceState {
		if d.state != nil && level >= getSourceState {
			for k, _ := range d.state.Attributes {
				if !strings.HasPrefix(k, prefix) {
					continue
				}

				single := k[len(prefix):]
				result[single] = d.getPrimitive(k, nil, elemSchema, source).Value
				resultSet = true
			}
		}
	}

	if d.config != nil && level == getSourceConfig {
		// For config, we always set the result to exactly what was requested
		if mraw, ok := d.config.Get(k); ok {
			result = make(map[string]interface{})
			switch m := mraw.(type) {
			case []interface{}:
				for _, innerRaw := range m {
					for k, v := range innerRaw.(map[string]interface{}) {
						result[k] = v
					}
				}

				resultSet = true
			case []map[string]interface{}:
				for _, innerRaw := range m {
					for k, v := range innerRaw {
						result[k] = v
					}
				}

				resultSet = true
			case map[string]interface{}:
				result = m
				resultSet = true
			default:
				panic(fmt.Sprintf("unknown type: %#v", mraw))
			}
		} else {
			result = nil
		}
	}

	if d.diff != nil && diff {
		for k, v := range d.diff.Attributes {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			resultSet = true

			single := k[len(prefix):]

			if v.NewRemoved {
				delete(result, single)
			} else {
				result[single] = d.getPrimitive(k, nil, elemSchema, source).Value
			}
		}
	}

	if !exact || level == getSourceSet {
		if d.setMap != nil && level >= getSourceSet {
			cleared := false
			if v, ok := d.setMap[k]; ok && v == "" {
				// We've cleared the map
				result = make(map[string]interface{})
				resultSet = true
			} else {
				for k, _ := range d.setMap {
					if !strings.HasPrefix(k, prefix) {
						continue
					}
					resultSet = true

					if !cleared {
						// We clear the results if they are in the set map
						result = make(map[string]interface{})
						cleared = true
					}

					single := k[len(prefix):]
					result[single] = d.getPrimitive(
						k, nil, elemSchema, source).Value
				}
			}
		}
	}

	// If we're requesting a specific element, return that
	var resultValue interface{} = result
	if len(parts) > 0 {
		resultValue = result[parts[0]]
	}

	return getResult{
		Value:  resultValue,
		Exists: resultSet,
		Schema: schema,
	}
}

func (d *ResourceData) getObject(
	k string,
	parts []string,
	schema map[string]*Schema,
	source getSource) getResult {
	if len(parts) > 0 {
		// We're requesting a specific key in an object
		key := parts[0]
		parts = parts[1:]
		s, ok := schema[key]
		if !ok {
			return getResultEmpty
		}

		if k != "" {
			// If we're not at the root, then we need to append
			// the key to get the full key path.
			key = fmt.Sprintf("%s.%s", k, key)
		}

		return d.get(key, parts, s, source)
	}

	// Get the entire object
	result := make(map[string]interface{})
	for field, _ := range schema {
		result[field] = d.getObject(k, []string{field}, schema, source).Value
	}

	return getResult{
		Value:  result,
		Exists: true,
		Schema: &Schema{
			Elem: schema,
		},
	}
}

func (d *ResourceData) getList(
	k string,
	parts []string,
	schema *Schema,
	source getSource) getResult {
	if len(parts) > 0 {
		// We still have parts left over meaning we're accessing an
		// element of this list.
		idx := parts[0]
		parts = parts[1:]

		// Special case if we're accessing the count of the list
		if idx == "#" {
			schema := &Schema{Type: TypeInt}
			return d.get(k+".#", parts, schema, source)
		}

		key := fmt.Sprintf("%s.%s", k, idx)
		switch t := schema.Elem.(type) {
		case *Resource:
			return d.getObject(key, parts, t.Schema, source)
		case *Schema:
			return d.get(key, parts, t, source)
		}
	}

	// Get the entire list.
	var result []interface{}
	count := d.getList(k, []string{"#"}, schema, source)
	if !count.Computed {
		result = make([]interface{}, count.Value.(int))
		for i, _ := range result {
			is := strconv.FormatInt(int64(i), 10)
			result[i] = d.getList(k, []string{is}, schema, source).Value
		}
	}

	return getResult{
		Value:    result,
		Computed: count.Computed,
		Exists:   count.Exists,
		Schema:   schema,
	}
}

func (d *ResourceData) getPrimitive(
	k string,
	parts []string,
	schema *Schema,
	source getSource) getResult {
	var result string
	var resultProcessed interface{}
	var resultComputed, resultSet bool
	flags := source & ^getSourceLevelMask
	source = source & getSourceLevelMask
	exact := flags&getSourceExact != 0
	diff := flags&getSourceDiff != 0

	if !exact || source == getSourceState {
		if d.state != nil && source >= getSourceState {
			result, resultSet = d.state.Attributes[k]
		}
	}

	// No exact check is needed here because config is always exact
	if d.config != nil && source == getSourceConfig {
		// For config, we always return the exact value
		if v, ok := d.config.Get(k); ok {
			if err := mapstructure.WeakDecode(v, &result); err != nil {
				panic(err)
			}

			resultSet = true
		} else {
			result = ""
			resultSet = false
		}

		// If it is computed, set that.
		resultComputed = d.config.IsComputed(k)
	}

	if d.diff != nil && diff {
		attrD, ok := d.diff.Attributes[k]
		if ok {
			if !attrD.NewComputed {
				result = attrD.New
				if attrD.NewExtra != nil {
					// If NewExtra != nil, then we have processed data as the New,
					// so we store that but decode the unprocessed data into result
					resultProcessed = result

					err := mapstructure.WeakDecode(attrD.NewExtra, &result)
					if err != nil {
						panic(err)
					}
				}

				resultSet = true
			} else {
				result = ""
				resultSet = false
			}
		}
	}

	if !exact || source == getSourceSet {
		if d.setMap != nil && source >= getSourceSet {
			if v, ok := d.setMap[k]; ok {
				result = v
				resultSet = true
			}
		}
	}

	if !resultSet {
		result = ""
	}

	var resultValue interface{}
	switch schema.Type {
	case TypeBool:
		if result == "" {
			resultValue = false
			break
		}

		v, err := strconv.ParseBool(result)
		if err != nil {
			panic(err)
		}

		resultValue = v
	case TypeString:
		// Use the value as-is. We just put this case here to be explicit.
		resultValue = result
	case TypeInt:
		if result == "" {
			resultValue = 0
			break
		}

		if resultComputed {
			break
		}

		v, err := strconv.ParseInt(result, 0, 0)
		if err != nil {
			panic(err)
		}

		resultValue = int(v)
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}

	return getResult{
		Value:          resultValue,
		ValueProcessed: resultProcessed,
		Computed:       resultComputed,
		Exists:         resultSet,
		Schema:         schema,
	}
}

func (d *ResourceData) set(
	k string,
	parts []string,
	schema *Schema,
	value interface{}) error {
	switch schema.Type {
	case TypeList:
		return d.setList(k, parts, schema, value)
	case TypeMap:
		return d.setMapValue(k, parts, schema, value)
	case TypeSet:
		return d.setSet(k, parts, schema, value)
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		return d.setPrimitive(k, schema, value)
	default:
		panic(fmt.Sprintf("%s: unknown type %#v", k, schema.Type))
	}
}

func (d *ResourceData) setList(
	k string,
	parts []string,
	schema *Schema,
	value interface{}) error {
	if len(parts) > 0 {
		return fmt.Errorf("%s: can only set the full list, not elements", k)
	}

	setElement := func(k string, idx string, value interface{}) error {
		if idx == "#" {
			return fmt.Errorf("%s: can't set count of list", k)
		}

		key := fmt.Sprintf("%s.%s", k, idx)
		switch t := schema.Elem.(type) {
		case *Resource:
			return d.setObject(key, nil, t.Schema, value)
		case *Schema:
			return d.set(key, nil, t, value)
		}

		return nil
	}

	var vs []interface{}
	if err := mapstructure.Decode(value, &vs); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	// Set the entire list.
	var err error
	for i, elem := range vs {
		is := strconv.FormatInt(int64(i), 10)
		err = setElement(k, is, elem)
		if err != nil {
			break
		}
	}
	if err != nil {
		for i, _ := range vs {
			is := strconv.FormatInt(int64(i), 10)
			setElement(k, is, nil)
		}

		return err
	}

	d.setMap[k+".#"] = strconv.FormatInt(int64(len(vs)), 10)
	return nil
}

func (d *ResourceData) setMapValue(
	k string,
	parts []string,
	schema *Schema,
	value interface{}) error {
	elemSchema := &Schema{Type: TypeString}
	if len(parts) > 0 {
		return fmt.Errorf("%s: full map must be set, no a single element", k)
	}

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Map {
		return fmt.Errorf("%s: must be a map", k)
	}
	if v.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("%s: keys must strings", k)
	}
	vs := make(map[string]interface{})
	for _, mk := range v.MapKeys() {
		mv := v.MapIndex(mk)
		vs[mk.String()] = mv.Interface()
	}

	if len(vs) == 0 {
		// The empty string here means the map is removed.
		d.setMap[k] = ""
		return nil
	}

	delete(d.setMap, k)
	for subKey, v := range vs {
		err := d.set(fmt.Sprintf("%s.%s", k, subKey), nil, elemSchema, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *ResourceData) setObject(
	k string,
	parts []string,
	schema map[string]*Schema,
	value interface{}) error {
	if len(parts) > 0 {
		// We're setting a specific key in an object
		key := parts[0]
		parts = parts[1:]

		s, ok := schema[key]
		if !ok {
			return fmt.Errorf("%s (internal): unknown key to set: %s", k, key)
		}

		if k != "" {
			// If we're not at the root, then we need to append
			// the key to get the full key path.
			key = fmt.Sprintf("%s.%s", k, key)
		}

		return d.set(key, parts, s, value)
	}

	// Set the entire object. First decode into a proper structure
	var v map[string]interface{}
	if err := mapstructure.Decode(value, &v); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	// Set each element in turn
	var err error
	for k1, v1 := range v {
		err = d.setObject(k, []string{k1}, schema, v1)
		if err != nil {
			break
		}
	}
	if err != nil {
		for k1, _ := range v {
			d.setObject(k, []string{k1}, schema, nil)
		}
	}

	return err
}

func (d *ResourceData) setPrimitive(
	k string,
	schema *Schema,
	v interface{}) error {
	if v == nil {
		delete(d.setMap, k)
		return nil
	}

	var set string
	switch schema.Type {
	case TypeBool:
		var b bool
		if err := mapstructure.Decode(v, &b); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}

		set = strconv.FormatBool(b)
	case TypeString:
		if err := mapstructure.Decode(v, &set); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}
	case TypeInt:
		var n int
		if err := mapstructure.Decode(v, &n); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}

		set = strconv.FormatInt(int64(n), 10)
	default:
		return fmt.Errorf("Unknown type: %#v", schema.Type)
	}

	d.setMap[k] = set
	return nil
}

func (d *ResourceData) setSet(
	k string,
	parts []string,
	schema *Schema,
	value interface{}) error {
	if len(parts) > 0 {
		return fmt.Errorf("%s: can only set the full set, not elements", k)
	}

	// If it is a slice, then we have to turn it into a *Set so that
	// we get the proper order back based on the hash code.
	if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
		// Build a temp *ResourceData to use for the conversion
		tempD := &ResourceData{
			setMap: make(map[string]string),
			schema: map[string]*Schema{k: schema},
		}
		tempD.once.Do(tempD.init)

		// Set the entire list, this lets us get sane values out of it
		if err := tempD.setList(k, nil, schema, value); err != nil {
			return err
		}

		// Build the set by going over the list items in order and
		// hashing them into the set. The reason we go over the list and
		// not the `value` directly is because this forces all types
		// to become []interface{} (generic) instead of []string, which
		// most hash functions are expecting.
		s := &Set{F: schema.Set}
		source := getSourceSet | getSourceExact
		for i := 0; i < v.Len(); i++ {
			is := strconv.FormatInt(int64(i), 10)
			result := tempD.get(k, []string{is}, schema, source)
			if !result.Exists {
				panic("just set item doesn't exist")
			}

			s.Add(result.Value)
		}

		value = s
	}

	switch t := schema.Elem.(type) {
	case *Resource:
		for code, elem := range value.(*Set).m {
			for field, _ := range t.Schema {
				subK := fmt.Sprintf("%s.%d", k, code)
				err := d.setObject(
					subK, []string{field}, t.Schema, elem.(map[string]interface{})[field])
				if err != nil {
					return err
				}
			}
		}
	case *Schema:
		for code, elem := range value.(*Set).m {
			subK := fmt.Sprintf("%s.%d", k, code)
			err := d.set(subK, nil, t, elem)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%s: unknown element type (internal)", k)
	}

	d.setMap[k+".#"] = strconv.Itoa(value.(*Set).Len())
	return nil
}

func (d *ResourceData) stateList(
	prefix string,
	schema *Schema) map[string]string {
	countRaw := d.get(prefix, []string{"#"}, schema, d.stateSource(prefix))
	if !countRaw.Exists {
		if schema.Computed {
			// If it is computed, then it always _exists_ in the state,
			// it is just empty.
			countRaw.Exists = true
			countRaw.Value = 0
		} else {
			return nil
		}
	}
	count := countRaw.Value.(int)

	result := make(map[string]string)
	if count > 0 || schema.Computed {
		result[prefix+".#"] = strconv.FormatInt(int64(count), 10)
	}
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%s.%d", prefix, i)

		var m map[string]string
		switch t := schema.Elem.(type) {
		case *Resource:
			m = d.stateObject(key, t.Schema)
		case *Schema:
			m = d.stateSingle(key, t)
		}

		for k, v := range m {
			result[k] = v
		}
	}

	return result
}

func (d *ResourceData) stateMap(
	prefix string,
	schema *Schema) map[string]string {
	v := d.getMap(prefix, nil, schema, d.stateSource(prefix))
	if !v.Exists {
		return nil
	}

	elemSchema := &Schema{Type: TypeString}
	result := make(map[string]string)
	for mk, _ := range v.Value.(map[string]interface{}) {
		mp := fmt.Sprintf("%s.%s", prefix, mk)
		for k, v := range d.stateSingle(mp, elemSchema) {
			result[k] = v
		}
	}

	return result
}

func (d *ResourceData) stateObject(
	prefix string,
	schema map[string]*Schema) map[string]string {
	result := make(map[string]string)
	for k, v := range schema {
		key := k
		if prefix != "" {
			key = prefix + "." + key
		}

		for k1, v1 := range d.stateSingle(key, v) {
			result[k1] = v1
		}
	}

	return result
}

func (d *ResourceData) statePrimitive(
	prefix string,
	schema *Schema) map[string]string {
	raw := d.getRaw(prefix, d.stateSource(prefix))
	if !raw.Exists {
		return nil
	}

	v := raw.Value
	if raw.ValueProcessed != nil {
		v = raw.ValueProcessed
	}

	var vs string
	switch schema.Type {
	case TypeBool:
		vs = strconv.FormatBool(v.(bool))
	case TypeString:
		vs = v.(string)
	case TypeInt:
		vs = strconv.FormatInt(int64(v.(int)), 10)
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}

	return map[string]string{
		prefix: vs,
	}
}

func (d *ResourceData) stateSet(
	prefix string,
	schema *Schema) map[string]string {
	raw := d.get(prefix, nil, schema, d.stateSource(prefix))
	if !raw.Exists {
		if schema.Computed {
			// If it is computed, then it always _exists_ in the state,
			// it is just empty.
			raw.Exists = true
			raw.Value = new(Set)
		} else {
			return nil
		}
	}

	set := raw.Value.(*Set)
	result := make(map[string]string)
	result[prefix+".#"] = strconv.Itoa(set.Len())

	for _, idx := range set.listCode() {
		key := fmt.Sprintf("%s.%d", prefix, idx)

		var m map[string]string
		switch t := schema.Elem.(type) {
		case *Resource:
			m = d.stateObject(key, t.Schema)
		case *Schema:
			m = d.stateSingle(key, t)
		}

		for k, v := range m {
			result[k] = v
		}
	}

	return result
}

func (d *ResourceData) stateSingle(
	prefix string,
	schema *Schema) map[string]string {
	switch schema.Type {
	case TypeList:
		return d.stateList(prefix, schema)
	case TypeMap:
		return d.stateMap(prefix, schema)
	case TypeSet:
		return d.stateSet(prefix, schema)
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		return d.statePrimitive(prefix, schema)
	default:
		panic(fmt.Sprintf("%s: unknown type %#v", prefix, schema.Type))
	}
}

func (d *ResourceData) stateSource(prefix string) getSource {
	// If we're not doing a partial apply, then get the set level
	if !d.partial {
		return getSourceSet | getSourceDiff
	}

	// Otherwise, only return getSourceSet if its in the partial map.
	// Otherwise we use state level only.
	for k, _ := range d.partialMap {
		if strings.HasPrefix(prefix, k) {
			return getSourceSet | getSourceDiff
		}
	}

	return getSourceState
}
