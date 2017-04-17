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

// hashcode returns the hashcode used for set elements.
func hashcode(v interface{}) interface{} {
	if h, ok := v.(Hashable); ok {
		return h.Hashcode()
	}

	return v
}

// Add adds an item to the set
func (s *Set) Add(v interface{}) {
	s.once.Do(s.init)
	s.m[hashcode(v)] = v
}

// Delete removes an item from the set.
func (s *Set) Delete(v interface{}) {
	s.once.Do(s.init)
	delete(s.m, hashcode(v))
}

// Include returns true/false of whether a value is in the set.
func (s *Set) Include(v interface{}) bool {
	s.once.Do(s.init)
	_, ok := s.m[hashcode(v)]
	return ok
}

// Intersection computes the set intersection with other.
func (s *Set) Intersection(other *Set) *Set {
	result := new(Set)
	if s == nil {
		return result
	}
	if other != nil {
		for _, v := range s.m {
			if other.Include(v) {
				result.Add(v)
			}
		}
	}

	return result
}

// Difference returns a set with the elements that s has but
// other doesn't.
func (s *Set) Difference(other *Set) *Set {
	result := new(Set)
	if s != nil {
		for k, v := range s.m {
			var ok bool
			if other != nil {
				_, ok = other.m[k]
			}
			if !ok {
				result.Add(v)
			}
		}
	}

	return result
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

func (s *Set) init() {
	s.m = make(map[interface{}]interface{})
}
