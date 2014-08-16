package schema

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceData is used to query and set the attributes of a resource.
type ResourceData struct {
	schema map[string]*Schema
	state  *terraform.ResourceState
	diff   *terraform.ResourceDiff
}

// Get returns the data for the given key, or nil if the key doesn't exist.
func (d *ResourceData) Get(key string) interface{} {
	parts := strings.Split(key, ".")
	return d.getObject("", parts, d.schema)
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
			return d.get(k+".#", parts, schema)
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
	if d.state != nil {
		result = d.state.Attributes[k]
	}

	if d.diff != nil {
		attrD, ok := d.diff.Attributes[k]
		if ok {
			result = attrD.New
		}
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
