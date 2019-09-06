package schema

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

func SerializeValueForHash(buf *bytes.Buffer, val interface{}, schema *Schema) {
	if val == nil {
		buf.WriteRune(';')
		return
	}

	switch schema.Type {
	case TypeBool:
		if val.(bool) {
			buf.WriteRune('1')
		} else {
			buf.WriteRune('0')
		}
	case TypeInt:
		buf.WriteString(strconv.Itoa(val.(int)))
	case TypeFloat:
		buf.WriteString(strconv.FormatFloat(val.(float64), 'g', -1, 64))
	case TypeString:
		buf.WriteString(val.(string))
	case TypeList:
		buf.WriteRune('(')
		l := val.([]interface{})
		for _, innerVal := range l {
			serializeCollectionMemberForHash(buf, innerVal, schema.Elem)
		}
		buf.WriteRune(')')
	case TypeMap:

		m := val.(map[string]interface{})
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteRune('[')
		for _, k := range keys {
			innerVal := m[k]
			if innerVal == nil {
				continue
			}
			buf.WriteString(k)
			buf.WriteRune(':')

			switch innerVal := innerVal.(type) {
			case int:
				buf.WriteString(strconv.Itoa(innerVal))
			case float64:
				buf.WriteString(strconv.FormatFloat(innerVal, 'g', -1, 64))
			case string:
				buf.WriteString(innerVal)
			default:
				panic(fmt.Sprintf("unknown value type in TypeMap %T", innerVal))
			}

			buf.WriteRune(';')
		}
		buf.WriteRune(']')
	case TypeSet:
		buf.WriteRune('{')
		s := val.(*Set)
		for _, innerVal := range s.List() {
			serializeCollectionMemberForHash(buf, innerVal, schema.Elem)
		}
		buf.WriteRune('}')
	default:
		panic("unknown schema type to serialize")
	}
	buf.WriteRune(';')
}

// SerializeValueForHash appends a serialization of the given resource config
// to the given buffer, guaranteeing deterministic results given the same value
// and schema.
//
// Its primary purpose is as input into a hashing function in order
// to hash complex substructures when used in sets, and so the serialization
// is not reversible.
func SerializeResourceForHash(buf *bytes.Buffer, val interface{}, resource *Resource) {
	if val == nil {
		return
	}
	sm := resource.Schema
	m := val.(map[string]interface{})
	var keys []string
	allComputed := true
	for k, v := range sm {
		if v.Optional || v.Required {
			allComputed = false
		}

		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		innerSchema := sm[k]
		// Skip attributes that are not user-provided. Computed attributes
		// do not contribute to the hash since their ultimate value cannot
		// be known at plan/diff time.
		if !allComputed && !(innerSchema.Required || innerSchema.Optional) {
			continue
		}

		buf.WriteString(k)
		buf.WriteRune(':')
		innerVal := m[k]
		SerializeValueForHash(buf, innerVal, innerSchema)
	}
}

func serializeCollectionMemberForHash(buf *bytes.Buffer, val interface{}, elem interface{}) {
	switch tElem := elem.(type) {
	case *Schema:
		SerializeValueForHash(buf, val, tElem)
	case *Resource:
		buf.WriteRune('<')
		SerializeResourceForHash(buf, val, tElem)
		buf.WriteString(">;")
	default:
		panic(fmt.Sprintf("invalid element type: %T", tElem))
	}
}
