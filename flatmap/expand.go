package flatmap

import (
	"fmt"
	"regexp"
	"sort"
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

	// Unless we actually have a list of our keys here, we are going to run into
	// trouble with hashed lists (ie: sets). Assemble and sort the list of
	// available keys in this attribute.
	keys := make([]int, 0)
	listElementKey := regexp.MustCompile("^" + prefix + "\\.([0-9]+)")
	for id := range m {
		if match := listElementKey.FindStringSubmatch(id); match != nil {
			index, err := strconv.Atoi(match[1])
			if err != nil {
				panic(err)
			}
			keys = append(keys, index)
		}
	}

	// sort the keys
	sort.Ints(keys)

	// remove duplicates
	n := 0
	for {
		if n >= len(keys)-1 {
			break
		}
		if keys[n+1] == keys[n] {
			keys = append(keys[:n], keys[n+1:]...)
		} else {
			n++
		}
	}

	result := make([]interface{}, num)
	for i := 0; i < int(num); i++ {
		result[i] = Expand(m, fmt.Sprintf("%s.%d", prefix, keys[i]))
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

		// skip the map count value
		if key == "%" {
			continue
		}
		result[key] = Expand(m, k[:len(prefix)+len(key)])
	}

	return result
}
