package nxrm

import "fmt"

func mismatchError(cfg map[string]interface{}, key string, got interface{}) string {
	return fmt.Sprintf("expected: %s got: %s", cfg[key], got.(string))
}

func InitTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"username":  "testymctestface",
		"password":  "mybigsecret",
		"url":       "http://localhost:8081/repository/tf-backend",
		"subpath":   "this/here",
		"stateName": "demo.tfstate",
		"timeout":   30,
	}
}
