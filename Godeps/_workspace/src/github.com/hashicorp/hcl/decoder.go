package hcl

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/hcl"
)

// This is the tag to use with structures to have settings for HCL
const tagName = "hcl"

// Decode reads the given input and decodes it into the structure
// given by `out`.
func Decode(out interface{}, in string) error {
	obj, err := Parse(in)
	if err != nil {
		return err
	}

	return DecodeObject(out, obj)
}

// DecodeObject is a lower-level version of Decode. It decodes a
// raw Object into the given output.
func DecodeObject(out interface{}, n *hcl.Object) error {
	var d decoder
	return d.decode("root", n, reflect.ValueOf(out).Elem())
}

type decoder struct {
	stack []reflect.Kind
}

func (d *decoder) decode(name string, o *hcl.Object, result reflect.Value) error {
	k := result

	// If we have an interface with a valid value, we use that
	// for the check.
	if result.Kind() == reflect.Interface {
		elem := result.Elem()
		if elem.IsValid() {
			k = elem
		}
	}

	// Push current onto stack unless it is an interface.
	if k.Kind() != reflect.Interface {
		d.stack = append(d.stack, k.Kind())

		// Schedule a pop
		defer func() {
			d.stack = d.stack[:len(d.stack)-1]
		}()
	}

	switch k.Kind() {
	case reflect.Bool:
		return d.decodeBool(name, o, result)
	case reflect.Float64:
		return d.decodeFloat(name, o, result)
	case reflect.Int:
		return d.decodeInt(name, o, result)
	case reflect.Interface:
		// When we see an interface, we make our own thing
		return d.decodeInterface(name, o, result)
	case reflect.Map:
		return d.decodeMap(name, o, result)
	case reflect.Ptr:
		return d.decodePtr(name, o, result)
	case reflect.Slice:
		return d.decodeSlice(name, o, result)
	case reflect.String:
		return d.decodeString(name, o, result)
	case reflect.Struct:
		return d.decodeStruct(name, o, result)
	default:
		return fmt.Errorf(
			"%s: unknown kind to decode into: %s", name, k.Kind())
	}

	return nil
}

func (d *decoder) decodeBool(name string, o *hcl.Object, result reflect.Value) error {
	switch o.Type {
	case hcl.ValueTypeBool:
		result.Set(reflect.ValueOf(o.Value.(bool)))
	default:
		return fmt.Errorf("%s: unknown type %s", name, o.Type)
	}

	return nil
}

func (d *decoder) decodeFloat(name string, o *hcl.Object, result reflect.Value) error {
	switch o.Type {
	case hcl.ValueTypeFloat:
		result.Set(reflect.ValueOf(o.Value.(float64)))
	default:
		return fmt.Errorf("%s: unknown type %s", name, o.Type)
	}

	return nil
}

func (d *decoder) decodeInt(name string, o *hcl.Object, result reflect.Value) error {
	switch o.Type {
	case hcl.ValueTypeInt:
		result.Set(reflect.ValueOf(o.Value.(int)))
	default:
		return fmt.Errorf("%s: unknown type %s", name, o.Type)
	}

	return nil
}

func (d *decoder) decodeInterface(name string, o *hcl.Object, result reflect.Value) error {
	var set reflect.Value
	redecode := true

	switch o.Type {
	case hcl.ValueTypeObject:
		// If we're at the root or we're directly within a slice, then we
		// decode objects into map[string]interface{}, otherwise we decode
		// them into lists.
		if len(d.stack) == 0 || d.stack[len(d.stack)-1] == reflect.Slice {
			var temp map[string]interface{}
			tempVal := reflect.ValueOf(temp)
			result := reflect.MakeMap(
				reflect.MapOf(
					reflect.TypeOf(""),
					tempVal.Type().Elem()))

			set = result
		} else {
			var temp []map[string]interface{}
			tempVal := reflect.ValueOf(temp)
			result := reflect.MakeSlice(
				reflect.SliceOf(tempVal.Type().Elem()), 0, int(o.Len()))
			set = result
		}
	case hcl.ValueTypeList:
		var temp []interface{}
		tempVal := reflect.ValueOf(temp)
		result := reflect.MakeSlice(
			reflect.SliceOf(tempVal.Type().Elem()), 0, 0)
		set = result
	case hcl.ValueTypeBool:
		var result bool
		set = reflect.Indirect(reflect.New(reflect.TypeOf(result)))
	case hcl.ValueTypeFloat:
		var result float64
		set = reflect.Indirect(reflect.New(reflect.TypeOf(result)))
	case hcl.ValueTypeInt:
		var result int
		set = reflect.Indirect(reflect.New(reflect.TypeOf(result)))
	case hcl.ValueTypeString:
		set = reflect.Indirect(reflect.New(reflect.TypeOf("")))
	default:
		return fmt.Errorf(
			"%s: cannot decode into interface: %T",
			name, o)
	}

	// Set the result to what its supposed to be, then reset
	// result so we don't reflect into this method anymore.
	result.Set(set)

	if redecode {
		// Revisit the node so that we can use the newly instantiated
		// thing and populate it.
		if err := d.decode(name, o, result); err != nil {
			return err
		}
	}

	return nil
}

func (d *decoder) decodeMap(name string, o *hcl.Object, result reflect.Value) error {
	if o.Type != hcl.ValueTypeObject {
		return fmt.Errorf("%s: not an object type for map (%s)", name, o.Type)
	}

	// If we have an interface, then we can address the interface,
	// but not the slice itself, so get the element but set the interface
	set := result
	if result.Kind() == reflect.Interface {
		result = result.Elem()
	}

	resultType := result.Type()
	resultElemType := resultType.Elem()
	resultKeyType := resultType.Key()
	if resultKeyType.Kind() != reflect.String {
		return fmt.Errorf(
			"%s: map must have string keys", name)
	}

	// Make a map if it is nil
	resultMap := result
	if result.IsNil() {
		resultMap = reflect.MakeMap(
			reflect.MapOf(resultKeyType, resultElemType))
	}

	// Go through each element and decode it.
	for _, o := range o.Elem(false) {
		if o.Value == nil {
			continue
		}

		for _, o := range o.Elem(true) {
			// Make the field name
			fieldName := fmt.Sprintf("%s.%s", name, o.Key)

			// Get the key/value as reflection values
			key := reflect.ValueOf(o.Key)
			val := reflect.Indirect(reflect.New(resultElemType))

			// If we have a pre-existing value in the map, use that
			oldVal := resultMap.MapIndex(key)
			if oldVal.IsValid() {
				val.Set(oldVal)
			}

			// Decode!
			if err := d.decode(fieldName, o, val); err != nil {
				return err
			}

			// Set the value on the map
			resultMap.SetMapIndex(key, val)
		}
	}

	// Set the final map if we can
	set.Set(resultMap)
	return nil
}

func (d *decoder) decodePtr(name string, o *hcl.Object, result reflect.Value) error {
	// Create an element of the concrete (non pointer) type and decode
	// into that. Then set the value of the pointer to this type.
	resultType := result.Type()
	resultElemType := resultType.Elem()
	val := reflect.New(resultElemType)
	if err := d.decode(name, o, reflect.Indirect(val)); err != nil {
		return err
	}

	result.Set(val)
	return nil
}

func (d *decoder) decodeSlice(name string, o *hcl.Object, result reflect.Value) error {
	// If we have an interface, then we can address the interface,
	// but not the slice itself, so get the element but set the interface
	set := result
	if result.Kind() == reflect.Interface {
		result = result.Elem()
	}

	// Create the slice if it isn't nil
	resultType := result.Type()
	resultElemType := resultType.Elem()
	if result.IsNil() {
		resultSliceType := reflect.SliceOf(resultElemType)
		result = reflect.MakeSlice(
			resultSliceType, 0, 0)
	}

	// Determine how we're doing this
	expand := true
	switch o.Type {
	case hcl.ValueTypeObject:
		expand = false
	default:
		// Array or anything else: we expand values and take it all
	}

	i := 0
	for _, o := range o.Elem(expand) {
		fieldName := fmt.Sprintf("%s[%d]", name, i)

		// Decode
		val := reflect.Indirect(reflect.New(resultElemType))
		if err := d.decode(fieldName, o, val); err != nil {
			return err
		}

		// Append it onto the slice
		result = reflect.Append(result, val)

		i += 1
	}

	set.Set(result)
	return nil
}

func (d *decoder) decodeString(name string, o *hcl.Object, result reflect.Value) error {
	switch o.Type {
	case hcl.ValueTypeInt:
		result.Set(reflect.ValueOf(
			strconv.FormatInt(int64(o.Value.(int)), 10)).Convert(result.Type()))
	case hcl.ValueTypeString:
		result.Set(reflect.ValueOf(o.Value.(string)).Convert(result.Type()))
	default:
		return fmt.Errorf("%s: unknown type to string: %s", name, o.Type)
	}

	return nil
}

func (d *decoder) decodeStruct(name string, o *hcl.Object, result reflect.Value) error {
	if o.Type != hcl.ValueTypeObject {
		return fmt.Errorf(
			"%s: not an object type for struct (%s)", name, o.Type)
	}

	// This slice will keep track of all the structs we'll be decoding.
	// There can be more than one struct if there are embedded structs
	// that are squashed.
	structs := make([]reflect.Value, 1, 5)
	structs[0] = result

	// Compile the list of all the fields that we're going to be decoding
	// from all the structs.
	fields := make(map[*reflect.StructField]reflect.Value)
	for len(structs) > 0 {
		structVal := structs[0]
		structs = structs[1:]

		structType := structVal.Type()
		for i := 0; i < structType.NumField(); i++ {
			fieldType := structType.Field(i)

			if fieldType.Anonymous {
				fieldKind := fieldType.Type.Kind()
				if fieldKind != reflect.Struct {
					return fmt.Errorf(
						"%s: unsupported type to struct: %s",
						fieldType.Name, fieldKind)
				}

				// We have an embedded field. We "squash" the fields down
				// if specified in the tag.
				squash := false
				tagParts := strings.Split(fieldType.Tag.Get(tagName), ",")
				for _, tag := range tagParts[1:] {
					if tag == "squash" {
						squash = true
						break
					}
				}

				if squash {
					structs = append(
						structs, result.FieldByName(fieldType.Name))
					continue
				}
			}

			// Normal struct field, store it away
			fields[&fieldType] = structVal.Field(i)
		}
	}

	usedKeys := make(map[string]struct{})
	decodedFields := make([]string, 0, len(fields))
	decodedFieldsVal := make([]reflect.Value, 0)
	unusedKeysVal := make([]reflect.Value, 0)
	for fieldType, field := range fields {
		if !field.IsValid() {
			// This should never happen
			panic("field is not valid")
		}

		// If we can't set the field, then it is unexported or something,
		// and we just continue onwards.
		if !field.CanSet() {
			continue
		}

		fieldName := fieldType.Name

		// This is whether or not we expand the object into its children
		// later.
		expand := false

		tagValue := fieldType.Tag.Get(tagName)
		tagParts := strings.SplitN(tagValue, ",", 2)
		if len(tagParts) >= 2 {
			switch tagParts[1] {
			case "expand":
				expand = true
			case "decodedFields":
				decodedFieldsVal = append(decodedFieldsVal, field)
				continue
			case "key":
				field.SetString(o.Key)
				continue
			case "unusedKeys":
				unusedKeysVal = append(unusedKeysVal, field)
				continue
			}
		}

		if tagParts[0] != "" {
			fieldName = tagParts[0]
		}

		// Find the element matching this name
		obj := o.Get(fieldName, true)
		if obj == nil {
			continue
		}

		// Track the used key
		usedKeys[fieldName] = struct{}{}

		// Create the field name and decode. We range over the elements
		// because we actually want the value.
		fieldName = fmt.Sprintf("%s.%s", name, fieldName)
		for _, obj := range obj.Elem(expand) {
			if err := d.decode(fieldName, obj, field); err != nil {
				return err
			}
		}

		decodedFields = append(decodedFields, fieldType.Name)
	}

	if len(decodedFieldsVal) > 0 {
		// Sort it so that it is deterministic
		sort.Strings(decodedFields)

		for _, v := range decodedFieldsVal {
			v.Set(reflect.ValueOf(decodedFields))
		}
	}

	// If we want to know what keys are unused, compile that
	if len(unusedKeysVal) > 0 {
		/*
			unusedKeys := make([]string, 0, int(obj.Len())-len(usedKeys))

			for _, elem := range obj.Elem {
				k := elem.Key()
				if _, ok := usedKeys[k]; !ok {
					unusedKeys = append(unusedKeys, k)
				}
			}

			if len(unusedKeys) == 0 {
				unusedKeys = nil
			}

			for _, v := range unusedKeysVal {
				v.Set(reflect.ValueOf(unusedKeys))
			}
		*/
	}

	return nil
}
