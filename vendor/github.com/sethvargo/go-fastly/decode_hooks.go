package fastly

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

// mapToHTTPHeaderHookFunc returns a function that converts maps into an
// http.Header value.
func mapToHTTPHeaderHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}
		if t != reflect.TypeOf(new(http.Header)) {
			return data, nil
		}

		typed, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot convert %T to http.Header", data)
		}

		n := map[string][]string{}
		for k, v := range typed {
			switch v.(type) {
			case string:
				n[k] = []string{v.(string)}
			case []string:
				n[k] = v.([]string)
			default:
				return nil, fmt.Errorf("cannot convert %T to http.Header", v)
			}
		}

		return n, nil
	}
}

// stringToTimeHookFunc returns a function that converts strings to a time.Time
// value.
func stringToTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(time.Now()) {
			return data, nil
		}

		// Convert it by parsing
		return time.Parse(time.RFC3339, data.(string))
	}
}
