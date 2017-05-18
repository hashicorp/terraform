package librato

import (
	"encoding/json"
	"fmt"
)

// Encodes a hash into a JSON string
func attributesFlatten(attrs map[string]string) (string, error) {
	byteArray, err := json.Marshal(attrs)
	if err != nil {
		return "", fmt.Errorf("Error encoding to JSON: %s", err)
	}

	return string(byteArray), nil
}

// Takes JSON in a string & decodes into a hash
func attributesExpand(raw string) (map[string]string, error) {
	attrs := make(map[string]string)
	err := json.Unmarshal([]byte(raw), &attrs)
	if err != nil {
		return nil, fmt.Errorf("Error decoding JSON: %s", err)
	}

	return attrs, err
}

func normalizeJSON(jsonString interface{}) string {
	if jsonString == nil || jsonString == "" {
		return ""
	}
	var j interface{}
	err := json.Unmarshal([]byte(jsonString.(string)), &j)
	if err != nil {
		return fmt.Sprintf("Error parsing JSON: %s", err)
	}
	b, _ := json.Marshal(j)
	return string(b[:])
}
