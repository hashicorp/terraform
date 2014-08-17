package schema

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// ResourceData is used to query and set the attributes of a resource.
type ResourceData struct {
	schema map[string]*Schema
	state  *terraform.ResourceState
	diff   *terraform.ResourceDiff
	setMap map[string]string
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

	return d.getObject("", parts, d.schema)
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

// State returns the new ResourceState after the diff and any Set
// calls.
func (d *ResourceData) State() *terraform.ResourceState {
	var result terraform.ResourceState
	result.Attributes = d.stateObject("", d.schema)
	return &result
}

func (d *ResourceData) get(
	k string,
	parts []string,
	schema *Schema) interface{} {
	switch schema.Type {
	case TypeList:
		return d.getList(k, parts, schema)
	default:
		return d.getPrimitive(k, parts, schema)
	}
}

func (d *ResourceData) getObject(
	k string,
	parts []string,
	schema map[string]*Schema) interface{} {
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

		return d.get(key, parts, s)
	}

	// Get the entire object
	result := make(map[string]interface{})
	for field, _ := range schema {
		result[field] = d.getObject(k, []string{field}, schema)
	}

	return result
}

func (d *ResourceData) getList(
	k string,
	parts []string,
	schema *Schema) interface{} {
	if len(parts) > 0 {
		// We still have parts left over meaning we're accessing an
		// element of this list.
		idx := parts[0]
		parts = parts[1:]

		// Special case if we're accessing the count of the list
		if idx == "#" {
			schema := &Schema{Type: TypeInt}
			result := d.get(k+".#", parts, schema)
			if result == nil {
				result = 0
			}

			return result
		}

		key := fmt.Sprintf("%s.%s", k, idx)
		switch t := schema.Elem.(type) {
		case *Resource:
			return d.getObject(key, parts, t.Schema)
		case *Schema:
			return d.get(key, parts, t)
		}
	}

	// Get the entire list.
	result := make([]interface{}, d.getList(k, []string{"#"}, schema).(int))
	for i, _ := range result {
		is := strconv.FormatInt(int64(i), 10)
		result[i] = d.getList(k, []string{is}, schema)
	}

	return result
}

func (d *ResourceData) getPrimitive(
	k string,
	parts []string,
	schema *Schema) interface{} {
	var result string
	var resultSet bool
	if d.state != nil {
		result, resultSet = d.state.Attributes[k]
	}

	if d.diff != nil {
		attrD, ok := d.diff.Attributes[k]
		if ok {
			result = attrD.New
			resultSet = true
		}
	}

	if d.setMap != nil {
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
	countRaw := d.get(prefix, []string{"#"}, schema)
	if countRaw == nil {
		return nil
	}
	count := countRaw.(int)

	result := make(map[string]string)
	result[prefix + ".#"] = strconv.FormatInt(int64(count), 10)
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
	v := d.getPrimitive(prefix, nil, schema)
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
	default:
		return d.statePrimitive(prefix, schema)
	}
}
