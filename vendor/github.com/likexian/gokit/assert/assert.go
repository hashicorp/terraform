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
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

// Version returns package version
func Version() string {
	return "0.10.1"
}

// Author returns package author
func Author() string {
	return "[Li Kexian](https://www.likexian.com/)"
}

// License returns package license
func License() string {
	return "Licensed under the Apache License 2.0"
}

// Equal assert test value to be equal
func Equal(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, got, exp, 1, args...)
}

// NotEqual assert test value to be not equal
func NotEqual(t *testing.T, got, exp interface{}, args ...interface{}) {
	notEqual(t, got, exp, 1, args...)
}

// Nil assert test value to be nil
func Nil(t *testing.T, got interface{}, args ...interface{}) {
	equal(t, got, nil, 1, args...)
}

// NotNil assert test value to be not nil
func NotNil(t *testing.T, got interface{}, args ...interface{}) {
	notEqual(t, got, nil, 1, args...)
}

// True assert test value to be true
func True(t *testing.T, got interface{}, args ...interface{}) {
	equal(t, got, true, 1, args...)
}

// False assert test value to be false
func False(t *testing.T, got interface{}, args ...interface{}) {
	notEqual(t, got, true, 1, args...)
}

// Zero assert test value to be zero value
func Zero(t *testing.T, got interface{}, args ...interface{}) {
	equal(t, IsZero(got), true, 1, args...)
}

// NotZero assert test value to be not zero value
func NotZero(t *testing.T, got interface{}, args ...interface{}) {
	notEqual(t, IsZero(got), true, 1, args...)
}

// Len assert length of test vaue to be exp
func Len(t *testing.T, got interface{}, exp int, args ...interface{}) {
	equal(t, Length(got), exp, 1, args...)
}

// NotLen assert length of test vaue to be not exp
func NotLen(t *testing.T, got interface{}, exp int, args ...interface{}) {
	notEqual(t, Length(got), exp, 1, args...)
}

// Contains assert test value to be contains
func Contains(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, IsContains(got, exp), true, 1, args...)
}

// NotContains assert test value to be contains
func NotContains(t *testing.T, got, exp interface{}, args ...interface{}) {
	notEqual(t, IsContains(got, exp), true, 1, args...)
}

// Match assert test value match exp pattern
func Match(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, IsMatch(got, exp), true, 1, args...)
}

// NotMatch assert test value not match exp pattern
func NotMatch(t *testing.T, got, exp interface{}, args ...interface{}) {
	notEqual(t, IsMatch(got, exp), true, 1, args...)
}

// Lt assert test value less than exp
func Lt(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, IsLt(got, exp), true, 1, args...)
}

// Le assert test value less than exp or equal
func Le(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, IsLe(got, exp), true, 1, args...)
}

// Gt assert test value greater than exp
func Gt(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, IsGt(got, exp), true, 1, args...)
}

// Ge assert test value greater than exp or equal
func Ge(t *testing.T, got, exp interface{}, args ...interface{}) {
	equal(t, IsGe(got, exp), true, 1, args...)
}

// Panic assert testing to be panic
func Panic(t *testing.T, fn func(), args ...interface{}) {
	defer func() {
		ff := func() {
			t.Error("! -", "assert expected to be panic")
			if len(args) > 0 {
				t.Error("! -", fmt.Sprint(args...))
			}
		}
		ok := recover() != nil
		assert(t, ok, ff, 2)
	}()

	fn()
}

// NotPanic assert testing to be panic
func NotPanic(t *testing.T, fn func(), args ...interface{}) {
	defer func() {
		ff := func() {
			t.Error("! -", "assert expected to be not panic")
			if len(args) > 0 {
				t.Error("! -", fmt.Sprint(args...))
			}
		}
		ok := recover() == nil
		assert(t, ok, ff, 3)
	}()

	fn()
}

func equal(t *testing.T, got, exp interface{}, step int, args ...interface{}) {
	fn := func() {
		switch got.(type) {
		case error:
			t.Errorf("! unexpected error: \"%s\"", got)
		default:
			t.Errorf("! expected %#v, but got %#v", exp, got)
		}
		if len(args) > 0 {
			t.Error("! -", fmt.Sprint(args...))
		}
	}
	ok := reflect.DeepEqual(exp, got)
	assert(t, ok, fn, step+1)
}

func notEqual(t *testing.T, got, exp interface{}, step int, args ...interface{}) {
	fn := func() {
		t.Errorf("! unexpected: %#v", got)
		if len(args) > 0 {
			t.Error("! -", fmt.Sprint(args...))
		}
	}
	ok := !reflect.DeepEqual(exp, got)
	assert(t, ok, fn, step+1)
}

func assert(t *testing.T, pass bool, fn func(), step int) {
	if !pass {
		_, file, line, ok := runtime.Caller(step + 1)
		if ok {
			t.Errorf("%s:%d", file, line)
		}
		fn()
		t.FailNow()
	}
}
