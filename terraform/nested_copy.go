package terraform

import "fmt"

// nestedCopy returns a deep copy of i, provided that all interface{} values
// are string, []string, []interface{}, map[string]string, or
// map[string]interface{}. This is an interim replacement for
// copystructure.Copy where we want to copy simple nested data structures, but
// don't want to risk walking over structs that may contain locks.
func nestedCopy(src interface{}) interface{} {
	switch v := src.(type) {
	case string:
		return v
	case []interface{}:
		var sliceCopy []interface{}
		for _, i := range v {
			sliceCopy = append(sliceCopy, i)
		}
		return sliceCopy
	case []string:
		var sliceCopy []string
		for _, s := range v {
			sliceCopy = append(sliceCopy, s)
		}
		return sliceCopy
	case map[string]interface{}:
		var mapCopy map[string]interface{}
		if v == nil {
			return mapCopy
		}

		mapCopy = make(map[string]interface{})
		for k, i := range v {
			mapCopy[k] = nestedCopy(i)
		}
		return mapCopy
	case map[string]string:
		var mapCopy map[string]string
		if v == nil {
			return mapCopy
		}

		mapCopy = make(map[string]string)
		for k, s := range v {
			mapCopy[k] = s
		}
		return mapCopy
	default:
		panic(fmt.Sprintf("unexpected type %T", src))
	}
}
