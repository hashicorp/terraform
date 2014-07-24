package flatmap

import (
	"fmt"
	"strconv"
	"strings"
)

// Expand takes a map and a key (prefix) and expands that value into
// a more complex structure. This is the reverse of the Flatten operation.
func Expand(m map[string]string, key string) interface{} {
	// If the key is exactly a key in the map, just return it
	if v, ok := m[key]; ok {
		if v == "true" {
			return true
		} else if v == "false" {
			return false
		}

		return v
	}

	// Check if the key is an array, and if so, expand the array
	if _, ok := m[key+".#"]; ok {
		return expandArray(m, key)
	}

	// Check if this is a prefix in the map
	prefix := key + "."
	for k, _ := range m {
		if strings.HasPrefix(k, prefix) {
			return expandMap(m, prefix)
		}
	}

	return nil
}

func expandArray(m map[string]string, prefix string) []interface{} {
	num, err := strconv.ParseInt(m[prefix+".#"], 0, 0)
	if err != nil {
		panic(err)
	}

	result := make([]interface{}, num)
	for i := 0; i < int(num); i++ {
		result[i] = Expand(m, fmt.Sprintf("%s.%d", prefix, i))
	}

	return result
}

func expandMap(m map[string]string, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, _ := range m {
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		key := k[len(prefix):]
		idx := strings.Index(key, ".")
		if idx != -1 {
			key = key[:idx]
		}
		if _, ok := result[key]; ok {
			continue
		}

		// It contains a period, so it is a more complex structure
		result[key] = Expand(m, k[:len(prefix)+len(key)])
	}

	return result
}
