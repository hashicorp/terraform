// Package query implements encoding of structs into http.Header fields.
//
// As a simple example:
//
// 	type Options struct {
// 		ContentType  string `header:"Content-Type"`
// 		Length       int
// 	}
//
// 	opt := Options{"application/json", 2}
// 	h, _ := httpheader.Header(opt)
// 	fmt.Printf("%#v", h)
// 	// will output:
// 	// http.Header{"Content-Type":[]string{"application/json"},"Length":[]string{"2"}}
//
// The exact mapping between Go values and http.Header is described in the
// documentation for the Header() function.
package httpheader

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const tagName = "header"

// Version ...
const Version = "0.2.1"

var timeType = reflect.TypeOf(time.Time{})
var headerType = reflect.TypeOf(http.Header{})

var encoderType = reflect.TypeOf(new(Encoder)).Elem()

// Encoder is an interface implemented by any type that wishes to encode
// itself into Header fields in a non-standard way.
type Encoder interface {
	EncodeHeader(key string, v *http.Header) error
}

// Header returns the http.Header encoding of v.
//
// Header expects to be passed a struct, and traverses it recursively using the
// following encoding rules.
//
// Each exported struct field is encoded as a Header field unless
//
//	- the field's tag is "-", or
//	- the field is empty and its tag specifies the "omitempty" option
//
// The empty values are false, 0, any nil pointer or interface value, any array
// slice, map, or string of length zero, and any time.Time that returns true
// for IsZero().
//
// The Header field name defaults to the struct field name but can be
// specified in the struct field's tag value.  The "header" key in the struct
// field's tag value is the key name, followed by an optional comma and
// options.  For example:
//
// 	// Field is ignored by this package.
// 	Field int `header:"-"`
//
// 	// Field appears as Header field "X-Name".
// 	Field int `header:"X-Name"`
//
// 	// Field appears as Header field "X-Name" and the field is omitted if
// 	// its value is empty
// 	Field int `header:"X-Name,omitempty"`
//
// 	// Field appears as Header field "Field" (the default), but the field
// 	// is skipped if empty.  Note the leading comma.
// 	Field int `header:",omitempty"`
//
// For encoding individual field values, the following type-dependent rules
// apply:
//
// Boolean values default to encoding as the strings "true" or "false".
// Including the "int" option signals that the field should be encoded as the
// strings "1" or "0".
//
// time.Time values default to encoding as RFC1123("Mon, 02 Jan 2006 15:04:05 GMT")
// timestamps. Including the "unix" option signals that the field should be
// encoded as a Unix time (see time.Unix())
//
// Slice and Array values default to encoding as multiple Header values of the
// same name. example:
// X-Name: []string{"Tom", "Jim"}, etc.
//
// http.Header values will be used to extend the Header fields.
//
// Anonymous struct fields are usually encoded as if their inner exported
// fields were fields in the outer struct, subject to the standard Go
// visibility rules. An anonymous struct field with a name given in its Header
// tag is treated as having that name, rather than being anonymous.
//
// Non-nil pointer values are encoded as the value pointed to.
//
// All other values are encoded using their default string representation.
//
// Multiple fields that encode to the same Header filed name will be included
// as multiple Header values of the same name.
func Header(v interface{}) (http.Header, error) {
	h := make(http.Header)
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return h, nil
		}
		val = val.Elem()
	}

	if v == nil {
		return h, nil
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("httpheader: Header() expects struct input. Got %v", val.Kind())
	}

	err := reflectValue(h, val)
	return h, err
}

// reflectValue populates the header fields from the struct fields in val.
// Embedded structs are followed recursively (using the rules defined in the
// Values function documentation) breadth-first.
func reflectValue(header http.Header, val reflect.Value) error {
	var embedded []reflect.Value

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous { // unexported
			continue
		}

		sv := val.Field(i)
		tag := sf.Tag.Get(tagName)
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)
		if name == "" {
			if sf.Anonymous && sv.Kind() == reflect.Struct {
				// save embedded struct for later processing
				embedded = append(embedded, sv)
				continue
			}

			name = sf.Name
		}

		if opts.Contains("omitempty") && isEmptyValue(sv) {
			continue
		}

		if sv.Type().Implements(encoderType) {
			if !reflect.Indirect(sv).IsValid() {
				sv = reflect.New(sv.Type().Elem())
			}

			m := sv.Interface().(Encoder)
			if err := m.EncodeHeader(name, &header); err != nil {
				return err
			}
			continue
		}

		if sv.Kind() == reflect.Slice || sv.Kind() == reflect.Array {
			for i := 0; i < sv.Len(); i++ {
				k := name
				header.Add(k, valueString(sv.Index(i), opts))
			}
			continue
		}

		for sv.Kind() == reflect.Ptr {
			if sv.IsNil() {
				break
			}
			sv = sv.Elem()
		}

		if sv.Type() == timeType {
			header.Add(name, valueString(sv, opts))
			continue
		}
		if sv.Type() == headerType {
			h := sv.Interface().(http.Header)
			for k, vs := range h {
				for _, v := range vs {
					header.Add(k, v)
				}
			}
			continue
		}

		if sv.Kind() == reflect.Struct {
			reflectValue(header, sv)
			continue
		}

		header.Add(name, valueString(sv, opts))
	}

	for _, f := range embedded {
		if err := reflectValue(header, f); err != nil {
			return err
		}
	}

	return nil
}

// valueString returns the string representation of a value.
func valueString(v reflect.Value, opts tagOptions) string {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Bool && opts.Contains("int") {
		if v.Bool() {
			return "1"
		}
		return "0"
	}

	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		if opts.Contains("unix") {
			return strconv.FormatInt(t.Unix(), 10)
		}
		return t.Format(http.TimeFormat)
	}

	return fmt.Sprint(v.Interface())
}

// isEmptyValue checks if a value should be considered empty for the purposes
// of omitting fields with the "omitempty" option.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	if v.Type() == timeType {
		return v.Interface().(time.Time).IsZero()
	}

	return false
}

// tagOptions is the string following a comma in a struct field's "header" tag, or
// the empty string. It does not include the leading comma.
type tagOptions []string

// parseTag splits a struct field's header tag into its name and comma-separated
// options.
func parseTag(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

// Contains checks whether the tagOptions contains the specified option.
func (o tagOptions) Contains(option string) bool {
	for _, s := range o {
		if s == option {
			return true
		}
	}
	return false
}
