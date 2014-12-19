package schema

import (
	"fmt"
	"strconv"
)

// FieldReaders are responsible for decoding fields out of data into
// the proper typed representation. ResourceData uses this to query data
// out of multiple sources: config, state, diffs, etc.
type FieldReader interface {
	ReadField([]string, *Schema) (interface{}, bool, bool, error)
}

// readListField is a generic method for reading a list field out of a
// a FieldReader. It does this based on the assumption that there is a key
// "foo.#" for a list "foo" and that the indexes are "foo.0", "foo.1", etc.
// after that point.
func readListField(
	r FieldReader, k string, schema *Schema) (interface{}, bool, bool, error) {
	// Get the number of elements in the list
	countRaw, countOk, countComputed, err := r.ReadField(
		[]string{k + ".#"}, &Schema{Type: TypeInt})
	if err != nil {
		return nil, false, false, err
	}
	if !countOk {
		// No count, means we have no list
		countRaw = 0
	}

	// If we have an empty list, then return an empty list
	if countComputed || countRaw.(int) == 0 {
		return []interface{}{}, true, countComputed, nil
	}

	// Get the schema for the elements
	var elemSchema *Schema
	switch t := schema.Elem.(type) {
	case *Resource:
		elemSchema = &Schema{
			Type: typeObject,
			Elem: t.Schema,
		}
	case *Schema:
		elemSchema = t
	}

	// Go through each count, and get the item value out of it
	result := make([]interface{}, countRaw.(int))
	for i, _ := range result {
		is := strconv.FormatInt(int64(i), 10)
		raw, ok, _, err := r.ReadField([]string{k, is}, elemSchema)
		if err != nil {
			return nil, false, false, err
		}
		if !ok {
			// This should never happen, because by the time the data
			// gets to the FieldReaders, all the defaults should be set by
			// Schema.
			raw = nil
		}

		result[i] = raw
	}

	return result, true, false, nil
}

// readObjectField is a generic method for reading objects out of FieldReaders
// based on the assumption that building an address of []string{k, FIELD}
// will result in the proper field data.
func readObjectField(
	r FieldReader,
	k string,
	schema map[string]*Schema) (interface{}, bool, bool, error) {
	result := make(map[string]interface{})
	for field, schema := range schema {
		v, ok, _, err := r.ReadField([]string{k, field}, schema)
		if err != nil {
			return nil, false, false, err
		}
		if !ok {
			continue
		}

		result[field] = v
	}

	return result, true, false, nil
}

func stringToPrimitive(
	value string, computed bool, schema *Schema) (interface{}, error) {
	var returnVal interface{}
	switch schema.Type {
	case TypeBool:
		if value == "" {
			returnVal = false
			break
		}

		v, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}

		returnVal = v
	case TypeInt:
		if value == "" {
			returnVal = 0
			break
		}
		if computed {
			break
		}

		v, err := strconv.ParseInt(value, 0, 0)
		if err != nil {
			return nil, err
		}

		returnVal = int(v)
	case TypeString:
		returnVal = value
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}

	return returnVal, nil
}
