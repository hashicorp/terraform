package flatmap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hil"
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
	if v, ok := m[key+".#"]; ok {
		// If the count of the key is unknown, then just put the unknown
		// value in the value itself. This will be detected by Terraform
		// core later.
		if v == hil.UnknownValue {
			return v
		}

		return expandArray(m, key)
	}

	// Check if this is a prefix in the map
	prefix := key + "."
	for k := range m {
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

	// If the number of elements in this array is 0, then return an
	// empty slice as there is nothing to expand. Trying to expand it
	// anyway could lead to crashes as any child maps, arrays or sets
	// that no longer exist are still shown as empty with a count of 0.
	if num == 0 {
		return []interface{}{}
	}

	// The Schema "Set" type stores its values in an array format, but
	// using numeric hash values instead of ordinal keys. Take the set
	// of keys regardless of value, and expand them in numeric order.
	// See GH-11042 for more details.
	keySet := map[int]bool{}
	computed := map[string]bool{}
	for k := range m {
		if !strings.HasPrefix(k, prefix+".") {
			continue
		}

		key := k[len(prefix)+1:]
		idx := strings.Index(key, ".")
		if idx != -1 {
			key = key[:idx]
		}

		// skip the count value
		if key == "#" {
			continue
		}

		// strip the computed flag if there is one
		if strings.HasPrefix(key, "~") {
			key = key[1:]
			computed[key] = true
		}

		k, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		keySet[int(k)] = true
	}

	keysList := make([]int, 0, num)
	for key := range keySet {
		keysList = append(keysList, key)
	}
	sort.Ints(keysList)

	result := make([]interface{}, num)
	for i, key := range keysList {
		keyString := strconv.Itoa(key)
		if computed[keyString] {
			keyString = "~" + keyString
		}
		result[i] = Expand(m, fmt.Sprintf("%s.%s", prefix, keyString))
	}

	return result
}

func expandMap(m map[string]string, prefix string) map[string]interface{} {
	// Submaps may not have a '%' key, so we can't count on this value being
	// here. If we don't have a count, just proceed as if we have have a map.
	if count, ok := m[prefix+"%"]; ok && count == "0" {
		return map[string]interface{}{}
	}

	result := make(map[string]interface{})
	for k := range m {
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
