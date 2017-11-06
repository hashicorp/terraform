package flatmap

import (
	"strings"
)

// Map is a wrapper around map[string]string that provides some helpers
// above it that assume the map is in the format that flatmap expects
// (the result of Flatten).
//
// All modifying functions such as Delete are done in-place unless
// otherwise noted.
type Map map[string]string

// Contains returns true if the map contains the given key.
func (m Map) Contains(key string) bool {
	for _, k := range m.Keys() {
		if k == key {
			return true
		}
	}

	return false
}

// Delete deletes a key out of the map with the given prefix.
func (m Map) Delete(prefix string) {
	for k := range m {
		match := k == prefix
		if !match {
			if !strings.HasPrefix(k, prefix) {
				continue
			}

			if k[len(prefix):len(prefix)+1] != "." {
				continue
			}
		}

		delete(m, k)
	}
}

// Keys returns all of the top-level keys in this map
func (m Map) Keys() []string {
	ks := make(map[string]struct{})
	for k := range m {
		idx := strings.Index(k, ".")
		if idx == -1 {
			idx = len(k)
		}

		ks[k[:idx]] = struct{}{}
	}

	result := make([]string, 0, len(ks))
	for k := range ks {
		result = append(result, k)
	}

	return result
}

// Merge merges the contents of the other Map into this one.
//
// This merge is smarter than a simple map iteration because it
// will fully replace arrays and other complex structures that
// are present in this map with the other map's. For example, if
// this map has a 3 element "foo" list, and m2 has a 2 element "foo"
// list, then the result will be that m has a 2 element "foo"
// list.
func (m Map) Merge(m2 Map) {
	for _, prefix := range m2.Keys() {
		m.Delete(prefix)

		for k, v := range m2 {
			if strings.HasPrefix(k, prefix) {
				m[k] = v
			}
		}
	}
}
