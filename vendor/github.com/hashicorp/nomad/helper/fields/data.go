package fields

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/mapstructure"
)

// FieldData contains the raw data and the schema that the data should adhere to
type FieldData struct {
	Raw    map[string]interface{}
	Schema map[string]*FieldSchema
}

// Validate cycles through the raw data and validates conversions in the schema.
// It also checks for the existence and value of required fields.
func (d *FieldData) Validate() error {
	var result *multierror.Error

	// Scan for missing required fields
	for field, schema := range d.Schema {
		if schema.Required {
			_, ok := d.Raw[field]
			if !ok {
				result = multierror.Append(result, fmt.Errorf(
					"field %q is required", field))
			}
		}
	}

	// Validate field type and value
	for field, value := range d.Raw {
		schema, ok := d.Schema[field]
		if !ok {
			result = multierror.Append(result, fmt.Errorf(
				"%q is an invalid field", field))
			continue
		}

		switch schema.Type {
		case TypeBool, TypeInt, TypeMap, TypeArray, TypeString:
			val, _, err := d.getPrimitive(field, schema)
			if err != nil {
				result = multierror.Append(result, fmt.Errorf(
					"field %q with input %q doesn't seem to be of type %s",
					field, value, schema.Type))
			}
			// Check that we don't have an empty value for required fields
			if schema.Required && val == schema.Type.Zero() {
				result = multierror.Append(result, fmt.Errorf(
					"field %q is required, but no value was found", field))
			}
		default:
			result = multierror.Append(result, fmt.Errorf(
				"unknown field type %s for field %s", schema.Type, field))
		}
	}

	return result.ErrorOrNil()
}

// Get gets the value for the given field. If the key is an invalid field,
// FieldData will panic. If you want a safer version of this method, use
// GetOk. If the field k is not set, the default value (if set) will be
// returned, otherwise the zero value will be returned.
func (d *FieldData) Get(k string) interface{} {
	schema, ok := d.Schema[k]
	if !ok {
		panic(fmt.Sprintf("field %s not in the schema", k))
	}

	value, ok := d.GetOk(k)
	if !ok {
		value = schema.DefaultOrZero()
	}

	return value
}

// GetOk gets the value for the given field. The second return value
// will be false if the key is invalid or the key is not set at all.
func (d *FieldData) GetOk(k string) (interface{}, bool) {
	schema, ok := d.Schema[k]
	if !ok {
		return nil, false
	}

	result, ok, err := d.GetOkErr(k)
	if err != nil {
		panic(fmt.Sprintf("error reading %s: %s", k, err))
	}

	if ok && result == nil {
		result = schema.DefaultOrZero()
	}

	return result, ok
}

// GetOkErr is the most conservative of all the Get methods. It returns
// whether key is set or not, but also an error value. The error value is
// non-nil if the field doesn't exist or there was an error parsing the
// field value.
func (d *FieldData) GetOkErr(k string) (interface{}, bool, error) {
	schema, ok := d.Schema[k]
	if !ok {
		return nil, false, fmt.Errorf("unknown field: %s", k)
	}

	switch schema.Type {
	case TypeBool, TypeInt, TypeMap, TypeArray, TypeString:
		return d.getPrimitive(k, schema)
	default:
		return nil, false,
			fmt.Errorf("unknown field type %s for field %s", schema.Type, k)
	}
}

// getPrimitive tries to convert the raw value of a field to its data type as
// defined in the schema. It does strict type checking, so the value will need
// to be able to convert to the appropriate type directly.
func (d *FieldData) getPrimitive(
	k string, schema *FieldSchema) (interface{}, bool, error) {
	raw, ok := d.Raw[k]
	if !ok {
		return nil, false, nil
	}

	switch schema.Type {
	case TypeBool:
		var result bool
		if err := mapstructure.Decode(raw, &result); err != nil {
			return nil, true, err
		}
		return result, true, nil

	case TypeInt:
		var result int
		if err := mapstructure.Decode(raw, &result); err != nil {
			return nil, true, err
		}
		return result, true, nil

	case TypeString:
		var result string
		if err := mapstructure.Decode(raw, &result); err != nil {
			return nil, true, err
		}
		return result, true, nil

	case TypeMap:
		var result map[string]interface{}
		if err := mapstructure.Decode(raw, &result); err != nil {
			return nil, true, err
		}
		return result, true, nil

	case TypeArray:
		var result []interface{}
		if err := mapstructure.Decode(raw, &result); err != nil {
			return nil, true, err
		}
		return result, true, nil

	default:
		panic(fmt.Sprintf("Unknown type: %s", schema.Type))
	}
}
