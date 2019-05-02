package schema

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// ConfigFieldReader reads fields out of an untyped map[string]string to the
// best of its ability. It also applies defaults from the Schema. (The other
// field readers do not need default handling because they source fully
// populated data structures.)
type ConfigFieldReader struct {
	Config *terraform.ResourceConfig
	Schema map[string]*Schema

	indexMaps map[string]map[string]int
	once      sync.Once
}

func (r *ConfigFieldReader) ReadField(address []string) (FieldReadResult, error) {
	r.once.Do(func() { r.indexMaps = make(map[string]map[string]int) })
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

			indexMap, ok := r.indexMaps[strings.Join(address[:i+1], ".")]
			if !ok {
				// Get the set so we can get the index map that tells us the
				// mapping of the hash code to the list index
				_, err := r.readSet(address[:i+1], v)
				if err != nil {
					return FieldReadResult{}, err
				}
				indexMap = r.indexMaps[strings.Join(address[:i+1], ".")]
			}

			index, ok := indexMap[address[i+1]]
			if !ok {
				return FieldReadResult{}, nil
			}

			address[i+1] = strconv.FormatInt(int64(index), 10)
		}
	}

	k := strings.Join(address, ".")
	schema := schemaList[len(schemaList)-1]

	// If we're getting the single element of a promoted list, then
	// check to see if we have a single element we need to promote.
	if address[len(address)-1] == "0" && len(schemaList) > 1 {
		lastSchema := schemaList[len(schemaList)-2]
		if lastSchema.Type == TypeList && lastSchema.PromoteSingle {
			k := strings.Join(address[:len(address)-1], ".")
			result, err := r.readPrimitive(k, schema)
			if err == nil {
				return result, nil
			}
		}
	}

	if protoVersion5 {
		// Check if the value itself is unknown.
		// The new protocol shims will add unknown values to this list of
		// ComputedKeys. THis is the only way we have to indicate that a
		// collection is unknown in the config
		for _, unknown := range r.Config.ComputedKeys {
			if k == unknown {
				return FieldReadResult{Computed: true, Exists: true}, nil
			}
		}
	}

	switch schema.Type {
	case TypeBool, TypeFloat, TypeInt, TypeString:
		return r.readPrimitive(k, schema)
	case TypeList:
		// If we support promotion then we first check if we have a lone
		// value that we must promote.
		// a value that is alone.
		if schema.PromoteSingle {
			result, err := r.readPrimitive(k, schema.Elem.(*Schema))
			if err == nil && result.Exists {
				result.Value = []interface{}{result.Value}
				return result, nil
			}
		}

		return readListField(&nestedConfigFieldReader{r}, address, schema)
	case TypeMap:
		return r.readMap(k, schema)
	case TypeSet:
		return r.readSet(address, schema)
	case typeObject:
		return readObjectField(
			&nestedConfigFieldReader{r},
			address, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %s", schema.Type))
	}
}

func (r *ConfigFieldReader) readMap(k string, schema *Schema) (FieldReadResult, error) {
	// We want both the raw value and the interpolated. We use the interpolated
	// to store actual values and we use the raw one to check for
	// computed keys. Actual values are obtained in the switch, depending on
	// the type of the raw value.
	mraw, ok := r.Config.GetRaw(k)
	if !ok {
		// check if this is from an interpolated field by seeing if it exists
		// in the config
		_, ok := r.Config.Get(k)
		if !ok {
			// this really doesn't exist
			return FieldReadResult{}, nil
		}

		// We couldn't fetch the value from a nested data structure, so treat the
		// raw value as an interpolation string. The mraw value is only used
		// for the type switch below.
		mraw = "${INTERPOLATED}"
	}

	result := make(map[string]interface{})
	computed := false
	switch m := mraw.(type) {
	case string:
		// This is a map which has come out of an interpolated variable, so we
		// can just get the value directly from config. Values cannot be computed
		// currently.
		v, _ := r.Config.Get(k)

		// If this isn't a map[string]interface, it must be computed.
		mapV, ok := v.(map[string]interface{})
		if !ok {
			return FieldReadResult{
				Exists:   true,
				Computed: true,
			}, nil
		}

		// Otherwise we can proceed as usual.
		for i, iv := range mapV {
			result[i] = iv
		}
	case []interface{}:
		for i, innerRaw := range m {
			for ik := range innerRaw.(map[string]interface{}) {
				key := fmt.Sprintf("%s.%d.%s", k, i, ik)
				if r.Config.IsComputed(key) {
					computed = true
					break
				}

				v, _ := r.Config.Get(key)
				result[ik] = v
			}
		}
	case []map[string]interface{}:
		for i, innerRaw := range m {
			for ik := range innerRaw {
				key := fmt.Sprintf("%s.%d.%s", k, i, ik)
				if r.Config.IsComputed(key) {
					computed = true
					break
				}

				v, _ := r.Config.Get(key)
				result[ik] = v
			}
		}
	case map[string]interface{}:
		for ik := range m {
			key := fmt.Sprintf("%s.%s", k, ik)
			if r.Config.IsComputed(key) {
				computed = true
				break
			}

			v, _ := r.Config.Get(key)
			result[ik] = v
		}
	default:
		panic(fmt.Sprintf("unknown type: %#v", mraw))
	}

	err := mapValuesToPrimitive(k, result, schema)
	if err != nil {
		return FieldReadResult{}, nil
	}

	var value interface{}
	if !computed {
		value = result
	}

	return FieldReadResult{
		Value:    value,
		Exists:   true,
		Computed: computed,
	}, nil
}

func (r *ConfigFieldReader) readPrimitive(
	k string, schema *Schema) (FieldReadResult, error) {
	raw, ok := r.Config.Get(k)
	if !ok {
		// Nothing in config, but we might still have a default from the schema
		var err error
		raw, err = schema.DefaultValue()
		if err != nil {
			return FieldReadResult{}, fmt.Errorf("%s, error loading default: %s", k, err)
		}

		if raw == nil {
			return FieldReadResult{}, nil
		}
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
	indexMap := make(map[string]int)
	// Create the set that will be our result
	set := schema.ZeroValue().(*Set)

	raw, err := readListField(&nestedConfigFieldReader{r}, address, schema)
	if err != nil {
		return FieldReadResult{}, err
	}
	if !raw.Exists {
		return FieldReadResult{Value: set}, nil
	}

	// If the list is computed, the set is necessarilly computed
	if raw.Computed {
		return FieldReadResult{
			Value:    set,
			Exists:   true,
			Computed: raw.Computed,
		}, nil
	}

	// Build up the set from the list elements
	for i, v := range raw.Value.([]interface{}) {
		// Check if any of the keys in this item are computed
		computed := r.hasComputedSubKeys(
			fmt.Sprintf("%s.%d", strings.Join(address, "."), i), schema)

		code := set.add(v, computed)
		indexMap[code] = i
	}

	r.indexMaps[strings.Join(address, ".")] = indexMap

	return FieldReadResult{
		Value:  set,
		Exists: true,
	}, nil
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
