package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type encodeFunc func(reflect.Value) ([]byte, error)

func marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte{}, nil
	}

	val := reflect.ValueOf(v)
	return encodeValue(val)
}

func encodeValue(val reflect.Value) ([]byte, error) {
	var b []byte
	var err error

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return []byte("<value/>"), nil
		}

		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		switch val.Interface().(type) {
		case time.Time:
			t := val.Interface().(time.Time)
			b = []byte(fmt.Sprintf("<dateTime.iso8601>%s</dateTime.iso8601>", t.Format(iso8601)))
		default:
			b, err = encodeStruct(val)
		}
	case reflect.Map:
		b, err = encodeMap(val)
	case reflect.Slice:
		b, err = encodeSlice(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b = []byte(fmt.Sprintf("<int>%s</int>", strconv.FormatInt(val.Int(), 10)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		b = []byte(fmt.Sprintf("<i4>%s</i4>", strconv.FormatUint(val.Uint(), 10)))
	case reflect.Float32, reflect.Float64:
		b = []byte(fmt.Sprintf("<double>%s</double>",
			strconv.FormatFloat(val.Float(), 'g', -1, val.Type().Bits())))
	case reflect.Bool:
		if val.Bool() {
			b = []byte("<boolean>1</boolean>")
		} else {
			b = []byte("<boolean>0</boolean>")
		}
	case reflect.String:
		var buf bytes.Buffer

		xml.Escape(&buf, []byte(val.String()))

		if _, ok := val.Interface().(Base64); ok {
			b = []byte(fmt.Sprintf("<base64>%s</base64>", buf.String()))
		} else {
			b = []byte(fmt.Sprintf("<string>%s</string>", buf.String()))
		}
	default:
		return nil, fmt.Errorf("xmlrpc encode error: unsupported type")
	}

	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("<value>%s</value>", string(b))), nil
}

func encodeStruct(value reflect.Value) ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("<struct>")

	vals := []reflect.Value{value}
	for j := 0; j < len(vals); j++ {
		val := vals[j]
		t := val.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("xmlrpc")
			name := f.Name
			fieldVal := val.FieldByName(f.Name)
			fieldValKind := fieldVal.Kind()

			// Omit unexported fields
			if !fieldVal.CanInterface() {
				continue
			}

			// Omit fields who are structs that contain no fields themselves
			if fieldValKind == reflect.Struct && fieldVal.NumField() == 0 {
				continue
			}

			// Omit empty slices
			if fieldValKind == reflect.Slice && fieldVal.Len() == 0 {
				continue
			}

			// Omit empty fields (defined as nil pointers)
			if tag != "" {
				parts := strings.Split(tag, ",")
				name = parts[0]
				if len(parts) > 1 && parts[1] == "omitempty" {
					if fieldValKind == reflect.Ptr && fieldVal.IsNil() {
						continue
					}
				}
			}

			// Drill down into anonymous/embedded structs and do not expose the
			// containing embedded struct in request.
			// This will effectively pull up fields in embedded structs to look
			// as part of the original struct in the request.
			if f.Anonymous {
				vals = append(vals, fieldVal)
				continue
			}

			b.WriteString("<member>")
			b.WriteString(fmt.Sprintf("<name>%s</name>", name))

			p, err := encodeValue(fieldVal)
			if err != nil {
				return nil, err
			}
			b.Write(p)

			b.WriteString("</member>")
		}
	}

	b.WriteString("</struct>")

	return b.Bytes(), nil
}

func encodeMap(val reflect.Value) ([]byte, error) {
	var t = val.Type()

	if t.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("xmlrpc encode error: only maps with string keys are supported")
	}

	var b bytes.Buffer

	b.WriteString("<struct>")

	keys := val.MapKeys()

	for i := 0; i < val.Len(); i++ {
		key := keys[i]
		kval := val.MapIndex(key)

		b.WriteString("<member>")
		b.WriteString(fmt.Sprintf("<name>%s</name>", key.String()))

		p, err := encodeValue(kval)

		if err != nil {
			return nil, err
		}

		b.Write(p)
		b.WriteString("</member>")
	}

	b.WriteString("</struct>")

	return b.Bytes(), nil
}

func encodeSlice(val reflect.Value) ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("<array><data>")

	for i := 0; i < val.Len(); i++ {
		p, err := encodeValue(val.Index(i))
		if err != nil {
			return nil, err
		}

		b.Write(p)
	}

	b.WriteString("</data></array>")

	return b.Bytes(), nil
}
