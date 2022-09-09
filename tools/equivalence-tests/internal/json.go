package internal

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	wildcard = "*"
)

// StripJson will parse a JSON object represented by data, remove all the
// requested fields.
//
// The fields parameter should be a slice of strings, where each string is a
// list of keys seperated by the '.' character. Each key should either be a
// number representing an index in a JSON list, or a string representing an
// actual key in JSON map.
//
// The wildcard character, '*', can also be used to represent all items in a
// JSON list or map.
//
// Have a look at the json_test.go file for some examples.
func StripJson(fields []string, data interface{}) (interface{}, error) {
	for _, field := range fields {
		var err error
		data, err = strip(strings.Split(field, "."), data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func strip(parts []string, current interface{}) (interface{}, error) {
	if current == nil {
		return nil, nil
	}

	if len(parts) == 1 {
		return stripLeaf(parts[0], current)
	}

	return stripNode(parts, current)
}

func stripLeaf(part string, current interface{}) (interface{}, error) {
	switch leaf := current.(type) {
	case map[string]interface{}:
		return stripMapLeaf(part, leaf), nil
	case []interface{}:
		return stripSliceLeaf(part, leaf)
	default:
		return nil, fmt.Errorf("unrecognised json type: %T", leaf)
	}
}

func stripMapLeaf(part string, current map[string]interface{}) map[string]interface{} {
	switch part {
	case wildcard:
		return map[string]interface{}{}
	default:
		delete(current, part)
		return current
	}
}

func stripSliceLeaf(part string, current []interface{}) ([]interface{}, error) {
	switch part {
	case wildcard:
		return []interface{}{}, nil
	default:
		ix, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("must specify an integer when referencing json arrays, instead specified %s", part)
		}
		return append(current[:ix], current[ix+1:]...), nil
	}
}

func stripNode(parts []string, current interface{}) (interface{}, error) {
	switch node := current.(type) {
	case map[string]interface{}:
		return stripMapNode(parts, node)
	case []interface{}:
		return stripSliceNode(parts, node)
	default:
		return nil, fmt.Errorf("unrecognised json type: %T", node)
	}
}

func stripMapNode(parts []string, current map[string]interface{}) (map[string]interface{}, error) {
	switch parts[0] {
	case wildcard:
		ret := map[string]interface{}{}
		for key, value := range current {
			var err error
			if ret[key], err = strip(parts[1:], value); err != nil {
				return nil, err
			}
		}
		return ret, nil
	default:
		var err error
		if current[parts[0]], err = strip(parts[1:], current[parts[0]]); err != nil {
			return nil, err
		}
		return current, nil
	}
}

func stripSliceNode(parts []string, current []interface{}) ([]interface{}, error) {
	switch parts[0] {
	case wildcard:
		var ret []interface{}
		for _, item := range current {
			stripped, err := strip(parts[1:], item)
			if err != nil {
				return nil, err
			}
			ret = append(ret, stripped)
		}
		return ret, nil
	default:
		ix, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("must specify an integer when referencing json arrays, instead specified %s", parts[0])
		}

		if current[ix], err = strip(parts[1:], current[ix]); err != nil {
			return nil, err
		}
		return current, nil
	}
}
