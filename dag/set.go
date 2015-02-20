package dag

import (
	"sync"
)

// Set is a set data structure.
type Set struct {
	m    map[interface{}]interface{}
	once sync.Once
}

// Hashable is the interface used by set to get the hash code of a value.
// If this isn't given, then the value of the item being added to the set
// itself is used as the comparison value.
type Hashable interface {
	Hashcode() interface{}
}

// Add adds an item to the set
func (s *Set) Add(v interface{}) {
	s.once.Do(s.init)
	s.m[s.code(v)] = v
}

// Delete removes an item from the set.
func (s *Set) Delete(v interface{}) {
	s.once.Do(s.init)
	delete(s.m, s.code(v))
}

// Include returns true/false of whether a value is in the set.
func (s *Set) Include(v interface{}) bool {
	s.once.Do(s.init)
	_, ok := s.m[s.code(v)]
	return ok
}

// Len is the number of items in the set.
func (s *Set) Len() int {
	if s == nil {
		return 0
	}

	return len(s.m)
}

// List returns the list of set elements.
func (s *Set) List() []interface{} {
	if s == nil {
		return nil
	}

	r := make([]interface{}, 0, len(s.m))
	for _, v := range s.m {
		r = append(r, v)
	}

	return r
}

func (s *Set) code(v interface{}) interface{} {
	if h, ok := v.(Hashable); ok {
		return h.Hashcode()
	}

	return v
}

func (s *Set) init() {
	s.m = make(map[interface{}]interface{})
}
