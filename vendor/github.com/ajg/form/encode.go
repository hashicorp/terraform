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

// NewEncoder returns a new form encoder.
func NewEncoder(w io.Writer) *encoder {
	return &encoder{w}
}

// encoder provides a way to encode to a Writer.
type encoder struct {
	w io.Writer
}

// Encode encodes dst as form and writes it out using the encoder's Writer.
func (e encoder) Encode(dst interface{}) error {
	v := reflect.ValueOf(dst)
	n, err := encodeToNode(v)
	if err != nil {
		return err
	}
	s := n.Values().Encode()
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
	n, err := encodeToNode(v)
	if err != nil {
		return "", err
	}
	return n.Values().Encode(), nil
}

// EncodeToValues encodes dst as a form and returns it as Values.
func EncodeToValues(dst interface{}) (url.Values, error) {
	v := reflect.ValueOf(dst)
	n, err := encodeToNode(v)
	if err != nil {
		return nil, err
	}
	return n.Values(), nil
}

func encodeToNode(v reflect.Value) (n node, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	return getNode(encodeValue(v)), nil
}

func encodeValue(v reflect.Value) interface{} {
	t := v.Type()
	k := v.Kind()

	if s, ok := marshalValue(v); ok {
		return s
	} else if isEmptyValue(v) {
		return "" // Treat the zero value as the empty string.
	}

	switch k {
	case reflect.Ptr, reflect.Interface:
		return encodeValue(v.Elem())
	case reflect.Struct:
		if t.ConvertibleTo(timeType) {
			return encodeTime(v)
		} else if t.ConvertibleTo(urlType) {
			return encodeURL(v)
		}
		return encodeStruct(v)
	case reflect.Slice:
		return encodeSlice(v)
	case reflect.Array:
		return encodeArray(v)
	case reflect.Map:
		return encodeMap(v)
	case reflect.Invalid, reflect.Uintptr, reflect.UnsafePointer, reflect.Chan, reflect.Func:
		panic(t.String() + " has unsupported kind " + t.Kind().String())
	default:
		return encodeBasic(v)
	}
}

func encodeStruct(v reflect.Value) interface{} {
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
			n[k] = encodeValue(fv)
		}
	}
	return n
}

func encodeMap(v reflect.Value) interface{} {
	n := node{}
	for _, i := range v.MapKeys() {
		k := getString(encodeValue(i))
		n[k] = encodeValue(v.MapIndex(i))
	}
	return n
}

func encodeArray(v reflect.Value) interface{} {
	n := node{}
	for i := 0; i < v.Len(); i++ {
		n[strconv.Itoa(i)] = encodeValue(v.Index(i))
	}
	return n
}

func encodeSlice(v reflect.Value) interface{} {
	t := v.Type()
	if t.Elem().Kind() == reflect.Uint8 {
		return string(v.Bytes()) // Encode byte slices as a single string by default.
	}
	n := node{}
	for i := 0; i < v.Len(); i++ {
		n[strconv.Itoa(i)] = encodeValue(v.Index(i))
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

func findField(v reflect.Value, n string) (reflect.Value, bool) {
	t := v.Type()
	l := v.NumField()
	// First try named fields.
	for i := 0; i < l; i++ {
		f := t.Field(i)
		k, _ := fieldInfo(f)
		if k == omittedKey {
			continue
		} else if n == k {
			return v.Field(i), true
		}
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
		if ev, ok := findField(fv, n); ok {
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
