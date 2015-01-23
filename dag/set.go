package dag

import (
	"sync"
)

// set is an internal Set data structure that is based on simply using
// pointers as the hash key into a map.
type set struct {
	m    map[interface{}]struct{}
	once sync.Once
}

// Add adds an item to the set
func (s *set) Add(v interface{}) {
	s.once.Do(s.init)
	s.m[v] = struct{}{}
}

// Delete removes an item from the set.
func (s *set) Delete(v interface{}) {
	s.once.Do(s.init)
	delete(s.m, v)
}

// Include returns true/false of whether a value is in the set.
func (s *set) Include(v interface{}) bool {
	s.once.Do(s.init)
	_, ok := s.m[v]
	return ok
}

// Len is the number of items in the set.
func (s *set) Len() int {
	if s == nil {
		return 0
	}

	return len(s.m)
}

// List returns the list of set elements.
func (s *set) List() []interface{} {
	if s == nil {
		return nil
	}

	r := make([]interface{}, 0, len(s.m))
	for k, _ := range s.m {
		r = append(r, k)
	}

	return r
}

func (s *set) init() {
	s.m = make(map[interface{}]struct{})
}
