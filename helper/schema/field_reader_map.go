package schema

import (
	"fmt"
	"strings"
)

// MapFieldReader reads fields out of an untyped map[string]string to
// the best of its ability.
type MapFieldReader struct {
	Map    map[string]string
	Schema map[string]*Schema
}

func (r *MapFieldReader) ReadField(address []string) (FieldReadResult, error) {
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
		return r.readSet(k, schema)
	case typeObject:
		return readObjectField(r, address, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}
}

func (r *MapFieldReader) readMap(k string) (FieldReadResult, error) {
	result := make(map[string]interface{})
	resultSet := false

	prefix := k + "."
	for k, v := range r.Map {
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		result[k[len(prefix):]] = v
		resultSet = true
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
	k string, schema *Schema) (FieldReadResult, error) {
	result, ok := r.Map[k]
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
	k string, schema *Schema) (FieldReadResult, error) {
	// Get the number of elements in the list
	countRaw, err := r.readPrimitive(k+".#", &Schema{Type: TypeInt})
	if err != nil {
		return FieldReadResult{}, err
	}
	if !countRaw.Exists {
		// No count, means we have no list
		countRaw.Value = 0
	}

	// Create the set that will be our result
	set := &Set{F: schema.Set}

	// If we have an empty list, then return an empty list
	if countRaw.Computed || countRaw.Value.(int) == 0 {
		return FieldReadResult{
			Value:    set,
			Exists:   true,
			Computed: countRaw.Computed,
		}, nil
	}

	// Go through the map and find all the set items
	prefix := k + "."
	for k, _ := range r.Map {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if strings.HasPrefix(k, prefix+"#") {
			// Ignore the count field
			continue
		}

		// Split the key, since it might be a sub-object like "idx.field"
		parts := strings.Split(k[len(prefix):], ".")
		idx := parts[0]

		raw, err := r.ReadField([]string{prefix[:len(prefix)-1], idx})
		if err != nil {
			return FieldReadResult{}, err
		}
		if !raw.Exists {
			// This shouldn't happen because we just verified it does exist
			panic("missing field in set: " + k + "." + idx)
		}

		set.Add(raw.Value)
	}

	return FieldReadResult{
		Value:  set,
		Exists: true,
	}, nil
}
