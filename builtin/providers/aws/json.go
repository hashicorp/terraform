package aws

import (
	"encoding/json"
	"fmt"
)

func normalizeJson(jsonString interface{}) string {
	if jsonString == nil || jsonString == "" {
		return ""
	}
	var j interface{}
	err := json.Unmarshal([]byte(jsonString.(string)), &j)
	if err != nil {
		return fmt.Sprintf("Error parsing JSON: %s", err)
	}
	b, err := json.Marshal(j)
	if err != nil {
		panic(err) // unexpected
	}
	return string(b)
}

func normalizePolicyDocument(policyString interface{}) string {
	if policyString == nil || policyString == "" {
		return ""
	}
	var policy IAMPolicyDoc
	err := json.Unmarshal([]byte(policyString.(string)), &policy)
	if err != nil {
		return fmt.Sprintf("Error parsing IAM policy document: %s", err)
	}
	b, err := json.Marshal(policy)
	if err != nil {
		panic(err) // unexpected
	}
	return string(b)
}
