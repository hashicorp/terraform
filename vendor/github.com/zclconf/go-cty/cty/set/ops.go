package set

import (
	"sort"
)

// Add inserts the given value into the receiving Set.
//
// This mutates the set in-place. This operation is not thread-safe.
func (s Set) Add(val interface{}) {
	hv := s.rules.Hash(val)
	if _, ok := s.vals[hv]; !ok {
		s.vals[hv] = make([]interface{}, 0, 1)
	}
	bucket := s.vals[hv]

	// See if an equivalent value is already present
	for _, ev := range bucket {
		if s.rules.Equivalent(val, ev) {
			return
		}
	}

	s.vals[hv] = append(bucket, val)
}

// Remove deletes the given value from the receiving set, if indeed it was
// there in the first place. If the value is not present, this is a no-op.
func (s Set) Remove(val interface{}) {
	hv := s.rules.Hash(val)
	bucket, ok := s.vals[hv]
	if !ok {
		return
	}

	for i, ev := range bucket {
		if s.rules.Equivalent(val, ev) {
			newBucket := make([]interface{}, 0, len(bucket)-1)
			newBucket = append(newBucket, bucket[:i]...)
			newBucket = append(newBucket, bucket[i+1:]...)
			if len(newBucket) > 0 {
				s.vals[hv] = newBucket
			} else {
				delete(s.vals, hv)
			}
			return
		}
	}
}

// Has returns true if the given value is in the receiving set, or false if
// it is not.
func (s Set) Has(val interface{}) bool {
	hv := s.rules.Hash(val)
	bucket, ok := s.vals[hv]
	if !ok {
		return false
	}

	for _, ev := range bucket {
		if s.rules.Equivalent(val, ev) {
			return true
		}
	}
	return false
}

// Iterator returns an iterator over values in the set, in an undefined order
// that callers should not depend on.
//
// The pattern for using the returned iterator is:
//
//     it := set.Iterator()
//     for it.Next() {
//         val := it.Value()
//         // ...
//     }
//
// Once an iterator has been created for a set, the set *must not* be mutated
// until the iterator is no longer in use.
func (s Set) Iterator() *Iterator {
	// Sort the bucketIds to ensure that we always traverse in a
	// consistent order.
	bucketIds := make([]int, 0, len(s.vals))
	for id := range s.vals {
		bucketIds = append(bucketIds, id)
	}
	sort.Ints(bucketIds)

	return &Iterator{
		bucketIds: bucketIds,
		vals:      s.vals,
		bucketIdx: -1,
	}
}

// EachValue calls the given callback once for each value in the set, in an
// undefined order that callers should not depend on.
func (s Set) EachValue(cb func(interface{})) {
	it := s.Iterator()
	for it.Next() {
		cb(it.Value())
	}
}

// Values returns a slice of all of the values in the set in no particular
// order. This is just a wrapper around EachValue that accumulates the results
// in a slice for caller convenience.
//
// The returned slice will be nil if there are no values in the set.
func (s Set) Values() []interface{} {
	var ret []interface{}
	s.EachValue(func(v interface{}) {
		ret = append(ret, v)
	})
	return ret
}

// Length returns the number of values in the set.
func (s Set) Length() int {
	var count int
	for _, bucket := range s.vals {
		count = count + len(bucket)
	}
	return count
}

// Union returns a new set that contains all of the members of both the
// receiving set and the given set. Both sets must have the same rules, or
// else this function will panic.
func (s1 Set) Union(s2 Set) Set {
	mustHaveSameRules(s1, s2)
	rs := NewSet(s1.rules)
	s1.EachValue(func(v interface{}) {
		rs.Add(v)
	})
	s2.EachValue(func(v interface{}) {
		rs.Add(v)
	})
	return rs
}

// Intersection returns a new set that contains the values that both the
// receiver and given sets have in common. Both sets must have the same rules,
// or else this function will panic.
func (s1 Set) Intersection(s2 Set) Set {
	mustHaveSameRules(s1, s2)
	rs := NewSet(s1.rules)
	s1.EachValue(func(v interface{}) {
		if s2.Has(v) {
			rs.Add(v)
		}
	})
	return rs
}

// Subtract returns a new set that contains all of the values from the receiver
// that are not also in the given set. Both sets must have the same rules,
// or else this function will panic.
func (s1 Set) Subtract(s2 Set) Set {
	mustHaveSameRules(s1, s2)
	rs := NewSet(s1.rules)
	s1.EachValue(func(v interface{}) {
		if !s2.Has(v) {
			rs.Add(v)
		}
	})
	return rs
}

// SymmetricDifference returns a new set that contains all of the values from
// both the receiver and given sets, except those that both sets have in
// common. Both sets must have the same rules, or else this function will
// panic.
func (s1 Set) SymmetricDifference(s2 Set) Set {
	mustHaveSameRules(s1, s2)
	rs := NewSet(s1.rules)
	s1.EachValue(func(v interface{}) {
		if !s2.Has(v) {
			rs.Add(v)
		}
	})
	s2.EachValue(func(v interface{}) {
		if !s1.Has(v) {
			rs.Add(v)
		}
	})
	return rs
}
