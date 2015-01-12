package schema

import (
	"fmt"
	"sort"
	"sync"
)

// Set is a set data structure that is returned for elements of type
// TypeSet.
type Set struct {
	F SchemaSetFunc

	m    map[int]interface{}
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

// Add adds an item to the set if it isn't already in the set.
func (s *Set) Add(item interface{}) {
	s.add(item)
}

// Contains checks if the set has the given item.
func (s *Set) Contains(item interface{}) bool {
	_, ok := s.m[s.F(item)]
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

// Differences performs a set difference of the two sets, returning
// a new third set that has only the elements unique to this set.
func (s *Set) Difference(other *Set) *Set {
	result := &Set{F: s.F}
	result.init()

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
	result.init()

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
	result.init()

	for k, v := range s.m {
		result.m[k] = v
	}
	for k, v := range other.m {
		result.m[k] = v
	}

	return result
}

func (s *Set) GoString() string {
	return fmt.Sprintf("*Set(%#v)", s.m)
}

func (s *Set) init() {
	s.m = make(map[int]interface{})
}

func (s *Set) add(item interface{}) int {
	s.once.Do(s.init)

	code := s.F(item)
	if _, ok := s.m[code]; !ok {
		s.m[code] = item
	}

	return code
}

func (s *Set) index(item interface{}) int {
	return sort.SearchInts(s.listCode(), s.F(item))
}

func (s *Set) listCode() []int {
	// Sort the hash codes so the order of the list is deterministic
	keys := make([]int, 0, len(s.m))
	for k, _ := range s.m {
		keys = append(keys, k)
	}
	sort.Sort(sort.IntSlice(keys))
	return keys
}
