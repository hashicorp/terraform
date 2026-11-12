// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package schema

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"

	"github.com/hashicorp/terraform/internal/legacy/helper/hashcode"
)

// HashResource hashes complex structures that are described using
// a *Resource. This is the default set implementation used when a set's
// element type is a full resource.
func HashResource(resource *Resource) SchemaSetFunc {
	return func(v interface{}) int {
		var buf bytes.Buffer
		SerializeResourceForHash(&buf, v, resource)
		return hashcode.String(buf.String())
	}
}

// HashSchema hashes values that are described using a *Schema. This is the
// default set implementation used when a set's element type is a single
// schema.
func HashSchema(schema *Schema) SchemaSetFunc {
	return func(v interface{}) int {
		var buf bytes.Buffer
		SerializeValueForHash(&buf, v, schema)
		return hashcode.String(buf.String())
	}
}

// Set is a set data structure that is returned for elements of type
// TypeSet.
type Set struct {
	F SchemaSetFunc

	m    map[string]interface{}
	once sync.Once
}

// Add adds an item to the set if it isn't already in the set.
func (s *Set) Add(item interface{}) {
	s.add(item, false)
}

// Len returns the amount of items in the set.
func (s *Set) Len() int {
	return len(s.m)
}

// List returns the elements of this set in slice format.
//
// The order of the returned elements is deterministic. Given the same
// set, the order of this will always be the same.
func (s *Set) List() []interface{} {
	result := make([]interface{}, len(s.m))
	for i, k := range s.listCode() {
		result[i] = s.m[k]
	}

	return result
}

// Difference performs a set difference of the two sets, returning
// a new third set that has only the elements unique to this set.
func (s *Set) Difference(other *Set) *Set {
	result := &Set{F: s.F}
	result.once.Do(result.init)

	for k, v := range s.m {
		if _, ok := other.m[k]; !ok {
			result.m[k] = v
		}
	}

	return result
}

func (s *Set) Equal(raw interface{}) bool {
	other, ok := raw.(*Set)
	if !ok {
		return false
	}

	return reflect.DeepEqual(s.m, other.m)
}

func (s *Set) GoString() string {
	return fmt.Sprintf("*Set(%#v)", s.m)
}

func (s *Set) init() {
	s.m = make(map[string]interface{})
}

func (s *Set) add(item interface{}, computed bool) string {
	s.once.Do(s.init)

	code := s.hash(item)
	if computed {
		code = "~" + code

		if isProto5() {
			tmpCode := code
			count := 0
			for _, exists := s.m[tmpCode]; exists; _, exists = s.m[tmpCode] {
				count++
				tmpCode = fmt.Sprintf("%s%d", code, count)
			}
			code = tmpCode
		}
	}

	if _, ok := s.m[code]; !ok {
		s.m[code] = item
	}

	return code
}

func (s *Set) hash(item interface{}) string {
	code := s.F(item)
	// Always return a nonnegative hashcode.
	if code < 0 {
		code = -code
	}
	return strconv.Itoa(code)
}

func (s *Set) listCode() []string {
	// Sort the hash codes so the order of the list is deterministic
	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))
	return keys
}
