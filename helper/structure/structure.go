package structure

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
)

// Takes a value containing JSON string and passes it through
// the JSON parser to normalize it, returns either a parsing
// error or normalized JSON string.
func NormalizeJsonString(jsonString interface{}) (string, error) {
	var j interface{}

	if jsonString == nil || jsonString.(string) == "" {
		return "", nil
	}

	s := jsonString.(string)

	err := json.Unmarshal([]byte(s), &j)
	if err != nil {
		return s, err
	}

	bytes, _ := json.Marshal(j)
	return string(bytes[:]), nil
}

func ExpandJsonFromString(jsonString string) (map[string]interface{}, error) {
	var result map[string]interface{}

	err := json.Unmarshal([]byte(jsonString), &result)

	return result, err
}

func FlattenJsonToString(input map[string]interface{}) (string, error) {

	if len(input) == 0 {
		return "", nil
	}

	result, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func SuppressJsonDiff(k, old, new string, d *schema.ResourceData) bool {
	oldMap, err := ExpandJsonFromString(old)
	if err != nil {
		return false
	}

	newMap, err := ExpandJsonFromString(new)
	if err != nil {
		return false
	}

	return reflect.DeepEqual(oldMap, newMap)
}
