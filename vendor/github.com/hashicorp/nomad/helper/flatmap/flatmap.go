package flatmap

import (
	"fmt"
	"reflect"
)

// Flatten takes an object and returns a flat map of the object. The keys of the
// map is the path of the field names until a primitive field is reached and the
// value is a string representation of the terminal field.
func Flatten(obj interface{}, filter []string, primitiveOnly bool) map[string]string {
	flat := make(map[string]string)
	v := reflect.ValueOf(obj)
	if !v.IsValid() {
		return nil
	}

	flatten("", v, primitiveOnly, false, flat)
	for _, f := range filter {
		if _, ok := flat[f]; ok {
			delete(flat, f)
		}
	}
	return flat
}

// flatten recursively calls itself to create a flatmap representation of the
// passed value. The results are stored into the output map and the keys are
// the fields prepended with the passed prefix.
// XXX: A current restriction is that maps only support string keys.
func flatten(prefix string, v reflect.Value, primitiveOnly, enteredStruct bool, output map[string]string) {
	switch v.Kind() {
	case reflect.Bool:
		output[prefix] = fmt.Sprintf("%v", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		output[prefix] = fmt.Sprintf("%v", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		output[prefix] = fmt.Sprintf("%v", v.Uint())
	case reflect.Float32, reflect.Float64:
		output[prefix] = fmt.Sprintf("%v", v.Float())
	case reflect.Complex64, reflect.Complex128:
		output[prefix] = fmt.Sprintf("%v", v.Complex())
	case reflect.String:
		output[prefix] = fmt.Sprintf("%v", v.String())
	case reflect.Invalid:
		output[prefix] = "nil"
	case reflect.Ptr:
		if primitiveOnly && enteredStruct {
			return
		}

		e := v.Elem()
		if !e.IsValid() {
			output[prefix] = "nil"
		}
		flatten(prefix, e, primitiveOnly, enteredStruct, output)
	case reflect.Map:
		for _, k := range v.MapKeys() {
			if k.Kind() == reflect.Interface {
				k = k.Elem()
			}

			if k.Kind() != reflect.String {
				panic(fmt.Sprintf("%q: map key is not string: %s", prefix, k))
			}

			flatten(getSubKeyPrefix(prefix, k.String()), v.MapIndex(k), primitiveOnly, enteredStruct, output)
		}
	case reflect.Struct:
		if primitiveOnly && enteredStruct {
			return
		}
		enteredStruct = true

		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			name := t.Field(i).Name
			val := v.Field(i)
			if val.Kind() == reflect.Interface && !val.IsNil() {
				val = val.Elem()
			}

			flatten(getSubPrefix(prefix, name), val, primitiveOnly, enteredStruct, output)
		}
	case reflect.Interface:
		if primitiveOnly {
			return
		}

		e := v.Elem()
		if !e.IsValid() {
			output[prefix] = "nil"
			return
		}
		flatten(prefix, e, primitiveOnly, enteredStruct, output)
	case reflect.Array, reflect.Slice:
		if primitiveOnly {
			return
		}

		if v.Kind() == reflect.Slice && v.IsNil() {
			output[prefix] = "nil"
			return
		}
		for i := 0; i < v.Len(); i++ {
			flatten(fmt.Sprintf("%s[%d]", prefix, i), v.Index(i), primitiveOnly, enteredStruct, output)
		}
	default:
		panic(fmt.Sprintf("prefix %q; unsupported type %v", prefix, v.Kind()))
	}
}

// getSubPrefix takes the current prefix and the next subfield and returns an
// appropriate prefix.
func getSubPrefix(curPrefix, subField string) string {
	newPrefix := ""
	if curPrefix != "" {
		newPrefix = fmt.Sprintf("%s.%s", curPrefix, subField)
	} else {
		newPrefix = fmt.Sprintf("%s", subField)
	}
	return newPrefix
}

// getSubKeyPrefix takes the current prefix and the next subfield and returns an
// appropriate prefix for a map field.
func getSubKeyPrefix(curPrefix, subField string) string {
	newPrefix := ""
	if curPrefix != "" {
		newPrefix = fmt.Sprintf("%s[%s]", curPrefix, subField)
	} else {
		newPrefix = fmt.Sprintf("%s", subField)
	}
	return newPrefix
}
