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

func IsHttpErrorCode(errorCode int) bool {
	if errorCode >= 400 {
		return true
	}

	return false
}
