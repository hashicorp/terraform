package schema

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// ConfigFieldReader reads fields out of an untyped map[string]string to
// the best of its ability.
type ConfigFieldReader struct {
	Config *terraform.ResourceConfig
	Schema map[string]*Schema

	lock sync.Mutex
}

func (r *ConfigFieldReader) ReadField(address []string) (FieldReadResult, error) {
	return r.readField(address, false)
}

func (r *ConfigFieldReader) readField(
	address []string, nested bool) (FieldReadResult, error) {
	schemaList := addrToSchema(address, r.Schema)
	if len(schemaList) == 0 {
		return FieldReadResult{}, nil
	}

	if !nested {
		// If we have a set anywhere in the address, then we need to
		// read that set out in order and actually replace that part of
		// the address with the real list index. i.e. set.50 might actually
		// map to set.12 in the config, since it is in list order in the
		// config, not indexed by set value.
		for i, v := range schemaList {
			// Sets are the only thing that cause this issue.
			if v.Type != TypeSet {
				continue
			}

			// If we're at the end of the list, then we don't have to worry
			// about this because we're just requesting the whole set.
			if i == len(schemaList)-1 {
				continue
			}

			// If we're looking for the count, then ignore...
			if address[i+1] == "#" {
				continue
			}

			// Get the code
			code, err := strconv.ParseInt(address[i+1], 0, 0)
			if err != nil {
				return FieldReadResult{}, err
			}

			// Get the set so we can get the index map that tells us the
			// mapping of the hash code to the list index
			_, indexMap, err := r.readSet(address[:i+1], v)
			if err != nil {
				return FieldReadResult{}, err
			}

			index, ok := indexMap[int(code)]
			if !ok {
				return FieldReadResult{}, nil
			}

			address[i+1] = strconv.FormatInt(int64(index), 10)
		}
	}

	k := strings.Join(address, ".")
	schema := schemaList[len(schemaList)-1]
	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		return r.readPrimitive(k, schema)
	case TypeList:
		return readListField(&nestedConfigFieldReader{r}, address, schema)
	case TypeMap:
		return r.readMap(k)
	case TypeSet:
		result, _, err := r.readSet(address, schema)
		return result, err
	case typeObject:
		return readObjectField(
			&nestedConfigFieldReader{r},
			address, schema.Elem.(map[string]*Schema))
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
	address []string, schema *Schema) (FieldReadResult, map[int]int, error) {
	indexMap := make(map[int]int)
	// Create the set that will be our result
	set := &Set{F: schema.Set}

	raw, err := readListField(&nestedConfigFieldReader{r}, address, schema)
	if err != nil {
		return FieldReadResult{}, indexMap, err
	}
	if !raw.Exists {
		return FieldReadResult{Value: set}, indexMap, nil
	}

	// If the list is computed, the set is necessarilly computed
	if raw.Computed {
		return FieldReadResult{
			Value:    set,
			Exists:   true,
			Computed: raw.Computed,
		}, indexMap, nil
	}

	// Build up the set from the list elements
	for i, v := range raw.Value.([]interface{}) {
		// Check if any of the keys in this item are computed
		computed := r.hasComputedSubKeys(
			fmt.Sprintf("%s.%d", strings.Join(address, "."), i), schema)

		code := set.add(v)
		indexMap[code] = i
		if computed {
			set.m[-code] = set.m[code]
			delete(set.m, code)
			code = -code
		}
	}

	return FieldReadResult{
		Value:  set,
		Exists: true,
	}, indexMap, nil
}

// hasComputedSubKeys walks through a schema and returns whether or not the
// given key contains any subkeys that are computed.
func (r *ConfigFieldReader) hasComputedSubKeys(key string, schema *Schema) bool {
	prefix := key + "."

	switch t := schema.Elem.(type) {
	case *Resource:
		for k, schema := range t.Schema {
			if r.Config.IsComputed(prefix + k) {
				return true
			}

			if r.hasComputedSubKeys(prefix+k, schema) {
				return true
			}
		}
	}

	return false
}

// nestedConfigFieldReader is a funny little thing that just wraps a
// ConfigFieldReader to call readField when ReadField is called so that
// we don't recalculate the set rewrites in the address, which leads to
// an infinite loop.
type nestedConfigFieldReader struct {
	Reader *ConfigFieldReader
}

func (r *nestedConfigFieldReader) ReadField(
	address []string) (FieldReadResult, error) {
	return r.Reader.readField(address, true)
}
