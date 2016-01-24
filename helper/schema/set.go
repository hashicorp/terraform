package schema

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"

	"github.com/hashicorp/terraform/helper/hashcode"
)

// HashString hashes strings. If you want a Set of strings, this is the
// SchemaSetFunc you want.
func HashString(v interface{}) int {
	return hashcode.String(v.(string))
}

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

// NewSet is a convenience method for creating a new set with the given
// items.
func NewSet(f SchemaSetFunc, items []interface{}) *Set {
	s := &Set{F: f}
	for _, i := range items {
		s.Add(i)
	}

	return s
}

// CopySet returns a copy of another set.
func CopySet(otherSet *Set) *Set {
	return NewSet(otherSet.F, otherSet.List())
}

// Add adds an item to the set if it isn't already in the set.
func (s *Set) Add(item interface{}) {
	s.add(item, false)
}

// Remove removes an item if it's already in the set. Idempotent.
func (s *Set) Remove(item interface{}) {
	s.remove(item)
}

// Contains checks if the set has the given item.
func (s *Set) Contains(item interface{}) bool {
	_, ok := s.m[s.hash(item)]
	return ok
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

// Intersection performs the set intersection of the two sets
// and returns a new third set.
func (s *Set) Intersection(other *Set) *Set {
	result := &Set{F: s.F}
	result.once.Do(result.init)

	for k, v := range s.m {
		if _, ok := other.m[k]; ok {
			result.m[k] = v
		}
	}

	return result
}

// Union performs the set union of the two sets and returns a new third
// set.
func (s *Set) Union(other *Set) *Set {
	result := &Set{F: s.F}
	result.once.Do(result.init)

	for k, v := range s.m {
		result.m[k] = v
	}
	for k, v := range other.m {
		result.m[k] = v
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

func (s *Set) remove(item interface{}) string {
	s.once.Do(s.init)

	code := s.hash(item)
	delete(s.m, code)

	return code
}

func (s *Set) index(item interface{}) int {
	return sort.SearchStrings(s.listCode(), s.hash(item))
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
