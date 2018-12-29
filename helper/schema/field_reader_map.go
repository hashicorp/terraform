package schema

import (
	"fmt"
	"strings"
)

// MapFieldReader reads fields out of an untyped map[string]string to
// the best of its ability.
type MapFieldReader struct {
	Map    MapReader
	Schema map[string]*Schema
}

func (r *MapFieldReader) ReadField(address []string) (FieldReadResult, error) {
	k := strings.Join(address, ".")
	schemaList := addrToSchema(address, r.Schema)
	if len(schemaList) == 0 {
		return FieldReadResult{}, nil
	}

	schema := schemaList[len(schemaList)-1]
	switch schema.Type {
	case TypeBool, TypeInt, TypeFloat, TypeString:
		return r.readPrimitive(address, schema)
	case TypeList:
		return readListField(r, address, schema)
	case TypeMap:
		return r.readMap(k, schema)
	case TypeSet:
		return r.readSet(address, schema)
	case typeObject:
		return readObjectField(r, address, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %s", schema.Type))
	}
}

func (r *MapFieldReader) readMap(k string, schema *Schema) (FieldReadResult, error) {
	result := make(map[string]interface{})
	resultSet := false

	// If the name of the map field is directly in the map with an
	// empty string, it means that the map is being deleted, so mark
	// that is is set.
	if v, ok := r.Map.Access(k); ok && v == "" {
		resultSet = true
	}

	prefix := k + "."
	r.Map.Range(func(k, v string) bool {
		if strings.HasPrefix(k, prefix) {
			resultSet = true

			key := k[len(prefix):]
			if key != "%" && key != "#" {
				result[key] = v
			}
		}

		return true
	})

	err := mapValuesToPrimitive(k, result, schema)
	if err != nil {
		return FieldReadResult{}, nil
	}

	var resultVal interface{}
	if resultSet {
		resultVal = result
	}

	return FieldReadResult{
		Value:  resultVal,
		Exists: resultSet,
	}, nil
}

func (r *MapFieldReader) readPrimitive(
	address []string, schema *Schema) (FieldReadResult, error) {
	k := strings.Join(address, ".")
	result, ok := r.Map.Access(k)
	if !ok {
		return FieldReadResult{}, nil
	}

	returnVal, err := stringToPrimitive(result, false, schema)
	if err != nil {
		return FieldReadResult{}, err
	}

	return FieldReadResult{
		Value:  returnVal,
		Exists: true,
	}, nil
}

func (r *MapFieldReader) readSet(
	address []string, schema *Schema) (FieldReadResult, error) {
	// Get the number of elements in the list
	countRaw, err := r.readPrimitive(
		append(address, "#"), &Schema{Type: TypeInt})
	if err != nil {
		return FieldReadResult{}, err
	}
	if !countRaw.Exists {
		// No count, means we have no list
		countRaw.Value = 0
	}

	// Create the set that will be our result
	set := schema.ZeroValue().(*Set)

	// If we have an empty list, then return an empty list
	if countRaw.Computed || countRaw.Value.(int) == 0 {
		return FieldReadResult{
			Value:    set,
			Exists:   countRaw.Exists,
			Computed: countRaw.Computed,
		}, nil
	}

	// Go through the map and find all the set items
	prefix := strings.Join(address, ".") + "."
	countExpected := countRaw.Value.(int)
	countActual := make(map[string]struct{})
	completed := r.Map.Range(func(k, _ string) bool {
		if !strings.HasPrefix(k, prefix) {
			return true
		}
		if strings.HasPrefix(k, prefix+"#") {
			// Ignore the count field
			return true
		}

		// Split the key, since it might be a sub-object like "idx.field"
		parts := strings.Split(k[len(prefix):], ".")
		idx := parts[0]

		var raw FieldReadResult
		raw, err = r.ReadField(append(address, idx))
		if err != nil {
			return false
		}
		if !raw.Exists {
			// This shouldn't happen because we just verified it does exist
			panic("missing field in set: " + k + "." + idx)
		}

		set.Add(raw.Value)

		// Due to the way multimap readers work, if we've seen the number
		// of fields we expect, then exit so that we don't read later values.
		// For example: the "set" map might have "ports.#", "ports.0", and
		// "ports.1", but the "state" map might have those plus "ports.2".
		// We don't want "ports.2"
		countActual[idx] = struct{}{}
		if len(countActual) >= countExpected {
			return false
		}

		return true
	})
	if !completed && err != nil {
		return FieldReadResult{}, err
	}

	return FieldReadResult{
		Value:  set,
		Exists: true,
	}, nil
}

// MapReader is an interface that is given to MapFieldReader for accessing
// a "map". This can be used to have alternate implementations. For a basic
// map[string]string, use BasicMapReader.
type MapReader interface {
	Access(string) (string, bool)
	Range(func(string, string) bool) bool
}

// BasicMapReader implements MapReader for a single map.
type BasicMapReader map[string]string

func (r BasicMapReader) Access(k string) (string, bool) {
	v, ok := r[k]
	return v, ok
}

func (r BasicMapReader) Range(f func(string, string) bool) bool {
	for k, v := range r {
		if cont := f(k, v); !cont {
			return false
		}
	}

	return true
}

// MultiMapReader reads over multiple maps, preferring keys that are
// founder earlier (lower number index) vs. later (higher number index)
type MultiMapReader []map[string]string

func (r MultiMapReader) Access(k string) (string, bool) {
	for _, m := range r {
		if v, ok := m[k]; ok {
			return v, ok
		}
	}

	return "", false
}

func (r MultiMapReader) Range(f func(string, string) bool) bool {
	done := make(map[string]struct{})
	for _, m := range r {
		for k, v := range m {
			if _, ok := done[k]; ok {
				continue
			}

			if cont := f(k, v); !cont {
				return false
			}

			done[k] = struct{}{}
		}
	}

	return true
}
