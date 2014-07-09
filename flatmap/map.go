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

// Delete deletes a key out of the map with the given prefix.
func (m Map) Delete(prefix string) {
	for k, _ := range m {
		match := k == prefix
		if !match && !strings.HasPrefix(k, prefix) {
			continue
		}
		if k[len(prefix):len(prefix)+1] != "." {
			continue
		}

		delete(m, k)
	}
}
