package schema

import (
	"fmt"
	"strconv"
)

// FieldReaders are responsible for decoding fields out of data into
// the proper typed representation. ResourceData uses this to query data
// out of multiple sources: config, state, diffs, etc.
type FieldReader interface {
	ReadField([]string, *Schema) (FieldReadResult, error)
}

// FieldReadResult encapsulates all the resulting data from reading
// a field.
type FieldReadResult struct {
	// Value is the actual read value. NegValue is the _negative_ value
	// or the items that should be removed (if they existed). NegValue
	// doesn't make sense for primitives but is important for any
	// container types such as maps, sets, lists.
	Value    interface{}
	NegValue interface{}

	// Exists is true if the field was found in the data. False means
	// it wasn't found if there was no error.
	Exists bool

	// Computed is true if the field was found but the value
	// is computed.
	Computed bool
}

// readListField is a generic method for reading a list field out of a
// a FieldReader. It does this based on the assumption that there is a key
// "foo.#" for a list "foo" and that the indexes are "foo.0", "foo.1", etc.
// after that point.
func readListField(
	r FieldReader, k string, schema *Schema) (FieldReadResult, error) {
	// Get the number of elements in the list
	countResult, err := r.ReadField([]string{k + ".#"}, &Schema{Type: TypeInt})
	if err != nil {
		return FieldReadResult{}, err
	}
	if !countResult.Exists {
		// No count, means we have no list
		countResult.Value = 0
	}

	// If we have an empty list, then return an empty list
	if countResult.Computed || countResult.Value.(int) == 0 {
		return FieldReadResult{
			Value:    []interface{}{},
			Exists:   true,
			Computed: countResult.Computed,
		}, nil
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
	result := make([]interface{}, countResult.Value.(int))
	for i, _ := range result {
		is := strconv.FormatInt(int64(i), 10)
		rawResult, err := r.ReadField([]string{k, is}, elemSchema)
		if err != nil {
			return FieldReadResult{}, err
		}
		if !rawResult.Exists {
			// This should never happen, because by the time the data
			// gets to the FieldReaders, all the defaults should be set by
			// Schema.
			rawResult.Value = nil
		}

		result[i] = rawResult.Value
	}

	return FieldReadResult{
		Value:  result,
		Exists: true,
	}, nil
}

// readObjectField is a generic method for reading objects out of FieldReaders
// based on the assumption that building an address of []string{k, FIELD}
// will result in the proper field data.
func readObjectField(
	r FieldReader,
	k string,
	schema map[string]*Schema) (FieldReadResult, error) {
	result := make(map[string]interface{})
	for field, schema := range schema {
		rawResult, err := r.ReadField([]string{k, field}, schema)
		if err != nil {
			return FieldReadResult{}, err
		}
		if !rawResult.Exists {
			continue
		}

		result[field] = rawResult.Value
	}

	return FieldReadResult{
		Value:  result,
		Exists: true,
	}, nil
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
