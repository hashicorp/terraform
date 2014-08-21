package schema

import (
	"sort"
	"sync"
)

// Set is a set data structure that is returned for elements of type
// TypeSet.
type Set struct {
	F   SchemaSetFunc

	m map[int]interface{}
	once sync.Once
}

// Add adds an item to the set if it isn't already in the set.
func (s *Set) Add(item interface{}) {
	s.add(item)
}

// List returns the elements of this set in slice format.
//
// The order of the returned elements is deterministic. Given the same
// set, the order of this will always be the same.
func (s *Set) List() []interface{}{
	result := make([]interface{}, len(s.m))
	for i, k := range s.listCode() {
		result[i] = s.m[k]
	}

	return result
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

func (s *Set) listCode() []int{
	// Sort the hash codes so the order of the list is deterministic
	keys := make([]int, 0, len(s.m))
	for k, _ := range s.m {
		keys = append(keys, k)
	}
	sort.Sort(sort.IntSlice(keys))
	return keys
}
