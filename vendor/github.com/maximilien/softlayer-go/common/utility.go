package common

import (
	"encoding/json"
)

func ValidateJson(s string) (bool, error) {
	var js map[string]interface{}

	err := json.Unmarshal([]byte(s), &js)
	if err != nil {
		return false, err
	}

	return true, nil
}
