package schema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// getSource represents the level we want to get for a value (internally).
// Any source less than or equal to the level will be loaded (whichever
// has a value first).
type getSource byte

const (
	getSourceState getSource = iota
	getSourceDiff
	getSourceSet
)

// ResourceData is used to query and set the attributes of a resource.
type ResourceData struct {
	schema map[string]*Schema
	state  *terraform.ResourceState
	diff   *terraform.ResourceDiff

	setMap   map[string]string
	newState *terraform.ResourceState
}

// Get returns the data for the given key, or nil if the key doesn't exist.
//
// The type of the data returned will be according to the schema specified.
// Primitives will be their respective types in Go, lists will always be
// []interface{}, and sub-resources will be map[string]interface{}.
func (d *ResourceData) Get(key string) interface{} {
	var parts []string
	if key != "" {
		parts = strings.Split(key, ".")
	}

	return d.getObject("", parts, d.schema, getSourceSet)
}

// GetChange returns the old and new value for a given key.
//
// If there is no change, then old and new will simply be the same.
func (d *ResourceData) GetChange(key string) (interface{}, interface{}) {
	var parts []string
	if key != "" {
		parts = strings.Split(key, ".")
	}

	o := d.getObject("", parts, d.schema, getSourceState)
	n := d.getObject("", parts, d.schema, getSourceDiff)
	return o, n
}

// HasChange returns whether or not the given key has been changed.
func (d *ResourceData) HasChange(key string) bool {
	o, n := d.GetChange(key)
	return !reflect.DeepEqual(o, n)
}

// Set sets the value for the given key.
//
// If the key is invalid or the value is not a correct type, an error
// will be returned.
func (d *ResourceData) Set(key string, value interface{}) error {
	if d.setMap == nil {
		d.setMap = make(map[string]string)
	}

	parts := strings.Split(key, ".")
	return d.setObject("", parts, d.schema, value)
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

// SetId sets the ID of the resource. If the value is blank, then the
// resource is destroyed.
func (d *ResourceData) SetId(v string) {
	if d.newState == nil {
		d.newState = new(terraform.ResourceState)
	}

	d.newState.ID = v
}

// State returns the new ResourceState after the diff and any Set
// calls.
func (d *ResourceData) State() *terraform.ResourceState {
	var result terraform.ResourceState
	result.ID = d.Id()
	result.Attributes = d.stateObject("", d.schema)

	return &result
}

func (d *ResourceData) get(
	k string,
	parts []string,
	schema *Schema,
	source getSource) interface{} {
	switch schema.Type {
	case TypeList:
		return d.getList(k, parts, schema, source)
	case TypeMap:
		return d.getMap(k, parts, schema, source)
	default:
		return d.getPrimitive(k, parts, schema, source)
	}
}

func (d *ResourceData) getMap(
	k string,
	parts []string,
	schema *Schema,
	source getSource) interface{} {
	elemSchema := &Schema{Type: TypeString}

	result := make(map[string]interface{})
	prefix := k + "."

	if d.state != nil && source >= getSourceState {
		for k, _ := range d.state.Attributes {
			if !strings.HasPrefix(k, prefix) {
				continue
			}

			single := k[len(prefix):]
			result[single] = d.getPrimitive(k, nil, elemSchema, source)
		}
	}

	if d.diff != nil && source >= getSourceDiff {
		for k, v := range d.diff.Attributes {
			if !strings.HasPrefix(k, prefix) {
				continue
			}

			single := k[len(prefix):]

			if v.NewRemoved {
				delete(result, single)
			} else {
				result[single] = d.getPrimitive(k, nil, elemSchema, source)
			}
		}
	}

	if d.setMap != nil && source >= getSourceSet {
		cleared := false
		for k, _ := range d.setMap {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			if !cleared {
				// We clear the results if they are in the set map
				result = make(map[string]interface{})
				cleared = true
			}

			single := k[len(prefix):]
			result[single] = d.getPrimitive(k, nil, elemSchema, source)
		}
	}

	return result
}

func (d *ResourceData) getObject(
	k string,
	parts []string,
	schema map[string]*Schema,
	source getSource) interface{} {
	if len(parts) > 0 {
		// We're requesting a specific key in an object
		key := parts[0]
		parts = parts[1:]
		s, ok := schema[key]
		if !ok {
			return nil
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
		result[field] = d.getObject(k, []string{field}, schema, source)
	}

	return result
}

func (d *ResourceData) getList(
	k string,
	parts []string,
	schema *Schema,
	source getSource) interface{} {
	if len(parts) > 0 {
		// We still have parts left over meaning we're accessing an
		// element of this list.
		idx := parts[0]
		parts = parts[1:]

		// Special case if we're accessing the count of the list
		if idx == "#" {
			schema := &Schema{Type: TypeInt}
			result := d.get(k+".#", parts, schema, source)
			if result == nil {
				result = 0
			}

			return result
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
	result := make(
		[]interface{},
		d.getList(k, []string{"#"}, schema, source).(int))
	for i, _ := range result {
		is := strconv.FormatInt(int64(i), 10)
		result[i] = d.getList(k, []string{is}, schema, source)
	}

	return result
}

func (d *ResourceData) getPrimitive(
	k string,
	parts []string,
	schema *Schema,
	source getSource) interface{} {
	var result string
	var resultSet bool
	if d.state != nil && source >= getSourceState {
		result, resultSet = d.state.Attributes[k]
	}

	if d.diff != nil && source >= getSourceDiff {
		attrD, ok := d.diff.Attributes[k]
		if ok && !attrD.NewComputed {
			result = attrD.New
			resultSet = true
		}
	}

	if d.setMap != nil && source >= getSourceSet {
		if v, ok := d.setMap[k]; ok {
			result = v
			resultSet = true
		}
	}

	if !resultSet {
		return nil
	}

	switch schema.Type {
	case TypeString:
		// Use the value as-is. We just put this case here to be explicit.
		return result
	case TypeInt:
		if result == "" {
			return 0
		}

		v, err := strconv.ParseInt(result, 0, 0)
		if err != nil {
			panic(err)
		}

		return int(v)
	default:
		panic(fmt.Sprintf("Unknown type: %s", schema.Type))
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
	default:
		return d.setPrimitive(k, schema, value)
	}
}

func (d *ResourceData) setList(
	k string,
	parts []string,
	schema *Schema,
	value interface{}) error {
	if len(parts) > 0 {
		// We're setting a specific element
		idx := parts[0]
		parts = parts[1:]

		// Special case if we're accessing the count of the list
		if idx == "#" {
			return fmt.Errorf("%s: can't set count of list", k)
		}

		key := fmt.Sprintf("%s.%s", k, idx)
		switch t := schema.Elem.(type) {
		case *Resource:
			return d.setObject(key, parts, t.Schema, value)
		case *Schema:
			return d.set(key, parts, t, value)
		}
	}

	var vs []interface{}
	if err := mapstructure.Decode(value, &vs); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	// Set the entire list.
	var err error
	for i, elem := range vs {
		is := strconv.FormatInt(int64(i), 10)
		err = d.setList(k, []string{is}, schema, elem)
		if err != nil {
			break
		}
	}
	if err != nil {
		for i, _ := range vs {
			is := strconv.FormatInt(int64(i), 10)
			d.setList(k, []string{is}, schema, nil)
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

	// Delete any prior map set
	/*
		v := d.getMap(k, nil, schema, getSourceSet)
		for subKey, _ := range v.(map[string]interface{}) {
			delete(d.setMap, fmt.Sprintf("%s.%s", k, subKey))
		}
	*/

	vs := value.(map[string]interface{})
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
		return fmt.Errorf("Unknown type: %s", schema.Type)
	}

	d.setMap[k] = set
	return nil
}

func (d *ResourceData) stateList(
	prefix string,
	schema *Schema) map[string]string {
	countRaw := d.get(prefix, []string{"#"}, schema, getSourceSet)
	if countRaw == nil {
		return nil
	}
	count := countRaw.(int)

	result := make(map[string]string)
	result[prefix+".#"] = strconv.FormatInt(int64(count), 10)
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
	v := d.getMap(prefix, nil, schema, getSourceSet)
	if v == nil {
		return nil
	}

	elemSchema := &Schema{Type: TypeString}
	result := make(map[string]string)
	for mk, _ := range v.(map[string]interface{}) {
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
	v := d.getPrimitive(prefix, nil, schema, getSourceSet)
	if v == nil {
		return nil
	}

	var vs string
	switch schema.Type {
	case TypeString:
		vs = v.(string)
	case TypeInt:
		vs = strconv.FormatInt(int64(v.(int)), 10)
	default:
		panic(fmt.Sprintf("Unknown type: %s", schema.Type))
	}

	return map[string]string{
		prefix: vs,
	}
}

func (d *ResourceData) stateSingle(
	prefix string,
	schema *Schema) map[string]string {
	switch schema.Type {
	case TypeList:
		return d.stateList(prefix, schema)
	case TypeMap:
		return d.stateMap(prefix, schema)
	default:
		return d.statePrimitive(prefix, schema)
	}
}
