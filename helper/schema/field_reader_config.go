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
	Schema map[string]*Schema
}

func (r *ConfigFieldReader) ReadField(address []string) (FieldReadResult, error) {
	k := strings.Join(address, ".")
	schema := addrToSchema(address, r.Schema)
	if schema == nil {
		return FieldReadResult{}, nil
	}

	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		return r.readPrimitive(k, schema)
	case TypeList:
		return readListField(r, address, schema)
	case TypeMap:
		return r.readMap(k)
	case TypeSet:
		return r.readSet(address, schema)
	case typeObject:
		return readObjectField(r, address, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}
}

func (r *ConfigFieldReader) readMap(k string) (FieldReadResult, error) {
	mraw, ok := r.Config.Get(k)
	if !ok {
		return FieldReadResult{}, nil
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

	return FieldReadResult{
		Value:  result,
		Exists: true,
	}, nil
}

func (r *ConfigFieldReader) readPrimitive(
	k string, schema *Schema) (FieldReadResult, error) {
	raw, ok := r.Config.Get(k)
	if !ok {
		return FieldReadResult{}, nil
	}

	var result string
	if err := mapstructure.WeakDecode(raw, &result); err != nil {
		return FieldReadResult{}, err
	}

	computed := r.Config.IsComputed(k)
	returnVal, err := stringToPrimitive(result, computed, schema)
	if err != nil {
		return FieldReadResult{}, err
	}

	return FieldReadResult{
		Value:    returnVal,
		Exists:   true,
		Computed: computed,
	}, nil
}

func (r *ConfigFieldReader) readSet(
	address []string, schema *Schema) (FieldReadResult, error) {
	raw, err := readListField(r, address, schema)
	if err != nil {
		return FieldReadResult{}, err
	}
	if !raw.Exists {
		return FieldReadResult{}, nil
	}

	// Create the set that will be our result
	set := &Set{F: schema.Set}

	// If the list is computed, the set is necessarilly computed
	if raw.Computed {
		return FieldReadResult{
			Value:    set,
			Exists:   true,
			Computed: raw.Computed,
		}, nil
	}

	// Build up the set from the list elements
	for _, v := range raw.Value.([]interface{}) {
		set.Add(v)
	}

	return FieldReadResult{
		Value:  set,
		Exists: true,
	}, nil
}
