package internal

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestStripJson(t *testing.T) {
	tcs := []struct {
		input    interface{}
		expected interface{}
		fields   []string
	}{
		{
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
			fields:   []string{},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			fields: []string{},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
			},
			fields: []string{
				"list",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{},
			},
			fields: []string{
				"list.*",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": []interface{}{
						"one",
						"two",
					},
					"two": []interface{}{
						"one",
						"two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"two": []interface{}{
						"one",
						"two",
					},
				},
			},
			fields: []string{
				"map.one",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": []interface{}{
						"one",
						"two",
					},
					"two": []interface{}{
						"one",
						"two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{},
			},
			fields: []string{
				"map.*",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": []interface{}{
						"one",
						"two",
					},
					"two": []interface{}{
						"one",
						"two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"one": []interface{}{},
					"two": []interface{}{
						"one",
						"two",
					},
				},
			},
			fields: []string{
				"map.one.*",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"two": "two",
					},
					map[string]interface{}{
						"two": "two",
					},
				},
			},
			fields: []string{
				"map.one",
				"list.*.one",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			expected: map[string]interface{}{
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			fields: []string{
				"map",
			},
		},
		{
			input: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
						"two": "two",
					},
				},
			},
			expected: map[string]interface{}{
				"map": map[string]interface{}{
					"one": "one",
					"two": "two",
				},
				"list": []interface{}{
					map[string]interface{}{
						"two": "two",
					},
					map[string]interface{}{
						"one": "one",
					},
				},
			},
			fields: []string{
				"list.0.one",
				"list.1.two",
			},
		},
	}
	for ix, tc := range tcs {
		t.Run(fmt.Sprintf("%d", ix), func(t *testing.T) {
			actual, err := StripJson(tc.fields, tc.input)
			if err != nil {
				t.Logf("call to StripJson failed unexpectedly: %v", err)
				t.Fail()
				return
			}

			actualStr, err := json.Marshal(actual)
			if err != nil {
				t.Fatalf("could not convert actual into bytes: %v", err)
			}

			expectedStr, err := json.Marshal(tc.expected)
			if err != nil {
				t.Fatalf("could not convert expected into bytes: %v", err)
			}

			if string(actualStr) != string(expectedStr) {
				t.Logf("actual does not equal expected\nexpected:\n\t%s\nactual:\n\t%s\n", string(expectedStr), string(actualStr))
				t.Fail()
			}
		})
	}
}
