// Copyright 2014 Alvaro J. Genial. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package form

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// NewEncoder returns a new form Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w, defaultDelimiter, defaultEscape, false}
}

// Encoder provides a way to encode to a Writer.
type Encoder struct {
	w io.Writer
	d rune
	e rune
	z bool
}

// DelimitWith sets r as the delimiter used for composite keys by Encoder e and returns the latter; it is '.' by default.
func (e *Encoder) DelimitWith(r rune) *Encoder {
	e.d = r
	return e
}

// EscapeWith sets r as the escape used for delimiters (and to escape itself) by Encoder e and returns the latter; it is '\\' by default.
func (e *Encoder) EscapeWith(r rune) *Encoder {
	e.e = r
	return e
}

// KeepZeros sets whether Encoder e should keep zero (default) values in their literal form when encoding, and returns the former; by default zero values are not kept, but are rather encoded as the empty string.
func (e *Encoder) KeepZeros(z bool) *Encoder {
	e.z = z
	return e
}

// Encode encodes dst as form and writes it out using the Encoder's Writer.
func (e Encoder) Encode(dst interface{}) error {
	v := reflect.ValueOf(dst)
	n, err := encodeToNode(v, e.z)
	if err != nil {
		return err
	}
	s := n.values(e.d, e.e).Encode()
	l, err := io.WriteString(e.w, s)
	switch {
	case err != nil:
		return err
	case l != len(s):
		return errors.New("could not write data completely")
	}
	return nil
}

// EncodeToString encodes dst as a form and returns it as a string.
func EncodeToString(dst interface{}) (string, error) {
	v := reflect.ValueOf(dst)
	n, err := encodeToNode(v, false)
	if err != nil {
		return "", err
	}
	vs := n.values(defaultDelimiter, defaultEscape)
	return vs.Encode(), nil
}

// EncodeToValues encodes dst as a form and returns it as Values.
func EncodeToValues(dst interface{}) (url.Values, error) {
	v := reflect.ValueOf(dst)
	n, err := encodeToNode(v, false)
	if err != nil {
		return nil, err
	}
	vs := n.values(defaultDelimiter, defaultEscape)
	return vs, nil
}

func encodeToNode(v reflect.Value, z bool) (n node, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	return getNode(encodeValue(v, z)), nil
}

func encodeValue(v reflect.Value, z bool) interface{} {
	t := v.Type()
	k := v.Kind()

	if s, ok := marshalValue(v); ok {
		return s
	} else if !z && isEmptyValue(v) {
		return "" // Treat the zero value as the empty string.
	}

	switch k {
	case reflect.Ptr, reflect.Interface:
		return encodeValue(v.Elem(), z)
	case reflect.Struct:
		if t.ConvertibleTo(timeType) {
			return encodeTime(v)
		} else if t.ConvertibleTo(urlType) {
			return encodeURL(v)
		}
		return encodeStruct(v, z)
	case reflect.Slice:
		return encodeSlice(v, z)
	case reflect.Array:
		return encodeArray(v, z)
	case reflect.Map:
		return encodeMap(v, z)
	case reflect.Invalid, reflect.Uintptr, reflect.UnsafePointer, reflect.Chan, reflect.Func:
		panic(t.String() + " has unsupported kind " + t.Kind().String())
	default:
		return encodeBasic(v)
	}
}

func encodeStruct(v reflect.Value, z bool) interface{} {
	t := v.Type()
	n := node{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		k, oe := fieldInfo(f)

		if k == "-" {
			continue
		} else if fv := v.Field(i); oe && isEmptyValue(fv) {
			delete(n, k)
		} else {
			n[k] = encodeValue(fv, z)
		}
	}
	return n
}

func encodeMap(v reflect.Value, z bool) interface{} {
	n := node{}
	for _, i := range v.MapKeys() {
		k := getString(encodeValue(i, z))
		n[k] = encodeValue(v.MapIndex(i), z)
	}
	return n
}

func encodeArray(v reflect.Value, z bool) interface{} {
	n := node{}
	for i := 0; i < v.Len(); i++ {
		n[strconv.Itoa(i)] = encodeValue(v.Index(i), z)
	}
	return n
}

func encodeSlice(v reflect.Value, z bool) interface{} {
	t := v.Type()
	if t.Elem().Kind() == reflect.Uint8 {
		return string(v.Bytes()) // Encode byte slices as a single string by default.
	}
	n := node{}
	for i := 0; i < v.Len(); i++ {
		n[strconv.Itoa(i)] = encodeValue(v.Index(i), z)
	}
	return n
}

func encodeTime(v reflect.Value) string {
	t := v.Convert(timeType).Interface().(time.Time)
	if t.Year() == 0 && (t.Month() == 0 || t.Month() == 1) && (t.Day() == 0 || t.Day() == 1) {
		return t.Format("15:04:05.999999999Z07:00")
	} else if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02T15:04:05.999999999Z07:00")
}

func encodeURL(v reflect.Value) string {
	u := v.Convert(urlType).Interface().(url.URL)
	return u.String()
}

func encodeBasic(v reflect.Value) string {
	t := v.Type()
	switch k := t.Kind(); k {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Complex64, reflect.Complex128:
		s := fmt.Sprintf("%g", v.Complex())
		return strings.TrimSuffix(strings.TrimPrefix(s, "("), ")")
	case reflect.String:
		return v.String()
	}
	panic(t.String() + " has unsupported kind " + t.Kind().String())
}

func isEmptyValue(v reflect.Value) bool {
	switch t := v.Type(); v.Kind() {
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
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		if t.ConvertibleTo(timeType) {
			return v.Convert(timeType).Interface().(time.Time).IsZero()
		}
		return reflect.DeepEqual(v, reflect.Zero(t))
	}
	return false
}

// canIndexOrdinally returns whether a value contains an ordered sequence of elements.
func canIndexOrdinally(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	switch t := v.Type(); t.Kind() {
	case reflect.Ptr, reflect.Interface:
		return canIndexOrdinally(v.Elem())
	case reflect.Slice, reflect.Array:
		return true
	}
	return false
}

func fieldInfo(f reflect.StructField) (k string, oe bool) {
	if f.PkgPath != "" { // Skip private fields.
		return omittedKey, oe
	}

	k = f.Name
	tag := f.Tag.Get("form")
	if tag == "" {
		return k, oe
	}

	ps := strings.SplitN(tag, ",", 2)
	if ps[0] != "" {
		k = ps[0]
	}
	if len(ps) == 2 {
		oe = ps[1] == "omitempty"
	}
	return k, oe
}

func findField(v reflect.Value, n string, ignoreCase bool) (reflect.Value, bool) {
	t := v.Type()
	l := v.NumField()

	var lowerN string
	caseInsensitiveMatch := -1
	if ignoreCase {
		lowerN = strings.ToLower(n)
	}

	// First try named fields.
	for i := 0; i < l; i++ {
		f := t.Field(i)
		k, _ := fieldInfo(f)
		if k == omittedKey {
			continue
		} else if n == k {
			return v.Field(i), true
		} else if ignoreCase && lowerN == strings.ToLower(k) {
			caseInsensitiveMatch = i
		}
	}

	// If no exact match was found try case insensitive match.
	if caseInsensitiveMatch != -1 {
		return v.Field(caseInsensitiveMatch), true
	}

	// Then try anonymous (embedded) fields.
	for i := 0; i < l; i++ {
		f := t.Field(i)
		k, _ := fieldInfo(f)
		if k == omittedKey || !f.Anonymous { // || k != "" ?
			continue
		}
		fv := v.Field(i)
		fk := fv.Kind()
		for fk == reflect.Ptr || fk == reflect.Interface {
			fv = fv.Elem()
			fk = fv.Kind()
		}

		if fk != reflect.Struct {
			continue
		}
		if ev, ok := findField(fv, n, ignoreCase); ok {
			return ev, true
		}
	}

	return reflect.Value{}, false
}

var (
	stringType    = reflect.TypeOf(string(""))
	stringMapType = reflect.TypeOf(map[string]interface{}{})
	timeType      = reflect.TypeOf(time.Time{})
	timePtrType   = reflect.TypeOf(&time.Time{})
	urlType       = reflect.TypeOf(url.URL{})
)

func skipTextMarshalling(t reflect.Type) bool {
	/*// Skip time.Time because its text unmarshaling is overly rigid:
	return t == timeType || t == timePtrType*/
	// Skip time.Time & convertibles because its text unmarshaling is overly rigid:
	return t.ConvertibleTo(timeType) || t.ConvertibleTo(timePtrType)
}

func unmarshalValue(v reflect.Value, x interface{}) bool {
	if skipTextMarshalling(v.Type()) {
		return false
	}

	tu, ok := v.Interface().(encoding.TextUnmarshaler)
	if !ok && !v.CanAddr() {
		return false
	} else if !ok {
		return unmarshalValue(v.Addr(), x)
	}

	s := getString(x)
	if err := tu.UnmarshalText([]byte(s)); err != nil {
		panic(err)
	}
	return true
}

func marshalValue(v reflect.Value) (string, bool) {
	if skipTextMarshalling(v.Type()) {
		return "", false
	}

	tm, ok := v.Interface().(encoding.TextMarshaler)
	if !ok && !v.CanAddr() {
		return "", false
	} else if !ok {
		return marshalValue(v.Addr())
	}

	bs, err := tm.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(bs), true
}
