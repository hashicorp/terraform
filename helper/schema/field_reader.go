package schema

import (
	"fmt"
	"strconv"
)

// FieldReaders are responsible for decoding fields out of data into
// the proper typed representation. ResourceData uses this to query data
// out of multiple sources: config, state, diffs, etc.
type FieldReader interface {
	ReadField([]string) (FieldReadResult, error)
}

// FieldReadResult encapsulates all the resulting data from reading
// a field.
type FieldReadResult struct {
	// Value is the actual read value. NegValue is the _negative_ value
	// or the items that should be removed (if they existed). NegValue
	// doesn't make sense for primitives but is important for any
	// container types such as maps, sets, lists.
	Value          interface{}
	ValueProcessed interface{}

	// Exists is true if the field was found in the data. False means
	// it wasn't found if there was no error.
	Exists bool

	// Computed is true if the field was found but the value
	// is computed.
	Computed bool
}

// ValueOrZero returns the value of this result or the zero value of the
// schema type, ensuring a consistent non-nil return value.
func (r *FieldReadResult) ValueOrZero(s *Schema) interface{} {
	if r.Value != nil {
		return r.Value
	}

	return s.ZeroValue()
}

// addrToSchema finds the final element schema for the given address
// and the given schema. It returns all the schemas that led to the final
// schema. These are in order of the address (out to in).
func addrToSchema(addr []string, schemaMap map[string]*Schema) []*Schema {
	current := &Schema{
		Type: typeObject,
		Elem: schemaMap,
	}

	// If we aren't given an address, then the user is requesting the
	// full object, so we return the special value which is the full object.
	if len(addr) == 0 {
		return []*Schema{current}
	}

	result := make([]*Schema, 0, len(addr))
	for len(addr) > 0 {
		k := addr[0]
		addr = addr[1:]

	REPEAT:
		// We want to trim off the first "typeObject" since its not a
		// real lookup that people do. i.e. []string{"foo"} in a structure
		// isn't {typeObject, typeString}, its just a {typeString}.
		if len(result) > 0 || current.Type != typeObject {
			result = append(result, current)
		}

		switch t := current.Type; t {
		case TypeBool, TypeInt, TypeFloat, TypeString:
			if len(addr) > 0 {
				return nil
			}
		case TypeList, TypeSet:
			isIndex := len(addr) > 0 && addr[0] == "#"

			switch v := current.Elem.(type) {
			case *Resource:
				current = &Schema{
					Type: typeObject,
					Elem: v.Schema,
				}
			case *Schema:
				current = v
			case ValueType:
				current = &Schema{Type: v}
			default:
				// we may not know the Elem type and are just looking for the
				// index
				if isIndex {
					break
				}

				if len(addr) == 0 {
					// we've processed the address, so return what we've
					// collected
					return result
				}

				if len(addr) == 1 {
					if _, err := strconv.Atoi(addr[0]); err == nil {
						// we're indexing a value without a schema. This can
						// happen if the list is nested in another schema type.
						// Default to a TypeString like we do with a map
						current = &Schema{Type: TypeString}
						break
					}
				}

				return nil
			}

			// If we only have one more thing and the next thing
			// is a #, then we're accessing the index which is always
			// an int.
			if isIndex {
				current = &Schema{Type: TypeInt}
				break
			}

		case TypeMap:
			if len(addr) > 0 {
				switch v := current.Elem.(type) {
				case ValueType:
					current = &Schema{Type: v}
				default:
					// maps default to string values. This is all we can have
					// if this is nested in another list or map.
					current = &Schema{Type: TypeString}
				}
			}
		case typeObject:
			// If we're already in the object, then we want to handle Sets
			// and Lists specially. Basically, their next key is the lookup
			// key (the set value or the list element). For these scenarios,
			// we just want to skip it and move to the next element if there
			// is one.
			if len(result) > 0 {
				lastType := result[len(result)-2].Type
				if lastType == TypeSet || lastType == TypeList {
					if len(addr) == 0 {
						break
					}

					k = addr[0]
					addr = addr[1:]
				}
			}

			m := current.Elem.(map[string]*Schema)
			val, ok := m[k]
			if !ok {
				return nil
			}

			current = val
			goto REPEAT
		}
	}

	return result
}

// readListField is a generic method for reading a list field out of a
// a FieldReader. It does this based on the assumption that there is a key
// "foo.#" for a list "foo" and that the indexes are "foo.0", "foo.1", etc.
// after that point.
func readListField(
	r FieldReader, addr []string, schema *Schema) (FieldReadResult, error) {
	addrPadded := make([]string, len(addr)+1)
	copy(addrPadded, addr)
	addrPadded[len(addrPadded)-1] = "#"

	// Get the number of elements in the list
	countResult, err := r.ReadField(addrPadded)
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
			Exists:   countResult.Exists,
			Computed: countResult.Computed,
		}, nil
	}

	// Go through each count, and get the item value out of it
	result := make([]interface{}, countResult.Value.(int))
	for i, _ := range result {
		is := strconv.FormatInt(int64(i), 10)
		addrPadded[len(addrPadded)-1] = is
		rawResult, err := r.ReadField(addrPadded)
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
	addr []string,
	schema map[string]*Schema) (FieldReadResult, error) {
	result := make(map[string]interface{})
	exists := false
	for field, s := range schema {
		addrRead := make([]string, len(addr), len(addr)+1)
		copy(addrRead, addr)
		addrRead = append(addrRead, field)
		rawResult, err := r.ReadField(addrRead)
		if err != nil {
			return FieldReadResult{}, err
		}
		if rawResult.Exists {
			exists = true
		}

		result[field] = rawResult.ValueOrZero(s)
	}

	return FieldReadResult{
		Value:  result,
		Exists: exists,
	}, nil
}

// convert map values to the proper primitive type based on schema.Elem
func mapValuesToPrimitive(m map[string]interface{}, schema *Schema) error {

	elemType := TypeString
	if et, ok := schema.Elem.(ValueType); ok {
		elemType = et
	}

	switch elemType {
	case TypeInt, TypeFloat, TypeBool:
		for k, v := range m {
			vs, ok := v.(string)
			if !ok {
				continue
			}

			v, err := stringToPrimitive(vs, false, &Schema{Type: elemType})
			if err != nil {
				return err
			}

			m[k] = v
		}
	}
	return nil
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
		if computed {
			break
		}

		v, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}

		returnVal = v
	case TypeFloat:
		if value == "" {
			returnVal = 0.0
			break
		}
		if computed {
			break
		}

		v, err := strconv.ParseFloat(value, 64)
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
		panic(fmt.Sprintf("Unknown type: %s", schema.Type))
	}

	return returnVal, nil
}
