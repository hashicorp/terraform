package schema

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// ConfigFieldReader reads fields out of an untyped map[string]string to
// the best of its ability.
type ConfigFieldReader struct {
	Config *terraform.ResourceConfig
}

func (r *ConfigFieldReader) ReadField(
	address []string, schema *Schema) (interface{}, bool, bool, error) {
	k := strings.Join(address, ".")

	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		return r.readPrimitive(k, schema)
	case TypeList:
		return readListField(r, k, schema)
	case TypeMap:
		return r.readMap(k)
	case TypeSet:
		return r.readSet(k, schema)
	case typeObject:
		return readObjectField(r, k, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}
}

func (r *ConfigFieldReader) readMap(k string) (interface{}, bool, bool, error) {
	mraw, ok := r.Config.Get(k)
	if !ok {
		return nil, false, false, nil
	}

	result := make(map[string]interface{})
	switch m := mraw.(type) {
	case []interface{}:
		for _, innerRaw := range m {
			for k, v := range innerRaw.(map[string]interface{}) {
				result[k] = v
			}
		}
	case []map[string]interface{}:
		for _, innerRaw := range m {
			for k, v := range innerRaw {
				result[k] = v
			}
		}
	case map[string]interface{}:
		result = m
	default:
		panic(fmt.Sprintf("unknown type: %#v", mraw))
	}

	return result, true, false, nil
}

func (r *ConfigFieldReader) readPrimitive(
	k string, schema *Schema) (interface{}, bool, bool, error) {
	raw, ok := r.Config.Get(k)
	if !ok {
		return nil, false, false, nil
	}

	var result string
	if err := mapstructure.WeakDecode(raw, &result); err != nil {
		return nil, false, false, err
	}

	computed := r.Config.IsComputed(k)
	returnVal, err := stringToPrimitive(result, computed, schema)
	if err != nil {
		return nil, false, false, err
	}

	return returnVal, true, computed, nil
}

func (r *ConfigFieldReader) readSet(
	k string, schema *Schema) (interface{}, bool, bool, error) {
	raw, ok, computed, err := readListField(r, k, schema)
	if err != nil {
		return nil, false, false, err
	}
	if !ok {
		return nil, false, false, nil
	}

	// Create the set that will be our result
	set := &Set{F: schema.Set}

	// If the list is computed, the set is necessarilly computed
	if computed {
		return set, true, computed, nil
	}

	// Build up the set from the list elements
	for _, v := range raw.([]interface{}) {
		set.Add(v)
	}

	return set, true, false, nil
}
