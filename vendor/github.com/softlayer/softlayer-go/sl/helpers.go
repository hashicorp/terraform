/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package sl has convenience functions for returning pointers to values
package sl

import (
	"reflect"
	"strings"
	"time"

	"github.com/softlayer/softlayer-go/datatypes"
)

// Int returns a pointer to the int value provided
func Int(v int) *int {
	return &v
}

// Uint returns a pointer to the uint value provided
func Uint(v uint) *uint {
	return &v
}

// String returns a pointer to the string value provided
func String(v string) *string {
	return &v
}

// Bool returns a pointer to the bool value provided
func Bool(v bool) *bool {
	return &v
}

// Time converts the time.Time value provided to a datatypes.Time value,
// and returns a pointer to it
func Time(v time.Time) *datatypes.Time {
	r := datatypes.Time{Time: v}
	return &r
}

// Float converts the float value provided to a datatypes.Float64 value,
// and returns a pointer to it
func Float(v float64) *datatypes.Float64 {
	r := datatypes.Float64(v)
	return &r
}

// Convenience functions to simplify dereference of datatype properties

// Get returns the value of p, either p itself, or, if p is a pointer, the
// value that p points to. d is an optional default value to be returned
// in the event that p is nil. If d is not specified, and p is nil, a
// type-appropriate zero-value is returned instead.
func Get(p interface{}, d ...interface{}) interface{} {
	var (
		val interface{}
		ok  bool
	)

	if val, ok = GetOk(p); ok {
		return val
	}

	if len(d) > 0 {
		return d[0]
	}

	return val
}

// GetOk returns the value of p, either p itself, or, if p is a pointer, the
// value that p points to. If d is not specified, and p is nil, a type-
// appropriate zero-value is returned instead. If p is a value or non-nil
// pointer, the second return value will be true.  Otherwise, it will be false
func GetOk(p interface{}) (interface{}, bool) {
	t := reflect.TypeOf(p)

	// if p is a non-pointer, just return it
	if t.Kind() != reflect.Ptr {
		return p, true
	}

	// p is a pointer.  If non-nil, return the value pointed to
	v := reflect.Indirect(reflect.ValueOf(p))
	if v.IsValid() {
		return v.Interface(), true
	}

	// p is a nil pointer.  Return the zero value for the pointed-to type
	return reflect.Zero(t.Elem()).Interface(), false
}

// Grab returns the value specified by the path given,
// starting from the struct s.
// If at any point in the path the lookup falls short
// (i.e. a field is not found), or if the last field in the path is nil
// itself, a type-appropriate zero-value is returned.
// This behavior can be overidden by providing a default value.
//
// This is useful for getting values our of deeply nested structures
// Example: val := sl.Grab(virtualGuest, "Datacenter.Name")
func Grab(s interface{}, path string, d ...interface{}) interface{} {
	var (
		val interface{}
		ok  bool
	)

	if val, ok = GrabOk(s, path); ok {
		return val
	}

	if len(d) > 0 {
		return d[0]
	}

	return val
}

// GrabOk returns the value specified by the path given,
// starting from the struct s.
// If at any point in the path the lookup falls short
// (i.e. a field is not found), or if the last field in the path is nil
// itself, a type-appropriate zero-value is returned.
// It returns a second value, a boolean, which will be false if it failed
// to lookup the value, including if the last field in the path was nil.
//
// This is useful for getting values our of deeply nested structures
// Example: val, ok := sl.GrabOk(virtualGuest, "Datacenter.Name")
func GrabOk(s interface{}, path string) (interface{}, bool) {
	t := reflect.TypeOf(s)
	if t.Kind() != reflect.Struct {
		return nil, false
	}

	dotIndex := strings.Index(path, ".")
	if dotIndex == -1 {
		dotIndex = len(path)
	}

	fieldName := path[0:dotIndex]
	val := reflect.ValueOf(s)
	fieldVal := val.FieldByName(fieldName)
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			return reflect.Zero(fieldVal.Type().Elem()).Interface(), false
		}

		fieldVal = reflect.Indirect(fieldVal)
	}

	result, ok := GetOk(fieldVal.Interface())
	if !ok {
		return result, ok
	}

	if dotIndex == len(path) {
		return result, ok
	}

	return GrabOk(result, path[dotIndex+1:len(path)])
}
