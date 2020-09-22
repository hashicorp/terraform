/*
 * Copyright 2012-2019 Li Kexian
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * A toolkit for Golang development
 * https://www.likexian.com/
 */

package assert

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalid is value invalid for operation
var ErrInvalid = errors.New("value if invalid")

// ErrLess is expect to be greater error
var ErrLess = errors.New("left is less the right")

// ErrGreater is expect to be less error
var ErrGreater = errors.New("left is greater then right")

// CMP is compare operation
var CMP = struct {
	LT string
	LE string
	GT string
	GE string
}{
	"<",
	"<=",
	">",
	">=",
}

// IsZero returns value is zero value
func IsZero(v interface{}) bool {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Bool:
		return !vv.Bool()
	case reflect.Ptr, reflect.Interface:
		return vv.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return vv.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return vv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return vv.Float() == 0
	default:
		return false
	}
}

// IsContains returns whether value is within array
func IsContains(array interface{}, value interface{}) bool {
	vv := reflect.ValueOf(array)
	if vv.Kind() == reflect.Ptr || vv.Kind() == reflect.Interface {
		if vv.IsNil() {
			return false
		}
		vv = vv.Elem()
	}

	switch vv.Kind() {
	case reflect.Invalid:
		return false
	case reflect.Slice:
		for i := 0; i < vv.Len(); i++ {
			if reflect.DeepEqual(value, vv.Index(i).Interface()) {
				return true
			}
		}
		return false
	case reflect.Map:
		s := vv.MapKeys()
		for i := 0; i < len(s); i++ {
			if reflect.DeepEqual(value, s[i].Interface()) {
				return true
			}
		}
		return false
	case reflect.String:
		ss := reflect.ValueOf(value)
		switch ss.Kind() {
		case reflect.String:
			return strings.Contains(vv.String(), ss.String())
		}
		return false
	default:
		return reflect.DeepEqual(array, value)
	}
}

// IsMatch returns if value v contains any match of pattern r
//   IsMatch(regexp.MustCompile("v\d+"), "v100")
//   IsMatch("v\d+", "v100")
//   IsMatch("\d+\.\d+", 100.1)
func IsMatch(r interface{}, v interface{}) bool {
	var re *regexp.Regexp

	if v, ok := r.(*regexp.Regexp); ok {
		re = v
	} else {
		re = regexp.MustCompile(fmt.Sprint(r))
	}

	return re.MatchString(fmt.Sprint(v))
}

// Length returns length of value
func Length(v interface{}) int {
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Ptr || vv.Kind() == reflect.Interface {
		if vv.IsNil() {
			return 0
		}
		vv = vv.Elem()
	}

	switch vv.Kind() {
	case reflect.Invalid:
		return 0
	case reflect.Ptr, reflect.Interface:
		return 0
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return vv.Len()
	default:
		return len(fmt.Sprintf("%#v", v))
	}
}

// IsLt returns if x less than y, value invalid will returns false
func IsLt(x, y interface{}) bool {
	return Compare(x, y, CMP.LT) == nil
}

// IsLe returns if x less than or equal to y, value invalid will returns false
func IsLe(x, y interface{}) bool {
	return Compare(x, y, CMP.LE) == nil
}

// IsGt returns if x greater than y, value invalid will returns false
func IsGt(x, y interface{}) bool {
	return Compare(x, y, CMP.GT) == nil
}

// IsGe returns if x greater than or equal to y, value invalid will returns false
func IsGe(x, y interface{}) bool {
	return Compare(x, y, CMP.GE) == nil
}

// Compare compare x and y, by operation
// It returns nil for true, ErrInvalid for invalid operation, err for false
//   Compare(1, 2, ">") // number compare -> true
//   Compare("a", "a", ">=") // string compare -> true
//   Compare([]string{"a", "b"}, []string{"a"}, "<") // slice len compare -> false
func Compare(x, y interface{}, op string) error {
	if !IsContains([]string{CMP.LT, CMP.LE, CMP.GT, CMP.GE}, op) {
		return ErrInvalid
	}

	vv := reflect.ValueOf(x)
	if vv.Kind() == reflect.Ptr || vv.Kind() == reflect.Interface {
		if vv.IsNil() {
			return ErrInvalid
		}
		vv = vv.Elem()
	}

	var c float64
	switch vv.Kind() {
	case reflect.Invalid:
		return ErrInvalid
	case reflect.String:
		yy := reflect.ValueOf(y)
		switch yy.Kind() {
		case reflect.String:
			c = float64(strings.Compare(vv.String(), yy.String()))
		default:
			return ErrInvalid
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		yy := reflect.ValueOf(y)
		switch yy.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			c = float64(vv.Len() - yy.Len())
		default:
			return ErrInvalid
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		yy, err := ToInt64(y)
		if err != nil {
			return ErrInvalid
		}
		c = float64(vv.Int() - yy)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		yy, err := ToUint64(y)
		if err != nil {
			return ErrInvalid
		}
		c = float64(vv.Uint()) - float64(yy)
	case reflect.Float32, reflect.Float64:
		yy, err := ToFloat64(y)
		if err != nil {
			return ErrInvalid
		}
		c = float64(vv.Float() - yy)
	default:
		return ErrInvalid
	}

	switch {
	case c < 0:
		switch op {
		case CMP.LT, CMP.LE:
			return nil
		default:
			return ErrLess
		}
	case c > 0:
		switch op {
		case CMP.GT, CMP.GE:
			return nil
		default:
			return ErrGreater
		}
	default:
		switch op {
		case CMP.LT:
			return ErrGreater
		case CMP.GT:
			return ErrLess
		default:
			return nil
		}
	}
}

// ToInt64 returns int value for int or uint or float
func ToInt64(v interface{}) (int64, error) {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int64(vv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int64(vv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return int64(vv.Float()), nil
	case reflect.String:
		r, err := strconv.ParseInt(vv.String(), 10, 64)
		if err != nil {
			return 0, ErrInvalid
		}
		return r, nil
	default:
		return 0, ErrInvalid
	}
}

// ToUint64 returns uint value for int or uint or float
func ToUint64(v interface{}) (uint64, error) {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(vv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uint64(vv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return uint64(vv.Float()), nil
	case reflect.String:
		r, err := strconv.ParseUint(vv.String(), 10, 64)
		if err != nil {
			return 0, ErrInvalid
		}
		return r, nil
	default:
		return 0, ErrInvalid
	}
}

// ToFloat64 returns float64 value for int or uint or float
func ToFloat64(v interface{}) (float64, error) {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(vv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(vv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return float64(vv.Float()), nil
	case reflect.String:
		r, err := strconv.ParseFloat(vv.String(), 64)
		if err != nil {
			return 0, ErrInvalid
		}
		return r, nil
	default:
		return 0, ErrInvalid
	}
}

// If returns x if c is true, else y
//   z = If(c, x, y)
// equal to:
//   z = c ? x : y
func If(c bool, x, y interface{}) interface{} {
	if c {
		return x
	}

	return y
}
