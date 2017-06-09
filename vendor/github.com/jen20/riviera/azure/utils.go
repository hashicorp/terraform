package azure

import (
	"encoding/json"
	"net/http"
)

// String returns a pointer to the input string. This is useful when initializing
// structures.
func String(input string) *string {
	return &input
}

// Int32 returns a pointer to the input int32. This is useful when initializing
// structures.
func Int32(input int32) *int32 {
	return &input
}

// Int64 returns a pointer to the input int64. This is useful when initializing
// structures.
func Int64(input int64) *int64 {
	return &input
}

// Bool returns a pointer to the input bool. This is useful when initializing
// structures.
func Bool(input bool) *bool {
	return &input
}

// isSuccessCode returns true for 200-range numbers which usually denote
// that an HTTP request was successful
func isSuccessCode(statusCode int) bool {
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return true
	}

	return false
}

// unmarshalFlattenPropertiesAndClose returns a map[string]interface{} with the
// "properties" key flattened for use with mapstructure. It closes the Body reader of
// the http.Response passed in.
func unmarshalFlattenPropertiesAndClose(response *http.Response) (map[string]interface{}, error) {
	return unmarshalNested(response, "properties")
}

// unmarshalFlattenErrorAndClose returns a map[string]interface{} with the
// "error" key flattened for use with mapstructure. It closes the Body reader of
// the http.Response passed in.
func unmarshalFlattenErrorAndClose(response *http.Response) (map[string]interface{}, error) {
	return unmarshalNested(response, "error")
}

func unmarshalNested(response *http.Response, key string) (map[string]interface{}, error) {
	defer response.Body.Close()

	var unmarshalled map[string]interface{}
	decoder := json.NewDecoder(response.Body)

	err := decoder.Decode(&unmarshalled)
	if err != nil {
		return nil, err
	}

	if properties, hasProperties := unmarshalled[key]; hasProperties {
		if propertiesMap, ok := properties.(map[string]interface{}); ok {
			for k, v := range propertiesMap {
				unmarshalled[k] = v
			}

			delete(propertiesMap, key)
		}
	}

	return unmarshalled, nil

}
